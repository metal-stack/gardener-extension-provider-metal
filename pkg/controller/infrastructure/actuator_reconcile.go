package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/client/firewall"
	"github.com/metal-stack/metal-go/api/client/image"
	"github.com/metal-stack/metal-go/api/client/ip"
	metalip "github.com/metal-stack/metal-go/api/client/ip"
	"github.com/metal-stack/metal-go/api/client/network"
	"github.com/metal-stack/metal-go/api/models"

	mn "github.com/metal-stack/metal-lib/pkg/net"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/controllerutils/reconciler"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/secrets"

	configlatest "k8s.io/client-go/tools/clientcmd/api/latest"
	configv1 "k8s.io/client-go/tools/clientcmd/api/v1"

	"github.com/flatcar/container-linux-config-transpiler/config/types"
)

const (
	caNameInfrastructure = "ca-" + metal.Name + "-infrastructure"
)

type firewallReconciler struct {
	logger               logr.Logger
	restConfig           *rest.Config
	c                    client.Client
	infrastructure       *extensionsv1alpha1.Infrastructure
	infrastructureConfig *metalapi.InfrastructureConfig
	providerStatus       *metalapi.InfrastructureStatus
	cluster              *extensionscontroller.Cluster
	mclient              metalgo.Client
	clusterID            string
	clusterTag           string
	machineIDInStatus    string
}

type egressIPReconciler struct {
	logger               logr.Logger
	infrastructureConfig *metalapi.InfrastructureConfig
	mclient              metalgo.Client
	clusterID            string
	egressTag            string
}

type firewallReconcileAction string

const (
	firewallControllerName = "firewall-controller"
	droptailerClientName   = "droptailer"
)

var (
	// firewallActionRecreate wipe infrastructure status and triggers creation of a new metal firewall
	firewallActionRecreate firewallReconcileAction = "recreate"
	// firewallActionDeleteAndRecreate deletes the firewall, wipe infrastructure status and triggers creation of a new metal firewall
	// occurs when someone changes the firewalltype, firewallimage or additionalnetworks
	firewallActionDeleteAndRecreate firewallReconcileAction = "delete"
	// firewallActionDoNothing nothing needs to be done for this firewall
	firewallActionDoNothing firewallReconcileAction = "nothing"
	// firewallActionCreate create a new firewall and write infrastructure status
	firewallActionCreate firewallReconcileAction = "create"
	// firewallActionStatusUpdateOnMigrate infrastructure status is not present, but a metal firewall machine is present.
	// this is the case during migration of the shoot to another seed because infrastructure status is not migrated by gardener
	// therefor the status needs to be recreated
	firewallActionStatusUpdateOnMigrate firewallReconcileAction = "migrate"
)

func (a *actuator) Reconcile(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	internalInfrastructureConfig, internalInfrastructureStatus, err := decodeInfrastructure(infrastructure, a.decoder)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, internalInfrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	mclient, err := metalclient.NewClient(ctx, a.client, metalControlPlane.Endpoint, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	egressIPReconciler := &egressIPReconciler{
		logger:               a.logger,
		infrastructureConfig: internalInfrastructureConfig,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
		egressTag:            egressTag(string(cluster.Shoot.GetUID())),
	}
	err = reconcileEgressIPs(ctx, egressIPReconciler)
	if err != nil {
		return err
	}

	reconciler := &firewallReconciler{
		logger:               a.logger,
		restConfig:           a.restConfig,
		c:                    a.client,
		infrastructure:       infrastructure,
		infrastructureConfig: internalInfrastructureConfig,
		providerStatus:       internalInfrastructureStatus,
		cluster:              cluster,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
		clusterTag:           ClusterTag(string(cluster.Shoot.GetUID())),
		machineIDInStatus:    decodeMachineID(internalInfrastructureStatus.Firewall.MachineID),
	}

	err = reconcileFirewall(ctx, reconciler)
	if err != nil {
		return err
	}

	return nil
}

// reconcileFirewall creates,recreates firewall and updates the infrastructure status accordingly
// TODO migrate to a state-machine or to a dedicated controller
func reconcileFirewall(ctx context.Context, r *firewallReconciler) error {
	// detect which next action is required
	action, status, err := firewallNextAction(ctx, r)
	if err != nil {
		return err
	}

	switch action {
	case firewallActionDoNothing:
		r.logger.Info("firewall reconciled, nothing to be done", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.providerStatus.Firewall.MachineID)
		return nil
	case firewallActionCreate:
		machineID, nodeCIDR, err := createFirewall(ctx, r)
		if err != nil {
			return err
		}
		r.logger.Info("firewall created", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.providerStatus.Firewall.MachineID)

		r.providerStatus.Firewall.MachineID = machineID
		return updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, &nodeCIDR)
	case firewallActionRecreate:
		err := deleteFirewallFromStatus(ctx, r)
		if err != nil {
			return err
		}
		r.logger.Info("firewall removed from status", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.machineIDInStatus)
		machineID, nodeCIDR, err := createFirewall(ctx, r)
		if err != nil {
			return err
		}
		r.logger.Info("firewall created", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.providerStatus.Firewall.MachineID)

		r.providerStatus.Firewall.MachineID = machineID

		return updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, &nodeCIDR)
	case firewallActionDeleteAndRecreate:
		err := deleteFirewall(ctx, r.machineIDInStatus, r.infrastructureConfig.ProjectID, r.clusterTag, r.mclient)
		if err != nil {
			return err
		}
		r.logger.Info("firewall deleted", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.machineIDInStatus)
		err = deleteFirewallFromStatus(ctx, r)
		if err != nil {
			return err
		}
		r.logger.Info("firewall removed from status", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.machineIDInStatus)
		machineID, nodeCIDR, err := createFirewall(ctx, r)
		if err != nil {
			return err
		}
		r.logger.Info("firewall created", "cluster-id", r.clusterID, "cluster", r.cluster.Shoot.Name, "machine-id", r.providerStatus.Firewall.MachineID)
		r.providerStatus.Firewall.MachineID = machineID
		return updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, &nodeCIDR)
	case firewallActionStatusUpdateOnMigrate:
		r.providerStatus.Firewall = *status
		return updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, r.cluster.Shoot.Spec.Networking.Nodes)
	default:
		return fmt.Errorf("unsupported firewall reconcile action: %s", action)
	}
}

func firewallNextAction(ctx context.Context, r *firewallReconciler) (firewallReconcileAction, *metalapi.FirewallStatus, error) {
	firewalls, err := metalclient.FindClusterFirewalls(ctx, r.mclient, r.clusterTag, r.infrastructureConfig.ProjectID)
	if err != nil {
		r.logger.Error(err, "unable to fetch cluster firewalls", "clustertag", r.clusterTag, "projectid", r.infrastructureConfig.ProjectID)
		return firewallActionDoNothing, nil, &reconciler.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	clusterFirewallAmount := len(firewalls)
	switch clusterFirewallAmount {
	case 0:
		// No machine created yet or existing machine was deleted, create one
		if r.machineIDInStatus == "" {
			return firewallActionCreate, nil, nil
		}
		r.logger.Info("firewall does not exist anymore, recreation required", "clusterid", r.clusterID, "machineid", r.machineIDInStatus)
		return firewallActionRecreate, nil, nil
	case 1:
		fw := firewalls[0]
		if r.machineIDInStatus == "" {
			r.logger.Info("firewall exists but status is empty, assuming migration", "clusterid", r.clusterID, "machineid", r.machineIDInStatus)
			return firewallActionStatusUpdateOnMigrate, &metalapi.FirewallStatus{
				MachineID: encodeMachineID(*fw.Partition.ID, *fw.ID),
			}, nil
		}

		if *fw.ID != r.machineIDInStatus {
			r.logger.Error(
				fmt.Errorf("machine id of this cluster's firewall differs from infrastructure status, not reconciling firewall anymore"),
				"leaving as it is, but something unexpected must have happened in the past. if you want to get to a clean state, remove the firewall by hand (causes downtime!) and reconcile infrastructure again",
				"clusterID", r.clusterID,
				"expectedMachineID", r.machineIDInStatus,
				"actualMachineID", *fw.ID,
			)
			return firewallActionDoNothing, nil, nil
		}

		if fw.Size.ID != nil && *fw.Size.ID != r.infrastructureConfig.Firewall.Size {
			r.logger.Info("firewall size has changed", "clusterid", r.clusterID, "machineid", r.machineIDInStatus, "current", *fw.Size.ID, "new", r.infrastructureConfig.Firewall.Size)
			return firewallActionDeleteAndRecreate, nil, nil
		}

		if fw.Allocation != nil && fw.Allocation.Image != nil && fw.Allocation.Image.ID != nil && *fw.Allocation.Image.ID != r.infrastructureConfig.Firewall.Image {
			want := r.infrastructureConfig.Firewall.Image
			image, err := r.mclient.Image().FindLatestImage(image.NewFindLatestImageParams().WithID(want).WithContext(ctx), nil)
			if err != nil {
				r.logger.Error(err, "firewall latest image not found", "clustertag", r.clusterTag, "projectid", r.infrastructureConfig.ProjectID, "image", want)
				return firewallActionDoNothing, nil, &reconciler.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}

			if image.Payload != nil && image.Payload.ID != nil && *image.Payload.ID != *fw.Allocation.Image.ID {
				r.logger.Info("firewall image has changed", "clusterid", r.clusterID, "machineid", r.machineIDInStatus, "current", *fw.Allocation.Image.ID, "new", *image.Payload.ID)
				return firewallActionDeleteAndRecreate, nil, nil
			}
		}

		currentNetworks := sets.NewString()
		for _, n := range fw.Allocation.Networks {
			if *n.Networktype == mn.PrivatePrimaryUnshared || *n.Networktype == mn.PrivatePrimaryShared {
				continue
			}
			if *n.Underlay {
				continue
			}
			currentNetworks.Insert(*n.Networkid)
		}
		wantNetworks := sets.NewString()
		for _, n := range r.infrastructureConfig.Firewall.Networks {
			wantNetworks.Insert(n)
		}
		if !currentNetworks.Equal(wantNetworks) {
			r.logger.Info("firewall networks have changed", "clusterid", r.clusterID, "machineid", r.machineIDInStatus, "current", currentNetworks.List(), "new", wantNetworks.List())
			return firewallActionDeleteAndRecreate, nil, nil
		}

		return firewallActionDoNothing, nil, nil
	default:
		err := fmt.Errorf("multiple firewalls exist for this cluster, which is currently unsupported. delete these firewalls and try to keep the one with machine id %q", r.machineIDInStatus)
		r.logger.Error(
			err,
			"multiple firewalls exist for this cluster",
			"clusterID", r.clusterID,
			"expectedMachineID", r.machineIDInStatus,
		)
		return firewallActionDoNothing, nil, err
	}
}

func createFirewall(ctx context.Context, r *firewallReconciler) (machineID string, nodeCIDR string, err error) {
	nodeCIDR, err = ensureNodeNetwork(ctx, r)
	if err != nil {
		r.logger.Error(err, "firewalls node network", "nodecidr", nodeCIDR)
		return "", "", &reconciler.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	err = updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, &nodeCIDR)
	if err != nil {
		r.logger.Error(err, "firewalls update provider status", "nodecidr", nodeCIDR)
		return "", "", &reconciler.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	uuid, err := uuid.NewUUID()
	if err != nil {
		return "", "", err
	}

	clusterName := r.cluster.ObjectMeta.Name
	name := clusterName + "-firewall-" + uuid.String()[:5]

	// find private network
	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(ctx, r.mclient, r.infrastructureConfig.ProjectID, nodeCIDR)
	if err != nil {
		return "", "", err
	}

	kubeconfig, err := createFirewallControllerKubeconfig(ctx, r)
	if err != nil {
		return "", "", err
	}

	firewallUserData, err := renderFirewallUserData(kubeconfig)
	if err != nil {
		return "", "", err
	}

	// assemble firewall allocation request
	networks := []*models.V1MachineAllocationNetwork{
		{
			Networkid:   privateNetwork.ID,
			Autoacquire: pointer.Bool(true),
		},
	}

	for _, n := range r.infrastructureConfig.Firewall.Networks {
		n := n
		network := &models.V1MachineAllocationNetwork{
			Networkid:   &n,
			Autoacquire: pointer.Bool(true),
		}
		networks = append(networks, network)
	}

	createRequest := &models.V1FirewallCreateRequest{
		Description: name + " created by Gardener",
		Name:        name,
		Hostname:    name,
		Sizeid:      &r.infrastructureConfig.Firewall.Size,
		Projectid:   &r.infrastructureConfig.ProjectID,
		Partitionid: &r.infrastructureConfig.PartitionID,
		Imageid:     &r.infrastructureConfig.Firewall.Image,
		SSHPubKeys:  []string{string(r.infrastructure.Spec.SSHPublicKey)},
		Networks:    networks,
		UserData:    firewallUserData,
		Tags:        []string{r.clusterTag},
	}

	fcr, err := r.mclient.Firewall().AllocateFirewall(firewall.NewAllocateFirewallParams().WithBody(createRequest).WithContext(ctx), nil)
	if err != nil {
		r.logger.Error(err, "firewall create error")
		return "", "", &reconciler.RequeueAfterError{
			Cause:        fmt.Errorf("failed to create firewall %w", err),
			RequeueAfter: 30 * time.Second,
		}
	}

	machineID = encodeMachineID(*fcr.Payload.Partition.ID, *fcr.Payload.ID)

	allocation := fcr.Payload.Allocation
	if allocation == nil {
		return "", "", fmt.Errorf("firewall %q was created but has no allocation", machineID)
	}

	return machineID, nodeCIDR, nil
}

func reconcileEgressIPs(ctx context.Context, r *egressIPReconciler) error {
	currentEgressIPs := sets.NewString()

	resp, err := r.mclient.IP().FindIPs(ip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
		Projectid: r.infrastructureConfig.ProjectID,
		Tags:      []string{r.egressTag},
		Type:      models.V1IPBaseTypeStatic,
	}).WithContext(ctx), nil)
	if err != nil {
		return &reconciler.RequeueAfterError{
			Cause:        fmt.Errorf("failed to list egress ips of cluster %w", err),
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range resp.Payload {
		currentEgressIPs.Insert(*ip.Ipaddress)
	}

	wantEgressIPs := sets.NewString()
	for _, egressRule := range r.infrastructureConfig.Firewall.EgressRules {
		wantEgressIPs.Insert(egressRule.IPs...)

		for _, ip := range egressRule.IPs {
			ip := ip
			if currentEgressIPs.Has(ip) {
				continue
			}

			resp, err := r.mclient.IP().FindIPs(metalip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
				Ipaddress: ip,
				Projectid: r.infrastructureConfig.ProjectID,
				Networkid: egressRule.NetworkID,
			}).WithContext(ctx), nil)
			if err != nil {
				return &reconciler.RequeueAfterError{
					Cause:        fmt.Errorf("error when retrieving ip %s for egress rule %w", ip, err),
					RequeueAfter: 30 * time.Second,
				}
			}

			switch len(resp.Payload) {
			case 0:
				return &reconciler.RequeueAfterError{
					Cause:        fmt.Errorf("ip %s for egress rule does not exist", ip),
					RequeueAfter: 30 * time.Second,
				}
			case 1:
			default:
				return fmt.Errorf("ip %s found multiple times", ip)
			}

			dbIP := resp.Payload[0]
			if dbIP.Type != nil && *dbIP.Type != models.V1IPBaseTypeStatic {
				return &reconciler.RequeueAfterError{
					Cause:        fmt.Errorf("ips for egress rule must be static, but %s is not static", ip),
					RequeueAfter: 30 * time.Second,
				}
			}

			if len(dbIP.Tags) > 0 {
				return &reconciler.RequeueAfterError{
					Cause:        fmt.Errorf("won't use ip %s for egress rules because it does not have an egress tag but it has other tags", *dbIP.Ipaddress),
					RequeueAfter: 30 * time.Second,
				}
			}

			_, err = r.mclient.IP().UpdateIP(metalip.NewUpdateIPParams().WithBody(&models.V1IPUpdateRequest{
				Ipaddress: dbIP.Ipaddress,
				Tags:      []string{r.egressTag},
			}).WithContext(ctx), nil)
			if err != nil {
				return &reconciler.RequeueAfterError{
					Cause:        fmt.Errorf("could not tag ip %s for egress usage %w", ip, err),
					RequeueAfter: 30 * time.Second,
				}
			}
		}
	}

	if !currentEgressIPs.Equal(wantEgressIPs) {
		toUnTag := currentEgressIPs.Difference(wantEgressIPs)
		for _, ip := range toUnTag.List() {
			err := clearIPTags(ctx, r.mclient, ip)
			if err != nil {
				return &reconciler.RequeueAfterError{
					Cause:        fmt.Errorf("could not remove egress tag from ip %s %w", ip, err),
					RequeueAfter: 30 * time.Second,
				}
			}
		}
	}

	return nil
}

func egressTag(clusterID string) string {
	return fmt.Sprintf("%s=%s", tag.ClusterEgress, clusterID)
}

func clearIPTags(ctx context.Context, mclient metalgo.Client, ip string) error {
	_, err := mclient.IP().UpdateIP(metalip.NewUpdateIPParams().WithBody(&models.V1IPUpdateRequest{
		Ipaddress: &ip,
		Tags:      []string{},
	}).WithContext(ctx), nil)

	return err
}

func ensureNodeNetwork(ctx context.Context, r *firewallReconciler) (string, error) {
	if r.cluster.Shoot.Spec.Networking.Nodes != nil {
		return *r.cluster.Shoot.Spec.Networking.Nodes, nil
	}
	if r.infrastructure.Status.NodesCIDR != nil {
		resp, err := r.mclient.Network().FindNetworks(network.NewFindNetworksParams().WithBody(&models.V1NetworkFindRequest{
			Projectid:   r.infrastructureConfig.ProjectID,
			Partitionid: r.infrastructureConfig.PartitionID,
			Labels:      map[string]string{tag.ClusterID: r.clusterID},
		}).WithContext(ctx), nil)
		if err != nil {
			return "", err
		}

		if len(resp.Payload) != 0 {
			return *r.infrastructure.Status.NodesCIDR, nil
		}

		return "", fmt.Errorf("node network disappeared from cloud provider: %s", *r.infrastructure.Status.NodesCIDR)
	}

	resp, err := r.mclient.Network().AllocateNetwork(network.NewAllocateNetworkParams().WithBody(&models.V1NetworkAllocateRequest{
		Projectid:   r.infrastructureConfig.ProjectID,
		Partitionid: r.infrastructureConfig.PartitionID,
		Name:        r.cluster.Shoot.GetName(),
		Description: r.clusterID,
		Labels:      map[string]string{tag.ClusterID: r.clusterID},
	}).WithContext(ctx), nil)
	if err != nil {
		return "", err
	}

	nodeCIDR := resp.Payload.Prefixes[0]

	return nodeCIDR, nil
}

func createFirewallControllerKubeconfig(ctx context.Context, r *firewallReconciler) (string, error) {
	manager, err := secretsmanager.New(ctx, r.logger.WithName("infrastructure-secrets-manager"), clock.RealClock{}, r.c, r.infrastructure.Namespace, metal.Type+"-provider-shoot-infrastructure", nil)
	if err != nil {
		return "", fmt.Errorf("unable to create secrets manager: %w", err)
	}

	secret, err := extensionssecretsmanager.GenerateAllSecrets(ctx, manager, []extensionssecretsmanager.SecretConfigWithOptions{
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:       caNameInfrastructure,
				CommonName: caNameInfrastructure,
				CertType:   secretutils.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        firewallControllerName,
				CommonName:                  fmt.Sprintf("system:%s", firewallControllerName),
				Organization:                []string{firewallControllerName},
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameInfrastructure), secretsmanager.Persist()},
		},
	})
	if err != nil {
		return "", err
	}

	ca, ok := secret[caNameInfrastructure]
	if !ok {
		return "", fmt.Errorf("infrastructure ca was not generated")
	}
	cert, ok := secret[firewallControllerName]
	if !ok {
		return "", fmt.Errorf("infrastructure firewall client cert was not generated")
	}

	config := &configv1.Config{
		CurrentContext: r.infrastructure.Name,
		Clusters: []configv1.NamedCluster{
			{
				Name: r.infrastructure.Name,
				Cluster: configv1.Cluster{
					CertificateAuthorityData: ca.Data[secrets.ControlPlaneSecretDataKeyCertificatePEM(ca.Name)],
					Server:                   fmt.Sprintf("https://api.%s", *r.cluster.Shoot.Spec.DNS.Domain),
				},
			},
		},
		Contexts: []configv1.NamedContext{
			{
				Name: r.infrastructure.Name,
				Context: configv1.Context{
					Cluster:  r.infrastructure.Name,
					AuthInfo: r.infrastructure.Name,
				},
			},
		},
		AuthInfos: []configv1.NamedAuthInfo{
			{
				Name: r.infrastructure.Name,
				AuthInfo: configv1.AuthInfo{
					ClientCertificateData: cert.Data[secrets.ControlPlaneSecretDataKeyCertificatePEM(cert.Name)],
					ClientKeyData:         cert.Data[secrets.ControlPlaneSecretDataKeyPrivateKey(cert.Name)],
				},
			},
		},
	}

	kubeconfig, err := runtime.Encode(configlatest.Codec, config)
	if err != nil {
		return "", fmt.Errorf("unable to encode kubeconfig for firewall: %w", err)
	}

	return string(kubeconfig), nil
}

func renderFirewallUserData(kubeconfig string) (string, error) {
	cfg := types.Config{}
	cfg.Systemd = types.Systemd{}

	enabled := true
	fcUnit := types.SystemdUnit{
		Name:    fmt.Sprintf("%s.service", firewallControllerName),
		Enable:  enabled,
		Enabled: &enabled,
	}
	dcUnit := types.SystemdUnit{
		Name:    fmt.Sprintf("%s.service", droptailerClientName),
		Enable:  enabled,
		Enabled: &enabled,
	}

	cfg.Systemd.Units = append(cfg.Systemd.Units, fcUnit, dcUnit)

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
	cfg.Storage.Files = append(cfg.Storage.Files, ignitionFile)

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

func deleteFirewallFromStatus(ctx context.Context, r *firewallReconciler) error {
	r.providerStatus.Firewall.MachineID = ""
	err := updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, r.infrastructure.Status.NodesCIDR)
	if err != nil {
		return err
	}
	return nil
}

func ClusterTag(clusterID string) string {
	return fmt.Sprintf("%s=%s", tag.ClusterID, clusterID)
}
