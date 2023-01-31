package helper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
)

func GetLatestSSHSecret(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		// TODO: migrate to secretsmanager constants on g/g v1.45
		secretsmanager.LabelKeyManagedBy:       secretsmanager.LabelValueSecretsManager,
		secretsmanager.LabelKeyManagerIdentity: "gardenlet",
		secretsmanager.LabelKeyName:            "ssh-keypair",
	}); err != nil {
		return nil, err
	}

	return getLatestIssuedSecret(secretList.Items)
}

func GetLatestCABundle(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		secretsmanager.LabelKeyManagedBy:       secretsmanager.LabelValueSecretsManager,
		secretsmanager.LabelKeyManagerIdentity: "gardenlet",
		secretsmanager.LabelKeyName:            "ca-bundle",
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
			return nil, fmt.Errorf("secret with no issues-at-time label: %s", secrets[i].Name)
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

	return newestSecret, nil
}
