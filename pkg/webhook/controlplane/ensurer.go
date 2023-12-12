package controlplane

import (
	"context"
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/coreos/go-systemd/v22/unit"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"

	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(logger logr.Logger, controllerConfig config.ControllerConfiguration) genericmutator.Ensurer {
	return &ensurer{
		logger:           logger.WithName("metal-controlplane-ensurer"),
		controllerConfig: controllerConfig,
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client           client.Client
	logger           logr.Logger
	controllerConfig config.ControllerConfiguration
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, gctx gcontext.GardenContext, new, _ *appsv1.Deployment) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	cpConfig, err := helper.ControlPlaneConfigFromClusterShootSpec(cluster)
	if err != nil {
		logger.Error(err, "could not read ControlPlaneConfig from cluster shoot spec", "Cluster name", cluster.ObjectMeta.Name)
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

	makeAuditForwarder := false
	if validation.ClusterAuditEnabled(&e.controllerConfig, cpConfig) {
		makeAuditForwarder = true
	}
	if makeAuditForwarder {
		audittailersecret := &corev1.Secret{}
		if err := e.client.Get(ctx, kutil.Key(cluster.ObjectMeta.Name, gutil.SecretNamePrefixShootAccess+metal.AudittailerClientSecretName), audittailersecret); err != nil {
			logger.Error(err, "could not get secret for cluster", "secret", gutil.SecretNamePrefixShootAccess+metal.AudittailerClientSecretName, "cluster name", cluster.ObjectMeta.Name)
			makeAuditForwarder = false
		}
		if len(audittailersecret.Data) == 0 {
			logger.Error(err, "token for secret not yet set in cluster", "secret", gutil.SecretNamePrefixShootAccess+metal.AudittailerClientSecretName, "cluster name", cluster.ObjectMeta.Name)
			makeAuditForwarder = false
		}
	}

	genericTokenKubeconfigSecretName := extensionscontroller.GenericTokenKubeconfigSecretNameFromCluster(cluster)

	auditToSplunk := false
	if validation.AuditToSplunkEnabled(&e.controllerConfig, cpConfig) {
		auditToSplunk = true
	}

	template := &new.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c, makeAuditForwarder)
		ensureVolumeMounts(c, makeAuditForwarder)
		ensureVolumes(ps, genericTokenKubeconfigSecretName, makeAuditForwarder, auditToSplunk)
	}
	if c := extensionswebhook.ContainerWithName(ps.Containers, "vpn-seed"); c != nil {
		ensureVPNSeedEnvVars(c, nodeCIDR)
	}
	if makeAuditForwarder {
		// required because auditforwarder uses kube-apiserver and not localhost
		template.Labels["networking.resources.gardener.cloud/to-kube-apiserver-tcp-443"] = "allowed"

		err := ensureAuditForwarder(ps, auditToSplunk)
		if err != nil {
			logger.Error(err, "could not ensure the audit forwarder", "Cluster name", cluster.ObjectMeta.Name)
			return err
		}
		if auditToSplunk {
			err := controlplane.EnsureConfigMapChecksumAnnotation(ctx, &new.Spec.Template, e.client, new.Namespace, metal.AuditForwarderSplunkConfigName)
			if err != nil {
				logger.Error(err, "could not ensure the splunk config map checksum annotation", "cluster name", cluster.ObjectMeta.Name, "configmap", metal.AuditForwarderSplunkConfigName)
				return err
			}
			err = controlplane.EnsureSecretChecksumAnnotation(ctx, &new.Spec.Template, e.client, new.Namespace, metal.AuditForwarderSplunkSecretName)
			if err != nil {
				logger.Error(err, "could not ensure the splunk secret checksum annotation", "cluster name", cluster.ObjectMeta.Name, "secret", metal.AuditForwarderSplunkSecretName)
				return err
			}
		}
	}

	return e.ensureChecksumAnnotations(ctx, &new.Spec.Template, new.Namespace)
}

var (
	// config mount for the audit policy; it gets mounted where the kube-apiserver expects its audit policy.
	auditPolicyVolumeMount = corev1.VolumeMount{
		Name:      metal.AuditPolicyName,
		MountPath: "/etc/kubernetes/audit-override",
		ReadOnly:  true,
	}
	auditPolicyVolume = corev1.Volume{
		Name: metal.AuditPolicyName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: metal.AuditPolicyName},
			},
		},
	}
	auditForwarderSplunkConfigVolumeMount = corev1.VolumeMount{
		Name:      metal.AuditForwarderSplunkConfigName,
		MountPath: "/fluent-bit/etc/add",
		ReadOnly:  true,
	}
	auditForwarderSplunkConfigVolume = corev1.Volume{
		Name: metal.AuditForwarderSplunkConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: metal.AuditForwarderSplunkConfigName},
			},
		},
	}
	auditForwarderSplunkSecretVolumeMount = corev1.VolumeMount{
		Name:      metal.AuditForwarderSplunkSecretName,
		MountPath: "/fluent-bit/etc/splunkca",
		ReadOnly:  true,
	}
	auditForwarderSplunkSecretVolume = corev1.Volume{
		Name: metal.AuditForwarderSplunkSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: metal.AuditForwarderSplunkSecretName,
			},
		},
	}
	auditForwarderSplunkPodNameEnvVar = corev1.EnvVar{
		Name: "MY_POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
		},
	}
	auditForwarderSplunkHECTokenEnvVar = corev1.EnvVar{
		Name: "SPLUNK_HEC_TOKEN",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: metal.AuditForwarderSplunkSecretName,
				},
				Key: "splunk_hec_token",
			},
		},
	}
	auditLogVolumeMount = corev1.VolumeMount{
		Name:      "auditlog",
		MountPath: "/auditlog",
		ReadOnly:  false,
	}
	auditLogVolume = corev1.Volume{
		Name: "auditlog",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	auditKubeconfig = func(genericKubeconfigSecretName string) corev1.Volume {
		return corev1.Volume{
			Name: "kubeconfig",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					DefaultMode: pointer.Pointer(int32(420)),
					Sources: []corev1.VolumeProjection{
						{
							Secret: &corev1.SecretProjection{
								Items: []corev1.KeyToPath{
									{
										Key:  "kubeconfig",
										Path: "kubeconfig",
									},
								},
								Optional: pointer.Pointer(false),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: genericKubeconfigSecretName,
								},
							},
						},
						{
							Secret: &corev1.SecretProjection{
								Items: []corev1.KeyToPath{
									{
										Key:  "token",
										Path: "token",
									},
								},
								Optional: pointer.Pointer(false),
								LocalObjectReference: corev1.LocalObjectReference{
									Name: gutil.SecretNamePrefixShootAccess + metal.AudittailerClientSecretName,
								},
							},
						},
					},
				},
			},
		}
	}
	reversedVpnVolumeMounts = []corev1.VolumeMount{
		{
			Name:      "ca-vpn",
			MountPath: "/proxy/ca",
			ReadOnly:  true,
		},
		{
			Name:      "http-proxy",
			MountPath: "/proxy/client",
			ReadOnly:  true,
		},
	}
	kubeAggregatorClientTlsEnvVars = []corev1.EnvVar{
		{
			Name:  "AUDIT_PROXY_CA_FILE",
			Value: "/proxy/ca/bundle.crt",
		},
		{
			Name:  "AUDIT_PROXY_CLIENT_CRT_FILE",
			Value: "/proxy/client/tls.crt",
		},
		{
			Name:  "AUDIT_PROXY_CLIENT_KEY_FILE",
			Value: "/proxy/client/tls.key",
		},
	}
	auditForwarderSidecarTemplate = corev1.Container{
		Name: "auditforwarder",
		// Image:   // is added from the image vector in the ensure function
		ImagePullPolicy: "Always",
		Env: []corev1.EnvVar{
			{
				Name:  "AUDIT_KUBECFG",
				Value: path.Join(gutil.VolumeMountPathGenericKubeconfig, "kubeconfig"),
			},
			{
				Name:  "AUDIT_NAMESPACE",
				Value: metal.AudittailerNamespace,
			},
			{
				Name:  "AUDIT_SERVICE_NAME",
				Value: "audittailer",
			},
			{
				Name:  "AUDIT_SECRET_NAME",
				Value: metal.AudittailerClientSecretName,
			},
			{
				Name:  "AUDIT_AUDIT_LOG_PATH",
				Value: "/auditlog/audit.log",
			},
			{
				Name:  "AUDIT_TLS_CA_FILE",
				Value: "ca.crt",
			},
			{
				Name:  "AUDIT_TLS_CRT_FILE",
				Value: "tls.crt",
			},
			{
				Name:  "AUDIT_TLS_KEY_FILE",
				Value: "tls.key",
			},
			{
				Name:  "AUDIT_TLS_VHOST",
				Value: "audittailer",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("100Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("500Mi"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "kubeconfig",
				MountPath: gutil.VolumeMountPathGenericKubeconfig,
				ReadOnly:  true,
			},
			auditLogVolumeMount,
		},
	}
)

func ensureVolumeMounts(c *corev1.Container, makeAuditForwarder bool) {
	if makeAuditForwarder {
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, auditPolicyVolumeMount)
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, auditLogVolumeMount)
	}
}

func ensureVolumes(ps *corev1.PodSpec, genericKubeconfigSecretName string, makeAuditForwarder, auditToSplunk bool) {
	if makeAuditForwarder {

		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditKubeconfig(genericKubeconfigSecretName))
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditPolicyVolume)
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditLogVolume)
	}
	if auditToSplunk {
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditForwarderSplunkConfigVolume)
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditForwarderSplunkSecretVolume)
	}
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container, makeAuditForwarder bool) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")

	if makeAuditForwarder {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-policy-file=", "/etc/kubernetes/audit-override/audit-policy.yaml")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-log-path=", "/auditlog/audit.log")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-log-maxsize=", "100")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-log-maxbackup=", "1")
	}
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

func ensureAuditForwarder(ps *corev1.PodSpec, auditToSplunk bool) error {
	auditForwarderSidecar := auditForwarderSidecarTemplate.DeepCopy()
	auditForwarderImage, err := imagevector.ImageVector().FindImage("auditforwarder")
	if err != nil {
		logger.Error(err, "Could not find auditforwarder image in imagevector")
		return err
	}
	auditForwarderSidecar.Image = auditForwarderImage.String()

	var proxyHost string

	for _, volume := range ps.Volumes {
		switch volume.Name {
		case "egress-selection-config":
			proxyHost = "vpn-seed-server"
		}
	}

	if proxyHost != "" {
		err := ensureAuditForwarderProxy(auditForwarderSidecar, proxyHost)
		if err != nil {
			logger.Error(err, "could not ensure auditForwarder proxy")
			return err
		}
	}

	if auditToSplunk {
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(auditForwarderSidecar.VolumeMounts, auditForwarderSplunkConfigVolumeMount)
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(auditForwarderSidecar.VolumeMounts, auditForwarderSplunkSecretVolumeMount)
		auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, auditForwarderSplunkPodNameEnvVar)
		auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, auditForwarderSplunkHECTokenEnvVar)
	}

	logger.Info("ensuring audit forwarder sidecar", "container", auditForwarderSidecar.Name)

	ps.Containers = extensionswebhook.EnsureContainerWithName(ps.Containers, *auditForwarderSidecar)
	return nil
}

func ensureAuditForwarderProxy(auditForwarderSidecar *corev1.Container, proxyHost string) error {
	logger.Info("ensureAuditForwarderProxy called", "proxyHost=", proxyHost)
	proxyEnvVars := []corev1.EnvVar{
		{
			Name:  "AUDIT_PROXY_HOST",
			Value: proxyHost,
		},
		{
			Name:  "AUDIT_PROXY_PORT",
			Value: "9443",
		},
	}

	for _, envVar := range proxyEnvVars {
		auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, envVar)
	}

	switch proxyHost {
	case "vpn-seed-server":
		for _, envVar := range kubeAggregatorClientTlsEnvVars {
			auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, envVar)
		}
		for _, mount := range reversedVpnVolumeMounts {
			auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(auditForwarderSidecar.VolumeMounts, mount)
		}
	default:
		return fmt.Errorf("%q is not a valid proxy name", proxyHost)
	}

	return nil
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

// EnsureAdditionalFiles adds additional files to override DNS and NTP configurations from the NetworkIsolation.
func (e *ensurer) EnsureAdditionalFiles(ctx context.Context, gctx gcontext.GardenContext, new, old *[]extensionsv1alpha1.File) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
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

	controlPlaneConfig, err := helper.ControlPlaneConfigFromClusterShootSpec(cluster)
	if err != nil {
		return err
	}

	networkAccessType := metalapi.NetworkAccessBaseline
	if controlPlaneConfig.NetworkAccessType != nil {
		networkAccessType = *controlPlaneConfig.NetworkAccessType
	}

	if networkAccessType != metalapi.NetworkAccessBaseline {
		dnsFiles := additionalDNSConfFiles(partition.NetworkIsolation.DNSServers)
		appendOrReplaceFile(new, dnsFiles...)

		ntpFiles := additionalNTPConfFiles(partition.NetworkIsolation.NTPServers)
		appendOrReplaceFile(new, ntpFiles...)

		containerdFiles := additionalContainterdConfigFiles("https://r.metal-stack.dev",
			[]string{
				"docker.lightbitslabs.com",
				"quay.io",
				"eu.gcr.io",
				"ghcr.io",
				"registry.k8s.io",
				"r.metal-stack.io",
			},
		)
		appendOrReplaceFile(new, containerdFiles...)
	}

	return nil
}

func appendOrReplaceFile(new *[]extensionsv1alpha1.File, additionals ...extensionsv1alpha1.File) {
	for _, additional := range additionals {
		var hasReplaced bool
		for i, f := range *new {
			if f.Path == additional.Path {
				(*new)[i] = additional
				hasReplaced = true
				break
			}
		}
		if !hasReplaced {
			*new = append(*new, additional)
		}
	}
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

func additionalContainterdConfigFiles(endpoint string, mirrors []string) []extensionsv1alpha1.File {
	if endpoint == "" || len(mirrors) == 0 {
		return nil
	}
	// TODO: other parties might also want to write to the containerd config.toml.
	// For this case we might want to unmarshal any existing new file to add and patch it with our changes.
	renderedContent := `# Generated by gardener-extension-provider-metal
version = 2

[plugins."io.containerd.grpc.v1.cri".registry]
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
`
	for _, m := range mirrors {
		renderedContent += fmt.Sprintf(`    [plugins."io.containerd.grpc.v1.cri".registry.mirrors.%q]
      endpoint = [%q]
`, m, endpoint)
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
