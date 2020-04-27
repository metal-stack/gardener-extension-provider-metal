package controlplaneexposure

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the metal exposure webhook to the manager.
type AddOptions struct {
	// ETCDStorage is the etcd storage configuration.
	ETCDStorage config.ETCDStorage
}

var logger = log.Log.WithName("metal-controlplaneexposure-webhook")

// AddToManagerWithOptions creates a webhook with the given options and adds it to the manager.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) (*extensionswebhook.Webhook, error) {
	logger.Info("Adding webhook to manager")
	return controlplane.Add(mgr, controlplane.AddArgs{
		Kind:     controlplane.KindSeed,
		Provider: metal.Type,
		Types:    []runtime.Object{&corev1.Service{}, &appsv1.Deployment{}, &appsv1.StatefulSet{}},
		Mutator:  genericmutator.NewMutator(NewEnsurer(&opts.ETCDStorage, logger), nil, nil, nil, logger),
	})
}

// AddToManager creates a webhook with the default options and adds it to the manager.
func AddToManager(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
