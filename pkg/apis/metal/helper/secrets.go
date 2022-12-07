package helper

import (
	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"
	secretutils "github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
)

const (
	caNameControlPlane = "ca-" + metal.Name + "-controlplane"
)

func SecretConfigsFunc(namespace string) []extensionssecretsmanager.SecretConfigWithOptions {
	return []extensionssecretsmanager.SecretConfigWithOptions{
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:       caNameControlPlane,
				CommonName: caNameControlPlane,
				CertType:   secretutils.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.CloudControllerManagerServerName,
				CommonName:                  metal.CloudControllerManagerDeploymentName,
				DNSNames:                    kutil.DNSNamesForService(metal.CloudControllerManagerDeploymentName, namespace),
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.DurosControllerDeploymentName,
				CommonName:                  metal.DurosControllerDeploymentName,
				DNSNames:                    kutil.DNSNamesForService(metal.DurosControllerDeploymentName, namespace),
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.FirewallControllerManagerDeploymentName,
				CommonName:                  metal.FirewallControllerManagerDeploymentName,
				DNSNames:                    kutil.DNSNamesForService(metal.FirewallControllerManagerDeploymentName, namespace),
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.AudittailerClientSecretName,
				CommonName:                  "audittailer",
				DNSNames:                    kutil.DNSNamesForService("audittailer", namespace),
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.GroupRolebindingControllerName,
				CommonName:                  "system:group-rolebinding-controller",
				DNSNames:                    kutil.DNSNamesForService(metal.GroupRolebindingControllerName, namespace),
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.AuthNWebhookServerName,
				CommonName:                  metal.AuthNWebhookDeploymentName,
				DNSNames:                    kutil.DNSNamesForService(metal.AuthNWebhookDeploymentName, namespace),
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: false,
			},
			// use current CA for signing server cert to prevent mismatches when dropping the old CA from the webhook
			// config in phase Completing
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane, secretsmanager.UseCurrentCA)},
		},
		{
			Config: &secretutils.CertificateSecretConfig{
				Name:                        metal.AccountingExporterName,
				CommonName:                  "system:accounting-exporter",
				DNSNames:                    kutil.DNSNamesForService(metal.AccountingExporterName, namespace),
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane)},
		},
	}
}

func ShootAccessSecretsFunc(namespace string) []*gutil.ShootAccessSecret {
	return []*gutil.ShootAccessSecret{
		gutil.NewShootAccessSecret(metal.CloudControllerManagerDeploymentName, namespace),
		gutil.NewShootAccessSecret(metal.DurosControllerDeploymentName, namespace),
		gutil.NewShootAccessSecret(metal.AudittailerClientSecretName, namespace),
		gutil.NewShootAccessSecret(metal.GroupRolebindingControllerName, namespace),
		gutil.NewShootAccessSecret(metal.AuthNWebhookDeploymentName, namespace),
		gutil.NewShootAccessSecret(metal.AccountingExporterName, namespace),
		gutil.NewShootAccessSecret(metal.FirewallControllerManagerDeploymentName, namespace),
	}
}

func GetSecretConfigByName(name string, namespace string) *extensionssecretsmanager.SecretConfigWithOptions {
	for _, config := range SecretConfigsFunc(namespace) {
		if name == config.Config.GetName() {
			return &config
		}
	}
	return nil
}
