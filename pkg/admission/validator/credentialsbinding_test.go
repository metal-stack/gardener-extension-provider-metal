package validator_test

import (
	"context"
	"errors"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/security"
	"github.com/gardener/gardener/pkg/utils/test"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/admission/validator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("CredentialsBinding validator", func() {
	Describe("#Validate", func() {
		const (
			namespace = "garden-dev"
			name      = "my-provider-account"
		)

		var (
			credentialsBindingValidator extensionswebhook.Validator

			mgr *test.FakeManager

			ctx                      = context.TODO()
			credentialsBindingSecret *security.CredentialsBinding

			scheme *runtime.Scheme
		)

		newValidator := func(objs ...client.Object) {
			builder := fakeclient.NewClientBuilder().WithScheme(scheme)
			for _, obj := range objs {
				builder = builder.WithObjects(obj)
			}
			apiReader := builder.Build()
			mgr = &test.FakeManager{APIReader: apiReader}
			credentialsBindingValidator = validator.NewCredentialsBindingValidator(
				mgr,
			)
		}

		BeforeEach(func() {
			scheme = runtime.NewScheme()
			Expect(corev1.AddToScheme(scheme)).To(Succeed())
			Expect(gardencorev1beta1.AddToScheme(scheme)).To(Succeed())

			newValidator()

			credentialsBindingSecret = &security.CredentialsBinding{
				CredentialsRef: corev1.ObjectReference{
					Name:       name,
					Namespace:  namespace,
					Kind:       "Secret",
					APIVersion: "v1",
				},
			}
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

		// It("should return err if it fails to get the corresponding Secret", func() {
		// 	apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).Return(fakeErr)

		// 	err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
		// 	Expect(err).To(MatchError(fakeErr))
		// })

		// It("should return err when the corresponding Secret does not contain a 'serviceaccount.json' field", func() {
		// 	apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).
		// 		DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
		// 			secret := &corev1.Secret{Data: map[string][]byte{
		// 				"foo": []byte("bar"),
		// 			}}
		// 			*obj = *secret
		// 			return nil
		// 		})

		// 	err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
		// 	Expect(err).To(MatchError("referenced secret garden-dev/my-provider-account is not valid: either hmac or api key must be set"))
		// })

		// It("should return err when the corresponding Secret does not contain a valid 'metalAPIHMac' field", func() {
		// 	apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).
		// 		DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
		// 			secret := &corev1.Secret{Data: map[string][]byte{
		// 				metal.APIHMac: []byte(``),
		// 			}}
		// 			*obj = *secret
		// 			return nil
		// 		})

		// 	err := credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)
		// 	Expect(err).To(MatchError("referenced secret garden-dev/my-provider-account is not valid: either hmac or api key must be set"))
		// })

		// It("should succeed when the corresponding Secret is valid", func() {
		// 	apiReader.EXPECT().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, gomock.AssignableToTypeOf(&corev1.Secret{})).
		// 		DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *corev1.Secret, _ ...client.GetOption) error {
		// 			secret := &corev1.Secret{Data: map[string][]byte{
		// 				metal.APIHMac: []byte(`a-secure-secret`),
		// 			}}
		// 			*obj = *secret
		// 			return nil
		// 		})

		// 	Expect(credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, nil)).To(Succeed())
		// })

		It("should return nil when the CredentialsBinding did not change", func() {
			old := credentialsBindingSecret.DeepCopy()

			Expect(credentialsBindingValidator.Validate(ctx, credentialsBindingSecret, old)).To(Succeed())
		})
	})
})
