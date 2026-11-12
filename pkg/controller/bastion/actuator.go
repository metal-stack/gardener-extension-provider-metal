package bastion

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"
)

type actuator struct {
	client              client.Client
	decoder             runtime.Decoder
	machineImageMapping []config.MachineImage
}

// newActuator creates a new bastion actuator.
func newActuator(mgr manager.Manager, machineImages []config.MachineImage) bastion.Actuator {
	return &actuator{
		client:              mgr.GetClient(),
		decoder:             serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
		machineImageMapping: machineImages,
	}
}

// clusterData bundles all data required to talk to the metal-api of the shoot's control plane.
type clusterData struct {
	cloudProfileConfig   *apismetal.CloudProfileConfig
	infrastructureConfig *apismetal.InfrastructureConfig
	mclient              metalgo.Client
	projectID            string
}

// getClusterData resolves the cloud profile, infrastructure config, metal control plane and a metal client
// for the given bastion.
func (a *actuator) getClusterData(ctx context.Context, bastion *extensionsv1alpha1.Bastion, cluster *extensionscontroller.Cluster) (*clusterData, error) {
	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	if cloudProfileConfig == nil {
		return nil, fmt.Errorf("cloud profile config is not set")
	}

	infrastructureConfig, err := getInfrastructureConfig(a.decoder, cluster)
	if err != nil {
		return nil, err
	}

	mcp, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return nil, err
	}

	secretRef := &corev1.SecretReference{
		Namespace: bastion.Namespace,
		Name:      v1beta1constants.SecretNameCloudProvider,
	}
	mclient, err := metalclient.NewClient(ctx, a.client, mcp.Endpoint, secretRef)
	if err != nil {
		return nil, fmt.Errorf("failed to create metal client: %w", err)
	}

	return &clusterData{
		cloudProfileConfig:   cloudProfileConfig,
		infrastructureConfig: infrastructureConfig,
		mclient:              mclient,
		projectID:            infrastructureConfig.ProjectID,
	}, nil
}

// getPrivateNetworkID resolves the private (node) network id the bastion should be allocated in.
func (a *actuator) getPrivateNetworkID(ctx context.Context, data *clusterData, bastion *extensionsv1alpha1.Bastion, cluster *extensionscontroller.Cluster) (string, error) {
	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := a.client.Get(ctx, client.ObjectKey{Namespace: bastion.Namespace, Name: cluster.Shoot.Name}, infrastructure); err != nil {
		return "", fmt.Errorf("failed to get infrastructure resource: %w", err)
	}

	nodeCIDR, err := helper.GetNodeCIDR(infrastructure, cluster)
	if err != nil {
		return "", err
	}

	nw, err := metalclient.GetPrivateNetworkFromNodeNetwork(ctx, data.mclient, data.projectID, nodeCIDR)
	if err != nil {
		return "", err
	}
	if nw.ID == nil {
		return "", fmt.Errorf("private network id is nil")
	}

	return *nw.ID, nil
}

// getInfrastructureConfig decodes the infrastructure config from the shoot spec.
func getInfrastructureConfig(decoder runtime.Decoder, cluster *extensionscontroller.Cluster) (*apismetal.InfrastructureConfig, error) {
	if cluster == nil || cluster.Shoot == nil || cluster.Shoot.Spec.Provider.InfrastructureConfig == nil || cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw == nil {
		return nil, fmt.Errorf("infrastructure config is not set on the shoot")
	}

	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := decoder.Decode(cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, fmt.Errorf("could not decode infrastructure config: %w", err)
	}

	return infrastructureConfig, nil
}
