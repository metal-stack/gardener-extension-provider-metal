package healthcheck

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/go-logr/logr"
	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// FirewallHealthChecker contains all the information for the Firewall HealthCheck
type FirewallHealthChecker struct {
	logger               logr.Logger
	shootClient          client.Client
	firewallResourceName string
}

// CheckFirewall is a healthCheck function to check Firewalls
func CheckFirewall(firewallResourceName string) healthcheck.HealthCheck {
	return &FirewallHealthChecker{
		firewallResourceName: firewallResourceName,
	}
}

// InjectShootClient injects the shoot client
func (healthChecker *FirewallHealthChecker) InjectShootClient(shootClient client.Client) {
	healthChecker.shootClient = shootClient
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
	firewall := &firewallv1.Firewall{}

	// TODO make namespace a const
	namespace := "firewall"
	if err := healthChecker.shootClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: healthChecker.firewallResourceName}, firewall); err != nil {
		err := fmt.Errorf("check firewall resource failed. Unable to retrieve firewall resource '%s' in namespace '%s': %v", healthChecker.firewallResourceName, namespace, err)
		healthChecker.logger.Error(err, "Health check failed")
		return nil, err
	}
	if isHealthy, err := firewallIsHealthy(firewall); !isHealthy {
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

func firewallIsHealthy(firewall *firewallv1.Firewall) (bool, error) {
	if firewall == nil {
		return false, fmt.Errorf("firewall resource not deployed")
	}

	// FIXME remove this once firewall-controller >= v1.1.3 is deployed to all clusters
	if firewall.Status.ControllerVersion == "" {
		return true, nil
	}

	if firewall.Spec.ControllerVersion != firewall.Status.ControllerVersion {
		return false, fmt.Errorf("firewall version specified at version:%s but still on:%s", firewall.Spec.ControllerVersion, firewall.Status.ControllerVersion)
	}
	return true, nil

}
