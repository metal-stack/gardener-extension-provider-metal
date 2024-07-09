package controlplane

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/coreos/go-systemd/v22/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/gardener/gardener/pkg/component/machinecontrollermanager"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	vpaautoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	if err := e.client.Get(ctx, kutil.Key(cluster.ObjectMeta.Name, cluster.Shoot.Name), infrastructure); err != nil {
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
	if err := e.client.Get(ctx, kutil.Key(cluster.ObjectMeta.Name, cluster.Shoot.Name), infrastructure); err != nil {
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

// TODO:
// - Write configuration also into shoot.spec.worker[n].image.providerconfig, but then the worker rolls ?
// - calculate hash over containerd/config.toml and add to containerd.service Unit to trigger restart on changes

// EnsureAdditionalFiles adds additional files to override DNS and NTP configurations from the NetworkIsolation.
func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx gcontext.GardenContext, new, old *[]extensionsv1alpha1.File) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	controlPlaneConfig, err := helper.ControlPlaneConfigFromClusterShootSpec(cluster)
	if err != nil {
		return err
	}

	networkAccessType := metalapi.NetworkAccessBaseline
	if controlPlaneConfig.NetworkAccessType != nil {
		networkAccessType = *controlPlaneConfig.NetworkAccessType
	}

	if networkAccessType == metalapi.NetworkAccessBaseline {
		return nil
	}

	infra := &extensionsv1alpha1.Infrastructure{}
	if err := e.client.Get(ctx, kutil.Key(cluster.ObjectMeta.Name, cluster.Shoot.Name), infra); err != nil {
		logger.Error(err, "could not read Infrastructure for cluster", "cluster name", cluster.ObjectMeta.Name)
		return err
	}

	infraConf, err := helper.InfrastructureConfigFromInfrastructure(infra)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return err
	}

	_, partition, err := helper.FindMetalControlPlane(cloudProfileConfig, infraConf.PartitionID)
	if err != nil {
		return err
	}

	if partition.NetworkIsolation == nil {
		return nil
	}

	if networkAccessType != metalapi.NetworkAccessBaseline {
		dnsFiles := additionalDNSConfFiles(partition.NetworkIsolation.DNSServers)
		for _, f := range dnsFiles {
			*new = extensionswebhook.EnsureFileWithPath(*new, f)
		}

		ntpFiles := additionalNTPConfFiles(partition.NetworkIsolation.NTPServers)
		for _, f := range ntpFiles {
			*new = extensionswebhook.EnsureFileWithPath(*new, f)
		}

		containerdFiles := additionalContainterdConfigFiles(partition.NetworkIsolation.RegistryMirrors)
		for _, f := range containerdFiles {
			*new = extensionswebhook.EnsureFileWithPath(*new, f)
		}
	}

	return nil
}

func additionalDNSConfFiles(dnsServers []string) []extensionsv1alpha1.File {
	resolveDNS := strings.Join(dnsServers, " ")
	systemdResolvedConfd := fmt.Sprintf(`# Generated by gardener-extension-provider-metal

[Resolve]
DNS=%s
Domain=~.

`, resolveDNS)
	resolvConf := "# Generated by gardener-extension-provider-metal\n"
	for _, ip := range dnsServers {
		resolvConf += fmt.Sprintf("nameserver %s\n", ip)
	}

	return []extensionsv1alpha1.File{
		{
			Path: "/etc/systemd/resolved.conf.d/dns.conf",
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: string(extensionsv1alpha1.B64FileCodecID),
					Data:     base64.StdEncoding.EncodeToString([]byte(systemdResolvedConfd)),
				},
			},
		},
		{
			Path: "/etc/resolv.conf",
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: string(extensionsv1alpha1.B64FileCodecID),
					Data:     base64.StdEncoding.EncodeToString([]byte(resolvConf)),
				},
			},
		},
	}
}

func additionalNTPConfFiles(ntpServers []string) []extensionsv1alpha1.File {
	ntps := strings.Join(ntpServers, " ")
	renderedContent := fmt.Sprintf(`# Generated by gardener-extension-provider-metal

[Time]
NTP=%s
`, ntps)

	return []extensionsv1alpha1.File{
		{
			Path: "/etc/systemd/timesyncd.conf",
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: string(extensionsv1alpha1.B64FileCodecID),
					Data:     base64.StdEncoding.EncodeToString([]byte(renderedContent)),
				},
			},
		},
	}
}

func additionalContainterdConfigFiles(mirrors []metalapi.RegistryMirror) []extensionsv1alpha1.File {
	if len(mirrors) == 0 {
		return nil
	}
	// TODO: other parties might also want to write to the containerd config.toml.
	// For this case we might want to unmarshal any existing new file to add and patch it with our changes.
	renderedContent := `# Generated by gardener-extension-provider-metal
imports = ["/etc/containerd/conf.d/*.toml"]
version = 2

[plugins."io.containerd.grpc.v1.cri".registry]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`
	for _, m := range mirrors {
		for _, of := range m.MirrorOf {
			renderedContent += fmt.Sprintf(`    [plugins."io.containerd.grpc.v1.cri".registry.mirrors.%q]
      endpoint = [%q]
`, of, m.Endpoint)
		}
	}

	return []extensionsv1alpha1.File{
		{
			Path: "/etc/containerd/config.toml",
			Content: extensionsv1alpha1.FileContent{
				Inline: &extensionsv1alpha1.FileContentInline{
					Encoding: string(extensionsv1alpha1.B64FileCodecID),
					Data:     base64.StdEncoding.EncodeToString([]byte(renderedContent)),
				},
			},
		},
	}
}

// ImageVector is exposed for testing.
var ImageVector = imagevector.ImageVector()

// EnsureMachineControllerManagerDeployment ensures that the machine-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerDeployment(_ context.Context, _ gcontext.GardenContext, newObj, _ *appsv1.Deployment) error {
	image, err := ImageVector.FindImage(metal.MCMProviderMetalImageName)
	if err != nil {
		return err
	}

	// TODO: Add back our settings
	// - --machine-drain-timeout=2h
	// - --machine-health-timeout=10080m
	// - --machine-safety-apiserver-statuscheck-timeout=30s
	// - --machine-safety-apiserver-statuscheck-period=1m
	// - --machine-safety-orphan-vms-period=30m

	newObj.Spec.Template.Spec.Containers = extensionswebhook.EnsureContainerWithName(
		newObj.Spec.Template.Spec.Containers,
		machinecontrollermanager.ProviderSidecarContainer(newObj.Namespace, metal.Name, image.String()),
	)
	return nil
}

// EnsureMachineControllerManagerVPA ensures that the machine-controller-manager VPA conforms to the provider requirements.
func (e *ensurer) EnsureMachineControllerManagerVPA(_ context.Context, _ gcontext.GardenContext, newObj, _ *vpaautoscalingv1.VerticalPodAutoscaler) error {
	var (
		minAllowed = corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
		}
		maxAllowed = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("5G"),
		}
	)

	if newObj.Spec.ResourcePolicy == nil {
		newObj.Spec.ResourcePolicy = &vpaautoscalingv1.PodResourcePolicy{}
	}

	newObj.Spec.ResourcePolicy.ContainerPolicies = extensionswebhook.EnsureVPAContainerResourcePolicyWithName(
		newObj.Spec.ResourcePolicy.ContainerPolicies,
		machinecontrollermanager.ProviderSidecarVPAContainerPolicy(metal.Name, minAllowed, maxAllowed),
	)
	return nil
}
