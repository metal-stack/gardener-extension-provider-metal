package controlplane

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *actuator) csiLVMReconcile(ctx context.Context, _ logr.Logger, _ *extensionsv1alpha1.ControlPlane, _ *extensionscontroller.Cluster) error {

	name := "csi-lvm"
	provisioner := "metal-stack.io/csi-lvm"

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	err := a.client.Get(ctx, client.ObjectKeyFromObject(namespace), namespace)
	if err == nil {
		a.client.Delete(ctx, namespace)
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("error while getting csi-lvm namespace: %w", err)
	}

	storageClass := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Provisioner: provisioner,
	}

	err = a.client.Get(ctx, client.ObjectKeyFromObject(storageClass), storageClass)
	if err == nil {
		a.client.Delete(ctx, storageClass)
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("error while getting csi-lvm storageclass: %w", err)
	}

	return nil

}
