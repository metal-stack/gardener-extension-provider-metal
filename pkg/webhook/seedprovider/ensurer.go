package seedprovider

import (
	"context"
	"fmt"
	"time"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
)

// NewEnsurer creates a new seedprovider ensurer.
func NewEnsurer(mgr manager.Manager, etcdStorage *config.ETCD, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		c:      etcdStorage,
		client: mgr.GetClient(),
		logger: logger.WithName("metal-seedprovider-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	c      *config.ETCD
	client client.Client
	logger logr.Logger
}

// EnsureETCD ensures that the etcd conform to the provider requirements.
func (e *ensurer) EnsureETCD(ctx context.Context, gctx gcontext.GardenContext, new, old *druidv1alpha1.Etcd) error {
	new.Spec.StorageCapacity = pointer.Pointer(resource.MustParse("16Gi"))

	if e.c == nil {
		return nil
	}

	if new.Name == v1beta1constants.ETCDMain {
		if old == nil {
			// capacity and storage class can only be set on initial deployment
			// after that the stateful set prevents the update.

			if e.c.Storage.Capacity != nil {
				new.Spec.StorageCapacity = e.c.Storage.Capacity
			}
			if e.c.Storage.ClassName != nil {
				new.Spec.StorageClass = e.c.Storage.ClassName
			}
		} else {
			// ensure values stay as the were

			new.Spec.StorageCapacity = old.Spec.StorageCapacity
			new.Spec.StorageClass = old.Spec.StorageClass
		}

		if e.c.Backup.DeltaSnapshotPeriod != nil {
			d, err := time.ParseDuration(*e.c.Backup.DeltaSnapshotPeriod)
			if err != nil {
				return fmt.Errorf("unable to set delta snapshot period %w", err)
			}
			new.Spec.Backup.DeltaSnapshotPeriod = &v1.Duration{Duration: d}
		}

		if e.c.Backup.Schedule != nil {
			new.Spec.Backup.FullSnapshotSchedule = e.c.Backup.Schedule
		}
	}

	return nil
}
