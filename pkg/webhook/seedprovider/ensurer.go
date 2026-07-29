package seedprovider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	druidcorev1alpha1 "github.com/gardener/etcd-druid/api/core/v1alpha1"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
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
func (e *ensurer) EnsureETCD(ctx context.Context, gctx gcontext.GardenContext, newObj, old *druidcorev1alpha1.Etcd) error {
	newObj.Spec.StorageCapacity = new(resource.MustParse("16Gi"))

	if e.c == nil {
		return nil
	}

	if newObj.Name == v1beta1constants.ETCDMain {
		if old == nil {
			// capacity and storage class can only be set on initial deployment
			// after that the stateful set prevents the update.

			if e.c.Storage.Capacity != nil {
				newObj.Spec.StorageCapacity = e.c.Storage.Capacity
			}
			if e.c.Storage.ClassName != nil {
				newObj.Spec.StorageClass = e.c.Storage.ClassName
			}
		} else {
			// ensure values stay as the were

			newObj.Spec.StorageCapacity = old.Spec.StorageCapacity
			newObj.Spec.StorageClass = old.Spec.StorageClass
		}

		if e.c.Backup.DeltaSnapshotPeriod != nil {
			d, err := time.ParseDuration(*e.c.Backup.DeltaSnapshotPeriod)
			if err != nil {
				return fmt.Errorf("unable to set delta snapshot period %w", err)
			}
			newObj.Spec.Backup.DeltaSnapshotPeriod = &v1.Duration{Duration: d}
		}

		if e.c.Backup.Schedule != nil {
			newObj.Spec.Backup.FullSnapshotSchedule = e.c.Backup.Schedule
		}
	}

	if e.c.IsEvictionAllowed {
		if newObj.Spec.Annotations == nil {
			newObj.Spec.Annotations = map[string]string{}
		}
		newObj.Spec.Annotations["metal-stack.io/csi-driver-lvm.is-eviction-allowed"] = strconv.FormatBool(true)

		if newObj.Annotations == nil {
			newObj.Annotations = map[string]string{}
		}
		newObj.Annotations[druidcorev1alpha1.DisableEtcdComponentProtectionAnnotation] = strconv.FormatBool(true)
	}

	return nil
}
