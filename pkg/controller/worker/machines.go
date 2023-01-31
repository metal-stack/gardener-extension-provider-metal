package worker

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Masterminds/semver"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/client/firewall"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/tag"
	metaltag "github.com/metal-stack/metal-lib/pkg/tag"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"

	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
)

// MachineClassKind yields the name of the machine class.
func (w *workerDelegate) MachineClassKind() string {
	return "MachineClass"
}

// MachineClass yields a newly initialized MachineClass object.
func (w *workerDelegate) MachineClass() client.Object {
	return &machinev1alpha1.MachineClass{}
}

// MachineClassList yields a newly initialized MachineClassList object.
func (w *workerDelegate) MachineClassList() client.ObjectList {
	return &machinev1alpha1.MachineClassList{}
}

// DeployMachineClasses generates and creates the metal specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	if w.machineClasses == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return err
		}
	}

	values := kubernetes.Values(map[string]interface{}{"machineClasses": w.machineClasses})

	return w.seedChartApplier.Apply(ctx, filepath.Join(metal.InternalChartsPath, "machineclass"), w.worker.Namespace, "machineclass", values)
}

// GenerateMachineDeployments generates the configuration for the desired machine deployments.
func (w *workerDelegate) GenerateMachineDeployments(ctx context.Context) (worker.MachineDeployments, error) {
	if w.machineDeployments == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return nil, err
		}
	}
	return w.machineDeployments, nil
}

func (w *workerDelegate) generateMachineConfig(ctx context.Context) error {
	var (
		machineDeployments = worker.MachineDeployments{}
		machineClasses     []map[string]interface{}
		machineImages      []apismetal.MachineImage
	)

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(w.cluster)
	if err != nil {
		return err
	}

	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := w.decoder.Decode(w.cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	credentials, err := metalclient.ReadCredentialsFromSecretRef(ctx, w.client, &w.worker.Spec.SecretRef)
	if err != nil {
		return err
	}

	mclient, err := metalclient.NewClientFromCredentials(metalControlPlane.Endpoint, credentials)
	if err != nil {
		return err
	}

	// TODO: this is a workaround to speed things for the time being...
	// the infrastructure controller writes the nodes cidr back into the infrastructure status, but the cluster resource does not contain it immediately
	// it would need the start of another reconcilation until the node cidr can be picked up from the cluster resource
	// therefore, we read it directly from the infrastructure status
	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := w.client.Get(ctx, kutil.Key(w.worker.Namespace, w.cluster.Shoot.Name), infrastructure); err != nil {
		return err
	}

	sshSecret, err := helper.GetLatestSSHSecret(ctx, w.client, w.cluster.ObjectMeta.Name)
	if err != nil {
		return fmt.Errorf("could not find current ssh secret: %w", err)
	}

	projectID := infrastructureConfig.ProjectID
	nodeCIDR := infrastructure.Status.NodesCIDR

	if nodeCIDR == nil {
		if w.cluster.Shoot.Spec.Networking.Nodes == nil {
			return fmt.Errorf("nodeCIDR was not yet set by infrastructure controller")
		}
		nodeCIDR = w.cluster.Shoot.Spec.Networking.Nodes
	}

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(ctx, mclient, projectID, *nodeCIDR)
	if err != nil {
		return err
	}

	err = w.migrateFirewall(ctx, metalControlPlane, infrastructureConfig, w.cluster, mclient, *privateNetwork.ID)
	if err != nil {
		return err
	}

	err = w.ensureFirewallDeployment(ctx, metalControlPlane, infrastructureConfig, w.cluster, sshSecret, *privateNetwork.ID)
	if err != nil {
		return err
	}

	for _, pool := range w.worker.Spec.Pools {
		workerPoolHash, err := worker.WorkerPoolHash(pool, w.cluster)
		if err != nil {
			return err
		}

		machineImage, err := w.findMachineImage(pool.MachineImage.Name, pool.MachineImage.Version)
		if err != nil {
			return err
		}
		machineImages = appendMachineImage(machineImages, apismetal.MachineImage{
			Name:    pool.MachineImage.Name,
			Version: pool.MachineImage.Version,
			Image:   machineImage,
		})

		var (
			metalClusterIDTag      = fmt.Sprintf("%s=%s", metaltag.ClusterID, w.cluster.Shoot.GetUID())
			metalClusterNameTag    = fmt.Sprintf("%s=%s", metaltag.ClusterName, w.worker.Namespace)
			metalClusterProjectTag = fmt.Sprintf("%s=%s", metaltag.ClusterProject, infrastructureConfig.ProjectID)

			kubernetesClusterTag        = fmt.Sprintf("kubernetes.io/cluster=%s", w.worker.Namespace)
			kubernetesRoleTag           = "kubernetes.io/role=node"
			kubernetesInstanceTypeTag   = fmt.Sprintf("node.kubernetes.io/instance-type=%s", pool.MachineType)
			kubernetesTopologyRegionTag = fmt.Sprintf("topology.kubernetes.io/region=%s", w.worker.Spec.Region)
			kubernetesTopologyZoneTag   = fmt.Sprintf("topology.kubernetes.io/zone=%s", infrastructureConfig.PartitionID)
		)

		tags := []string{
			kubernetesClusterTag,
			kubernetesRoleTag,
			kubernetesInstanceTypeTag,
			kubernetesTopologyRegionTag,
			kubernetesTopologyZoneTag,

			metalClusterIDTag,
			metalClusterNameTag,
			metalClusterProjectTag,
		}

		for k, v := range pool.Labels {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}

		machineClassSpec := map[string]interface{}{
			"partition": infrastructureConfig.PartitionID,
			"size":      pool.MachineType,
			"project":   projectID,
			"network":   privateNetwork.ID,
			"image":     machineImage,
			"tags":      tags,
			"sshkeys":   []string{string(w.worker.Spec.SSHPublicKey)},
			"secret": map[string]interface{}{
				"cloudConfig": string(pool.UserData),
			},
			"credentialsSecretRef": map[string]interface{}{
				"name":      w.worker.Spec.SecretRef.Name,
				"namespace": w.worker.Spec.SecretRef.Namespace,
			},
		}

		var (
			deploymentName = fmt.Sprintf("%s-%s", w.worker.Namespace, pool.Name)
			className      = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
		)

		machineDeployments = append(machineDeployments, worker.MachineDeployment{
			Name:                 deploymentName,
			ClassName:            className,
			SecretName:           className,
			Minimum:              pool.Minimum,
			Maximum:              pool.Maximum,
			MaxSurge:             pool.MaxSurge,
			MaxUnavailable:       pool.MaxUnavailable,
			Labels:               pool.Labels,
			Annotations:          pool.Annotations,
			Taints:               pool.Taints,
			MachineConfiguration: genericworkeractuator.ReadMachineConfiguration(pool),
		})

		machineClassSpec["name"] = className
		machineClassSpec["labels"] = map[string]string{
			v1beta1constants.GardenerPurpose: genericworkeractuator.GardenPurposeMachineClass,
		}

		// if we'd move the endpoint out of this secret into the deployment spec (which would be the way to go)
		// it would roll all worker nodes...
		machineClassSpec["secret"].(map[string]interface{})["metalAPIURL"] = metalControlPlane.Endpoint
		machineClassSpec["secret"].(map[string]interface{})[metal.APIKey] = credentials.MetalAPIKey
		machineClassSpec["secret"].(map[string]interface{})[metal.APIHMac] = credentials.MetalAPIHMac

		machineClasses = append(machineClasses, machineClassSpec)
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses

	return nil
}

// migrateFirewall can be removed along with the deployment of the old firewall resource after all firewall are running firewall-controller >= v0.2.0
func (w *workerDelegate) migrateFirewall(ctx context.Context, metalControlPlane *apismetal.MetalControlPlane, infrastructureConfig *apismetal.InfrastructureConfig, cluster *extensionscontroller.Cluster, mclient metalgo.Client, privateNetworkID string) error {
	var (
		clusterID = string(cluster.Shoot.GetUID())
		projectID = infrastructureConfig.ProjectID
		namespace = cluster.ObjectMeta.Name
	)

	resp, err := mclient.Firewall().FindFirewalls(firewall.NewFindFirewallsParams().WithBody(&models.V1FirewallFindRequest{
		AllocationProject: projectID,
		Tags:              []string{clusterTag(clusterID)},
	}).WithContext(ctx), nil)
	if err != nil {
		return fmt.Errorf("error finding firewall: %w", err)
	}

	var toMigrate []*models.V1FirewallResponse

	for _, fw := range resp.Payload {
		tm := tag.NewTagMap(fw.Tags)

		if _, ok := tm.Value(fcmv2.FirewallControllerSetAnnotation); ok {
			// firewall is already owned by the firewall-cotroller-manager, does not need migration
			continue
		}

		toMigrate = append(toMigrate, fw)
	}

	if len(toMigrate) == 0 {
		w.logger.Info("no firewalls to be migrated to firewall-controller-manager")
		return nil
	}

	rateLimit := func(limits []apismetal.RateLimit) []fcmv2.RateLimit {
		var result []fcmv2.RateLimit
		for _, l := range limits {
			result = append(result, fcmv2.RateLimit{
				NetworkID: l.NetworkID,
				Rate:      l.RateLimit,
			})
		}
		return result
	}

	egressRules := func(egress []apismetal.EgressRule) []fcmv2.EgressRuleSNAT {
		var result []fcmv2.EgressRuleSNAT
		for _, rule := range egress {
			rule := rule
			result = append(result, fcmv2.EgressRuleSNAT{
				NetworkID: rule.NetworkID,
				IPs:       rule.IPs,
			})
		}
		return result
	}

	fwcv, err := validation.ValidateFirewallControllerVersion(metalControlPlane.FirewallControllerVersions, infrastructureConfig.Firewall.ControllerVersion)
	if err != nil {
		return err
	}

	internalPrefixes := []string{}
	if w.controllerConfig.AccountingExporter.Enabled && w.controllerConfig.AccountingExporter.NetworkTraffic.Enabled {
		internalPrefixes = w.controllerConfig.AccountingExporter.NetworkTraffic.InternalNetworks
	}

	for _, fw := range toMigrate {
		fw := fw

		f := &fcmv2.Firewall{
			ObjectMeta: metav1.ObjectMeta{
				Name:      *fw.Allocation.Name,
				Namespace: namespace,
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, w.client, f, func() error {
			f.Labels = map[string]string{
				tag.ClusterID: clusterID,
			}
			if v, err := semver.NewVersion(fwcv.Version); err == nil && v.LessThan(semver.MustParse("v2.0.0")) {
				f.Annotations = map[string]string{
					fcmv2.FirewallNoControllerConnectionAnnotation: "true",
				}
			}
			f.Spec = fcmv2.FirewallSpec{
				Size:                   *fw.Size.ID,
				Image:                  *fw.Allocation.Image.ID,
				Partition:              *fw.Partition.ID,
				Project:                *fw.Allocation.Project,
				Networks:               append(infrastructureConfig.Firewall.Networks, privateNetworkID),
				Userdata:               fw.Allocation.UserData,
				SSHPublicKeys:          fw.Allocation.SSHPubKeys,
				RateLimits:             rateLimit(infrastructureConfig.Firewall.RateLimits),
				InternalPrefixes:       internalPrefixes,
				EgressRules:            egressRules(infrastructureConfig.Firewall.EgressRules),
				Interval:               "10s",
				DryRun:                 false,
				Ipv4RuleFile:           "",
				ControllerVersion:      fwcv.Version,
				ControllerURL:          fwcv.URL,
				LogAcceptedConnections: infrastructureConfig.Firewall.LogAcceptedConnections,
				DNSServerAddress:       "",
				DNSPort:                nil,
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error creating firewall resource for firewall migration: %w", err)
		}

		w.logger.Info("created firewall migration", "id", fw.ID, "cluster-id", clusterID)

	}

	return nil
}

func (w *workerDelegate) ensureFirewallDeployment(ctx context.Context, metalControlPlane *apismetal.MetalControlPlane, infrastructureConfig *apismetal.InfrastructureConfig, cluster *extensionscontroller.Cluster, sshSecret *corev1.Secret, privateNetworkID string) error {
	// why is this code here and not in the controlplane controller?
	// the controlplane controller deploys the firewall-controller-manager including validating and mutating webhooks
	// this has to be running before we can create a firewall deployment because the mutating webhook is creating the userdata
	// the worker controller acts after the controlplane controller, also the terms and responsibilities are pretty similar between machine-controller-manager and firewall-controller-manager
	var (
		clusterID = string(cluster.Shoot.GetUID())
		namespace = cluster.ObjectMeta.Name

		rateLimit = func(limits []apismetal.RateLimit) []fcmv2.RateLimit {
			var result []fcmv2.RateLimit
			for _, l := range limits {
				result = append(result, fcmv2.RateLimit{
					NetworkID: l.NetworkID,
					Rate:      l.RateLimit,
				})
			}
			return result
		}

		egressRules = func(egress []apismetal.EgressRule) []fcmv2.EgressRuleSNAT {
			var result []fcmv2.EgressRuleSNAT
			for _, rule := range egress {
				rule := rule
				result = append(result, fcmv2.EgressRuleSNAT{
					NetworkID: rule.NetworkID,
					IPs:       rule.IPs,
				})
			}
			return result
		}
	)

	internalPrefixes := []string{}
	if w.controllerConfig.AccountingExporter.Enabled && w.controllerConfig.AccountingExporter.NetworkTraffic.Enabled {
		internalPrefixes = w.controllerConfig.AccountingExporter.NetworkTraffic.InternalNetworks
	}

	fwcv, err := validation.ValidateFirewallControllerVersion(metalControlPlane.FirewallControllerVersions, infrastructureConfig.Firewall.ControllerVersion)
	if err != nil {
		return err
	}

	deploy := &fcmv2.FirewallDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shoot-firewall",
			Namespace: namespace,
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, w.client, deploy, func() error {
		deploy.Spec = fcmv2.FirewallDeploymentSpec{
			Strategy: fcmv2.StrategyRollingUpdate,
			Replicas: 1,
			Selector: map[string]string{
				tag.ClusterID: clusterID,
			},
			Template: fcmv2.FirewallTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						tag.ClusterID: clusterID,
					},
				},
				Spec: fcmv2.FirewallSpec{
					Size:                   infrastructureConfig.Firewall.Size,
					Image:                  infrastructureConfig.Firewall.Image,
					Partition:              infrastructureConfig.PartitionID,
					Project:                infrastructureConfig.ProjectID,
					Networks:               append(infrastructureConfig.Firewall.Networks, privateNetworkID),
					SSHPublicKeys:          []string{string(sshSecret.Data["id_rsa.pub"])},
					RateLimits:             rateLimit(infrastructureConfig.Firewall.RateLimits),
					InternalPrefixes:       internalPrefixes,
					EgressRules:            egressRules(infrastructureConfig.Firewall.EgressRules),
					Interval:               "10s",
					DryRun:                 false,
					Ipv4RuleFile:           "",
					ControllerVersion:      fwcv.Version,
					ControllerURL:          fwcv.URL,
					LogAcceptedConnections: infrastructureConfig.Firewall.LogAcceptedConnections,
					DNSServerAddress:       "",
					DNSPort:                nil,
				},
			},
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating firewall deployment: %w", err)
	}

	w.logger.Info("created firewall deployment", "name", deploy.Name, "cluster-id", clusterID)

	return nil
}

func clusterTag(clusterID string) string {
	return fmt.Sprintf("%s=%s", tag.ClusterID, clusterID)
}
