package controlplane

import (
	"context"

	"github.com/coreos/go-systemd/v22/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"

	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"
	"github.com/go-logr/logr"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
	cluster, _ := gctx.GetCluster(ctx)
	makeAuditForwarder := false
	auditToSplunk := false

	if e.controllerConfig.ClusterAudit.Enabled {
		cpConfig, err := helper.ControlPlaneConfigFromClusterShootSpec(cluster)
		if err != nil {
			logger.Error(err, "Could not read ControlPlaneConfig from cluster shoot spec", "Cluster name", cluster.ObjectMeta.Name)
			return err
		}
		if cpConfig.FeatureGates.ClusterAudit == nil || *cpConfig.FeatureGates.ClusterAudit {
			makeAuditForwarder = true
			if cpConfig.FeatureGates.AuditToSplunk == nil || *cpConfig.FeatureGates.AuditToSplunk {
				auditToSplunk = true
			}
		}
	}

	template := &new.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c, makeAuditForwarder, e.controllerConfig)
		ensureVolumeMounts(c, makeAuditForwarder, e.controllerConfig)
		ensureVolumes(ps, makeAuditForwarder, auditToSplunk, e.controllerConfig)
		fixKonnektivityHostPort(ps, e.logger)
	}
	if makeAuditForwarder {
		err := ensureAuditForwarder(ps, auditToSplunk, e.controllerConfig)
		if err != nil {
			logger.Error(err, "Could not ensure the audit forwarder", "Cluster name", cluster.ObjectMeta.Name)
			return err
		}
	}

	return e.ensureChecksumAnnotations(ctx, &new.Spec.Template, new.Namespace)
}

var (
	// config mount for authn-webhook-config that is specified at kube-apiserver commandline
	authnWebhookConfigVolumeMount = corev1.VolumeMount{
		Name:      metal.AuthNWebHookConfigName,
		MountPath: "/etc/webhook/config",
		ReadOnly:  true,
	}
	authnWebhookConfigVolume = corev1.Volume{
		Name: metal.AuthNWebHookConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: metal.AuthNWebHookConfigName},
			},
		},
	}
	// cert mount "kube-jwt-authn-webhook-server" that is referenced from the authn-webhook-config
	authnWebhookCertVolumeMount = corev1.VolumeMount{
		Name:      metal.AuthNWebHookCertName,
		MountPath: "/etc/webhook/certs",
		ReadOnly:  true,
	}
	authnWebhookCertVolume = corev1.Volume{
		Name: metal.AuthNWebHookCertName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: "kube-jwt-authn-webhook-server",
			},
		},
	}
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
	audittailerClientSecretVolume = corev1.Volume{
		Name: metal.AudittailerClientSecretName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: metal.AudittailerClientSecretName,
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
	konnectivityUdsVolumeMount = corev1.VolumeMount{
		Name:      "konnectivity-uds",
		MountPath: "/konnectivity-uds",
		ReadOnly:  false,
	}
	konnectivityEnvVar = corev1.EnvVar{
		Name:  "AUDIT_KONNECTIVITY_UDS_SOCKET",
		Value: "/konnectivity-uds/konnectivity-server.socket",
	}
	auditForwarderSidecar = corev1.Container{
		Name: "auditforwarder",
		// Image:   // is added from the image vector in the ensure function
		ImagePullPolicy: "Always",
		Env: []corev1.EnvVar{
			{
				Name:  "AUDIT_KUBECFG",
				Value: "/shootconfig/kubeconfig",
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
				Value: "audittailer-client.crt",
			},
			{
				Name:  "AUDIT_TLS_KEY_FILE",
				Value: "audittailer-client.key",
			},
			{
				Name:  "AUDIT_TLS_VHOST",
				Value: "audittailer",
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      audittailerClientSecretVolume.Name,
				ReadOnly:  true,
				MountPath: "/shootconfig",
			},
			auditLogVolumeMount,
		},
	}
)

func ensureVolumeMounts(c *corev1.Container, makeAuditForwarder bool, controllerConfig config.ControllerConfiguration) {
	if controllerConfig.Auth.Enabled {
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, authnWebhookConfigVolumeMount)
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, authnWebhookCertVolumeMount)
	}
	if makeAuditForwarder {
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, auditPolicyVolumeMount)
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, auditLogVolumeMount)
	}
}

func ensureVolumes(ps *corev1.PodSpec, makeAuditForwarder, auditToSplunk bool, controllerConfig config.ControllerConfiguration) {
	if controllerConfig.Auth.Enabled {
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, authnWebhookConfigVolume)
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, authnWebhookCertVolume)
	}
	if makeAuditForwarder {
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditPolicyVolume)
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditLogVolume)
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, audittailerClientSecretVolume)
	}
	if auditToSplunk {
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditForwarderSplunkConfigVolume)
		ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, auditForwarderSplunkSecretVolume)
	}
}

// fixKonnektivityHostPort fixes a Gardener bug introduced in v1.16 where host port is preventing multiple
// API servers in a seed to be scheduled because host ports can only be taken once
// TODO: Remove when a fix is available from Gardener upstream
func fixKonnektivityHostPort(ps *corev1.PodSpec, log logr.Logger) {
	var containers []corev1.Container
	for _, c := range ps.Containers {
		if c.Name != "konnectivity-server" {
			containers = append(containers, c)
			continue
		}

		var ports []corev1.ContainerPort
		for _, p := range c.Ports {
			p := p

			if p.Name == "adminport" || p.Name == "healthport" {
				p = corev1.ContainerPort{
					Name:          p.Name,
					Protocol:      p.Protocol,
					ContainerPort: p.ContainerPort,
				}
			}

			ports = append(ports, p)
		}

		c.Ports = ports
		c.LivenessProbe.HTTPGet.Host = ""

		containers = append(containers, c)
	}

	ps.Containers = containers
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container, makeAuditForwarder bool, controllerConfig config.ControllerConfiguration) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")

	if controllerConfig.Auth.Enabled {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--authentication-token-webhook-config-file=", "/etc/webhook/config/authn-webhook-config.json")
	}

	if makeAuditForwarder {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-policy-file=", "/etc/kubernetes/audit-override/audit-policy.yaml")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-log-path=", "/auditlog/audit.log")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-log-maxsize=", "100")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-log-maxbackup=", "1")
	}

}

func ensureAuditForwarder(ps *corev1.PodSpec, auditToSplunk bool, controllerConfig config.ControllerConfiguration) error {
	auditForwarderImage, err := imagevector.ImageVector().FindImage("auditforwarder")
	if err != nil {
		logger.Error(err, "Could not find auditforwarder image in imagevector")
		return err
	}
	auditForwarderSidecar.Image = auditForwarderImage.String()

	udsVolumeFound := false
	for _, volume := range ps.Volumes {
		if volume.Name == "konnectivity-uds" {
			udsVolumeFound = true
			break
		}
	}
	if udsVolumeFound {
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(auditForwarderSidecar.VolumeMounts, konnectivityUdsVolumeMount)
		auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, konnectivityEnvVar)
	} else {
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureNoVolumeMountWithName(auditForwarderSidecar.VolumeMounts, konnectivityUdsVolumeMount.Name)
		auditForwarderSidecar.Env = extensionswebhook.EnsureNoEnvVarWithName(auditForwarderSidecar.Env, konnectivityEnvVar.Name)
	}

	if auditToSplunk {
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(auditForwarderSidecar.VolumeMounts, auditForwarderSplunkConfigVolumeMount)
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(auditForwarderSidecar.VolumeMounts, auditForwarderSplunkSecretVolumeMount)
		auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, auditForwarderSplunkPodNameEnvVar)
		auditForwarderSidecar.Env = extensionswebhook.EnsureEnvVarWithName(auditForwarderSidecar.Env, auditForwarderSplunkHECTokenEnvVar)
	} else {
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureNoVolumeMountWithName(auditForwarderSidecar.VolumeMounts, auditForwarderSplunkConfigVolumeMount.Name)
		auditForwarderSidecar.VolumeMounts = extensionswebhook.EnsureNoVolumeMountWithName(auditForwarderSidecar.VolumeMounts, auditForwarderSplunkSecretVolumeMount.Name)
		auditForwarderSidecar.Env = extensionswebhook.EnsureNoEnvVarWithName(auditForwarderSidecar.Env, auditForwarderSplunkPodNameEnvVar.Name)
		auditForwarderSidecar.Env = extensionswebhook.EnsureNoEnvVarWithName(auditForwarderSidecar.Env, auditForwarderSplunkHECTokenEnvVar.Name)
	}

	ps.Containers = extensionswebhook.EnsureContainerWithName(ps.Containers, auditForwarderSidecar)
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
	return controlplane.EnsureSecretChecksumAnnotation(ctx, template, e.client, namespace, v1alpha1constants.SecretNameCloudProvider)
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *ensurer) EnsureKubeletServiceUnitOptions(ctx context.Context, gctx gcontext.GardenContext, new, _ []*unit.UnitOption) ([]*unit.UnitOption, error) {
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
func (e *ensurer) EnsureKubeletConfiguration(ctx context.Context, gctx gcontext.GardenContext, new, _ *kubeletconfigv1beta1.KubeletConfiguration) error {
	// Make sure CSI-related feature gates are not enabled
	// TODO Leaving these enabled shouldn't do any harm, perhaps remove this code when properly tested?
	delete(new.FeatureGates, "VolumeSnapshotDataSource")
	delete(new.FeatureGates, "CSINodeInfo")
	delete(new.FeatureGates, "CSIDriverRegistry")
	return nil
}
