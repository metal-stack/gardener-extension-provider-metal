package metalgo

import (
	"fmt"

	"github.com/metal-pod/metal-go/api/client/ip"
	"github.com/metal-pod/metal-go/api/client/network"
	"github.com/metal-pod/metal-go/api/models"
)

// NetworkGetResponse contains the network get result
type NetworkGetResponse struct {
	Network *models.V1NetworkResponse
}

// NetworkListResponse is the response of a NetworkList action
type NetworkListResponse struct {
	Networks []*models.V1NetworkResponse
}

// NetworkCreateRequest is the request for create a new network
type NetworkCreateRequest struct {
	ID *string `json:"id"`
	// a description for this entity
	Description string `json:"description,omitempty"`

	// the readable name
	Name string `json:"name,omitempty"`

	// if set to true, packets leaving this network get masqueraded behind interface ip.
	// Required: true
	Nat bool `json:"nat"`

	// the partition this network belongs to, TODO: can be empty ?
	// Required: true
	Partitionid string `json:"partitionid"`

	// the prefixes of this network, required.
	// Required: true
	Prefixes []string `json:"prefixes"`

	// the destination prefixes of this network
	// Required: true
	Destinationprefixes []string `json:"destinationprefixes"`

	// if set to true, this network is attached to a machine/firewall
	// Required: true
	Primary bool `json:"primary"`

	// the project this network belongs to, can be empty if globally available.
	// Required: true
	Projectid string `json:"projectid"`

	// if set to true, this network can be used for underlay communication
	// Required: true
	Underlay bool `json:"underlay"`

	// the vrf this network is associated with
	Vrf int64 `json:"vrf,omitempty"`
}

// NetworkDetailResponse is the response of a NetworkList action
type NetworkDetailResponse struct {
	Network *models.V1NetworkResponse
}

// NetworkUpdateRequest is the request to update the Network
type NetworkUpdateRequest struct {
	// the network id for this update request.
	Networkid string `json:"networkid"`
	// Prefix the prefix to add/remove
	Prefix string
}

// IPUpdateRequest is the request to update an IP
type IPUpdateRequest struct {
	// the ip address for this ip update request.
	IPAddress string `json:"ipaddress"`
	// a description for this entity
	Description string `json:"description,omitempty"`
	// the readable name
	Name string `json:"name,omitempty"`
	// the machine id that is associated to this ip
	MachineID string `json:"machineid,omitempty"`
}

// IPListResponse is the response when ips are listed
type IPListResponse struct {
	IPs []*models.V1IPResponse
}

// IPAcquireRequest is the request to acquire an IP
type IPAcquireRequest struct {

	// a description for this entity
	Description string `json:"description,omitempty"`

	// the readable name
	Name string `json:"name,omitempty"`

	// the network this ip allocate request address belongs to, required.
	// Required: true
	Networkid string `json:"networkid"`

	// the project this ip address belongs to, required.
	// Required: true
	Projectid string `json:"projectid"`
	// SpecificIP tries to acquire this ip.
	// Required: false
	SpecificIP string `json:"specificip"`
}

// NetworkFindRequest contains criteria for a network listing
type NetworkFindRequest struct {
	ID                  *string
	Name                *string
	PartitionID         *string
	ProjectID           *string
	Prefixes            []string
	DestinationPrefixes []string
	Nat                 *bool
	Primary             *bool
	Underlay            *bool
	Vrf                 *int64
	ParentNetworkID     *string
	TenantID            *string
}

// IPFindRequest contains criteria for a ip listing
type IPFindRequest struct {
	IPAddress        *string
	ProjectID        *string
	ParentPrefixCidr *string
	NetworkID        *string
	MachineID        *string
}

// IPDetailResponse is the response to an IP detail request.
type IPDetailResponse struct {
	IP *models.V1IPResponse
}

// NetworkGet returns the network with the given ID
func (d *Driver) NetworkGet(id string) (*NetworkGetResponse, error) {
	findNetwork := network.NewFindNetworkParams()
	findNetwork.ID = id

	response := &NetworkGetResponse{}
	resp, err := d.network.FindNetwork(findNetwork, d.auth)
	if err != nil {
		return response, err
	}
	response.Network = resp.Payload

	return response, nil
}

// NetworkList returns all networks
func (d *Driver) NetworkList() (*NetworkListResponse, error) {
	response := &NetworkListResponse{}
	listNetworks := network.NewListNetworksParams()
	resp, err := d.network.ListNetworks(listNetworks, d.auth)
	if err != nil {
		return response, err
	}
	response.Networks = resp.Payload
	return response, nil
}

// NetworkFind returns all networks that match given properties
func (d *Driver) NetworkFind(nfr *NetworkFindRequest) (*NetworkListResponse, error) {
	if nfr == nil {
		return d.NetworkList()
	}

	response := &NetworkListResponse{}
	var err error
	var resp *network.FindNetworksOK

	findNetworks := network.NewFindNetworksParams()
	req := &models.V1FindNetworksRequest{
		ID:                  nfr.ID,
		Name:                nfr.Name,
		Partitionid:         nfr.PartitionID,
		Projectid:           nfr.ProjectID,
		Prefixes:            nfr.Prefixes,
		Destinationprefixes: nfr.DestinationPrefixes,
		Nat:                 nfr.Nat,
		Primary:             nfr.Primary,
		Underlay:            nfr.Underlay,
		Vrf:                 nfr.Vrf,
		Parentnetworkid:     nfr.ParentNetworkID,
		Tenantid:            nfr.TenantID,
	}
	findNetworks.SetBody(req)

	resp, err = d.network.FindNetworks(findNetworks, d.auth)
	if err != nil {
		return response, err
	}
	response.Networks = resp.Payload

	return response, nil
}

// NetworkCreate creates a new network
func (d *Driver) NetworkCreate(ncr *NetworkCreateRequest) (*NetworkDetailResponse, error) {
	response := &NetworkDetailResponse{}
	createNetwork := network.NewCreateNetworkParams()

	createRequest := &models.V1NetworkCreateRequest{
		ID:                  ncr.ID,
		Description:         ncr.Description,
		Name:                ncr.Name,
		Nat:                 &ncr.Nat,
		Partitionid:         ncr.Partitionid,
		Prefixes:            ncr.Prefixes,
		Destinationprefixes: ncr.Destinationprefixes,
		Vrf:                 ncr.Vrf,
		Primary:             &ncr.Primary,
		Projectid:           ncr.Projectid,
		Underlay:            &ncr.Underlay,
	}
	createNetwork.SetBody(createRequest)
	resp, err := d.network.CreateNetwork(createNetwork, d.auth)
	if err != nil {
		return response, err
	}
	response.Network = resp.Payload
	return response, nil
}

// NetworkUpdate creates a new network
func (d *Driver) NetworkUpdate(ncr *NetworkCreateRequest) (*NetworkDetailResponse, error) {
	response := &NetworkDetailResponse{}
	updateNetwork := network.NewUpdateNetworkParams()

	updateRequest := &models.V1NetworkUpdateRequest{
		ID:          ncr.ID,
		Description: ncr.Description,
		Name:        ncr.Name,
		Prefixes:    ncr.Prefixes,
	}
	updateNetwork.SetBody(updateRequest)
	resp, err := d.network.UpdateNetwork(updateNetwork, d.auth)
	if err != nil {
		return response, err
	}
	response.Network = resp.Payload
	return response, nil
}

// NetworkAddPrefix adds a prefix to a network
func (d *Driver) NetworkAddPrefix(nur *NetworkUpdateRequest) (*NetworkDetailResponse, error) {
	old, err := d.NetworkGet(nur.Networkid)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch network: %s to update:%v", nur.Networkid, err)
	}
	oldNetwork := old.Network
	newPrefixes := append(oldNetwork.Prefixes, nur.Prefix)

	response := &NetworkDetailResponse{}
	updateNetwork := network.NewUpdateNetworkParams()
	updateRequest := &models.V1NetworkUpdateRequest{
		ID:       &nur.Networkid,
		Prefixes: newPrefixes,
	}
	updateNetwork.SetBody(updateRequest)
	resp, err := d.network.UpdateNetwork(updateNetwork, d.auth)
	if err != nil {
		return response, err
	}
	response.Network = resp.Payload
	return response, nil
}

// NetworkRemovePrefix removes a prefix from a network
func (d *Driver) NetworkRemovePrefix(nur *NetworkUpdateRequest) (*NetworkDetailResponse, error) {
	old, err := d.NetworkGet(nur.Networkid)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch network: %s to update:%v", nur.Networkid, err)
	}
	oldNetwork := old.Network
	var newPrefixes []string
	for _, p := range oldNetwork.Prefixes {
		if p == nur.Prefix {
			continue
		}
		newPrefixes = append(newPrefixes, p)
	}

	response := &NetworkDetailResponse{}
	updateNetwork := network.NewUpdateNetworkParams()
	updateRequest := &models.V1NetworkUpdateRequest{
		ID:       &nur.Networkid,
		Prefixes: newPrefixes,
	}
	updateNetwork.SetBody(updateRequest)
	resp, err := d.network.UpdateNetwork(updateNetwork, d.auth)
	if err != nil {
		return response, err
	}
	response.Network = resp.Payload
	return response, nil
}

// IPGet gets a given IP
func (d *Driver) IPGet(ipaddress string) (*IPDetailResponse, error) {
	response := &IPDetailResponse{}
	findIP := ip.NewFindIPParams()
	findIP.ID = ipaddress
	resp, err := d.ip.FindIP(findIP, d.auth)
	if err != nil {
		return response, err
	}
	response.IP = resp.Payload
	return response, nil
}

// IPUpdate updates an IP
func (d *Driver) IPUpdate(iur *IPUpdateRequest) (*IPDetailResponse, error) {
	response := &IPDetailResponse{}
	updateIP := ip.NewUpdateIPParams()

	updateRequest := &models.V1IPUpdateRequest{
		Ipaddress:   &iur.IPAddress,
		Description: iur.Description,
		Name:        iur.Name,
		Machineid:   &iur.MachineID,
	}
	updateIP.SetBody(updateRequest)
	resp, err := d.ip.UpdateIP(updateIP, d.auth)
	if err != nil {
		return response, err
	}
	response.IP = resp.Payload
	return response, nil
}

// IPList lists all IPs
func (d *Driver) IPList() (*IPListResponse, error) {
	response := &IPListResponse{}
	listIPs := ip.NewListIpsParams()
	resp, err := d.ip.ListIps(listIPs, d.auth)
	if err != nil {
		return response, err
	}
	response.IPs = resp.Payload
	return response, nil
}

// IPFind returns all ips that match given properties
func (d *Driver) IPFind(ifr *IPFindRequest) (*IPListResponse, error) {
	if ifr == nil {
		return d.IPList()
	}

	response := &IPListResponse{}
	var err error
	var resp *ip.FindIpsOK

	findIPs := ip.NewFindIpsParams()
	req := &models.V1FindIpsRequest{
		Ipaddress:     ifr.IPAddress,
		Projectid:     ifr.ProjectID,
		Networkprefix: ifr.ParentPrefixCidr,
		Networkid:     ifr.NetworkID,
		Machineid:     ifr.MachineID,
	}
	findIPs.SetBody(req)

	resp, err = d.ip.FindIps(findIPs, d.auth)
	if err != nil {
		return response, err
	}
	response.IPs = resp.Payload

	return response, nil
}

// IPAcquire acquires an IP in a network for a project
func (d *Driver) IPAcquire(iar *IPAcquireRequest) (*IPDetailResponse, error) {
	response := &IPDetailResponse{}
	acquireIPRequest := &models.V1IPAllocateRequest{
		Description: iar.Description,
		Name:        iar.Name,
		Networkid:   &iar.Networkid,
		Projectid:   &iar.Projectid,
	}
	if iar.SpecificIP == "" {
		acquireIP := ip.NewAllocateIPParams()
		acquireIP.SetBody(acquireIPRequest)
		resp, err := d.ip.AllocateIP(acquireIP, d.auth)
		if err != nil {
			return response, err
		}
		response.IP = resp.Payload
	} else {
		acquireIP := ip.NewAllocateSpecificIPParams()
		acquireIP.IP = iar.SpecificIP
		acquireIP.SetBody(acquireIPRequest)
		resp, err := d.ip.AllocateSpecificIP(acquireIP, d.auth)
		if err != nil {
			return response, err
		}
		response.IP = resp.Payload
	}
	return response, nil
}

// IPDelete releases an IP
func (d *Driver) IPDelete(id string) (*IPDetailResponse, error) {
	response := &IPDetailResponse{}
	deleteIP := ip.NewDeleteIPParams()
	deleteIP.ID = id
	resp, err := d.ip.DeleteIP(deleteIP, d.auth)
	if err != nil {
		return response, err
	}
	response.IP = resp.Payload
	return response, nil
}
