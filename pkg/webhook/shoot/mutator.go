package shoot

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
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
		case "metrics-server":
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateMetricsServerDeployment(ctx, x)
		}
	}
	return nil
}

// TODO: can be removed in Gardener v1.34: https://github.com/gardener/gardener/pull/4884
func (m *mutator) mutateMetricsServerDeployment(_ context.Context, deployment *appsv1.Deployment) error {
	if c := extensionswebhook.ContainerWithName(deployment.Spec.Template.Spec.Containers, "metrics-server"); c != nil {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, "--kubelet-preferred-address-types=", "InternalIP,InternalDNS,ExternalDNS,ExternalIP,Hostname")
	}

	return nil
}
