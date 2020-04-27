package controlplaneexposure

import (
	"context"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
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
func (e *ensurer) EnsureKubeAPIServerService(ctx context.Context, ectx genericmutator.EnsurerContext, new, old *corev1.Service) error {
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, ectx genericmutator.EnsurerContext, new, old *appsv1.Deployment) error {
	// Get load balancer address of the kube-apiserver service
	address, err := kutil.GetLoadBalancerIngress(ctx, e.client, new.Namespace, v1beta1constants.DeploymentNameKubeAPIServer)
	if err != nil {
		return errors.Wrap(err, "could not get kube-apiserver service load balancer address")
	}

	if c := extensionswebhook.ContainerWithName(new.Spec.Template.Spec.Containers, "kube-apiserver"); c != nil {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--advertise-address=", address)
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--external-hostname=", address)
	}
	return nil
}

// EnsureETCD ensures that the etcd conform to the provider requirements.
func (e *ensurer) EnsureETCD(ctx context.Context, ectx genericmutator.EnsurerContext, new, old *druidv1alpha1.Etcd) error {
	capacity := resource.MustParse("10Gi")
	class := ""

	if new.Name == v1beta1constants.ETCDMain && e.etcdStorage != nil {
		if e.etcdStorage.Capacity != nil {
			capacity = *e.etcdStorage.Capacity
		}
		if e.etcdStorage.ClassName != nil {
			class = *e.etcdStorage.ClassName
		}
	}

	new.Spec.StorageClass = &class
	new.Spec.StorageCapacity = &capacity

	return nil
}
