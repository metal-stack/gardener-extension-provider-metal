package healthcheck

import (
	"context"
	"fmt"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	durosv1 "github.com/metal-stack/duros-controller/api/v1"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DurosHealthChecker contains all the information for the Duros HealthCheck
type DurosHealthChecker struct {
	logger            logr.Logger
	seedClient        client.Client
	durosResourceName string
}

// CheckDuros is a healthCheck function to check Duross
func CheckDuros(durosResourceName string) healthcheck.HealthCheck {
	return &DurosHealthChecker{
		durosResourceName: durosResourceName,
	}
}

// InjectSeedClient injects the seed client
func (healthChecker *DurosHealthChecker) InjectSeedClient(seedClient client.Client) {
	healthChecker.seedClient = seedClient
}

// SetLoggerSuffix injects the logger
func (healthChecker *DurosHealthChecker) SetLoggerSuffix(provider, extension string) {
	healthChecker.logger = log.Log.WithName(fmt.Sprintf("%s-%s-healthcheck-duros", provider, extension))
}

// DeepCopy clones the healthCheck struct by making a copy and returning the pointer to that new copy
func (healthChecker *DurosHealthChecker) DeepCopy() healthcheck.HealthCheck {
	copy := *healthChecker
	return &copy
}

// Check executes the health check
func (healthChecker *DurosHealthChecker) Check(ctx context.Context, request types.NamespacedName) (*healthcheck.SingleCheckResult, error) {
	duros := &durosv1.Duros{}

	if err := healthChecker.seedClient.Get(ctx, client.ObjectKey{Namespace: request.Namespace, Name: healthChecker.durosResourceName}, duros); err != nil {
		if apierrors.IsNotFound(err) {
			// we skip the health check when there is no duros resource deployed
			return &healthcheck.SingleCheckResult{
				Status: gardencorev1beta1.ConditionTrue,
			}, nil
		}

		err := fmt.Errorf("check duros resource failed. Unable to retrieve duros resource '%s' in namespace '%s': %w", healthChecker.durosResourceName, request.Namespace, err)
		healthChecker.logger.Error(err, "Health check failed")
		return nil, err
	}
	if isHealthy, err := DurosIsHealthy(duros); !isHealthy {
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

func DurosIsHealthy(duros *durosv1.Duros) (bool, error) {
	if duros == nil {
		return false, fmt.Errorf("duros resource not deployed")
	}

	var problems []string

	if duros.Status.ReconcileStatus.Error != nil {
		problems = append(problems, fmt.Sprintf("reconcile error: %s (at %s)", *duros.Status.ReconcileStatus.Error, pointer.SafeDeref(duros.Status.ReconcileStatus.LastReconcile).String()))
	}

	for _, r := range duros.Status.ManagedResourceStatuses {
		if r.State == durosv1.HealthStateRunning {
			continue
		}

		problems = append(problems, fmt.Sprintf("%s is not running because: %s", r.Name, r.Description))
	}

	if len(problems) > 0 {
		err := fmt.Errorf("duros resource %s in namespace %s is unhealthy: %v", duros.Name, duros.Namespace, strings.Join(problems, ", "))
		return false, err
	}

	return true, nil
}
