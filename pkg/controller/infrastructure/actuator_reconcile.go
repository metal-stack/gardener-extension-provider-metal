package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	metalapi "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"

	metalgo "github.com/metal-pod/metal-go"
	metalfirewall "github.com/metal-pod/metal-go/api/client/firewall"

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

	a.logger.Info("InfrastructureConfig", "config", infrastructureConfig)

	infrastructureStatus := &metalapi.InfrastructureStatus{}
	if infrastructure.Status.ProviderStatus != nil {
		if _, _, err := a.decoder.Decode(infrastructure.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return fmt.Errorf("could not decode infrastructure status: %+v", err)
		}
	}

	providerSecret := &corev1.Secret{}
	if err := a.client.Get(ctx, kutil.Key(infrastructure.Spec.SecretRef.Namespace, infrastructure.Spec.SecretRef.Name), providerSecret); err != nil {
		return err
	}

	firewallStatus := infrastructureStatus.Firewall
	if firewallStatus.Succeeded {
		return nil
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

	if firewallStatus.MachineID != "" {
		// firewall was already created

		machineID := decodeMachineID(firewallStatus.MachineID)

		resp, err := svc.FirewallGet(machineID)
		if err != nil {
			switch e := err.(type) {
			case *metalfirewall.FindFirewallDefault:
				if e.Code() >= 500 {
					return &controllererrors.RequeueAfterError{
						Cause:        e,
						RequeueAfter: 30 * time.Second,
					}
				}
			default:
				return e
			}
		}

		allocation := resp.Firewall.Allocation
		if allocation == nil {
			return fmt.Errorf("firewall %q was created but has no allocation", machineID)
		}

		firewallStatus.Succeeded = *resp.Firewall.Allocation.Succeeded
		return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus)
	}

	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	// Example values:
	// cluster.Shoot.Namespace "garden-dev"
	// cluster.Seed.Namespace  "shoot--dev--johndoe-metal"
	name := cluster.Seed.Namespace + "-firewall-" + uuid.String()[:5]
	project := cluster.Seed.Namespace

	createRequest := &metalgo.FirewallCreateRequest{
		MachineCreateRequest: metalgo.MachineCreateRequest{
			Description: name + " created by Gardener",
			Name:        name,
			Hostname:    name,
			Size:        infrastructureConfig.Firewall.Size,
			Project:     project,
			Tenant:      string(providerSecret.Data[metal.TenantID]),
			Partition:   infrastructureConfig.Firewall.Partition,
			Image:       infrastructureConfig.Firewall.Image,
		},

		NetworkIDs: infrastructureConfig.Firewall.Networks,
	}

	a.logger.Info("create firewall from", "request", createRequest)

	fcr, err := svc.FirewallCreate(createRequest)
	if err != nil {
		a.logger.Error(err, "failed to create firewall", "infrastructure", infrastructure.Name)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	machineID := encodeMachineID(*fcr.Firewall.Partition.ID, *fcr.Firewall.ID)

	allocation := fcr.Firewall.Allocation
	if allocation == nil {
		return fmt.Errorf("firewall %q was created but has no allocation", machineID)
	}

	firewallStatus.MachineID = machineID
	firewallStatus.Succeeded = *allocation.Succeeded

	return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus)
}

func (a *actuator) updateProviderStatus(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureConfig *metalapi.InfrastructureConfig, status metalapi.FirewallStatus) error {
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
