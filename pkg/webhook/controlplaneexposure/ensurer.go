package controlplaneexposure

import (
	"context"
	"fmt"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	v1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
)

// NewEnsurer creates a new controlplaneexposure ensurer.
func NewEnsurer(mgr manager.Manager, etcdStorage *config.ETCD, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		c:      etcdStorage,
		client: mgr.GetClient(),
		logger: logger.WithName("metal-controlplaneexposure-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	c      *config.ETCD
	client client.Client
	logger logr.Logger
}

// EnsureKubeAPIServerService ensures that the kube-apiserver service conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerService(ctx context.Context, gctx gcontext.GardenContext, new, old *corev1.Service) error {
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
		return fmt.Errorf("could not get kube-apiserver service load balancer address %w", err)
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

	defer func() {
		new.Spec.StorageClass = &class
		new.Spec.StorageCapacity = &capacity
	}()

	if e.c == nil {
		return nil
	}

	if old == nil {
		// capacity and storage class can only be set on initial deployment
		// after that the stateful set prevents the update.

		if e.c.Storage.Capacity != nil {
			capacity = *e.c.Storage.Capacity
			e.logger.Info("mutating capacity", "capacity", capacity)
		}
		if e.c.Storage.ClassName != nil {
			class = *e.c.Storage.ClassName
			e.logger.Info("mutating storage class", "class", class)
		}
	}

	if new.Name == v1beta1constants.ETCDMain {
		if e.c.Backup.DeltaSnapshotPeriod != nil {
			d, err := time.ParseDuration(*e.c.Backup.DeltaSnapshotPeriod)
			if err != nil {
				return fmt.Errorf("unable to set delta snapshot period %w", err)
			}
			new.Spec.Backup.DeltaSnapshotPeriod = &v1.Duration{Duration: d}

			e.logger.Info("mutating delta snapshot period", "period", d.String())
		}

		if e.c.Backup.Schedule != nil {
			new.Spec.Backup.FullSnapshotSchedule = e.c.Backup.Schedule
		}
	}

	return nil
}
