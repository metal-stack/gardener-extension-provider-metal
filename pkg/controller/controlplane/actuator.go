package controlplane

import (
	"context"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/cache"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type (
	// actuator reconciles the cluster's controle plane
	// we are wrapping gardener's control plane actuator here because we need to intercept the migrate call from the actuator.
	// unfortunately, there is no callback provided which we could use for this.
	actuator struct {
		controllerConfig config.ControllerConfiguration

		controlplaneActuator controlplane.Actuator

		networkCache *cache.Cache[*cacheKey, *models.V1NetworkResponse]

		client     client.Client
		decoder    runtime.Decoder
		restConfig *rest.Config
		scheme     *runtime.Scheme
	}
)

func NewActuator(mgr manager.Manager, opts AddOptions) (controlplane.Actuator, error) {
	a := &actuator{
		controllerConfig: opts.ControllerConfig,
		networkCache: cache.New(15*time.Minute, func(ctx context.Context, accessor *cacheKey) (*models.V1NetworkResponse, error) {
			mclient, ok := ctx.Value(ClientKey).(metalgo.Client)
			if !ok {
				return nil, fmt.Errorf("no client passed in context")
			}
			privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(ctx, mclient, accessor.projectID, accessor.nodeCIDR)
			if err != nil {
				return nil, err
			}
			return privateNetwork, nil
		}),
		client:     mgr.GetClient(),
		decoder:    serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		restConfig: mgr.GetConfig(),
		scheme:     mgr.GetScheme(),
	}

	controlplaneActuator, err := genericactuator.NewActuator(
		mgr, metal.Name, secretConfigsFunc, shootAccessSecretsFunc,
		nil, nil, nil, controlPlaneChart, cpShootChart,
		nil, storageClassChart, nil, NewValuesProvider(mgr, opts.ControllerConfig),
		extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
		imagevector.ImageVector(), "", opts.ShootWebhookConfig, opts.WebhookServerNamespace,
	)

	if err != nil {
		return nil, err
	}

	a.controlplaneActuator = controlplaneActuator

	return a, nil
}

func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, controlPlane *extensionsv1alpha1.ControlPlane, cluster *extensions.Cluster) (bool, error) {
	err := a.csiLVMReconcile(ctx, log, controlPlane, cluster)
	if err != nil {
		return false, err
	}
	return a.controlplaneActuator.Restore(ctx, log, controlPlane, cluster)
}

func (a *actuator) Delete(ctx context.Context, log logr.Logger, controlPlane *extensionsv1alpha1.ControlPlane, cluster *extensions.Cluster) error {
	return a.controlplaneActuator.Delete(ctx, log, controlPlane, cluster)
}

func (a *actuator) ForceDelete(ctx context.Context, log logr.Logger, controlPlane *extensionsv1alpha1.ControlPlane, cluster *extensions.Cluster) error {
	return a.controlplaneActuator.ForceDelete(ctx, log, controlPlane, cluster)
}

func (a *actuator) Migrate(ctx context.Context, log logr.Logger, controlPlane *extensionsv1alpha1.ControlPlane, cluster *extensions.Cluster) error {
	return a.controlplaneActuator.Migrate(ctx, log, controlPlane, cluster)
}

func (a *actuator) Restore(ctx context.Context, log logr.Logger, controlPlane *extensionsv1alpha1.ControlPlane, cluster *extensions.Cluster) (bool, error) {
	return a.controlplaneActuator.Restore(ctx, log, controlPlane, cluster)
}
