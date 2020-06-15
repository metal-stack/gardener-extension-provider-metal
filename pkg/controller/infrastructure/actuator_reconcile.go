package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/metal-stack/metal-lib/pkg/tag"

	"github.com/google/uuid"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalgo "github.com/metal-stack/metal-go"
	metalfirewall "github.com/metal-stack/metal-go/api/client/firewall"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener/extensions/pkg/controller/error"

	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/coreos/container-linux-config-transpiler/config/types"
)

const (
	firewallControllerName = "firewall-controller"
	// firewallPolicyControllerNameCompatibility TODO can be removed in a future version when all firewalls migrated to firewall-controller
	firewallPolicyControllerNameCompatibility = "firewall-policy-controller"
	droptailerClientName                      = "droptailer"
)

func (a *actuator) reconcile(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	infrastructureConfig, infrastructureStatus, err := a.decodeInfrastructure(infrastructure)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	var (
		clusterID      = string(cluster.Shoot.GetUID())
		clusterTag     = fmt.Sprintf("%s=%s", tag.ClusterID, clusterID)
		firewallStatus = infrastructureStatus.Firewall
	)

	mclient, err := metalclient.NewClient(ctx, a.client, metalControlPlane.Endpoint, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	nodeCIDR, err := a.ensureNodeNetwork(ctx, clusterID, mclient, infrastructure, infrastructureConfig, cluster)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	infrastructure.Status.NodesCIDR = &nodeCIDR
	err = a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus, &nodeCIDR)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	if firewallStatus.Succeeded {
		// verify that the firewall is still there and correctly reconciled

		resp, err := mclient.FirewallFind(&metalgo.FirewallFindRequest{
			MachineFindRequest: metalgo.MachineFindRequest{
				AllocationProject: &infrastructureConfig.ProjectID,
				Tags:              []string{clusterTag},
			},
		})
		if err != nil {
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}

		machineID := decodeMachineID(firewallStatus.MachineID)
		clusterFirewallAmount := len(resp.Firewalls)

		if clusterFirewallAmount == 0 {
			a.logger.Error(err, "firewall does not exist anymore, creating new firewall", "clusterid", clusterID, "machineid", machineID)
		}
		if clusterFirewallAmount > 1 {
			return fmt.Errorf("multiple firewalls exist for this cluster, which is currently unsupported. delete these firewalls and try to keep the one with machine id %q", machineID)
		}
		if clusterFirewallAmount == 1 {
			fw := resp.Firewalls[0]
			if *fw.ID != machineID {
				a.logger.Error(fmt.Errorf("machine id of this cluster's firewall differs from infrastructure status"), "leaving as it is, but something unexpected must have happened in the past. if you want to get to a clean state, remove the firewall by hand (causes downtime!) and reconcile infrastructure again", "clusterID", clusterID, "expectedMachineID", machineID, "actualMachineID", *fw.ID)
			}

			if *fw.Size.ID == infrastructureConfig.Firewall.Size && *fw.Allocation.Image.ID == infrastructureConfig.Firewall.Image {
				return nil
			}

			a.logger.Info("firewall spec has changed. deleting old firewall and creating a new one", "clusterid", clusterID, "machineid", machineID)

			_, err = mclient.MachineDelete(*fw.ID)
			if err != nil {
				return &controllererrors.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}
		}

		firewallStatus.MachineID = ""
		firewallStatus.Succeeded = false
		err = a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus, &nodeCIDR)
		if err != nil {
			return err
		}
	}

	if firewallStatus.MachineID != "" {
		// firewall was created, waiting for completion
		machineID := decodeMachineID(firewallStatus.MachineID)

		resp, err := mclient.FirewallGet(machineID)
		if err != nil {
			switch e := err.(type) {
			case *metalfirewall.FindFirewallDefault:
				if e.Code() >= 500 {
					return &controllererrors.RequeueAfterError{
						Cause:        e,
						RequeueAfter: 5 * time.Second,
					}
				}
			default:
				return e
			}
		}

		allocation := resp.Firewall.Allocation
		if allocation == nil {
			return fmt.Errorf("firewall %q was created but has no allocation", machineID)
		}

		firewallStatus.Succeeded = *resp.Firewall.Allocation.Succeeded
		return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus, &nodeCIDR)
	}

	// we need to create a firewall
	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	clusterName := cluster.ObjectMeta.Name
	name := clusterName + "-firewall-" + uuid.String()[:5]

	// find private network
	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, infrastructureConfig.ProjectID, nodeCIDR)
	if err != nil {
		return err
	}

	kubeconfig, kubeconfigCompatibility, err := a.createFirewallControllerKubeconfig(ctx, infrastructure, cluster)
	if err != nil {
		return err
	}

	firewallUserData, err := a.renderFirewallUserData(kubeconfig, kubeconfigCompatibility)
	if err != nil {
		return err
	}

	// assemble firewall allocation request
	var networks []metalgo.MachineAllocationNetwork
	network := metalgo.MachineAllocationNetwork{
		NetworkID:   *privateNetwork.ID,
		Autoacquire: true,
	}
	networks = append(networks, network)
	for _, n := range infrastructureConfig.Firewall.Networks {
		network := metalgo.MachineAllocationNetwork{
			NetworkID:   n,
			Autoacquire: true,
		}
		networks = append(networks, network)
	}

	createRequest := &metalgo.FirewallCreateRequest{
		MachineCreateRequest: metalgo.MachineCreateRequest{
			Description:   name + " created by Gardener",
			Name:          name,
			Hostname:      name,
			Size:          infrastructureConfig.Firewall.Size,
			Project:       infrastructureConfig.ProjectID,
			Partition:     infrastructureConfig.PartitionID,
			Image:         infrastructureConfig.Firewall.Image,
			SSHPublicKeys: []string{string(infrastructure.Spec.SSHPublicKey)},
			Networks:      networks,
			UserData:      firewallUserData,
			Tags:          []string{clusterTag},
		},
	}

	a.logger.Info("create firewall", "name", createRequest.Name)

	fcr, err := mclient.FirewallCreate(createRequest)
	if err != nil {
		a.logger.Error(err, "failed to create firewall", "infrastructure", infrastructure.Name)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	machineID := encodeMachineID(*fcr.Firewall.Partition.ID, *fcr.Firewall.ID)

	allocation := fcr.Firewall.Allocation
	if allocation == nil {
		return fmt.Errorf("firewall %q was created but has no allocation", machineID)
	}

	firewallStatus.MachineID = machineID
	firewallStatus.Succeeded = true

	return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus, &nodeCIDR)
}

func (a *actuator) ensureNodeNetwork(ctx context.Context, clusterID string, mclient *metalgo.Driver, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureConfig *metalapi.InfrastructureConfig, cluster *extensionscontroller.Cluster) (string, error) {
	if cluster.Shoot.Spec.Networking.Nodes != nil {
		return *cluster.Shoot.Spec.Networking.Nodes, nil
	}
	if infrastructure.Status.NodesCIDR != nil {
		resp, err := mclient.NetworkFind(&metalgo.NetworkFindRequest{
			ProjectID:   &infrastructureConfig.ProjectID,
			PartitionID: &infrastructureConfig.PartitionID,
			Labels:      map[string]string{tag.ClusterID: clusterID},
		})
		if err != nil {
			return "", err
		}

		if len(resp.Networks) != 0 {
			return *infrastructure.Status.NodesCIDR, nil
		}

		return "", fmt.Errorf("node network disappeared from cloud provider: %s", *infrastructure.Status.NodesCIDR)
	}

	resp, err := mclient.NetworkAllocate(&metalgo.NetworkAllocateRequest{
		ProjectID:   infrastructureConfig.ProjectID,
		PartitionID: infrastructureConfig.PartitionID,
		Name:        cluster.Shoot.GetName(),
		Description: clusterID,
		Labels:      map[string]string{tag.ClusterID: clusterID},
	})
	if err != nil {
		return "", err
	}

	nodeCIDR := resp.Network.Prefixes[0]
	a.logger.Info("dynamically allocated node network", "nodeCIDR", nodeCIDR)

	return nodeCIDR, nil
}

func (a *actuator) createFirewallControllerKubeconfig(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) (string, string, error) {
	apiServerURL := fmt.Sprintf("api.%s", *cluster.Shoot.Spec.DNS.Domain)
	infrastructureSecrets := &secrets.Secrets{
		CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
			v1alpha1constants.SecretNameCACluster: {
				Name:       v1alpha1constants.SecretNameCACluster,
				CommonName: "kubernetes",
				CertType:   secrets.CACert,
			},
		},
		SecretConfigsFunc: func(cas map[string]*secrets.Certificate, clusterName string) []secrets.ConfigInterface {
			return []secrets.ConfigInterface{
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         firewallControllerName,
						CommonName:   fmt.Sprintf("system:%s", firewallControllerName),
						Organization: []string{firewallControllerName},
						CertType:     secrets.ClientCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
					KubeConfigRequest: &secrets.KubeConfigRequest{
						ClusterName:  clusterName,
						APIServerURL: apiServerURL,
					},
				},
				// TODO: can be removed in a future version
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         firewallPolicyControllerNameCompatibility,
						CommonName:   fmt.Sprintf("system:%s", firewallPolicyControllerNameCompatibility),
						Organization: []string{firewallPolicyControllerNameCompatibility},
						CertType:     secrets.ClientCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
					KubeConfigRequest: &secrets.KubeConfigRequest{
						ClusterName:  clusterName,
						APIServerURL: apiServerURL,
					},
				},
			}
		},
	}

	secret, err := infrastructureSecrets.Deploy(ctx, a.clientset, a.gardenerClientset, infrastructure.Namespace)
	if err != nil {
		return "", "", err
	}

	kubeconfig, ok := secret[firewallControllerName].Data["kubeconfig"]
	if !ok {
		return "", "", fmt.Errorf("kubeconfig not part of generated firewall controller secret")
	}

	kubeconfigCompatibility, ok := secret[firewallPolicyControllerNameCompatibility].Data["kubeconfig"]
	if !ok {
		return "", "", fmt.Errorf("kubeconfig not part of generated firewall policy controller secret")
	}

	return string(kubeconfig), string(kubeconfigCompatibility), nil
}

func (a *actuator) renderFirewallUserData(kubeconfig, kubeconfigCompatibility string) (string, error) {
	cfg := types.Config{}
	cfg.Systemd = types.Systemd{}

	enabled := true
	fcUnit := types.SystemdUnit{
		Name:    fmt.Sprintf("%s.service", firewallControllerName),
		Enable:  enabled,
		Enabled: &enabled,
	}
	// TODO can be removed in a future version
	fpcUnitCompatibility := types.SystemdUnit{
		Name:    fmt.Sprintf("%s.service", firewallPolicyControllerNameCompatibility),
		Enable:  enabled,
		Enabled: &enabled,
	}
	dcUnit := types.SystemdUnit{
		Name:    fmt.Sprintf("%s.service", droptailerClientName),
		Enable:  enabled,
		Enabled: &enabled,
	}

	cfg.Systemd.Units = append(cfg.Systemd.Units, fcUnit, fpcUnitCompatibility, dcUnit)

	cfg.Storage = types.Storage{}

	mode := 0600
	id := 0
	ignitionFile := types.File{
		Path:       "/etc/firewall-controller/.kubeconfig",
		Filesystem: "root",
		Mode:       &mode,
		User: &types.FileUser{
			Id: &id,
		},
		Group: &types.FileGroup{
			Id: &id,
		},
		Contents: types.FileContents{
			Inline: string(kubeconfig),
		},
	}
	// TODO can be removed in a future version
	ignitionFileCompatibility := types.File{
		Path:       "/etc/firewall-policy-controller/.kubeconfig",
		Filesystem: "root",
		Mode:       &mode,
		User: &types.FileUser{
			Id: &id,
		},
		Group: &types.FileGroup{
			Id: &id,
		},
		Contents: types.FileContents{
			Inline: string(kubeconfigCompatibility),
		},
	}
	cfg.Storage.Files = append(cfg.Storage.Files, ignitionFile, ignitionFileCompatibility)

	outCfg, report := types.Convert(cfg, "", nil)
	if report.IsFatal() {
		return "", fmt.Errorf("could not transpile ignition config: %s", report.String())
	}

	userData, err := json.Marshal(outCfg)
	if err != nil {
		return "", err
	}

	return string(userData), nil
}
