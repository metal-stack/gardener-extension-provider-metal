package controlplaneexposure

import (
	"context"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
)

// NewEnsurer creates a new controlplaneexposure ensurer.
func NewEnsurer(etcdStorage *config.ETCD, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		c:      etcdStorage,
		logger: logger.WithName("metal-controlplaneexposure-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	c      *config.ETCD
	client client.Client
	logger logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureKubeAPIServerService ensures that the kube-apiserver service conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerService(ctx context.Context, gctx gcontext.GardenContext, new, old *corev1.Service) error {
	cluster, _ := gctx.GetCluster(ctx)

	if old != nil {
		e.logger.Info("kube-apiserver service externalTrafficPolicy of", "shoot", cluster.ObjectMeta.Name, "old", old.Spec.ExternalTrafficPolicy, "new", new.Spec.ExternalTrafficPolicy)
		e.logger.Info("kube-apiserver service healthCheckNodePort of", "shoot", cluster.ObjectMeta.Name, "old", old.Spec.HealthCheckNodePort, "new", new.Spec.HealthCheckNodePort)
	} else {
		e.logger.Info("kube-apiserver service externalTrafficPolicy of", "shoot", cluster.ObjectMeta.Name, "old", "nil", "new", new.Spec.ExternalTrafficPolicy)
		e.logger.Info("kube-apiserver service healthCheckNodePort of", "shoot", cluster.ObjectMeta.Name, "old", "nil", "new", new.Spec.HealthCheckNodePort)
	}
	// We ensure that the externalTrafficPolicy of the kube-apiserver service is Local, and that the healthCheckNodePort is preserved.
	if new.Spec.ExternalTrafficPolicy != corev1.ServiceExternalTrafficPolicyType("Local") {
		e.logger.Info("kube-apiserver service externalTrafficPolicy of", "shoot", cluster.ObjectMeta.Name, "now", new.Spec.ExternalTrafficPolicy, "changing to", corev1.ServiceExternalTrafficPolicyType("Local"))
		new.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyType("Local")
	}
	if new.Spec.HealthCheckNodePort == 0 {
		if old != nil {
			e.logger.Info("kube-apiserver service healthCheckNodePort of", "shoot", cluster.ObjectMeta.Name, "now", new.Spec.HealthCheckNodePort, "changing to", old.Spec.HealthCheckNodePort)
			new.Spec.HealthCheckNodePort = old.Spec.HealthCheckNodePort
		} else {
			e.logger.Info("new kube-apiserver service healthCheckNodePort is 0, can not set it because there is no old kube-apiserver service (nil). Hope that's okay.")
		}
	}

	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, old *appsv1.Deployment) error {
	// ignore gardener managed (APIServerSNI-enabled) apiservers.
	if v1beta1helper.IsAPIServerExposureManaged(new) {
		return nil
	}

	// Get load balancer address of the kube-apiserver service
	address, err := kutil.GetLoadBalancerIngress(ctx, e.client, &corev1.Service{ObjectMeta: v1.ObjectMeta{Namespace: new.Namespace, Name: v1beta1constants.DeploymentNameKubeAPIServer}})
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
func (e *ensurer) EnsureETCD(ctx context.Context, gctx gcontext.GardenContext, new, old *druidv1alpha1.Etcd) error {
	capacity := resource.MustParse("16Gi")
	class := ""

	if new.Name == v1beta1constants.ETCDMain && e.c != nil {
		if e.c.Storage.Capacity != nil {
			capacity = *e.c.Storage.Capacity
		}
		if e.c.Storage.ClassName != nil {
			class = *e.c.Storage.ClassName
		}
		if e.c.Backup.DeltaSnapshotPeriod != nil {
			d, err := time.ParseDuration(*e.c.Backup.DeltaSnapshotPeriod)
			if err != nil {
				return errors.Wrap(err, "unable to set delta snapshot period")
			}
			new.Spec.Backup.DeltaSnapshotPeriod = &v1.Duration{Duration: d}
		}
		if e.c.Backup.Schedule != nil {
			new.Spec.Backup.FullSnapshotSchedule = e.c.Backup.Schedule
		}
	}

	new.Spec.StorageClass = &class
	new.Spec.StorageCapacity = &capacity

	return nil
}
