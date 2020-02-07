dpackage controlplaneexposure

import (
	"context"

	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
)

// NewEnsurer creates a new controlplaneexposure ensurer.
func NewEnsurer(etcdStorage *config.ETCDStorage, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		etcdStorage: etcdStorage,
		logger:      logger.WithName("metal-controlplaneexposure-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	etcdStorage *config.ETCDStorage
	client      client.Client
	logger      logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureKubeAPIServerService ensures that the kube-apiserver service conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerService(ctx context.Context, ectx genericmutator.EnsurerContext, svc *corev1.Service) error {
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, ectx genericmutator.EnsurerContext, dep *appsv1.Deployment) error {
	// Get load balancer address of the kube-apiserver service
	address, err := kutil.GetLoadBalancerIngress(ctx, e.client, dep.Namespace, v1beta1constants.DeploymentNameKubeAPIServer)
	if err != nil {
		return errors.Wrap(err, "could not get kube-apiserver service load balancer address")
	}

	if c := extensionswebhook.ContainerWithName(dep.Spec.Template.Spec.Containers, "kube-apiserver"); c != nil {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--advertise-address=", address)
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--external-hostname=", address)
	}
	return nil
}

// EnsureETCDStatefulSet ensures that the etcd stateful sets conform to the provider requirements.
func (e *ensurer) EnsureETCDStatefulSet(ctx context.Context, ectx genericmutator.EnsurerContext, ss *appsv1.StatefulSet) error {
	e.ensureVolumeClaimTemplates(&ss.Spec, ss.Name)
	return nil
}

func (e *ensurer) ensureVolumeClaimTemplates(spec *appsv1.StatefulSetSpec, name string) {
	// use default storage class
	// t := e.getVolumeClaimTemplate(name)
	// spec.VolumeClaimTemplates = controlplane.EnsurePVCWithName(spec.VolumeClaimTemplates, *t)
}

func (e *ensurer) getVolumeClaimTemplate(name string) *corev1.PersistentVolumeClaim {
	var (
		etcdStorage             config.ETCDStorage
		volumeClaimTemplateName = name
	)

	if name == v1beta1constants.ETCDMain {
		etcdStorage = *e.etcdStorage
		volumeClaimTemplateName = controlplane.EtcdMainVolumeClaimTemplateName
	}

	return controlplane.GetETCDVolumeClaimTemplate(volumeClaimTemplateName, etcdStorage.ClassName, etcdStorage.Capacity)
}
