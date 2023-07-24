package client

import (
	"context"
	"fmt"
	"strings"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/client/firewall"
	metalip "github.com/metal-stack/metal-go/api/client/ip"
	"github.com/metal-stack/metal-go/api/client/network"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/tag"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient returns a new metal client with the provider credentials from a given secret reference.
func NewClient(ctx context.Context, k8sClient client.Client, endpoint string, secretRef *corev1.SecretReference) (metalgo.Client, error) {
	credentials, err := ReadCredentialsFromSecretRef(ctx, k8sClient, secretRef)
	if err != nil {
		return nil, err
	}

	return NewClientFromCredentials(endpoint, credentials)
}

// NewClientFromCredentials returns a new metal client with the client constructed from the given credentials.
func NewClientFromCredentials(endpoint string, credentials *metal.Credentials) (metalgo.Client, error) {
	client, err := metalgo.NewDriver(endpoint, credentials.MetalAPIKey, credentials.MetalAPIHMac)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ReadCredentialsFromSecretRef returns metal credentials from the provider credentials from a given secret reference.
func ReadCredentialsFromSecretRef(ctx context.Context, k8sClient client.Client, secretRef *corev1.SecretReference) (*metal.Credentials, error) {
	providerSecret, err := extensionscontroller.GetSecretByReference(ctx, k8sClient, secretRef)
	if err != nil {
		return nil, err
	}

	credentials, err := metal.ReadCredentialsSecret(providerSecret)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// GetPrivateNetworksFromNodeNetwork returns the private network that belongs to the given node network cidr and project.
func GetPrivateNetworksFromNodeNetwork(ctx context.Context, client metalgo.Client, projectID string, nodeNetworkCIDR string) ([]*models.V1NetworkResponse, error) {
	if nodeNetworkCIDR == "" {
		return nil, fmt.Errorf("node network cidr is empty")
	}

	networkFindRequest := &models.V1NetworkFindRequest{
		Projectid: projectID,
		Prefixes:  []string{nodeNetworkCIDR},
	}
	networkFindResponse, err := client.Network().FindNetworks(network.NewFindNetworksParams().WithBody(networkFindRequest).WithContext(ctx), nil)
	if err != nil {
		return nil, err
	}

	return networkFindResponse.Payload, nil
}

// GetPrivateNetworkFromNodeNetwork returns the private network that belongs to the given node network cidr and project.
func GetPrivateNetworkFromNodeNetwork(ctx context.Context, client metalgo.Client, projectID string, nodeNetworkCIDR string) (*models.V1NetworkResponse, error) {
	privateNetworks, err := GetPrivateNetworksFromNodeNetwork(ctx, client, projectID, nodeNetworkCIDR)
	if err != nil {
		return nil, err
	}
	if len(privateNetworks) != 1 {
		return nil, fmt.Errorf("no distinct private network for project id %q and prefix %s found", projectID, nodeNetworkCIDR)
	}
	return privateNetworks[0], nil
}

// GetEphemeralIPsFromCluster return all ephemeral IPs for given project and cluster
func GetEphemeralIPsFromCluster(ctx context.Context, client metalgo.Client, projectID, clusterID string) ([]*models.V1IPResponse, []*models.V1IPResponse, error) {
	ipFindResponse, err := client.IP().FindIPs(metalip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
		Projectid: projectID,
		Type:      models.V1IPBaseTypeEphemeral,
	}).WithContext(ctx), nil)
	if err != nil {
		return nil, nil, err
	}

	// only these who are member of one cluster are freed
	ipsToFree := []*models.V1IPResponse{}
	// those who are member of more clusters must be updated and the tags which references this cluster must be removed.
	ipsToUpdate := []*models.V1IPResponse{}
	for _, ip := range ipFindResponse.Payload {
		clusterCount := 0
		for _, t := range ip.Tags {

			if isMemberOfCluster(t, clusterID) {
				clusterCount++
			}
		}
		if clusterCount == 1 {
			ipsToFree = append(ipsToFree, ip)
			continue
		}
		// IPs which are used in more than one cluster must be updated to get the tags with this clusterid removed
		ipsToUpdate = append(ipsToUpdate, ip)
	}
	return ipsToFree, ipsToUpdate, nil
}

// UpdateIPInCluster update the IP in the cluster to have only these tags left which are not from this cluster
func UpdateIPInCluster(ctx context.Context, client metalgo.Client, ip *models.V1IPResponse, clusterID string) error {
	var newTags []string
	for _, t := range ip.Tags {
		if strings.HasPrefix(t, tag.ClusterServiceFQN+"="+clusterID) {
			continue
		}
		newTags = append(newTags, t)
	}

	_, err := client.IP().UpdateIP(metalip.NewUpdateIPParams().WithBody(&models.V1IPUpdateRequest{
		Ipaddress: ip.Ipaddress,
		Tags:      newTags,
	}).WithContext(ctx), nil)
	if err != nil {
		return err
	}

	return nil
}

func isMemberOfCluster(t, clusterID string) bool {
	if strings.HasPrefix(t, tag.ClusterID) {
		parts := strings.Split(t, "=")
		if len(parts) != 2 {
			return false
		}
		if strings.HasPrefix(parts[1], clusterID) {
			return true
		}
	}
	return false
}

func FindClusterFirewalls(ctx context.Context, client metalgo.Client, clusterTag, projectID string) ([]*models.V1FirewallResponse, error) {
	resp, err := client.Firewall().FindFirewalls(firewall.NewFindFirewallsParams().WithBody(&models.V1FirewallFindRequest{
		AllocationProject: projectID,
		Tags:              []string{clusterTag},
	}).WithContext(ctx), nil)
	if err != nil {
		return nil, err
	}

	return resp.Payload, nil
}
