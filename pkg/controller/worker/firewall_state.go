package worker

import (
	"context"
	"encoding/json"
	"fmt"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InfrastructureState represents the last known State of an Infrastructure resource.
// It is saved after a reconciliation and used during restore operations.
// We use this for restoring firewalls, which are actually maintained by the worker controller
// because the worker controller does not allow adding our state to the worker resource as it
// is used by the MCM already.
type InfrastructureState struct {
	// Firewalls contains the running firewalls.
	Firewalls []string `json:"firewalls"`

	SeedAccess []SeedAccessState `json:"seedAccess"`
}

type SeedAccessState struct {
	ServiceAccount        string   `json:"serviceAccount"`
	ServiceAccountSecrets []string `json:"serviceAccountSecrets"`
}

func (a *actuator) updateState(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure) error {
	patch := client.MergeFrom(infrastructure.DeepCopy())

	var (
		namespace = infrastructure.Namespace

		infraState = &InfrastructureState{}
		fwdeploys  = &fcmv2.FirewallDeploymentList{}
		firewalls  = &fcmv2.FirewallList{}
	)

	err := a.client.List(ctx, firewalls, client.InNamespace(infrastructure.Namespace))
	if err != nil {
		return fmt.Errorf("unable to list firewalls: %w", err)
	}

	for _, fw := range firewalls.Items {
		fw := fw

		fw.ResourceVersion = ""
		fw.OwnerReferences = nil
		fw.Status = fcmv2.FirewallStatus{}

		raw, err := yaml.Marshal(fw)
		if err != nil {
			return err
		}

		infraState.Firewalls = append(infraState.Firewalls, string(raw))
	}

	err = a.client.List(ctx, fwdeploys, client.InNamespace(infrastructure.Namespace))
	if err != nil {
		return fmt.Errorf("unable to list firewall deployments: %w", err)
	}

	for _, fwdeploy := range fwdeploys.Items {
		saName := fmt.Sprintf("firewall-controller-seed-access-%s", fwdeploy.Name) // TODO: name should be exposed by fcm
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: namespace,
			},
		}

		err := a.client.Get(ctx, client.ObjectKeyFromObject(sa), sa)
		if err != nil {
			continue
		}

		secrets := []string{}

		for _, ref := range sa.Secrets {

			saSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ref.Name,
					Namespace: namespace,
				},
			}

			err = a.client.Get(ctx, client.ObjectKeyFromObject(saSecret), saSecret)
			if err != nil {
				return fmt.Errorf("error getting service account secret: %w", err)
			}

			saSecret.ResourceVersion = ""

			raw, err := yaml.Marshal(*saSecret)
			if err != nil {
				return err
			}

			secrets = append(secrets, string(raw))
		}

		sa.ResourceVersion = ""

		raw, err := yaml.Marshal(*sa)
		if err != nil {
			return err
		}

		infraState.SeedAccess = append(infraState.SeedAccess, SeedAccessState{
			ServiceAccount:        string(raw),
			ServiceAccountSecrets: secrets,
		})
	}

	infraStateBytes, err := json.Marshal(infraState)
	if err != nil {
		return err
	}

	infrastructure.Status.State = &runtime.RawExtension{Raw: infraStateBytes}

	return a.client.Status().Patch(ctx, infrastructure, patch)
}

func (a *actuator) restoreState(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure) error {
	infraState := &InfrastructureState{}
	err := json.Unmarshal(infrastructure.Status.State.Raw, infraState)
	if err != nil {
		return fmt.Errorf("unable to decode infrastructure status: %w", err)
	}

	a.logger.Info("restoring firewalls and service accounts", "firewalls", len(infraState.Firewalls), "service-accounts", len(infraState.SeedAccess))

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

	for _, seedAccess := range infraState.SeedAccess {

		sa := &corev1.ServiceAccount{}
		err := yaml.Unmarshal([]byte(seedAccess.ServiceAccount), sa)
		if err != nil {
			return err
		}

		err = a.client.Create(ctx, sa)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("unable restoring service account: %w", err)
		}

		for _, raw := range seedAccess.ServiceAccountSecrets {
			raw := raw

			secret := &corev1.Secret{}
			err := yaml.Unmarshal([]byte(raw), secret)
			if err != nil {
				return err
			}

			secret.Annotations["kubernetes.io/service-account.uid"] = string(sa.UID)

			err = a.client.Create(ctx, secret)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("unable restoring service account secret %q: %w", secret.Name, err)
			}
		}
	}

	return nil
}
