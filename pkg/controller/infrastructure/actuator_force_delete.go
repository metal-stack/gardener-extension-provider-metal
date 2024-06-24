package infrastructure

import (
	"context"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
)

func (a *actuator) ForceDelete(context.Context, logr.Logger, *extensionsv1alpha1.Infrastructure, *extensionscontroller.Cluster) error {
	// TODO: implement
	return nil
}
