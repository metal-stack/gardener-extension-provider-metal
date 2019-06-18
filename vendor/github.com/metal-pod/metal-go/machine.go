package metalgo

import (
	"net/http"
	"time"

	"github.com/metal-pod/metal-go/api/client/machine"
	"github.com/metal-pod/metal-go/api/models"
)

// MachineCreateRequest contains data for a machine creation
type MachineCreateRequest struct {
	Description   string
	Hostname      string
	Name          string
	UserData      string
	Size          string
	Project       string
	Tenant        string
	Partition     string
	Image         string
	Tags          []string
	SSHPublicKeys []string
	UUID          string
}

// MachineListRequest contains data for a machine listing
type MachineListRequest struct {
	Tenant    string
	Project   string
	Partition string
	Tags      []string
}

// MachineCreateResponse is returned when a machine was created
type MachineCreateResponse struct {
	Machine *models.V1MachineResponse
}

// MachineListResponse contains the machine list result
type MachineListResponse struct {
	Machines []*models.V1MachineResponse
}

// MachineGetResponse contains the machine get result
type MachineGetResponse struct {
	Machine *models.V1MachineResponse
}

// MachineIpmiResponse contains the machine get result
type MachineIpmiResponse struct {
	IPMI *models.V1MachineIPMI
}

// MachineDeleteResponse contains the machine delete result
type MachineDeleteResponse struct {
	Machine *models.V1MachineResponse
}

// MachinePowerResponse contains the machine power result
type MachinePowerResponse struct {
	MachineAllocation *models.V1MachineResponse
}

// MachineBiosResponse contains the machine bios result
type MachineBiosResponse struct {
	MachineAllocation *models.V1MachineResponse
}

// MachineStateResponse contains the machine bios result
type MachineStateResponse struct {
	Machine *models.V1MachineResponse
}

// MachineCreate will create a single metal machine
func (d *Driver) MachineCreate(mcr *MachineCreateRequest) (*MachineCreateResponse, error) {
	response := &MachineCreateResponse{}

	allocateRequest := &models.V1MachineAllocateRequest{
		Description: mcr.Description,
		Partitionid: &mcr.Partition,
		Hostname:    mcr.Hostname,
		Imageid:     &mcr.Image,
		Name:        mcr.Name,
		UUID:        mcr.UUID,
		Projectid:   &mcr.Project,
		Tenant:      &mcr.Tenant,
		Sizeid:      &mcr.Size,
		SSHPubKeys:  mcr.SSHPublicKeys,
		UserData:    mcr.UserData,
		Tags:        mcr.Tags,
	}
	allocMachine := machine.NewAllocateMachineParams()
	allocMachine.SetBody(allocateRequest)

	retryOnlyOnStatusNotFound := func(sc int) bool {
		return sc == http.StatusNotFound
	}
	allocMachine.WithContext(newRetryContext(3, 5*time.Second, retryOnlyOnStatusNotFound))

	resp, err := d.machine.AllocateMachine(allocMachine, d.auth)
	if err != nil {
		return response, err
	}

	response.Machine = resp.Payload

	return response, nil
}

// MachineDelete will delete a single metal machine
func (d *Driver) MachineDelete(machineID string) (*MachineDeleteResponse, error) {
	freeMachine := machine.NewFreeMachineParams()
	freeMachine.ID = machineID

	response := &MachineDeleteResponse{}
	resp, err := d.machine.FreeMachine(freeMachine, d.auth)
	if err != nil {
		return response, err
	}
	response.Machine = resp.Payload
	return response, nil
}

// MachineList will list all machines
func (d *Driver) MachineList(mcr *MachineListRequest) (*MachineListResponse, error) {
	response := &MachineListResponse{}

	listMachine := machine.NewListMachinesParams()
	resp, err := d.machine.ListMachines(listMachine, d.auth)
	if err != nil {
		return response, err
	}
	response.Machines = resp.Payload
	return response, nil
}

// MachineSearch will search for machines for given criteria
func (d *Driver) MachineSearch(mac, partition, project *string) (*MachineListResponse, error) {
	response := &MachineListResponse{}

	searchMachine := machine.NewSearchMachineParams()
	searchMachine.WithMac(mac)
	searchMachine.WithPartition(partition)
	searchMachine.WithProject(project)
	resp, err := d.machine.SearchMachine(searchMachine, d.auth)
	if err != nil {
		return response, err
	}
	response.Machines = resp.Payload
	return response, nil
}

// MachineGet will only return one machine
func (d *Driver) MachineGet(machineID string) (*MachineGetResponse, error) {
	findMachine := machine.NewFindMachineParams()
	findMachine.ID = machineID

	response := &MachineGetResponse{}
	resp, err := d.machine.FindMachine(findMachine, d.auth)
	if err != nil {
		return response, err
	}
	response.Machine = resp.Payload

	return response, nil
}

// MachineIpmi will only return one machine
func (d *Driver) MachineIpmi(machineID string) (*MachineIpmiResponse, error) {
	ipmiMachine := machine.NewIPMIDataParams()
	ipmiMachine.ID = machineID

	response := &MachineIpmiResponse{}
	resp, err := d.machine.IPMIData(ipmiMachine, d.auth)
	if err != nil {
		return response, err
	}
	response.IPMI = resp.Payload

	return response, nil
}

// MachinePowerOn will power on a single metal machine
func (d *Driver) MachinePowerOn(machineID string) (*MachinePowerResponse, error) {
	machineOn := machine.NewMachineOnParams()
	machineOn.ID = machineID
	machineOn.Body = []string{}

	response := &MachinePowerResponse{}
	resp, err := d.machine.MachineOn(machineOn, d.auth)
	if err != nil {
		return response, err
	}
	response.MachineAllocation = resp.Payload
	return response, nil
}

// MachinePowerOff will power off a single metal machine
func (d *Driver) MachinePowerOff(machineID string) (*MachinePowerResponse, error) {
	machineOff := machine.NewMachineOffParams()
	machineOff.ID = machineID
	machineOff.Body = []string{}

	response := &MachinePowerResponse{}
	resp, err := d.machine.MachineOff(machineOff, d.auth)
	if err != nil {
		return response, err
	}
	response.MachineAllocation = resp.Payload
	return response, nil
}

// MachinePowerReset will power reset a single metal machine
func (d *Driver) MachinePowerReset(machineID string) (*MachinePowerResponse, error) {
	machineReset := machine.NewMachineResetParams()
	machineReset.ID = machineID
	machineReset.Body = []string{}

	response := &MachinePowerResponse{}
	resp, err := d.machine.MachineReset(machineReset, d.auth)
	if err != nil {
		return response, err
	}
	response.MachineAllocation = resp.Payload
	return response, nil
}

// MachineBootBios will boot a single metal machine into BIOS
func (d *Driver) MachineBootBios(machineID string) (*MachineBiosResponse, error) {
	machineBios := machine.NewMachineBiosParams()
	machineBios.ID = machineID
	machineBios.Body = []string{}

	response := &MachineBiosResponse{}
	resp, err := d.machine.MachineBios(machineBios, d.auth)
	if err != nil {
		return response, err
	}
	response.MachineAllocation = resp.Payload
	return response, nil
}

// MachineReserve will reserve a machine for single allocation
func (d *Driver) MachineReserve(machineID, description string) (*MachineStateResponse, error) {
	machineState := machine.NewSetMachineStateParams()
	machineState.ID = machineID
	reserved := "RESERVED"
	machineState.Body = &models.V1MachineState{
		Value:       &reserved,
		Description: &description,
	}

	response := &MachineStateResponse{}
	resp, err := d.machine.SetMachineState(machineState, d.auth)
	if err != nil {
		return response, err
	}
	response.Machine = resp.Payload
	return response, nil
}

// MachineUnReserve will unreserve a machine
func (d *Driver) MachineUnReserve(machineID string) (*MachineStateResponse, error) {
	machineState := machine.NewSetMachineStateParams()
	machineState.ID = machineID
	reserved := ""
	machineState.Body = &models.V1MachineState{
		Value: &reserved,
	}

	response := &MachineStateResponse{}
	resp, err := d.machine.SetMachineState(machineState, d.auth)
	if err != nil {
		return response, err
	}
	response.Machine = resp.Payload
	return response, nil
}
