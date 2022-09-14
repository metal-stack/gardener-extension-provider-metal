package shoot

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type mutator struct {
	seedClient client.Client
	logger     logr.Logger
}

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator() extensionswebhook.Mutator {
	return &mutator{
		logger: log.Log.WithName("shoot-mutator"),
	}
}

// InjectSeedClient injects the seed client
func (m *mutator) InjectSeedClient(seedClient client.Client) {
	m.seedClient = seedClient
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

	gctx := gcontext.NewGardenContext(m.seedClient, new)

	switch x := new.(type) {
	case *appsv1.Deployment:
		switch x.Name {
		case "vpn-shoot":
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateVPNShootDeployment(ctx, gctx, x)
		}
	}
	return nil
}

func (m *mutator) mutateVPNShootDeployment(ctx context.Context, gctx gcontext.GardenContext, deployment *appsv1.Deployment) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := m.seedClient.Get(ctx, kutil.Key(cluster.ObjectMeta.Name, cluster.Shoot.Name), infrastructure); err != nil {
		logger.Error(err, "could not read Infrastructure for cluster", "cluster name", cluster.ObjectMeta.Name)
		return err
	}

	if infrastructure == nil || infrastructure.Status.NodesCIDR == nil {
		return fmt.Errorf("nodeCIDR was not yet set by infrastructure controller")
	}

	nodeCIDR := *infrastructure.Status.NodesCIDR

	if c := extensionswebhook.ContainerWithName(deployment.Spec.Template.Spec.Containers, "vpn-shoot"); c != nil {
		// fixes a regression from https://github.com/gardener/gardener/pull/4691
		// raising the timeout to 15 minutes leads to additional 15 minutes of provisioning time because
		// the nodes cidr will only be set on next shoot reconcile
		// with the following mutation we can immediately provide the proper nodes cidr and save time
		logger.Info("ensuring nodes cidr in vpn-shoot deployment", "cidr", nodeCIDR)
		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name:  "NODE_NETWORK",
			Value: nodeCIDR,
		})
	}

	return nil
}
