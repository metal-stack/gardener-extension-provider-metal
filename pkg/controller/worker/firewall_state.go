package worker

import (
	"context"
	"encoding/json"
	"fmt"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// InfrastructureState represents the last known State of an Infrastructure resource.
// It is saved after a reconciliation and used during restore operations.
// We use this for restoring firewalls, which are actually maintained by the worker controller
// because the worker controller does not allow adding our state to the worker resource as it
// is used by the MCM already.
type InfrastructureState struct {
	// Firewalls contains the running firewalls.
	Firewalls []string `json:"firewalls"`
}

func (a *actuator) updateState(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure) error {
	patch := client.MergeFrom(infrastructure.DeepCopy())

	var (
		namespace  = infrastructure.Namespace
		infraState = &InfrastructureState{}
		firewalls  = &fcmv2.FirewallList{}
	)

	err := a.client.List(ctx, firewalls, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("unable to list firewalls: %w", err)
	}

	for _, fw := range firewalls.Items {
		fw := fw

		fw.ResourceVersion = ""
		fw.OwnerReferences = nil
		fw.Status = fcmv2.FirewallStatus{}
		fw.ManagedFields = nil

		raw, err := yaml.Marshal(fw)
		if err != nil {
			return err
		}

		infraState.Firewalls = append(infraState.Firewalls, string(raw))
	}

	infraStateBytes, err := json.Marshal(infraState)
	if err != nil {
		return err
	}

	infrastructure.Status.State = &runtime.RawExtension{Raw: infraStateBytes}

	err = a.client.Status().Patch(ctx, infrastructure, patch)
	if err != nil {
		return err
	}

	log.Info("firewall state updated in infrastructure status", "firewalls", len(infraState.Firewalls))

	return nil
}

func (a *actuator) restoreState(ctx context.Context, log logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure) error {
	infraState := &InfrastructureState{}
	err := json.Unmarshal(infrastructure.Status.State.Raw, infraState)
	if err != nil {
		return fmt.Errorf("unable to decode infrastructure status: %w", err)
	}

	log.Info("restoring firewalls from infrastructure status", "firewalls", len(infraState.Firewalls))

	for _, raw := range infraState.Firewalls {
		raw := raw

		fw := &fcmv2.Firewall{}
		err := yaml.Unmarshal([]byte(raw), fw)
		if err != nil {
			return err
		}

		err = a.client.Create(ctx, fw)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable restoring firewall resource: %w", err)
		}
	}

	return nil
}
