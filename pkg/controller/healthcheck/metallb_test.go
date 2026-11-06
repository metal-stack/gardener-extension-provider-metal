package healthcheck

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_checkConfigurationStates(t *testing.T) {
	tests := []struct {
		name        string
		states      map[string]configurationState
		wantInvalid []string
		wantPending []string
	}{
		{
			name: "all valid",
			states: map[string]configurationState{
				"controller":    {result: "Valid"},
				"speaker-node1": {result: "Valid"},
			},
		},
		{
			name: "invalid controller configuration",
			states: map[string]configurationState{
				"controller":    {result: "Invalid", errorSummary: "duplicate ip address"},
				"speaker-node1": {result: "Valid"},
			},
			wantInvalid: []string{`"controller" reports result "Invalid": duplicate ip address`},
		},
		{
			name: "invalid speaker configuration",
			states: map[string]configurationState{
				"controller":    {result: "Valid"},
				"speaker-node1": {result: "Invalid"},
				"speaker-node2": {result: "Valid"},
			},
			wantInvalid: []string{`"speaker-node1" reports result "Invalid"`},
		},
		{
			name: "controller did not report yet",
			states: map[string]configurationState{
				"speaker-node1": {result: "Valid"},
			},
			wantPending: []string{"the metallb controller did not report a configuration state yet"},
		},
		{
			name: "no speaker did report yet",
			states: map[string]configurationState{
				"controller": {result: "Valid"},
			},
			wantPending: []string{"no metallb speaker reported a configuration state yet"},
		},
		{
			name: "unknown result is pending",
			states: map[string]configurationState{
				"controller":    {result: "Valid"},
				"speaker-node1": {result: "Unknown"},
			},
			wantPending: []string{`"speaker-node1" did not report a configuration result yet`},
		},
		{
			name: "unrelated objects are ignored",
			states: map[string]configurationState{
				"controller":     {result: "Valid"},
				"speaker-node1":  {result: "Valid"},
				"something-else": {result: "Invalid"},
			},
		},
		{
			name:        "nothing reported yet",
			states:      map[string]configurationState{},
			wantPending: []string{"the metallb controller did not report a configuration state yet", "no metallb speaker reported a configuration state yet"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invalid, pending := checkConfigurationStates(tt.states)
			if !reflect.DeepEqual(invalid, tt.wantInvalid) {
				t.Errorf("invalid = %v, want %v", invalid, tt.wantInvalid)
			}
			if !reflect.DeepEqual(pending, tt.wantPending) {
				t.Errorf("pending = %v, want %v", pending, tt.wantPending)
			}
		})
	}
}

func Test_checkFRRNodeStates(t *testing.T) {
	success := frrNodeState{lastConversionResult: "success", lastReloadResult: "success"}

	tests := []struct {
		name       string
		states     map[string]frrNodeState
		nodes      []string
		wantFailed []string
		wantPend   []string
	}{
		{
			name:   "all nodes applied the configuration",
			states: map[string]frrNodeState{"node1": success, "node2": success},
			nodes:  []string{"node1", "node2"},
		},
		{
			name:     "node did not report yet",
			states:   map[string]frrNodeState{"node1": success},
			nodes:    []string{"node1", "node2"},
			wantPend: []string{`frr-k8s did not report a node state for node "node2" yet`},
		},
		{
			name:       "conversion failed",
			states:     map[string]frrNodeState{"node1": {lastConversionResult: "failed: unknown field", lastReloadResult: "success"}},
			nodes:      []string{"node1"},
			wantFailed: []string{`frr-k8s on node "node1" could not apply the configuration (conversion: "failed: unknown field", reload: "success")`},
		},
		{
			name:       "reload failed",
			states:     map[string]frrNodeState{"node1": {lastConversionResult: "success", lastReloadResult: ""}},
			nodes:      []string{"node1"},
			wantFailed: []string{`frr-k8s on node "node1" could not apply the configuration (conversion: "success", reload: "")`},
		},
		{
			name:   "nodes without a speaker are ignored",
			states: map[string]frrNodeState{"node1": {}},
			nodes:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, pending := checkFRRNodeStates(tt.states, tt.nodes)
			if !reflect.DeepEqual(failed, tt.wantFailed) {
				t.Errorf("failed = %v, want %v", failed, tt.wantFailed)
			}
			if !reflect.DeepEqual(pending, tt.wantPend) {
				t.Errorf("pending = %v, want %v", pending, tt.wantPend)
			}
		})
	}
}

func Test_speakerNodes(t *testing.T) {
	states := map[string]configurationState{
		"controller":     {result: "Valid"},
		"speaker-node2":  {result: "Valid"},
		"speaker-node1":  {result: "Valid"},
		"something-else": {result: "Valid"},
	}

	want := []string{"node1", "node2"}
	if got := speakerNodes(states); !reflect.DeepEqual(got, want) {
		t.Errorf("speakerNodes() = %v, want %v", got, want)
	}
}

func Test_toConfigurationStates(t *testing.T) {
	list := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		*configurationStateObject("controller", "Valid", ""),
		*configurationStateObject("speaker-node1", "Invalid", "duplicate ip address"),
	}}

	want := map[string]configurationState{
		"controller":    {result: "Valid"},
		"speaker-node1": {result: "Invalid", errorSummary: "duplicate ip address"},
	}
	if got := toConfigurationStates(list); !reflect.DeepEqual(got, want) {
		t.Errorf("toConfigurationStates() = %v, want %v", got, want)
	}
}

func Test_toFRRNodeStates(t *testing.T) {
	list := &unstructured.UnstructuredList{Items: []unstructured.Unstructured{
		*frrNodeStateObject("node1", "success", "success"),
		*frrNodeStateObject("node2", "failed: unknown field", "success"),
	}}

	want := map[string]frrNodeState{
		"node1": {lastConversionResult: "success", lastReloadResult: "success"},
		"node2": {lastConversionResult: "failed: unknown field", lastReloadResult: "success"},
	}
	if got := toFRRNodeStates(list); !reflect.DeepEqual(got, want) {
		t.Errorf("toFRRNodeStates() = %v, want %v", got, want)
	}
}

func Test_Check(t *testing.T) {
	tests := []struct {
		name        string
		objects     []client.Object
		oldMetallb  bool
		wantStatus  gardencorev1beta1.ConditionStatus
		wantDetail  string
		wantPending bool
		wantErr     string
	}{
		{
			name: "all components healthy",
			objects: []client.Object{
				configurationStateObject("controller", "Valid", ""),
				configurationStateObject("speaker-node1", "Valid", ""),
				frrNodeStateObject("node1", "success", "success"),
			},
			wantStatus: gardencorev1beta1.ConditionTrue,
		},
		{
			name: "invalid metallb configuration",
			objects: []client.Object{
				configurationStateObject("controller", "Valid", ""),
				configurationStateObject("speaker-node1", "Invalid", "duplicate ip address"),
				frrNodeStateObject("node1", "success", "success"),
			},
			wantStatus: gardencorev1beta1.ConditionFalse,
			wantDetail: "duplicate ip address",
		},
		{
			name: "frr-k8s could not apply the configuration",
			objects: []client.Object{
				configurationStateObject("controller", "Valid", ""),
				configurationStateObject("speaker-node1", "Valid", ""),
				frrNodeStateObject("node1", "success", "failed: reload timeout"),
			},
			wantStatus: gardencorev1beta1.ConditionFalse,
			wantDetail: "failed: reload timeout",
		},
		{
			name: "frr-k8s did not report yet",
			objects: []client.Object{
				configurationStateObject("controller", "Valid", ""),
				configurationStateObject("speaker-node1", "Valid", ""),
			},
			wantStatus:  gardencorev1beta1.ConditionProgressing,
			wantDetail:  `did not report a node state for node "node1"`,
			wantPending: true,
		},
		{
			name: "shoot still runs the old metallb deployment",
			objects: []client.Object{
				legacyHealthConfigMapObject("true", "false"),
			},
			oldMetallb: true,
			wantStatus: gardencorev1beta1.ConditionTrue,
		},
		{
			name: "shoot still runs the old metallb deployment with a configuration which is not loaded yet",
			objects: []client.Object{
				legacyHealthConfigMapObject("false", "false"),
			},
			oldMetallb: true,
			wantStatus: gardencorev1beta1.ConditionFalse,
			wantDetail: "not loaded yet",
		},
		{
			name: "shoot still runs the old metallb deployment with a stale configuration",
			objects: []client.Object{
				legacyHealthConfigMapObject("true", "true"),
			},
			oldMetallb: true,
			wantStatus: gardencorev1beta1.ConditionFalse,
			wantDetail: "stale",
		},
		{
			name:       "shoot still runs the old metallb deployment whose health sidecar did not write a configmap yet",
			oldMetallb: true,
			wantErr:    `configmaps "health" not found`,
		},
		{
			name: "an existing health sidecar configmap does not hide an invalid configuration",
			objects: []client.Object{
				configurationStateObject("controller", "Valid", ""),
				configurationStateObject("speaker-node1", "Invalid", "duplicate ip address"),
				frrNodeStateObject("node1", "success", "success"),
				legacyHealthConfigMapObject("true", "false"),
			},
			wantStatus: gardencorev1beta1.ConditionFalse,
			wantDetail: "duplicate ip address",
		},
		{
			name: "the obsolete health sidecar configmap is removed from a shoot which reports its health",
			objects: []client.Object{
				configurationStateObject("controller", "Valid", ""),
				configurationStateObject("speaker-node1", "Valid", ""),
				frrNodeStateObject("node1", "success", "success"),
				// the configmap reports a problem, but it is not evaluated anymore as soon as the metallb components
				// report their health.
				legacyHealthConfigMapObject("true", "true"),
			},
			wantStatus: gardencorev1beta1.ConditionTrue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shootClient := newFakeClient(t, tt.objects...)
			if tt.oldMetallb {
				// shoots which were not yet reconciled with the new metallb deployment do not ship the
				// `configurationstates.metallb.io` CRD, hence listing the configuration states fails.
				shootClient = &clientWithoutMetallbCRDs{Client: shootClient}
			}

			healthChecker := &MetalLBHealthChecker{shootClient: shootClient}
			healthChecker.SetLoggerSuffix("provider", "metal")

			result, err := healthChecker.Check(context.Background(), types.NamespacedName{})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Check() error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Check() returned an unexpected error: %v", err)
			}
			if result.Status != tt.wantStatus {
				t.Errorf("Check() status = %v, want %v (detail: %s)", result.Status, tt.wantStatus, result.Detail)
			}
			if tt.wantDetail != "" && !strings.Contains(result.Detail, tt.wantDetail) {
				t.Errorf("Check() detail = %q, want it to contain %q", result.Detail, tt.wantDetail)
			}
			if tt.wantPending && result.ProgressingThreshold == nil {
				t.Errorf("Check() did not set a progressing threshold")
			}

			// the configmap of the deprecated health sidecar is only needed as long as the shoot cannot report its
			// health through `ConfigurationState` objects, hence it has to be removed on all other shoots.
			legacyConfigMap := &corev1.ConfigMap{}
			err = shootClient.Get(context.Background(), client.ObjectKey{Namespace: metallbNamespace, Name: legacyHealthConfigMapName}, legacyConfigMap)
			switch {
			case !tt.oldMetallb && !apierrors.IsNotFound(err):
				t.Errorf("Check() did not remove the obsolete configmap of the deprecated metallb health sidecar: %v", err)
			case tt.oldMetallb && err != nil:
				t.Errorf("Check() removed the configmap the health of the shoot is derived from: %v", err)
			}
		})
	}
}

func newFakeClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	s := runtime.NewScheme()
	s.AddKnownTypeWithName(configurationStateGVK, &unstructured.Unstructured{})
	s.AddKnownTypeWithName(configurationStateListGVK, &unstructured.UnstructuredList{})
	s.AddKnownTypeWithName(frrNodeStateGVK, &unstructured.Unstructured{})
	s.AddKnownTypeWithName(frrNodeStateListGVK, &unstructured.UnstructuredList{})
	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("unable to add the core types to the scheme: %v", err)
	}

	return fake.NewClientBuilder().WithScheme(s).WithObjects(objects...).Build()
}

func configurationStateObject(name, result, errorSummary string) *unstructured.Unstructured {
	status := map[string]any{"result": result}
	if errorSummary != "" {
		status["errorSummary"] = errorSummary
	}

	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "metallb.io/v1beta1",
		"kind":       "ConfigurationState",
		"metadata": map[string]any{
			"name":      name,
			"namespace": "metallb-system",
		},
		"status": status,
	}}
}

func frrNodeStateObject(name, lastConversionResult, lastReloadResult string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "frrk8s.metallb.io/v1beta1",
		"kind":       "FRRNodeState",
		"metadata": map[string]any{
			"name": name,
		},
		"status": map[string]any{
			"lastConversionResult": lastConversionResult,
			"lastReloadResult":     lastReloadResult,
		},
	}}
}

// clientWithoutMetallbCRDs simulates the shoot client of a shoot which still runs the metallb deployment which does not
// ship the `configurationstates.metallb.io` and `frrk8s.metallb.io` CRDs. For such a shoot listing the configuration
// states fails with the missing kind error the rest mapper of the shoot client returns. Any other error can be injected
// through err.
type clientWithoutMetallbCRDs struct {
	client.Client
	err error
}

func (c *clientWithoutMetallbCRDs) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	switch list.GetObjectKind().GroupVersionKind().Group {
	case "metallb.io", "frrk8s.metallb.io":
		if c.err != nil {
			return c.err
		}
		return &meta.NoKindMatchError{
			GroupKind:        schema.GroupKind{Group: list.GetObjectKind().GroupVersionKind().Group, Kind: list.GetObjectKind().GroupVersionKind().Kind},
			SearchedVersions: []string{list.GetObjectKind().GroupVersionKind().Version},
		}
	default:
		return c.Client.List(ctx, list, opts...)
	}
}

// clientWithFailingDelete simulates a shoot client which is unable to remove the configmap of the deprecated metallb
// health sidecar, for example because the api server is unavailable.
type clientWithFailingDelete struct {
	client.Client
	err error
}

func (c *clientWithFailingDelete) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	return c.err
}

// legacyHealthConfigMapObject returns the configmap as the deprecated metallb health sidecar wrote it.
func legacyHealthConfigMapObject(configLoaded, configStale string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "health",
			Namespace: "metallb-system",
		},
		Data: map[string]string{
			"configLoaded": configLoaded,
			"configStale":  configStale,
		},
	}
}

func Test_legacyHealth(t *testing.T) {
	tests := []struct {
		name      string
		data      map[string]string
		wantError string
	}{
		{
			name: "healthy configuration",
			data: map[string]string{"configLoaded": "true", "configStale": "false"},
		},
		{
			name:      "configuration is not loaded yet",
			data:      map[string]string{"configLoaded": "false", "configStale": "false"},
			wantError: "not loaded yet",
		},
		{
			name:      "configuration is stale",
			data:      map[string]string{"configLoaded": "true", "configStale": "true"},
			wantError: "stale",
		},
		{
			name:      "configLoaded is missing",
			data:      map[string]string{"configStale": "false"},
			wantError: `unable to parse "configLoaded" of the deprecated metallb health configmap`,
		},
		{
			name:      "configStale is missing",
			data:      map[string]string{"configLoaded": "true"},
			wantError: `unable to parse "configStale" of the deprecated metallb health configmap`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := legacyHealth(&corev1.ConfigMap{Data: tt.data})
			switch {
			case tt.wantError == "" && err != nil:
				t.Errorf("legacyHealth() = %v, want no error", err)
			case tt.wantError != "" && err == nil:
				t.Errorf("legacyHealth() = nil, want an error containing %q", tt.wantError)
			case tt.wantError != "" && !strings.Contains(err.Error(), tt.wantError):
				t.Errorf("legacyHealth() = %v, want it to contain %q", err, tt.wantError)
			}
		})
	}
}

func Test_Check_reportsErrorsOfTheComponents(t *testing.T) {
	// a healthy configmap of the deprecated health sidecar must not hide an error the metallb components report
	shootClient := &clientWithoutMetallbCRDs{
		Client: newFakeClient(t, legacyHealthConfigMapObject("true", "false")),
		err:    errors.New("connection refused"),
	}
	healthChecker := &MetalLBHealthChecker{shootClient: shootClient}
	healthChecker.SetLoggerSuffix("provider", "metal")

	_, err := healthChecker.Check(context.Background(), types.NamespacedName{})
	if err == nil || !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("Check() error = %v, want it to contain %q", err, "connection refused")
	}
}

func Test_Check_ignoresErrorsWhileRemovingTheLegacyHealthConfigMap(t *testing.T) {
	// the configmap of the deprecated health sidecar is not evaluated anymore as soon as the metallb components report
	// their health, hence a failing removal must not fail the health check of the shoot.
	shootClient := &clientWithFailingDelete{
		Client: newFakeClient(t,
			configurationStateObject("controller", "Valid", ""),
			configurationStateObject("speaker-node1", "Valid", ""),
			frrNodeStateObject("node1", "success", "success"),
			legacyHealthConfigMapObject("true", "false"),
		),
		err: apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, legacyHealthConfigMapName, errors.New("not allowed")),
	}
	healthChecker := &MetalLBHealthChecker{shootClient: shootClient}
	healthChecker.SetLoggerSuffix("provider", "metal")

	result, err := healthChecker.Check(context.Background(), types.NamespacedName{})
	if err != nil {
		t.Fatalf("Check() returned an unexpected error: %v", err)
	}
	if result.Status != gardencorev1beta1.ConditionTrue {
		t.Errorf("Check() status = %v, want %v (detail: %s)", result.Status, gardencorev1beta1.ConditionTrue, result.Detail)
	}

	legacyConfigMap := &corev1.ConfigMap{}
	if err := shootClient.Get(context.Background(), client.ObjectKey{Namespace: metallbNamespace, Name: legacyHealthConfigMapName}, legacyConfigMap); err != nil {
		t.Errorf("the configmap of the deprecated metallb health sidecar is not retained for the next health check: %v", err)
	}
}

func Test_configurationStateUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "the kind is not shipped by the shoot",
			err: &meta.NoKindMatchError{
				GroupKind:        schema.GroupKind{Group: "metallb.io", Kind: "ConfigurationStateList"},
				SearchedVersions: []string{"v1beta1"},
			},
			want: true,
		},
		{
			name: "the api server does not know the resource",
			err:  apierrors.NewNotFound(schema.GroupResource{Group: "metallb.io", Resource: "configurationstates"}, ""),
			want: true,
		},
		{
			name: "any other error",
			err:  errors.New("connection refused"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := configurationStateUnavailable(tt.err); got != tt.want {
				t.Errorf("configurationStateUnavailable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
