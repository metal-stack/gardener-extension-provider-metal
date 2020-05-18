package controlplane

import (
	"context"

	"github.com/coreos/go-systemd/unit"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
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
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, ectx genericmutator.EnsurerContext, new, _ *appsv1.Deployment) error {
	template := &new.Spec.Template
	ps := &template.Spec
	if c := extensionswebhook.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c, e.controllerConfig)
		ensureVolumeMounts(c)
		ensureVolumes(ps)
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
)

func ensureVolumeMounts(c *corev1.Container) {
	c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, authnWebhookConfigVolumeMount)
	c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, authnWebhookCertVolumeMount)
}

func ensureVolumes(ps *corev1.PodSpec) {
	ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, authnWebhookConfigVolume)
	ps.Volumes = extensionswebhook.EnsureVolumeWithName(ps.Volumes, authnWebhookCertVolume)
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container, controllerConfig config.ControllerConfiguration) {
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")

	// activate AuthN Webhook with mounted Webhook-Config
	c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--authentication-token-webhook-config-file=", "/etc/webhook/config/authn-webhook-config.json")
	// activate dynamic auditing if enabled
	if controllerConfig.SplunkAudit.Enabled {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--audit-dynamic-configuration=", "true")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--feature-gates=", "DynamicAuditing=true")
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--runtime-config=", "auditregistration.k8s.io/v1alpha1=true")
	}
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeControllerManagerDeployment(ctx context.Context, ectx genericmutator.EnsurerContext, new, _ *appsv1.Deployment) error {
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
func (e *ensurer) EnsureKubeletServiceUnitOptions(ctx context.Context, ectx genericmutator.EnsurerContext, new, _ []*unit.UnitOption) ([]*unit.UnitOption, error) {
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
func (e *ensurer) EnsureKubeletConfiguration(ctx context.Context, ectx genericmutator.EnsurerContext, new, _ *kubeletconfigv1beta1.KubeletConfiguration) error {
	// Make sure CSI-related feature gates are not enabled
	// TODO Leaving these enabled shouldn't do any harm, perhaps remove this code when properly tested?
	delete(new.FeatureGates, "VolumeSnapshotDataSource")
	delete(new.FeatureGates, "CSINodeInfo")
	delete(new.FeatureGates, "CSIDriverRegistry")
	return nil
}
