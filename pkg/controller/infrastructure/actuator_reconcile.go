package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/google/uuid"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalgo "github.com/metal-stack/metal-go"
	metalfirewall "github.com/metal-stack/metal-go/api/client/firewall"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener/extensions/pkg/controller/error"

	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"
	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/coreos/container-linux-config-transpiler/config/types"
)

type firewallReconciler struct {
	logger               logr.Logger
	restConfig           *rest.Config
	c                    client.Client
	clientset            kubernetes.Interface
	gc                   gardenerkubernetes.Interface
	infrastructure       *extensionsv1alpha1.Infrastructure
	infrastructureConfig *metalapi.InfrastructureConfig
	providerStatus       *metalapi.InfrastructureStatus
	cluster              *extensionscontroller.Cluster
	mclient              *metalgo.Driver
	clusterID            string
	clusterTag           string
	machineID            string
}

type firewallReconcileAction string

const (
	firewallControllerName = "firewall-controller"
	// firewallPolicyControllerNameCompatibility TODO can be removed in a future version when all firewalls migrated to firewall-controller
	firewallPolicyControllerNameCompatibility = "firewall-policy-controller"
	droptailerClientName                      = "droptailer"
)

var (
	firewallActionRecreate          firewallReconcileAction = "recreate"
	firewallActionDeleteAndRecreate firewallReconcileAction = "delete"
	firewallActionDoNothing         firewallReconcileAction = "nothing"
)

func (a *actuator) Reconcile(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	return Reconcile(ctx, a.logger, a.restConfig, a.client, a.decoder, cluster, infrastructure)
}

func Reconcile(
	ctx context.Context,
	logger logr.Logger,
	restConfig *rest.Config,
	c client.Client,
	decoder runtime.Decoder,
	cluster *extensionscontroller.Cluster,
	infrastructure *extensionsv1alpha1.Infrastructure,
) error {
	internalInfrastructureConfig, internalInfrastructureStatus, err := decodeInfrastructure(infrastructure, decoder)
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

	mclient, err := metalclient.NewClient(ctx, c, metalControlPlane.Endpoint, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "could not create kubernetes clientset")
	}

	gardenerClientset, err := gardenerkubernetes.NewWithConfig(gardenerkubernetes.WithRESTConfig(restConfig))
	if err != nil {
		return errors.Wrap(err, "could not create gardener clientset")
	}

	reconciler := &firewallReconciler{
		logger:               logger,
		restConfig:           restConfig,
		c:                    c,
		clientset:            clientset,
		gc:                   gardenerClientset,
		infrastructure:       infrastructure,
		infrastructureConfig: internalInfrastructureConfig,
		providerStatus:       internalInfrastructureStatus,
		cluster:              cluster,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
		clusterTag:           fmt.Sprintf("%s=%s", tag.ClusterID, string(cluster.Shoot.GetUID())),
		machineID:            decodeMachineID(internalInfrastructureStatus.Firewall.MachineID),
	}

	err = reconcileFirewall(ctx, reconciler)
	if err != nil {
		return err
	}

	return nil
}

func reconcileFirewall(ctx context.Context, r *firewallReconciler) error {
	var (
		firewallInProgress = r.machineID != "" && !r.providerStatus.Firewall.Succeeded
	)

	if r.providerStatus.Firewall.Succeeded {
		action, err := firewallNextAction(ctx, r)
		if err != nil {
			return err
		}

		switch action {
		case firewallActionDoNothing:
			return nil
		case firewallActionRecreate:
		case firewallActionDeleteAndRecreate:
			err := deleteFirewall(r.logger, r.machineID, r.infrastructureConfig.ProjectID, r.clusterTag, r.mclient)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported firewall action: %s", action)
		}

		r.providerStatus.Firewall.MachineID = ""
		r.providerStatus.Firewall.Succeeded = false
		err = updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, r.infrastructure.Status.NodesCIDR)
		if err != nil {
			return err
		}
	}

	if firewallInProgress {
		succeeded, err := hasFirewallSucceeded(r.machineID, r.mclient)
		if err != nil {
			return err
		}

		r.providerStatus.Firewall.Succeeded = succeeded
		return updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, r.infrastructure.Status.NodesCIDR)
	}

	err := createFirewall(ctx, r)
	if err != nil {
		return err
	}

	return nil
}

func firewallNextAction(ctx context.Context, r *firewallReconciler) (firewallReconcileAction, error) {
	firewalls, err := metalclient.FindClusterFirewalls(r.mclient, r.clusterTag, r.infrastructureConfig.ProjectID)
	if err != nil {
		return firewallActionDoNothing, &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	clusterFirewallAmount := len(firewalls)
	switch clusterFirewallAmount {
	case 0:
		r.logger.Info("firewall does not exist anymore, recreation required", "clusterid", r.clusterID, "machineid", r.machineID)
		return firewallActionRecreate, nil
	case 1:
		fw := firewalls[0]
		if *fw.ID != r.machineID {
			r.logger.Error(
				fmt.Errorf("machine id of this cluster's firewall differs from infrastructure status"),
				"leaving as it is, but something unexpected must have happened in the past. if you want to get to a clean state, remove the firewall by hand (causes downtime!) and reconcile infrastructure again",
				"clusterID", r.clusterID,
				"expectedMachineID", r.machineID,
				"actualMachineID", *fw.ID,
			)
			return firewallActionDoNothing, nil
		}

		if fw.Size.ID != nil && *fw.Size.ID != r.infrastructureConfig.Firewall.Size {
			r.logger.Info("firewall size has changed, recreating firewall", "clusterid", r.clusterID, "machineid", r.machineID, "current", *fw.Size.ID, "new", r.infrastructureConfig.Firewall.Size)
			return firewallActionDeleteAndRecreate, nil
		}

		if fw.Allocation != nil && fw.Allocation.Image != nil && fw.Allocation.Image.ID != nil && *fw.Allocation.Image.ID != r.infrastructureConfig.Firewall.Image {
			want := r.infrastructureConfig.Firewall.Image
			image, err := r.mclient.ImageGetLatest(want)
			if err != nil {
				return firewallActionDoNothing, &controllererrors.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}

			if image.Image != nil && image.Image.ID != nil && *image.Image.ID != *fw.Allocation.Image.ID {
				r.logger.Info("firewall image has changed, recreating firewall", "clusterid", r.clusterID, "machineid", r.machineID, "current", *fw.Allocation.Image.ID, "new", *image.Image.ID)
				return firewallActionDeleteAndRecreate, nil
			}
		}

		return firewallActionDoNothing, nil
	default:
		r.logger.Error(
			fmt.Errorf("multiple firewalls exist for this cluster, which is currently unsupported. delete these firewalls and try to keep the one with machine id %q", r.machineID),
			"clusterID", r.clusterID,
			"expectedMachineID", r.machineID,
		)
		return firewallActionDoNothing, nil
	}
}

func hasFirewallSucceeded(machineID string, mclient *metalgo.Driver) (bool, error) {
	resp, err := mclient.FirewallGet(machineID)
	if err != nil {
		switch e := err.(type) {
		case *metalfirewall.FindFirewallDefault:
			if e.Code() >= 500 {
				return false, &controllererrors.RequeueAfterError{
					Cause:        e,
					RequeueAfter: 5 * time.Second,
				}
			}
		default:
			return false, e
		}
	}

	if resp.Firewall == nil || resp.Firewall.Allocation == nil || resp.Firewall.Allocation.Succeeded == nil {
		return false, fmt.Errorf("firewall %q was created but has no allocation", machineID)
	}

	return *resp.Firewall.Allocation.Succeeded, nil
}

func createFirewall(ctx context.Context, r *firewallReconciler) error {
	nodeCIDR, err := ensureNodeNetwork(ctx, r.clusterID, r.mclient, r.infrastructure, r.infrastructureConfig, r.cluster)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	err = updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, &nodeCIDR)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	clusterName := r.cluster.ObjectMeta.Name
	name := clusterName + "-firewall-" + uuid.String()[:5]

	// find private network
	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(r.mclient, r.infrastructureConfig.ProjectID, nodeCIDR)
	if err != nil {
		return err
	}

	kubeconfig, kubeconfigCompatibility, err := createFirewallControllerKubeconfig(ctx, r.clientset, r.gc, r.infrastructure, r.cluster)
	if err != nil {
		return err
	}

	firewallUserData, err := renderFirewallUserData(kubeconfig, kubeconfigCompatibility)
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
	for _, n := range r.infrastructureConfig.Firewall.Networks {
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
			Size:          r.infrastructureConfig.Firewall.Size,
			Project:       r.infrastructureConfig.ProjectID,
			Partition:     r.infrastructureConfig.PartitionID,
			Image:         r.infrastructureConfig.Firewall.Image,
			SSHPublicKeys: []string{string(r.infrastructure.Spec.SSHPublicKey)},
			Networks:      networks,
			UserData:      firewallUserData,
			Tags:          []string{r.clusterTag},
		},
	}

	r.logger.Info("create firewall", "name", createRequest.Name)
	fcr, err := r.mclient.FirewallCreate(createRequest)
	if err != nil {
		r.logger.Error(err, "failed to create firewall", "infrastructure", r.infrastructure.Name)
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

	r.providerStatus.Firewall.MachineID = machineID
	r.providerStatus.Firewall.Succeeded = true

	return updateProviderStatus(ctx, r.c, r.infrastructure, r.providerStatus, &nodeCIDR)
}

func ensureNodeNetwork(ctx context.Context, clusterID string, mclient *metalgo.Driver, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureConfig *metalapi.InfrastructureConfig, cluster *extensionscontroller.Cluster) (string, error) {
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

	return nodeCIDR, nil
}

func createFirewallControllerKubeconfig(
	ctx context.Context,
	c kubernetes.Interface,
	gc gardenerkubernetes.Interface,
	infrastructure *extensionsv1alpha1.Infrastructure,
	cluster *extensionscontroller.Cluster,
) (string, string, error) {
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

	secret, err := infrastructureSecrets.Deploy(ctx, c, gc, infrastructure.Namespace)
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

func renderFirewallUserData(kubeconfig, kubeconfigCompatibility string) (string, error) {
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
