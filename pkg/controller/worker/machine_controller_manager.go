package worker

import (
	"context"
	"fmt"
	"path/filepath"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"github.com/pkg/errors"

	"github.com/gardener/gardener/pkg/utils/chart"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

var (
	mcmChart = &chart.Chart{
		Name:   metal.MachineControllerManagerName,
		Path:   filepath.Join(metal.InternalChartsPath, metal.MachineControllerManagerName, "seed"),
		Images: []string{metal.MachineControllerManagerImageName, metal.MCMProviderMetalImageName},
		Objects: []*chart.Object{
			{Type: &appsv1.Deployment{}, Name: metal.MachineControllerManagerName},
			{Type: &corev1.Service{}, Name: metal.MachineControllerManagerName},
			{Type: &corev1.ServiceAccount{}, Name: metal.MachineControllerManagerName},
			{Type: &corev1.Secret{}, Name: metal.MachineControllerManagerName},
		},
	}

	mcmShootChart = &chart.Chart{
		Name: metal.MachineControllerManagerName,
		Path: filepath.Join(metal.InternalChartsPath, metal.MachineControllerManagerName, "shoot"),
		Objects: []*chart.Object{
			{Type: &rbacv1.ClusterRole{}, Name: fmt.Sprintf("extensions.gardener.cloud:%s:%s", metal.Name, metal.MachineControllerManagerName)},
			{Type: &rbacv1.ClusterRoleBinding{}, Name: fmt.Sprintf("extensions.gardener.cloud:%s:%s", metal.Name, metal.MachineControllerManagerName)},
		},
	}
)

func (w *workerDelegate) GetMachineControllerManagerChartValues(ctx context.Context) (map[string]interface{}, error) {
	namespace := &corev1.Namespace{}
	if err := w.client.Get(ctx, kutil.Key(w.worker.Namespace), namespace); err != nil {
		return nil, err
	}

	ootDeployment, err := w.isOOTDeployment()
	if err != nil {
		return nil, err
	}

	if !ootDeployment {
		err := w.errorWhenAlreadyMigrated(ctx)
		if err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"providerName": metal.Name,
		"namespace": map[string]interface{}{
			"uid": namespace.UID,
		},
		"deployOOT": ootDeployment,
	}, nil
}

func (w *workerDelegate) isOOTDeployment() (bool, error) {
	controlPlaneConfig := &apismetal.ControlPlaneConfig{}
	if w.cluster != nil && w.cluster.Shoot != nil && w.cluster.Shoot.Spec.Provider.ControlPlaneConfig != nil {
		if _, _, err := w.decoder.Decode(w.cluster.Shoot.Spec.Provider.ControlPlaneConfig.Raw, nil, controlPlaneConfig); err != nil {
			return false, errors.Wrapf(err, "could not decode providerConfig of control plane")
		}
	}

	if controlPlaneConfig.FeatureGates.MachineControllerManagerOOT != nil && *controlPlaneConfig.FeatureGates.MachineControllerManagerOOT {
		return true, nil
	}

	return false, nil
}

func (w *workerDelegate) GetMachineControllerManagerShootChartValues(ctx context.Context) (map[string]interface{}, error) {
	ootDeployment, err := w.isOOTDeployment()
	if err != nil {
		return nil, err
	}

	if !ootDeployment {
		err := w.errorWhenAlreadyMigrated(ctx)
		if err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"providerName": metal.Name,
		"deployOOT":    ootDeployment,
	}, nil
}
