package metalgo

import (
	"net/http"
	"time"

	"github.com/metal-pod/metal-go/api/client/firewall"
	"github.com/metal-pod/metal-go/api/models"
)

// FirewallCreateRequest contains data for a machine creation
type FirewallCreateRequest struct {
	MachineCreateRequest
	NetworkIDs []string
}

// FirewallCreateResponse is returned when a machine was created
type FirewallCreateResponse struct {
	Firewall *models.V1FirewallResponse
}

// FirewallListResponse contains the machine list result
type FirewallListResponse struct {
	Firewalls []*models.V1FirewallResponse
}

// FirewallGetResponse contains the machine get result
type FirewallGetResponse struct {
	Firewall *models.V1FirewallResponse
}

// FirewallCreate will create a single metal machine
func (d *Driver) FirewallCreate(fcr *FirewallCreateRequest) (*FirewallCreateResponse, error) {
	response := &FirewallCreateResponse{}

	allocateRequest := &models.V1FirewallCreateRequest{
		Description: fcr.Description,
		Partitionid: &fcr.Partition,
		Hostname:    fcr.Hostname,
		Imageid:     &fcr.Image,
		Name:        fcr.Name,
		UUID:        fcr.UUID,
		Projectid:   &fcr.Project,
		Tenant:      &fcr.Tenant,
		Sizeid:      &fcr.Size,
		SSHPubKeys:  fcr.SSHPublicKeys,
		UserData:    fcr.UserData,
		Tags:        fcr.Tags,
		Networks:    fcr.NetworkIDs,
	}

	allocFirewall := firewall.NewAllocateFirewallParams()
	allocFirewall.SetBody(allocateRequest)

	retryOnlyOnStatusNotFound := func(sc int) bool {
		return sc == http.StatusNotFound
	}
	allocFirewall.WithContext(newRetryContext(3, 5*time.Second, retryOnlyOnStatusNotFound))

	resp, err := d.firewall.AllocateFirewall(allocFirewall, d.auth)
	if err != nil {
		return response, err
	}
	response.Firewall = resp.Payload

	return response, nil
}

// FirewallList will list all machines
func (d *Driver) FirewallList() (*FirewallListResponse, error) {
	response := &FirewallListResponse{}

	listFirewall := firewall.NewListFirewallsParams()
	resp, err := d.firewall.ListFirewalls(listFirewall, d.auth)
	if err != nil {
		return response, err
	}
	response.Firewalls = resp.Payload
	return response, nil
}

// FirewallSearch will search for firewalls for given criteria
func (d *Driver) FirewallSearch(partition, project *string) (*FirewallListResponse, error) {
	response := &FirewallListResponse{}

	searchFirewall := firewall.NewSearchFirewallParams()
	searchFirewall.WithPartition(partition)
	searchFirewall.WithProject(project)
	resp, err := d.firewall.SearchFirewall(searchFirewall, d.auth)
	if err != nil {
		return response, err
	}
	response.Firewalls = resp.Payload
	return response, nil
}

// FirewallGet will only return one machine
func (d *Driver) FirewallGet(machineID string) (*FirewallGetResponse, error) {
	findFirewall := firewall.NewFindFirewallParams()
	findFirewall.ID = machineID

	response := &FirewallGetResponse{}
	resp, err := d.firewall.FindFirewall(findFirewall, d.auth)
	if err != nil {
		return response, err
	}
	response.Firewall = resp.Payload

	return response, nil
}
