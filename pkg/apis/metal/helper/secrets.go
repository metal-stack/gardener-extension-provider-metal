package helper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
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

	return getLatestIssuedSecret(secretList.Items)
}

func GetLatestCABundle(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		secretsmanager.LabelKeyManagedBy: secretsmanager.LabelValueSecretsManager,
		secretsmanager.LabelKeyName:      metal.FirewallControllerManagerDeploymentName,
	}); err != nil {
		return nil, err
	}

	return getLatestIssuedSecret(secretList.Items)
}

func GetLatestSecret(ctx context.Context, c client.Client, namespace string, name string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		secretsmanager.LabelKeyManagedBy: secretsmanager.LabelValueSecretsManager,
		secretsmanager.LabelKeyName:      name,
	}); err != nil {
		return nil, err
	}

	return getLatestIssuedSecret(secretList.Items)
}

// getLatestIssuedSecret returns the secret with the "issued-at-time" label that represents the latest point in time
func getLatestIssuedSecret(secrets []corev1.Secret) (*corev1.Secret, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("no secret found")
	}

	var newestSecret *corev1.Secret
	var currentIssuedAtTime time.Time
	for i := 0; i < len(secrets); i++ {
		// if some of the secrets have no "issued-at-time" label
		// we have a problem since this is the source of truth
		issuedAt, ok := secrets[i].Labels[secretsmanager.LabelKeyIssuedAtTime]
		if !ok {
			// there are some old secrets from ancient gardener versions which have to be skipped... (e.g. ssh-keypair.old)
			continue
		}

		issuedAtUnix, err := strconv.ParseInt(issuedAt, 10, 64)
		if err != nil {
			return nil, err
		}

		issuedAtTime := time.Unix(issuedAtUnix, 0).UTC()
		if newestSecret == nil || issuedAtTime.After(currentIssuedAtTime) {
			newestSecret = &secrets[i]
			currentIssuedAtTime = issuedAtTime
		}
	}

	if newestSecret == nil {
		return nil, fmt.Errorf("no secret found")
	}

	return newestSecret, nil
}
