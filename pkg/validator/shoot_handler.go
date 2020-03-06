package validator

import (
	"context"
	"net/http"

	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/go-logr/logr"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Shoot validates shoots
type Shoot struct {
	client  client.Client
	decoder runtime.Decoder
	Logger  logr.Logger
}

// Handle implements Handler.Handle
func (v *Shoot) Handle(ctx context.Context, req admission.Request) admission.Response {
	shoot := &core.Shoot{}
	if err := util.Decode(v.decoder, req.Object.Raw, shoot); err != nil {
		v.Logger.Error(err, "failed to decode shoot", string(req.Object.Raw))
		return admission.Errored(http.StatusBadRequest, err)
	}

	if shoot.Spec.Provider.Type != metal.Type {
		return admission.Allowed("webhook not responsible for this provider")
	}

	switch req.Operation {
	case admissionv1beta1.Create:
		if err := v.validateShoot(ctx, shoot); err != nil {
			v.Logger.Error(err, "denied request")
			return admission.Errored(http.StatusBadRequest, err)
		}
	case admissionv1beta1.Update:
		oldShoot := &core.Shoot{}
		if err := util.Decode(v.decoder, req.OldObject.Raw, oldShoot); err != nil {
			v.Logger.Error(err, "failed to decode old shoot", string(req.OldObject.Raw))
			return admission.Errored(http.StatusBadRequest, err)
		}

		if err := v.validateShootUpdate(ctx, oldShoot, shoot); err != nil {
			v.Logger.Error(err, "denied request")
			return admission.Errored(http.StatusBadRequest, err)
		}
	default:
		v.Logger.Info("Webhook not responsible", "Operation", req.Operation)
	}

	return admission.Allowed("validations succeeded")
}

// InjectClient injects the client.
func (v *Shoot) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectScheme injects the scheme.
func (v *Shoot) InjectScheme(s *runtime.Scheme) error {
	v.decoder = serializer.NewCodecFactory(s).UniversalDecoder()
	return nil
}
