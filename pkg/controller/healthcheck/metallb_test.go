package healthcheck

import (
	"context"
	"reflect"
	"strings"
	"testing"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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
		wantStatus  gardencorev1beta1.ConditionStatus
		wantDetail  string
		wantPending bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthChecker := &MetalLBHealthChecker{shootClient: newFakeClient(t, tt.objects...)}
			healthChecker.SetLoggerSuffix("provider", "metal")

			result, err := healthChecker.Check(context.Background(), types.NamespacedName{})
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
