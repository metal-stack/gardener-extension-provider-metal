package healthcheck

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/go-logr/logr"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FirewallHealthChecker contains all the information for the Firewall HealthCheck
type FirewallHealthChecker struct {
	logger     logr.Logger
	seedClient client.Client
}

// CheckFirewall is a healthCheck function to check Firewalls
func CheckFirewall() healthcheck.HealthCheck {
	return &FirewallHealthChecker{}
}

// InjectSeedClient injects the seed client
func (healthChecker *FirewallHealthChecker) InjectSeedClient(seedClient client.Client) {
	healthChecker.seedClient = seedClient
}

// SetLoggerSuffix injects the logger
func (healthChecker *FirewallHealthChecker) SetLoggerSuffix(provider, extension string) {
	healthChecker.logger = log.Log.WithName(fmt.Sprintf("%s-%s-healthcheck-firewall", provider, extension))
}

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *FirewallHealthChecker) DeepCopy() healthcheck.HealthCheck {
	copy := *healthChecker
	return &copy
}

// Check executes the health check
func (healthChecker *FirewallHealthChecker) Check(ctx context.Context, request types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	fwdeploy := &fcmv2.FirewallDeployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      metal.FirewallDeploymentName,
			Namespace: request.Namespace,
		},
	}

	if err := healthChecker.seedClient.Get(ctx, client.ObjectKeyFromObject(fwdeploy), fwdeploy); err != nil {
		err := fmt.Errorf("check firewall deployment resource failed. Unable to retrieve firewall deployment resource '%s' in namespace '%s': %w", fwdeploy.Name, request.Namespace, err)
		healthChecker.logger.Error(err, "Health check failed")
		return nil, err
	}

	if isHealthy, err := firewallIsHealthy(fwdeploy); !isHealthy {
		healthChecker.logger.Error(err, "Health check failed")
		return &healthcheck.SingleCheckResult{
			Status: gardencorev1beta1.ConditionFalse,
			Detail: err.Error(),
		}, nil
	}

	return &healthcheck.SingleCheckResult{
		Status: gardencorev1beta1.ConditionTrue,
	}, nil
}

func firewallIsHealthy(fwdeploy *fcmv2.FirewallDeployment) (bool, error) {
	if fwdeploy == nil {
		return false, fmt.Errorf("firewall deployment resource not deployed")
	}

	if fwdeploy.Status.UnhealthyReplicas > 0 {
		return false, fmt.Errorf("firewall deployment has %d unhealthy replicas", fwdeploy.Status.UnhealthyReplicas)
	}

	if fwdeploy.Status.ReadyReplicas != fwdeploy.Status.TargetReplicas {
		return false, fmt.Errorf("firewall deployment only has %d/%d ready replicas", fwdeploy.Status.ReadyReplicas, fwdeploy.Status.TargetReplicas)
	}

	// TODO: Decide if we need more specific information to make problems easier to identify when looking at the shoot state
	// e.g. gather problematic conditions from the managed firewall resources

	return true, nil

}
