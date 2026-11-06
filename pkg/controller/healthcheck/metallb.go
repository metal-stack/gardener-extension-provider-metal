package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// metallbNamespace is the namespace all metallb components are deployed to.
	metallbNamespace = "metallb-system"

	// configurationStateNameController is the name of the configuration state reported by the metallb controller.
	configurationStateNameController = "controller"
	// configurationStateNameSpeakerPrefix is the prefix of the configuration states reported by the metallb speakers.
	// There is one per node, suffixed with the node name.
	configurationStateNameSpeakerPrefix = "speaker-"

	// configurationResultValid is the result reported by a component which validated its configuration successfully.
	configurationResultValid = "Valid"
	// configurationResultInvalid is the result reported by a component which failed to validate its configuration.
	configurationResultInvalid = "Invalid"

	// frrK8sResultSuccess is the result reported by frr-k8s once the configuration conversion and reload succeeded.
	frrK8sResultSuccess = "success"

	// progressingThreshold is the duration after which components which did not report a healthy state yet are considered unhealthy.
	progressingThreshold = 10 * time.Minute

	// legacyHealthConfigMapName is the name of the configmap the deprecated metallb health sidecar wrote the result of
	// its configuration validation into. It is only evaluated for shoots which still run the metallb deployment which
	// does not report its health through `ConfigurationState` objects.
	legacyHealthConfigMapName = "health"
	// legacyHealthConfigMapConfigLoadedKey is the key of the health sidecar configmap which reports whether the metallb
	// components loaded their configuration.
	legacyHealthConfigMapConfigLoadedKey = "configLoaded"
	// legacyHealthConfigMapConfigStaleKey is the key of the health sidecar configmap which reports whether the
	// configuration of the metallb components is stale or erroneous.
	legacyHealthConfigMapConfigStaleKey = "configStale"
)

var (
	// configurationStateGVK is the group version kind of the configuration states reported by the metallb components.
	configurationStateGVK     = schema.GroupVersionKind{Group: "metallb.io", Version: "v1beta1", Kind: "ConfigurationState"}
	configurationStateListGVK = configurationStateGVK.GroupVersion().WithKind("ConfigurationStateList")
	// frrNodeStateGVK is the group version kind of the node states reported by frr-k8s.
	frrNodeStateGVK     = schema.GroupVersionKind{Group: "frrk8s.metallb.io", Version: "v1beta1", Kind: "FRRNodeState"}
	frrNodeStateListGVK = frrNodeStateGVK.GroupVersion().WithKind("FRRNodeStateList")
)

// MetalLBHealthChecker contains all the information for the MetalLB HealthCheck
type MetalLBHealthChecker struct {
	logger      logr.Logger
	shootClient client.Client
}

// CheckMetalLB is a healthCheck function to check MetalLBs
func CheckMetalLB() healthcheck.HealthCheck {
	return &MetalLBHealthChecker{}
}

// InjectTargetClient injects the shoot client
func (healthChecker *MetalLBHealthChecker) InjectTargetClient(targetClient client.Client) {
	healthChecker.shootClient = targetClient
}

// SetLoggerSuffix injects the logger
func (healthChecker *MetalLBHealthChecker) SetLoggerSuffix(provider, extension string) {
	healthChecker.logger = log.Log.WithName(fmt.Sprintf("%s-%s-healthcheck-metallb", provider, extension))
}

// Check executes the health check.
//
// MetalLB components report the result of their configuration validation through `ConfigurationState` objects, one for
// the controller and one per node for the speaker. The BGP configuration is applied by frr-k8s, which reports the
// conversion and reload results through `FRRNodeState` objects, one per node. All of them have to be healthy, otherwise
// a stale or invalid configuration might interrupt the workload traffic of announced services.
//
// Shoots which were not yet reconciled with the metallb deployment shipped by this extension version still run the old
// deployment, which reported its health through the configmap written by the deprecated metallb health sidecar. Those
// shoots do not ship the `configurationstates.metallb.io` CRD and hence cannot report a state at all, so their health is
// evaluated with the health sidecar configmap instead, see checkLegacyHealth.
func (healthChecker *MetalLBHealthChecker) Check(ctx context.Context, _ types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	configurationStates, err := healthChecker.configurationStates(ctx)
	if err != nil {
		if !configurationStateUnavailable(err) {
			return nil, fmt.Errorf("unable to retrieve metallb configuration states: %w", err)
		}

		return healthChecker.checkLegacyHealth(ctx, err)
	}

	// The metallb components are able to report their configuration states, hence the shoot was reconciled with the
	// metallb deployment shipped by this extension version. Its components do not write the configmap of the deprecated
	// metallb health sidecar anymore, which is not evaluated as soon as the components report their health, so the
	// configmap is obsolete and can be removed.
	healthChecker.removeLegacyHealthConfigMap(ctx)

	return healthChecker.checkMetalLBComponents(ctx, configurationStates)
}

// checkLegacyHealth checks the health of a shoot which still runs the metallb deployment which does not report the
// result of its configuration validation through `ConfigurationState` objects. The health of such a shoot is derived
// from the configmap written by the deprecated metallb health sidecar, which was the health check before.
//
// reason is the error the metallb components reported when they were asked for their configuration state. It is logged
// to make the shoots which still run the old metallb deployment observable.
//
// TODO(metal-stack): Remove this function as soon as all shoots were reconciled with the metallb deployment which
// reports its health through `ConfigurationState` objects, because only then the deprecated metallb health sidecar and
// the configmap it writes are gone from all shoots.
func (healthChecker *MetalLBHealthChecker) checkLegacyHealth(ctx context.Context, reason error) (*healthcheck.SingleCheckResult, error) {
	// listing the configuration states fails if the shoot does not ship the `configurationstates.metallb.io` CRD, which
	// is the case for the shoots which still run the old metallb deployment. Those shoots never report a configuration
	// state, so the only way to check their health is the configmap written by the deprecated metallb health sidecar.
	// Shoots which ship the CRD are never checked with this configmap, a stale configmap of a reconciled shoot can
	// therefore not hide a problem which the metallb components reported.
	healthChecker.logger.V(1).Info("metallb components cannot report a health state, checking the deprecated metallb health sidecar", "reason", reason.Error())

	legacyResult, legacyErr := healthChecker.checkLegacyHealthConfigMap(ctx)
	if legacyErr != nil {
		// the metallb components cannot report a state and the health sidecar is gone or could not be evaluated. For the
		// shoots which still run the old deployment the health sidecar is the only source of truth, hence its error is
		// reported, because it describes best why the health of the shoot cannot be determined.
		healthChecker.logger.Error(legacyErr, "unable to evaluate the configmap of the deprecated metallb health sidecar")
		return nil, legacyErr
	}

	return legacyResult, nil
}

// checkMetalLBComponents checks the health of the metallb components and of the frr-k8s daemon which applies the BGP
// configuration to the nodes.
func (healthChecker *MetalLBHealthChecker) checkMetalLBComponents(ctx context.Context, configurationStates map[string]configurationState) (*healthcheck.SingleCheckResult, error) {
	invalid, pending := checkConfigurationStates(configurationStates)
	if len(invalid) > 0 {
		detail := fmt.Sprintf("metallb has an invalid configuration: %s", strings.Join(invalid, "; "))
		healthChecker.logger.Error(errors.New(detail), "Health check failed")
		return &healthcheck.SingleCheckResult{Status: gardencorev1beta1.ConditionFalse, Detail: detail}, nil
	}

	frrNodeStates, err := healthChecker.frrNodeStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve frr-k8s node states: %w", err)
	}

	failed, frrPending := checkFRRNodeStates(frrNodeStates, speakerNodes(configurationStates))
	if len(failed) > 0 {
		detail := fmt.Sprintf("frr-k8s failed to apply the BGP configuration: %s", strings.Join(failed, "; "))
		healthChecker.logger.Error(errors.New(detail), "Health check failed")
		return &healthcheck.SingleCheckResult{Status: gardencorev1beta1.ConditionFalse, Detail: detail}, nil
	}

	if pending = append(pending, frrPending...); len(pending) > 0 {
		threshold := progressingThreshold
		detail := fmt.Sprintf("metallb components did not report a health state yet: %s", strings.Join(pending, "; "))
		healthChecker.logger.Info("Health check progressing", "detail", detail)
		return &healthcheck.SingleCheckResult{
			Status:               gardencorev1beta1.ConditionProgressing,
			Detail:               detail,
			ProgressingThreshold: &threshold,
		}, nil
	}

	return &healthcheck.SingleCheckResult{Status: gardencorev1beta1.ConditionTrue}, nil
}

// configurationStates returns the configuration states of all metallb components, keyed by their name.
func (healthChecker *MetalLBHealthChecker) configurationStates(ctx context.Context) (map[string]configurationState, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(configurationStateListGVK)
	if err := healthChecker.shootClient.List(ctx, list, client.InNamespace(metallbNamespace)); err != nil {
		return nil, err
	}
	return toConfigurationStates(list), nil
}

// frrNodeStates returns the frr-k8s node states of all nodes, keyed by the node name.
func (healthChecker *MetalLBHealthChecker) frrNodeStates(ctx context.Context) (map[string]frrNodeState, error) {
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(frrNodeStateListGVK)
	if err := healthChecker.shootClient.List(ctx, list); err != nil {
		return nil, err
	}
	return toFRRNodeStates(list), nil
}

// configurationState holds the configuration validation result of a single metallb component.
type configurationState struct {
	result       string
	errorSummary string
}

// frrNodeState holds the configuration application result of frr-k8s on a single node.
type frrNodeState struct {
	lastConversionResult string
	lastReloadResult     string
}

func toConfigurationStates(list *unstructured.UnstructuredList) map[string]configurationState {
	states := make(map[string]configurationState, len(list.Items))
	for _, item := range list.Items {
		result, _, _ := unstructured.NestedString(item.Object, "status", "result")
		errorSummary, _, _ := unstructured.NestedString(item.Object, "status", "errorSummary")
		states[item.GetName()] = configurationState{result: result, errorSummary: errorSummary}
	}
	return states
}

func toFRRNodeStates(list *unstructured.UnstructuredList) map[string]frrNodeState {
	states := make(map[string]frrNodeState, len(list.Items))
	for _, item := range list.Items {
		conversion, _, _ := unstructured.NestedString(item.Object, "status", "lastConversionResult")
		reload, _, _ := unstructured.NestedString(item.Object, "status", "lastReloadResult")
		states[item.GetName()] = frrNodeState{lastConversionResult: conversion, lastReloadResult: reload}
	}
	return states
}

// checkConfigurationStates evaluates the configuration states reported by the metallb controller and the speakers.
// It returns the states which are invalid, as well as those which did not report a result yet.
func checkConfigurationStates(states map[string]configurationState) (invalid, pending []string) {
	names := make([]string, 0, len(states))
	for name := range states {
		names = append(names, name)
	}
	slices.Sort(names)

	var controllerReported, speakerReported bool
	for _, name := range names {
		state := states[name]

		switch {
		case name == configurationStateNameController:
			controllerReported = true
		case strings.HasPrefix(name, configurationStateNameSpeakerPrefix):
			speakerReported = true
		default:
			continue
		}

		switch state.result {
		case configurationResultValid:
			// nothing to do
		case configurationResultInvalid:
			invalid = append(invalid, describeConfigurationState(name, state))
		default:
			pending = append(pending, fmt.Sprintf("%q did not report a configuration result yet", name))
		}
	}

	if !controllerReported {
		pending = append(pending, "the metallb controller did not report a configuration state yet")
	}
	if !speakerReported {
		pending = append(pending, "no metallb speaker reported a configuration state yet")
	}

	return invalid, pending
}

// checkFRRNodeStates evaluates the frr-k8s node states of the given nodes.
// It returns the nodes on which frr-k8s failed to apply the configuration, as well as those which did not report a result yet.
func checkFRRNodeStates(states map[string]frrNodeState, nodes []string) (failed, pending []string) {
	for _, node := range nodes {
		state, ok := states[node]
		if !ok {
			pending = append(pending, fmt.Sprintf("frr-k8s did not report a node state for node %q yet", node))
			continue
		}
		if state.lastConversionResult != frrK8sResultSuccess || state.lastReloadResult != frrK8sResultSuccess {
			failed = append(failed, fmt.Sprintf("frr-k8s on node %q could not apply the configuration (conversion: %q, reload: %q)", node, state.lastConversionResult, state.lastReloadResult))
		}
	}
	return failed, pending
}

// speakerNodes returns the nodes on which a metallb speaker reported a configuration state.
func speakerNodes(states map[string]configurationState) []string {
	var nodes []string
	for name := range states {
		if node, ok := strings.CutPrefix(name, configurationStateNameSpeakerPrefix); ok {
			nodes = append(nodes, node)
		}
	}
	slices.Sort(nodes)
	return nodes
}

func describeConfigurationState(name string, state configurationState) string {
	if state.errorSummary == "" {
		return fmt.Sprintf("%q reports result %q", name, state.result)
	}
	return fmt.Sprintf("%q reports result %q: %s", name, state.result, state.errorSummary)
}

// configurationStateUnavailable returns true if the metallb components cannot report a configuration state. This is the
// case if the shoot does not ship the `configurationstates.metallb.io` CRD, which is true for all shoots which still run
// the metallb deployment which was replaced by the one shipped with this extension version. Depending on whether the
// shoot client resolves the resource through the rest mapper or asks the api server directly, this is reported as a
// missing kind or as a not found error.
func configurationStateUnavailable(err error) bool {
	return meta.IsNoMatchError(err) || apierrors.IsNotFound(err)
}

// checkLegacyHealthConfigMap evaluates the configmap written by the deprecated metallb health sidecar. It is evaluated
// as long as a shoot was not yet reconciled with the metallb deployment which reports its health through
// `ConfigurationState` objects, because the health sidecar was removed from the chart together with the configmap it
// wrote.
//
// TODO(metal-stack): Remove this function as soon as all shoots were reconciled, together with checkLegacyHealth.
func (healthChecker *MetalLBHealthChecker) checkLegacyHealthConfigMap(ctx context.Context) (*healthcheck.SingleCheckResult, error) {
	configMap := &corev1.ConfigMap{}
	if err := healthChecker.shootClient.Get(ctx, client.ObjectKey{Namespace: metallbNamespace, Name: legacyHealthConfigMapName}, configMap); err != nil {
		return nil, fmt.Errorf("unable to retrieve the configmap written by the deprecated metallb health sidecar (%s/%s): %w", metallbNamespace, legacyHealthConfigMapName, err)
	}

	if err := legacyHealth(configMap); err != nil {
		healthChecker.logger.Error(err, "Health check failed", "configmap", fmt.Sprintf("%s/%s", metallbNamespace, legacyHealthConfigMapName))
		return &healthcheck.SingleCheckResult{Status: gardencorev1beta1.ConditionFalse, Detail: err.Error()}, nil
	}

	healthChecker.logger.Info("metallb is still healthy according to the deprecated metallb health sidecar, waiting for the shoot to be reconciled")
	return &healthcheck.SingleCheckResult{Status: gardencorev1beta1.ConditionTrue}, nil
}

// legacyHealth returns the reason why the metallb components are considered unhealthy according to the configmap written
// by the deprecated metallb health sidecar, or nil if the health sidecar reported a healthy configuration. This is the
// health check which was used before the metallb components reported their configuration state themselves.
func legacyHealth(configMap *corev1.ConfigMap) error {
	configLoaded, err := strconv.ParseBool(configMap.Data[legacyHealthConfigMapConfigLoadedKey])
	if err != nil {
		return fmt.Errorf("unable to parse %q of the deprecated metallb health configmap: %w", legacyHealthConfigMapConfigLoadedKey, err)
	}
	if !configLoaded {
		return errors.New("the metallb configuration is not loaded yet")
	}

	configStale, err := strconv.ParseBool(configMap.Data[legacyHealthConfigMapConfigStaleKey])
	if err != nil {
		return fmt.Errorf("unable to parse %q of the deprecated metallb health configmap: %w", legacyHealthConfigMapConfigStaleKey, err)
	}
	if configStale {
		return errors.New("the metallb configuration is stale / erroneous, the next speaker reload may interrupt workload traffic")
	}

	return nil
}

// removeLegacyHealthConfigMap removes the configmap written by the deprecated metallb health sidecar from shoots which
// were reconciled with the metallb deployment which reports its health through `ConfigurationState` objects. The health
// sidecar does not exist anymore on those shoots, so the configmap is a leftover which is not evaluated anymore, see
// checkLegacyHealth.
//
// The removal is best effort: it must not fail the health check of the shoot, because the configmap is not taken into
// account anymore as soon as the metallb components report their health. If it cannot be removed it is retried with the
// next health check.
//
// TODO(metal-stack): Remove this function as soon as all shoots were reconciled, together with checkLegacyHealth.
func (healthChecker *MetalLBHealthChecker) removeLegacyHealthConfigMap(ctx context.Context) {
	configMap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: legacyHealthConfigMapName, Namespace: metallbNamespace}}
	if err := healthChecker.shootClient.Delete(ctx, configMap); err != nil {
		if !apierrors.IsNotFound(err) {
			healthChecker.logger.Error(err, "unable to remove the obsolete configmap of the deprecated metallb health sidecar", "configmap", fmt.Sprintf("%s/%s", configMap.Namespace, configMap.Name))
		}

		return
	}

	healthChecker.logger.Info("removed the obsolete configmap of the deprecated metallb health sidecar", "configmap", fmt.Sprintf("%s/%s", configMap.Namespace, configMap.Name))
}

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *MetalLBHealthChecker) DeepCopy() healthcheck.HealthCheck {
	copy := *healthChecker
	return &copy
}
