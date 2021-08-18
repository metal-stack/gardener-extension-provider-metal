package controlplane

import (
	"context"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	KonnectivityDeploymentName = "konnectivity-server"
	KonnectivityContainerName  = "konnectivity-server"
)

type mutator struct {
	logger         logr.Logger
	genericMutator extensionswebhook.Mutator
}

// NewMutator creates a new Mutator that mutates resources in the control plane.
func NewMutator(opts AddOptions) extensionswebhook.Mutator {
	return &mutator{
		logger: log.Log.WithName("controlplane-mutator"),
	}
}

func (m *mutator) Mutate(ctx context.Context, new, old client.Object) error {
	acc, err := meta.Accessor(new)
	if err != nil {
		return errors.Wrapf(err, "could not create accessor during webhook")
	}
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if acc.GetDeletionTimestamp() != nil {
		return nil
	}
	switch x := new.(type) {
	case *appsv1.Deployment:
		switch x.Name {
		case KonnectivityDeploymentName:
			// konnektivity support will be dropped in the near future, so this hack / mutator hook should be removed
			extensionswebhook.LogMutation(m.logger, x.Kind, x.Namespace, x.Name)
			return fixKonnektivityHostPort(&x.Spec.Template.Spec, m.logger)
		}
	}

	return nil
}

// fixKonnektivityHostPort fixes a Gardener bug introduced in v1.16 where host port is preventing multiple
// API servers in a seed to be scheduled because host ports can only be taken once
// TODO: Remove when a fix is available from Gardener upstream
func fixKonnektivityHostPort(ps *corev1.PodSpec, log logr.Logger) error {
	var containers []corev1.Container
	for _, c := range ps.Containers {
		if c.Name != KonnectivityContainerName {
			containers = append(containers, c)
			continue
		}

		var ports []corev1.ContainerPort
		for _, p := range c.Ports {
			p := p

			if p.Name == "server" || p.Name == "agent" || p.Name == "admin" || p.Name == "health" {
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

	return nil
}
