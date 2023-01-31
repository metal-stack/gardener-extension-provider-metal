package helper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetLatestSSHSecret(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		// TODO: migrate to secretsmanager constants on g/g v1.45
		"managed-by":       "secrets-manager",
		"manager-identity": "gardenlet",
		"name":             "ssh-keypair",
	}); err != nil {
		return nil, err
	}

	return GetLatestIssuedSecret(secretList.Items)
}

func GetLatestCABundle(ctx context.Context, c client.Client, namespace string) (*corev1.Secret, error) {
	secretList := &corev1.SecretList{}
	if err := c.List(ctx, secretList, client.InNamespace(namespace), client.MatchingLabels{
		// TODO: migrate to secretsmanager constants on g/g v1.45
		"managed-by":       "secrets-manager",
		"manager-identity": "gardenlet",
		"name":             "ca-bundle",
	}); err != nil {
		return nil, err
	}

	return GetLatestIssuedSecret(secretList.Items)
}

// getLatestIssuedSecret returns the secret with the "issued-at-time" label that represents the latest point in time
func GetLatestIssuedSecret(secrets []corev1.Secret) (*corev1.Secret, error) {
	if len(secrets) == 0 {
		return nil, fmt.Errorf("no secret found")
	}

	var newestSecret *corev1.Secret
	var currentIssuedAtTime time.Time
	for i := 0; i < len(secrets); i++ {
		// if some of the secrets have no "issued-at-time" label
		// we have a problem since this is the source of truth
		issuedAt, ok := secrets[i].Labels["issued-at-time"]
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
