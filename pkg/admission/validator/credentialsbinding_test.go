package validator_test

import (
	"context"
	"errors"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/security"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	mockmanager "github.com/gardener/gardener/third_party/mock/controller-runtime/manager"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/admission/validator"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("CredentialsBinding validator", func() {
	Describe("#Validate", func() {
		const (
			namespace = "garden-dev"
			name      = "my-provider-account"
		)

		var (
			credentialsBindingValidator extensionswebhook.Validator

			ctrl      *gomock.Controller
			mgr       *mockmanager.MockManager
			apiReader *mockclient.MockReader

			ctx                      = context.TODO()
			credentialsBindingSecret *security.CredentialsBinding

			fakeErr = fmt.Errorf("fake err")
		)

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())

			mgr = mockmanager.NewMockManager(ctrl)

			apiReader = mockclient.NewMockReader(ctrl)
			mgr.EXPECT().GetAPIReader().Return(apiReader)

			credentialsBindingValidator = validator.NewCredentialsBindingValidator(
				mgr,
			)

			credentialsBindingSecret = &security.CredentialsBinding{
				CredentialsRef: corev1.ObjectReference{
					Name:       name,
					Namespace:  namespace,
					Kind:       "Secret",
					APIVersion: "v1",
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should return err when obj is not a CredentialsBinding", func() {
			err := credentialsBindingValidator.Validate(ctx, &corev1.Secret{}, nil)
			Expect(err).To(MatchError("wrong object type *v1.Secret"))
		})

		It("should return err when oldObj is not a CredentialsBinding", func() {
			err := credentialsBindingValidator.Validate(ctx, &security.CredentialsBinding{}, &corev1.Secret{})
			Expect(err).To(MatchError("wrong object type *v1.Secret for old object"))
		})

		It("should return err if the CredentialsBinding references unknown credentials type", func() {
			credentialsBindingSecret.CredentialsRef.APIVersion = "unknown"
			err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
			Expect(err).To(MatchError(errors.New(`unsupported credentials reference: version "unknown", kind "Secret"`)))
		})

		It("should return err if it fails to get the corresponding Secret", func() {
			apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).Return(fakeErr)

			err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
			Expect(err).To(MatchError(fakeErr))
		})

		It("should return err when the corresponding Secret does not contain a 'serviceaccount.json' field", func() {
			apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					secret := &corev1.Secret{Data: map[string][]byte{
						"foo": []byte("bar"),
					}}
					*obj = *secret
					return nil
				})

			err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
			Expect(err).To(MatchError("referenced secret garden-dev/my-provider-account is not valid: either hmac or api key must be set"))
		})

		It("should return err when the corresponding Secret does not contain a valid 'metalAPIHMac' field", func() {
			apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					secret := &corev1.Secret{Data: map[string][]byte{
						metal.APIHMac: []byte(``),
					}}
					*obj = *secret
					return nil
				})

			err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
			Expect(err).To(MatchError("referenced secret garden-dev/my-provider-account is not valid: either hmac or api key must be set"))
		})

		It("should succeed when the corresponding Secret is valid", func() {
			apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
					secret := &corev1.Secret{Data: map[string][]byte{
						metal.APIHMac: []byte(`a-secure-secret`),
					}}
					*obj = *secret
					return nil
				})

			Expect(credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)).To(Succeed())
		})

		It("should return nil when the CredentialsBinding did not change", func() {
			old := credentialsBindingSecret.DeepCopy()

			Expect(credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, old)).To(Succeed())
		})
	})
})
