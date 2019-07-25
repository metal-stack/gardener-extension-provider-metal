package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	metalapi "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"

	metalgo "github.com/metal-pod/metal-go"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener-extensions/pkg/controller/error"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	corev1 "k8s.io/api/core/v1"
)

func (a *actuator) delete(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	infrastructureConfig := &metalapi.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infrastructure.Spec.ProviderConfig.Raw, nil, infrastructureConfig); err != nil {
		return fmt.Errorf("could not decode provider config: %+v", err)
	}

	providerSecret := &corev1.Secret{}
	if err := a.client.Get(ctx, kutil.Key(infrastructure.Spec.SecretRef.Namespace, infrastructure.Spec.SecretRef.Name), providerSecret); err != nil {
		return err
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

	partition := infrastructureConfig.Firewall.Partition
	project := cluster.Shoot.Status.TechnicalID
	a.logger.Info("search firewalls:", "partition", partition, "project", project)
	fws, err := svc.FirewallSearch(&partition, &project)
	a.logger.Info("found firewalls:", "count", len(fws.Firewalls), "", fws.Firewalls)
	if err != nil {
		a.logger.Error(err, "failed get firewalls", "infrastructure", infrastructure.Name)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, fw := range fws.Firewalls {
		_, err := svc.MachineDelete(*fw.ID)
		a.logger.Error(err, "failed to delete firewall", "infrastructure", infrastructure.Name, "firewallID", *fw.ID)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	return nil
}
