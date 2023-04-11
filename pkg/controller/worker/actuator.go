package worker

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/util"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	"github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalv1alpha1 "github.com/metal-stack/machine-controller-manager-provider-metal/pkg/provider/migration/legacy-api/machine/v1alpha1"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/client/kubernetes"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type delegateFactory struct {
	logger logr.Logger

	restConfig *rest.Config

	client  client.Client
	scheme  *runtime.Scheme
	decoder runtime.Decoder

	machineImageMapping []config.MachineImage
	controllerConfig    config.ControllerConfiguration
}

// NewActuator creates a new Actuator that updates the status of the handled WorkerPoolConfigs.
func NewActuator(machineImages []config.MachineImage, controllerConfig config.ControllerConfiguration) worker.Actuator {
	delegateFactory := &delegateFactory{
		logger:              log.Log.WithName("worker-actuator"),
		machineImageMapping: machineImages,
		controllerConfig:    controllerConfig,
	}
	return genericactuator.NewActuator(
		log.Log.WithName("metal-worker-actuator"),
		delegateFactory,
		metal.MachineControllerManagerName,
		mcmChart,
		mcmShootChart,
		imagevector.ImageVector(),
		extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
		false,
		false,
	)
}

func (d *delegateFactory) InjectScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(metalv1alpha1.SchemeGroupVersion,
		&metalv1alpha1.MetalMachineClass{},
		&metalv1alpha1.MetalMachineClassList{},
	)
	d.scheme = scheme
	d.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (d *delegateFactory) InjectConfig(restConfig *rest.Config) error {
	d.restConfig = restConfig
	return nil
}

func (d *delegateFactory) InjectClient(client client.Client) error {
	d.client = client
	return nil
}

func (d *delegateFactory) WorkerDelegate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (genericactuator.WorkerDelegate, error) {
	clientset, err := kubernetes.NewForConfig(d.restConfig)
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	seedChartApplier, err := gardener.NewChartApplierForConfig(d.restConfig)
	if err != nil {
		return nil, err
	}

	return NewWorkerDelegate(
		d.logger,
		d.client,
		d.scheme,
		d.decoder,

		d.machineImageMapping,
		seedChartApplier,
		serverVersion.GitVersion,

		worker,
		cluster,
		d.controllerConfig,
	), nil
}

type workerDelegate struct {
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
}

// NewWorkerDelegate creates a new context for a worker reconciliation.
func NewWorkerDelegate(
	logger logr.Logger,
	client client.Client,
	scheme *runtime.Scheme,
	decoder runtime.Decoder,

	machineImageMapping []config.MachineImage,
	seedChartApplier gardener.ChartApplier,
	serverVersion string,

	worker *extensionsv1alpha1.Worker,
	cluster *extensionscontroller.Cluster,
	controllerConfig config.ControllerConfiguration,

) genericactuator.WorkerDelegate {
	return &workerDelegate{
		logger:  logger,
		client:  client,
		scheme:  scheme,
		decoder: decoder,

		machineImageMapping: machineImageMapping,
		seedChartApplier:    seedChartApplier,
		serverVersion:       serverVersion,

		cluster: cluster,
		worker:  worker,
	}
}
