package shoot

import (
	"context"
	"fmt"
	"strings"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type mutator struct {
	logger logr.Logger
}

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator() extensionswebhook.Mutator {
	return &mutator{
		logger: log.Log.WithName("shoot-mutator"),
	}
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
		// FIXME only for isolated clusters
		if x.Labels["gardener.cloud/role"] == "cloud-config" {
			raw, ok := x.Data["script"]
			if ok {
				rawScript, err := utils.DecodeBase64(string(raw))
				if err != nil {
					return fmt.Errorf("unable to decode script %w", err)
				}
				script := string(rawScript)
				// FIXME use registry from networkisolation.RegistryMirrors
				newScript := strings.ReplaceAll(script, "eu.gcr.io/gardener-project/hyperkube", "r.metal-stack.dev/gardener-project/hyperkube")
				x.StringData["script"] = newScript
				x.Annotations["checksum/data-script"] = utils.ComputeChecksum(newScript)
			}
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
