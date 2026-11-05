package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/go-logr/logr"
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
func (healthChecker *MetalLBHealthChecker) Check(ctx context.Context, _ types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	configurationStates, err := healthChecker.configurationStates(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve metallb configuration states: %w", err)
	}

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

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *MetalLBHealthChecker) DeepCopy() healthcheck.HealthCheck {
	copy := *healthChecker
	return &copy
}
