package shoot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/util"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	resourcesv1alpha1 "github.com/gardener/gardener/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/downloader"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type mutator struct {
	client  client.Client
	decoder runtime.Decoder
	logger  logr.Logger
}

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator() extensionswebhook.Mutator {
	return &mutator{
		logger: log.Log.WithName("shoot-mutator"),
	}
}

// InjectScheme injects the given scheme into the validator.
func (s *mutator) InjectScheme(scheme *runtime.Scheme) error {
	s.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

// InjectClient injects the given client into the mutator.
func (s *mutator) InjectClient(client client.Client) error {
	s.client = client
	return nil
}

func (m *mutator) Mutate(ctx context.Context, new, _ client.Object) error {
	acc, err := meta.Accessor(new)
	if err != nil {
		return fmt.Errorf("could not create accessor during webhook %w", err)
	}
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if acc.GetDeletionTimestamp() != nil {
		return nil
	}

	switch x := new.(type) {
	case *appsv1.Deployment:
		switch x.Name {
		case "vpn-shoot":
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateVPNShootDeployment(ctx, x)
		}
	case *corev1.Secret:
		// TODO: remove this once gardener-node-agent is in use
		// the purpose of this hack is to enable the cloud-config-downloader to pull the hyperkube image from
		// a registry mirror in case this shoot cluster is configured with networkaccesstype restricted/forbidden
		err = m.mutateCloudConfigDownloaderHyperkubeImage(ctx, x)
		if err != nil {
			// FIXME is this correct
			m.logger.Error(err, "mutation did not work")
		}
	}
	return nil
}

func (m *mutator) mutateVPNShootDeployment(_ context.Context, deployment *appsv1.Deployment) error {
	if c := extensionswebhook.ContainerWithName(deployment.Spec.Template.Spec.Containers, "vpn-shoot"); c != nil {
		// fixes a regression from https://github.com/gardener/gardener/pull/4691
		// raising the timeout to 15 minutes leads to additional 15 minutes of provisioning time because
		// the nodes cidr will only be set on next shoot reconcile
		// with the following mutation we can immediately provide the proper nodes cidr and save time
		logger.Info("ensuring nodes cidr from shoot-node-cidr configmap in vpn-shoot deployment")
		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name:  "NODE_NETWORK",
			Value: "",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "shoot-info-node-cidr",
					},
					Key: "node-cidr",
				},
			},
		})
	}

	return nil
}

const (
	gardenerRegistry = "eu.gcr.io"
	hyperkubeImage   = "/gardener-project/hyperkube"

	// this should be the final destination
	newGardenerRegistry = "europe-docker.pkg.dev"
	newHyperkubeImage   = "/gardener-project/releases/hyperkube"
)

func (m *mutator) mutateCloudConfigDownloaderHyperkubeImage(ctx context.Context, secret *corev1.Secret) error {
	if secret.Labels["gardener.cloud/role"] != "cloud-config" {
		return nil
	}

	shootName, err := extractShootNameFromSecret(secret)
	if err != nil {
		return err
	}

	cluster := &extensionsv1alpha1.Cluster{}
	if err := m.client.Get(ctx, kutil.Key(shootName), cluster); err != nil {
		return err
	}

	shoot, err := extensionscontroller.ShootFromCluster(cluster)
	if err != nil {
		return fmt.Errorf("unable to decode cluster.Spec.Shoot.Raw %w", err)
	}

	if len(shoot.Spec.Provider.Workers) == 0 {
		m.logger.Info("workerless shoot, nothing to do here", "shoot", shootName)
		return nil
	}

	controlPlaneConfig := &metal.ControlPlaneConfig{}
	if shoot.Spec.Provider.ControlPlaneConfig != nil && len(shoot.Spec.Provider.ControlPlaneConfig.Raw) > 0 {
		if err := util.Decode(m.decoder, shoot.Spec.Provider.ControlPlaneConfig.Raw, controlPlaneConfig); err != nil {
			return fmt.Errorf("unable to decode shoot.spec.provider.controlplaneconfig %w", err)
		}
	}

	if controlPlaneConfig.NetworkAccessType == nil || *controlPlaneConfig.NetworkAccessType == metal.NetworkAccessBaseline {
		// this shoot does not have networkaccesstype restricted or forbidden specified, nothing to do here
		return nil
	}

	imageProviderConfig := &metalv1alpha1.ImageProviderConfig{}
	for _, w := range shoot.Spec.Provider.Workers {
		if w.Machine.Image != nil && w.Machine.Image.ProviderConfig != nil && len(w.Machine.Image.ProviderConfig.Raw) > 0 {
			if err := json.Unmarshal(w.Machine.Image.ProviderConfig.Raw, imageProviderConfig); err != nil {
				return fmt.Errorf("unable to decode worker.machine.image.providerconfig %w (%s)", err, string(w.Machine.Image.ProviderConfig.Raw))
			}
			break
		}
	}

	if imageProviderConfig.NetworkIsolation == nil || len(imageProviderConfig.NetworkIsolation.RegistryMirrors) == 0 {
		m.logger.Info("no registry mirrors specified in this shoot, nothing to do here", "shoot", shootName)
		return nil
	}

	var (
		networkIsolation    = imageProviderConfig.NetworkIsolation
		destinationRegistry string
	)

	for _, registry := range networkIsolation.RegistryMirrors {
		if slices.Contains(registry.MirrorOf, gardenerRegistry) {
			parsed, err := url.Parse(registry.Endpoint)
			if err != nil {
				return fmt.Errorf("unable to parse registry endpoint:%w", err)
			}
			destinationRegistry = parsed.Host
			break
		}
	}
	if destinationRegistry == "" {
		err := errors.New("no matching destination registry detected for the hyperkube image")
		m.logger.Error(err, "please check the networkisolation configuration", "shoot", shootName)
		return err
	}

	m.logger.Info("mutate secret", "shoot", shootName, "secret", secret.Name)

	raw, ok := secret.Data[downloader.DataKeyScript]
	if ok {
		script := string(raw)
		newScript := strings.ReplaceAll(script, gardenerRegistry+hyperkubeImage, destinationRegistry+hyperkubeImage)
		newScript = strings.ReplaceAll(newScript, newGardenerRegistry+newHyperkubeImage, destinationRegistry+newHyperkubeImage)
		secret.Data[downloader.DataKeyScript] = []byte(newScript)
		secret.Annotations[downloader.AnnotationKeyChecksum] = utils.ComputeChecksum(newScript)
	}
	return nil
}

func extractShootNameFromSecret(secret *corev1.Secret) (string, error) {
	// resources.gardener.cloud/origin: shoot--test--fra-equ01-8fef639c-bbe4-4c6f-9656-617dc4a4efd8-gardener-soil-test:shoot--pjb9j2--forbidden/shoot-cloud-config-execution
	origin, ok := secret.Annotations[resourcesv1alpha1.OriginAnnotation]
	if !ok {
		return "", fmt.Errorf("no matching annotation found to identify the shoot namespace")
	}

	// does not work
	// shootName, _, err := resourcesv1alpha1helper.SplitOrigin(origin)
	// if err != nil {
	// 	return "", fmt.Errorf("no matching content found in origin annotation to get shoot namespace %w", err)
	// }

	// resources.gardener.cloud/origin: shoot--test--fra-equ01-8fef639c-bbe4-4c6f-9656-617dc4a4efd8-gardener-soil-test:shoot--pjb9j2--forbidden/shoot-cloud-config-execution
	_, firstpart, found := strings.Cut(origin, ":")
	if !found {
		return "", fmt.Errorf("no matching content found in origin annotation to get shoot namespace")
	}
	shootName, _, found := strings.Cut(firstpart, "/")
	if !found {
		return "", fmt.Errorf("no matching content found in origin annotation to get shoot namespace")
	}
	if len(shootName) == 0 {
		return "", fmt.Errorf("could not find shoot name for webhook request")
	}
	return shootName, nil
}
