package healthcheck

import (
	"context"
	"time"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	genericcontrolplaneactuator "github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck"
	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config"
	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck/general"
	"github.com/gardener/gardener/extensions/pkg/controller/healthcheck/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"

	extensionspredicate "github.com/gardener/gardener/extensions/pkg/predicate"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	defaultSyncPeriod = time.Second * 30
	// DefaultAddOptions are the default DefaultAddArgs for AddToManager.
	DefaultAddOptions = AddOptions{
		HealthCheckDefaults: healthcheck.DefaultAddArgs{
			HealthCheckConfig: healthcheckconfig.HealthCheckConfig{SyncPeriod: metav1.Duration{Duration: defaultSyncPeriod}},
		},
	}
)

type AddOptions struct {
	ControllerConfig config.ControllerConfiguration

	HealthCheckDefaults healthcheck.DefaultAddArgs
}

// RegisterHealthChecks registers health checks for each extension resource
func RegisterHealthChecks(mgr manager.Manager, opts AddOptions) error {
	accountingPreCheck := func(_ context.Context, _ client.Client, _ client.Object, _ *extensionscontroller.Cluster) bool {
		return opts.ControllerConfig.AccountingExporter.Enabled
	}
	authPreCheck := func(_ context.Context, _ client.Client, _ client.Object, _ *extensionscontroller.Cluster) bool {
		return opts.ControllerConfig.Auth.Enabled
	}
	durosPreCheck := func(_ context.Context, _ client.Client, _ client.Object, _ *extensionscontroller.Cluster) bool {
		return opts.ControllerConfig.Storage.Duros.Enabled
	}

	if err := healthcheck.DefaultRegistration(
		metal.Type,
		extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.ControlPlaneResource),
		func() client.ObjectList { return &extensionsv1alpha1.ControlPlaneList{} },
		func() extensionsv1alpha1.Object { return &extensionsv1alpha1.ControlPlane{} },
		mgr,
		opts.HealthCheckDefaults,
		[]predicate.Predicate{extensionspredicate.HasPurpose(extensionsv1alpha1.Normal)},
		[]healthcheck.ConditionTypeToHealthCheck{
			{
				ConditionType: string(gardencorev1beta1.ShootControlPlaneHealthy),
				HealthCheck:   general.NewSeedDeploymentHealthChecker(metal.CloudControllerManagerDeploymentName),
			},
			{
				ConditionType: string(gardencorev1beta1.ShootControlPlaneHealthy),
				HealthCheck:   general.NewSeedDeploymentHealthChecker(metal.AccountingExporterName),
				PreCheckFunc:  accountingPreCheck,
			},
			{
				ConditionType: string(gardencorev1beta1.ShootControlPlaneHealthy),
				HealthCheck:   general.NewSeedDeploymentHealthChecker(metal.GroupRolebindingControllerName),
				PreCheckFunc:  authPreCheck,
			},
			{
				ConditionType: string(gardencorev1beta1.ShootControlPlaneHealthy),
				HealthCheck:   general.NewSeedDeploymentHealthChecker(metal.AuthNWebhookDeploymentName),
				PreCheckFunc:  authPreCheck,
			},
			{
				ConditionType: string(gardencorev1beta1.ShootSystemComponentsHealthy),
				HealthCheck:   general.CheckManagedResource(genericcontrolplaneactuator.ControlPlaneShootChartResourceName),
			},
			{
				ConditionType: string(gardencorev1beta1.ShootSystemComponentsHealthy),
				HealthCheck:   general.CheckManagedResource(genericcontrolplaneactuator.StorageClassesChartResourceName),
			},
			{
				ConditionType: string(gardencorev1beta1.ShootSystemComponentsHealthy),
				HealthCheck:   CheckDuros(metal.DurosResourceName),
				PreCheckFunc:  durosPreCheck,
			},
			{
				ConditionType: string(gardencorev1beta1.ShootSystemComponentsHealthy),
				HealthCheck:   CheckFirewall(metal.FirewallResourceName),
			},
			{
				ConditionType: string(gardencorev1beta1.ShootSystemComponentsHealthy),
				HealthCheck:   CheckMetalLB(),
			},
		}); err != nil {
		return err
	}

	return healthcheck.DefaultRegistration(
		metal.Type,
		extensionsv1alpha1.SchemeGroupVersion.WithKind(extensionsv1alpha1.WorkerResource),
		func() client.ObjectList { return &extensionsv1alpha1.WorkerList{} },
		func() extensionsv1alpha1.Object { return &extensionsv1alpha1.Worker{} },
		mgr,
		opts.HealthCheckDefaults,
		nil,
		[]healthcheck.ConditionTypeToHealthCheck{
			{
				ConditionType: string(gardencorev1beta1.ShootSystemComponentsHealthy),
				HealthCheck:   general.CheckManagedResource(genericworkeractuator.McmShootResourceName),
			},
			{
				ConditionType: string(gardencorev1beta1.ShootControlPlaneHealthy),
				HealthCheck:   general.NewSeedDeploymentHealthChecker("machine-controller-manager"),
			},
			{
				ConditionType: string(gardencorev1beta1.ShootEveryNodeReady),
				HealthCheck:   worker.NewNodesChecker(),
			},
		})
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return RegisterHealthChecks(mgr, DefaultAddOptions)
}
