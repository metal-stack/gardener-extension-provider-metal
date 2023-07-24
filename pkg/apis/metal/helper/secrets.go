package helper

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/secret"
)

func GetLatestSSHSecret(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		secretsmanager.LabelKeyManagedBy:       secretsmanager.LabelValueSecretsManager,
		secretsmanager.LabelKeyManagerIdentity: constants.SecretManagerIdentityGardenlet,
		secretsmanager.LabelKeyName:            constants.SecretNameSSHKeyPair,
	}); err != nil {
		return nil, err
	}

	return secret.GetLatestIssuedSecret(secretList.Items)
}

func GetLatestSecret(ctx context.Context, c client.Client, namespace string, name string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		secretsmanager.LabelKeyManagedBy:       secretsmanager.LabelValueSecretsManager,
		secretsmanager.LabelKeyManagerIdentity: metal.ManagerIdentity,
		secretsmanager.LabelKeyName:            name,
	}); err != nil {
		return nil, err
	}

	return secret.GetLatestIssuedSecret(secretList.Items)
}
