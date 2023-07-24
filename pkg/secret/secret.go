// Package secret is extracted to be reused by other metal components which require to gather the latest secret.
// which was created by the gardener secretsmanager.
// We dont want to include the whole gardener dependency for this sole purpose.
package secret

import (
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const labelKeyIssuedAtTime = "issued-at-time"

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
		issuedAt, ok := secrets[i].Labels[labelKeyIssuedAtTime]
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
