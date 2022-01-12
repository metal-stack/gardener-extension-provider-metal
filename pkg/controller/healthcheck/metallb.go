package healthcheck

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

// shootClient injects the shoot client
func (healthChecker *MetalLBHealthChecker) InjectShootClient(shootClient client.Client) {
	healthChecker.shootClient = shootClient
}

// SetLoggerSuffix injects the logger
func (healthChecker *MetalLBHealthChecker) SetLoggerSuffix(provider, extension string) {
	healthChecker.logger = log.Log.WithName(fmt.Sprintf("%s-%s-healthcheck-metallb", provider, extension))
}

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *MetalLBHealthChecker) DeepCopy() healthcheck.HealthCheck {
	copy := *healthChecker
	return &copy
}

// Check executes the health check
func (healthChecker *MetalLBHealthChecker) Check(ctx context.Context, request types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	health := &v1.ConfigMap{}

	if err := healthChecker.shootClient.Get(ctx, client.ObjectKey{Namespace: "metallb-system", Name: "health"}, health); err != nil {
		err := fmt.Errorf("check metallb health configmap failed. Unable to retrieve 'health' in namespace 'metallb-system': %v", err)
		healthChecker.logger.Error(err, "Health check failed")
		return nil, err
	}
	if isHealthy, err := IsHealthy(health); !isHealthy {
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

func IsHealthy(health *v1.ConfigMap) (bool, error) {
	isLoaded := health.Data["configLoaded"]
	if isLoaded != "1" {
		return false, fmt.Errorf("metallb configmap is not loaded")
	}

	isStale := health.Data["configStale"]
	if isStale == "1" {
		return false, fmt.Errorf("metallb configmap is stale / erroneous, next speaker reload may interrupt workload traffic")
	}

	return true, nil
}
