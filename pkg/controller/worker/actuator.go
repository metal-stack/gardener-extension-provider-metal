package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener/extensions/pkg/util"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	"github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalv1alpha1 "github.com/metal-stack/machine-controller-manager-provider-metal/pkg/provider/migration/legacy-api/machine/v1alpha1"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/models"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/metal-stack/metal-lib/pkg/cache"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type (
	// actuator reconciles the cluster's worker nodes and the firewalls.
	//
	// why is the firewall reconciliation here and not in the controlplane controller?
	// the controlplane controller deploys the firewall-controller-manager including validating and mutating webhooks
	// this has to be running before we can create a firewall deployment because the mutating webhook is creating the userdata
	// the worker controller acts after the controlplane controller, also the terms and responsibilities are pretty similar between machine-controller-manager and firewall-controller-manager,
	// so this place seems to be a valid fit.
	actuator struct {
		workerActuator worker.Actuator

		logger           logr.Logger
		controllerConfig config.ControllerConfiguration
		networkCache     *cache.Cache[*cacheKey, *models.V1NetworkResponse]

		restConfig *rest.Config
		client     client.Client
		scheme     *runtime.Scheme
		decoder    runtime.Decoder
	}

	delegateFactory struct {
		logger logr.Logger

		clientGetter func() (*rest.Config, client.Client, *runtime.Scheme, runtime.Decoder)

		dataGetter       func(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (*additionalData, error)
		controllerConfig config.ControllerConfiguration

		machineImageMapping []config.MachineImage
	}

	workerDelegate struct {
		logger logr.Logger

		client  client.Client
		scheme  *runtime.Scheme
		decoder runtime.Decoder

		machineImageMapping []config.MachineImage
		seedChartApplier    gardener.ChartApplier
		serverVersion       string

		cluster *extensionscontroller.Cluster
		worker  *extensionsv1alpha1.Worker

		machineClasses     []map[string]interface{}
		machineDeployments worker.MachineDeployments
		machineImages      []apismetal.MachineImage

		controllerConfig config.ControllerConfiguration
		additionalData   *additionalData
	}
)

func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(metalv1alpha1.SchemeGroupVersion,
		&metalv1alpha1.MetalMachineClass{},
		&metalv1alpha1.MetalMachineClassList{},
	)
	a.scheme = scheme
	a.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

func (a *actuator) InjectConfig(restConfig *rest.Config) error {
	a.restConfig = restConfig
	return nil
}

// NewActuator creates a new Actuator that updates the status of the handled WorkerPoolConfigs.
func NewActuator(machineImages []config.MachineImage, controllerConfig config.ControllerConfiguration) worker.Actuator {
	logger := log.Log.WithName("metal-worker-actuator")

	a := &actuator{
		logger:           logger,
		controllerConfig: controllerConfig,
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
	}

	delegateFactory := &delegateFactory{
		logger: log.Log.WithName("worker-actuator"),
		clientGetter: func() (*rest.Config, client.Client, *runtime.Scheme, runtime.Decoder) {
			return a.restConfig, a.client, a.scheme, a.decoder
		},
		dataGetter:          a.getAdditionalData,
		controllerConfig:    controllerConfig,
		machineImageMapping: machineImages,
	}

	a.workerActuator = genericactuator.NewActuator(
		logger,
		delegateFactory,
		metal.MachineControllerManagerName,
		mcmChart,
		mcmShootChart,
		imagevector.ImageVector(),
		extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
	)

	return a
}

func (a *actuator) Reconcile(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	err := a.firewallReconcile(ctx, worker, cluster)
	if err != nil {
		return err
	}

	return a.workerActuator.Reconcile(ctx, worker, cluster)
}

func (a *actuator) Delete(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	err := a.workerActuator.Delete(ctx, worker, cluster)
	if err != nil {
		return err
	}

	return a.firewallDelete(ctx, cluster)
}

func (a *actuator) Migrate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	err := a.workerActuator.Migrate(ctx, worker, cluster)
	if err != nil {
		return err
	}

	return a.firewallMigrate(ctx, cluster)
}

func (a *actuator) Restore(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	err := a.firewallRestore(ctx, worker, cluster)
	if err != nil {
		return err
	}

	return a.workerActuator.Restore(ctx, worker, cluster)
}

func (d *delegateFactory) WorkerDelegate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (genericactuator.WorkerDelegate, error) {
	config, client, scheme, decoder := d.clientGetter()

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	seedChartApplier, err := gardener.NewChartApplierForConfig(config)
	if err != nil {
		return nil, err
	}

	additionalData, err := d.dataGetter(ctx, worker, cluster)
	if err != nil {
		return nil, err
	}

	return &workerDelegate{
		logger:  d.logger,
		client:  client,
		scheme:  scheme,
		decoder: decoder,

		machineImageMapping: d.machineImageMapping,
		seedChartApplier:    seedChartApplier,
		serverVersion:       serverVersion.GitVersion,

		cluster: cluster,
		worker:  worker,

		controllerConfig: d.controllerConfig,
		additionalData:   additionalData,
	}, nil
}
