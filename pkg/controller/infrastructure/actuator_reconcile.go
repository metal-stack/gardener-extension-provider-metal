package infrastructure

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	metalapi "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	"strings"
	"time"

	metalgo "github.com/metal-pod/metal-go"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener-extensions/pkg/controller/error"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
)

func (a *actuator) reconcile(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {

	infrastructureConfig := &metalapi.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infrastructure.Spec.ProviderConfig.Raw, nil, infrastructureConfig); err != nil {
		return fmt.Errorf("could not decode provider config: %+v", err)
	}

	infrastructureStatus := &metalapi.InfrastructureStatus{}
	if _, _, err := a.decoder.Decode(infrastructure.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
		return fmt.Errorf("could not decode infrastructure status: %+v", err)
	}

	providerSecret := &corev1.Secret{}
	if err := a.client.Get(ctx, kutil.Key(infrastructure.Spec.SecretRef.Namespace, infrastructure.Spec.SecretRef.Name), providerSecret); err != nil {
		return err
	}

	firewallStatus := infrastructureStatus.Firewall
	if firewallStatus.Succeeded {
		return nil
	}

	if firewallStatus.MachineID != "" {
		// determine if succeeded

	}

	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	name := cluster.Shoot.Namespace + "-" + uuid.String()[:5]

	createRequest := &metalgo.FirewallCreateRequest{
		MachineCreateRequest: metalgo.MachineCreateRequest{
			Description: name + " created by Gardener",
			Name:        name,
			Hostname:    name,
			Size:        infrastructureConfig.Firewall.Size,
			Project:     cluster.Shoot.Namespace,
			Tenant:      string(providerSecret.Data[metal.TenantID]),
			Partition:   infrastructureConfig.Firewall.Partition,
			Image:       infrastructureConfig.Firewall.Image,
		},

		NetworkIDs: infrastructureConfig.Firewall.Networks,
	}

	token := strings.TrimSpace(string(providerSecret.Data[metal.APIKey]))
	hmac := strings.TrimSpace(string(providerSecret.Data[metal.APIHMac]))

	u, ok := providerSecret.Data[metal.APIURL]
	if !ok {
		return fmt.Errorf("missing %s in secret", metal.APIURL)
	}
	url := strings.TrimSpace(string(u))

	svc, err := metalgo.NewDriver(url, token, hmac)
	if err != nil {
		return err
	}

	fcr, err := svc.FirewallCreate(createRequest)
	if err != nil {
		a.logger.Error(err, "failed to create firewall", "infrastructure", infrastructure.Name)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	machineName := *fcr.Firewall.Allocation.Name
	machineId := encodeMachineID(*fcr.Firewall.Partition.ID, *fcr.Firewall.ID)

	return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus)
}

func (a *actuator) updateProviderStatus(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureConfig *metalapi.InfrastructureConfig, status metalapi.FirewallStatus) error {

	// FIXME compute status

	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.client, infrastructure, func() error {
		infrastructure.Status.ProviderStatus = &runtime.RawExtension{
			Object: &metalapi.InfrastructureStatus{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metalapi.SchemeGroupVersion.String(),
					Kind:       "InfrastructureStatus",
				},
				Firewall: status,
			},
		}
		return nil
	})
}

func encodeMachineID(partition, machineID string) string {
	return fmt.Sprintf("metal:///%s/%s", partition, machineID)
}

func decodeMachineID(id string) string {
	splitProviderID := strings.Split(id, "/")
	return splitProviderID[len(splitProviderID)-1]
}
