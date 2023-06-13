package worker

import (
	"context"
	"fmt"

	api "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	additionalData struct {
		privateNetworkID     string
		infrastructure       *extensionsv1alpha1.Infrastructure
		infrastructureConfig *apismetal.InfrastructureConfig
		mcp                  *apismetal.MetalControlPlane
		credentials          *metal.Credentials
		mclient              metalgo.Client
	}

	key int

	cacheKey struct {
		nodeCIDR  string
		projectID string
	}
)

const (
	ClientKey key = iota
)

func (a *actuator) getAdditionalData(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (*additionalData, error) {
	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return nil, err
	}

	credentials, err := metalclient.ReadCredentialsFromSecretRef(ctx, a.client, &worker.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	mclient, err := metalclient.NewClientFromCredentials(metalControlPlane.Endpoint, credentials)
	if err != nil {
		return nil, err
	}

	// TODO: this is a workaround to speed things for the time being...
	// the infrastructure controller writes the nodes cidr back into the infrastructure status, but the cluster resource does not contain it immediately
	// it would need the start of another reconcilation until the node cidr can be picked up from the cluster resource
	// therefore, we read it directly from the infrastructure status
	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := a.client.Get(ctx, kutil.Key(worker.Namespace, cluster.Shoot.Name), infrastructure); err != nil {
		return nil, err
	}

	projectID := infrastructureConfig.ProjectID

	nodeCIDR, err := helper.GetNodeCIDR(infrastructure, cluster)
	if err != nil {
		return nil, err
	}

	nw, err := a.networkCache.Get(context.WithValue(ctx, ClientKey, mclient), &cacheKey{
		projectID: projectID,
		nodeCIDR:  nodeCIDR,
	})
	if err != nil {
		return nil, err
	}

	if nw.ID == nil {
		return nil, fmt.Errorf("private network id is nil")
	}

	return &additionalData{
		mcp:                  metalControlPlane,
		infrastructure:       infrastructure,
		infrastructureConfig: infrastructureConfig,
		privateNetworkID:     *nw.ID,
		credentials:          credentials,
		mclient:              mclient,
	}, nil
}

func (w *workerDelegate) decodeWorkerProviderStatus() (*api.WorkerStatus, error) {
	workerStatus := &api.WorkerStatus{}

	if w.worker.Status.ProviderStatus == nil {
		return workerStatus, nil
	}

	if _, _, err := w.decoder.Decode(w.worker.Status.ProviderStatus.Raw, nil, workerStatus); err != nil {
		return nil, fmt.Errorf("could not decode WorkerStatus '%s' %w", kutil.ObjectName(w.worker), err)
	}

	return workerStatus, nil
}

func (w *workerDelegate) updateWorkerProviderStatus(ctx context.Context, workerStatus *api.WorkerStatus) error {
	var workerStatusV1alpha1 = &v1alpha1.WorkerStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "WorkerStatus",
		},
	}

	if err := w.scheme.Convert(workerStatus, workerStatusV1alpha1, nil); err != nil {
		return err
	}
	patch := cclient.MergeFrom(w.worker.DeepCopy())
	w.worker.Status.ProviderStatus = &runtime.RawExtension{Object: workerStatusV1alpha1}
	return w.client.Status().Patch(ctx, w.worker, patch)
}
