package controlplane

import (
	"context"
	"fmt"
	"net/url"
	"slices"

	"github.com/Masterminds/semver/v3"
	"github.com/coreos/go-systemd/v22/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/gardener/gardener/pkg/component/nodemanagement/machinecontrollermanager"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/go-logr/logr"

	apimetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(mgr manager.Manager, logger logr.Logger, controllerConfig config.ControllerConfiguration) genericmutator.Ensurer {
	return &ensurer{
		logger:           logger.WithName("metal-controlplane-ensurer"),
		controllerConfig: controllerConfig,
		client:           mgr.GetClient(),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client           client.Client
	logger           logr.Logger
	controllerConfig config.ControllerConfiguration
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, _ *appsv1.Deployment) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := e.client.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, infrastructure); err != nil {
		logger.Error(err, "could not read Infrastructure for cluster", "cluster name", cluster.ObjectMeta.Name)
		return err
	}

	nodeCIDR, err := helper.GetNodeCIDR(infrastructure, cluster)
	if err != nil {
		return err
	}

	template := &new.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c)
	}
	if c := extensionswebhook.ContainerWithName(ps.Containers, "vpn-seed"); c != nil {
		ensureVPNSeedEnvVars(c, nodeCIDR)
	}

	return e.ensureChecksumAnnotations(ctx, &new.Spec.Template, new.Namespace)
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")
}

func ensureVPNSeedEnvVars(c *corev1.Container, nodeCIDR string) {
	// fixes a regression from https://github.com/gardener/gardener/pull/4691
	// raising the timeout to 15 minutes leads to additional 15 minutes of provisioning time because
	// the nodes cidr will only be set on next shoot reconcile
	// with the following mutation we can immediately provide the proper nodes cidr and save time
	logger.Info("ensuring nodes cidr in container", "container", c.Name, "cidr", nodeCIDR)
	c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
		Name:  "NODE_NETWORK",
		Value: nodeCIDR,
	})
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeControllerManagerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, _ *appsv1.Deployment) error {
	template := &new.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-controller-manager"); c != nil {
		ensureKubeControllerManagerCommandLineArgs(c)
	}
	ensureKubeControllerManagerAnnotations(template)
	return e.ensureChecksumAnnotations(ctx, &new.Spec.Template, new.Namespace)
}

func ensureKubeControllerManagerCommandLineArgs(c *corev1.Container) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")
}

func ensureKubeControllerManagerAnnotations(t *corev1.PodTemplateSpec) {
	// TODO: These labels should be exposed as constants in Gardener
	t.Labels = extensionswebhook.EnsureAnnotationOrLabel(t.Labels, "networking.gardener.cloud/to-public-networks", "allowed")
	t.Labels = extensionswebhook.EnsureAnnotationOrLabel(t.Labels, "networking.gardener.cloud/to-private-networks", "allowed")
	t.Labels = extensionswebhook.EnsureAnnotationOrLabel(t.Labels, "networking.gardener.cloud/to-blocked-cidrs", "allowed")
}

func (e *ensurer) ensureChecksumAnnotations(ctx context.Context, template *corev1.PodTemplateSpec, namespace string) error {
	return controlplane.EnsureSecretChecksumAnnotation(ctx, template, e.client, namespace, v1beta1constants.SecretNameCloudProvider)
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *ensurer) EnsureKubeletServiceUnitOptions(ctx context.Context, gctx gcontext.GardenContext, kubeletVersion *semver.Version, new, _ []*unit.UnitOption) ([]*unit.UnitOption, error) {

	// FIXME Why ?
	if opt := extensionswebhook.UnitOptionWithSectionAndName(new, "Service", "ExecStart"); opt != nil {
		command := extensionswebhook.DeserializeCommandLine(opt.Value)
		command = ensureKubeletCommandLineArgs(command)
		opt.Value = extensionswebhook.SerializeCommandLine(command, 1, " \\\n    ")
	}
	return new, nil
}

func ensureKubeletCommandLineArgs(command []string) []string {
	command = extensionswebhook.EnsureStringWithPrefix(command, "--cloud-provider=", "external")
	return command
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *ensurer) EnsureKubeletConfiguration(ctx context.Context, gctx gcontext.GardenContext, kubeletVersion *semver.Version, new, _ *kubeletconfigv1beta1.KubeletConfiguration) error {

	// Make sure CSI-related feature gates are not enabled
	// TODO Leaving these enabled shouldn't do any harm, perhaps remove this code when properly tested?
	// FIXME Why ?
	delete(new.FeatureGates, "VolumeSnapshotDataSource")
	delete(new.FeatureGates, "CSINodeInfo")
	delete(new.FeatureGates, "CSIDriverRegistry")
	return nil
}

// EnsureVPNSeedServerDeployment ensures that the vpn seed server deployment configuration conforms to the provider requirements.
func (e *ensurer) EnsureVPNSeedServerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, _ *appsv1.Deployment) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := e.client.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, infrastructure); err != nil {
		logger.Error(err, "could not read Infrastructure for cluster", "cluster name", cluster.ObjectMeta.Name)
		return err
	}

	nodeCIDR, err := helper.GetNodeCIDR(infrastructure, cluster)
	if err != nil {
		return err
	}

	template := &new.Spec.Template
	ps := &template.Spec

	if c := extensionswebhook.ContainerWithName(ps.Containers, "vpn-seed-server"); c != nil {
		ensureVPNSeedEnvVars(c, nodeCIDR)
	}

	return nil
}

// ImageVector is exposed for testing.
var ImageVector = imagevector.ImageVector()

// EnsureMachineControllerManagerDeployment ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerDeployment(_ context.Context, _ gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
	image, err := ImageVector.FindImage(metal.MCMProviderMetalImageName)
	if err != nil {
		return err
	}

	c := machinecontrollermanager.ProviderSidecarContainer(newObj.Namespace, metal.Name, image.String())
	c.Args = extensionswebhook.EnsureStringWithPrefix(c.Args, "--machine-health-timeout=", "10080m")

	newObj.Spec.Template.Spec.Containers = extensionswebhook.EnsureContainerWithName(
		newObj.Spec.Template.Spec.Containers,
		c,
	)
	return nil
}

// EnsureMachineControllerManagerVPA ensures that the machine-controller-manager VPA conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerVPA(_ context.Context, _ gcontext.GardenContext, newObj, _ *vpaautoscalingv1.VerticalPodAutoscaler) error {
	if newObj.Spec.ResourcePolicy == nil {
		newObj.Spec.ResourcePolicy = &vpaautoscalingv1.PodResourcePolicy{}
	}

	newObj.Spec.ResourcePolicy.ContainerPolicies = extensionswebhook.EnsureVPAContainerResourcePolicyWithName(
		newObj.Spec.ResourcePolicy.ContainerPolicies,
		machinecontrollermanager.ProviderSidecarVPAContainerPolicy(metal.Name),
	)
	return nil
}

func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx gcontext.GardenContext, new, old *[]extensionsv1alpha1.File) error {
	if new == nil {
		return nil
	}

	var files []extensionsv1alpha1.File
	for _, f := range *new {
		if f.Path == v1beta1constants.OperatingSystemConfigFilePathKubeletConfig {
			// for cis benchmark this needs to be 600
			f.Permissions = pointer.Pointer(int32(0600))
		}

		files = append(files, f)
	}

	*new = files

	return nil
}

// EnsureCRIConfig ensures the CRI config.
// "old" might be "nil" and must always be checked.
func (e *ensurer) EnsureCRIConfig(ctx context.Context, gctx gcontext.GardenContext, new, _ *extensionsv1alpha1.CRIConfig) error {
	if new == nil || new.Name != extensionsv1alpha1.CRINameContainerD {
		return nil
	}

	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	cpConfig, err := helper.ControlPlaneConfigFromClusterShootSpec(cluster)
	if err != nil {
		return err
	}

	if cpConfig.NetworkAccessType == nil || *cpConfig.NetworkAccessType == apimetal.NetworkAccessBaseline {
		return nil
	}

	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := e.client.Get(ctx, client.ObjectKey{Namespace: cluster.ObjectMeta.Name, Name: cluster.Shoot.Name}, infrastructure); err != nil {
		logger.Error(err, "could not read Infrastructure for cluster", "cluster name", cluster.ObjectMeta.Name)
		return err
	}

	infrastructureConfig, err := helper.InfrastructureConfigFromInfrastructure(infrastructure)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return err
	}

	_, p, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	if p.NetworkIsolation == nil {
		return nil
	}

	if new.Containerd == nil {
		new.Containerd = &extensionsv1alpha1.ContainerdConfig{}
	}

	new.Containerd.Registries, err = ensureContainerdRegistries(p.NetworkIsolation.RegistryMirrors, new.Containerd.Registries)
	if err != nil {
		return err
	}

	return nil
}

func ensureContainerdRegistries(mirrors []apimetal.RegistryMirror, configs []extensionsv1alpha1.RegistryConfig) ([]extensionsv1alpha1.RegistryConfig, error) {
	var result []extensionsv1alpha1.RegistryConfig
	for _, c := range configs {
		result = append(result, *c.DeepCopy())
	}

	for _, r := range mirrors {
		url, err := url.Parse(r.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("unable to parse registry endpoint: %w", err)
		}

		registryConfig := extensionsv1alpha1.RegistryConfig{
			Upstream:       url.Host,
			ReadinessProbe: pointer.Pointer(false),
		}

		for _, mirrorOf := range r.MirrorOf {
			registryConfig.Hosts = append(registryConfig.Hosts, extensionsv1alpha1.RegistryHost{
				URL:          mirrorOf,
				Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
			})
		}

		idx := slices.IndexFunc(result, func(rc extensionsv1alpha1.RegistryConfig) bool {
			return rc.Upstream == url.Host
		})

		if idx < 0 {
			result = append(result, registryConfig)
		} else {
			result[idx] = registryConfig
		}
	}

	return result, nil
}
