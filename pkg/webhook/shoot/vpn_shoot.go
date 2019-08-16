// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shoot

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	metalgo "github.com/metal-pod/metal-go"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
)

func (m *mutator) determineNodeNetwork(ctx context.Context, logger logr.Logger) (string, error) {
	providerSecret := &corev1.Secret{}
	//if err := m.client.Get(ctx, kutil.Key(infrastructure.Spec.SecretRef.Namespace, infrastructure.Spec.SecretRef.Name), providerSecret); err != nil {
	if err := m.client.Get(ctx, kutil.Key("secretNamespace", "secretkey"), providerSecret); err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(providerSecret.Data[metal.APIKey]))
	hmac := strings.TrimSpace(string(providerSecret.Data[metal.APIHMac]))

	u, ok := providerSecret.Data[metal.APIURL]
	if !ok {
		return "", fmt.Errorf("missing %s in secret", metal.APIURL)
	}
	url := strings.TrimSpace(string(u))

	svc, err := metalgo.NewDriver(url, token, hmac)
	if err != nil {
		return "", err
	}

	p := "test"
	primary := true
	fr := &metalgo.NetworkFindRequest{
		ProjectID: &p,
		Primary:   &primary,
	}
	resp, err := svc.NetworkFind(fr)
	if err != nil {
		return "", fmt.Errorf("could not find primary network project: %s, error: %s", p, err)
	}

	nets := resp.Networks
	if len(nets) == 0 {
		logger.Info("could not find primary network for project network list is empty", p)
		return "", nil
	}

	if len(nets) > 1 {
		return "", fmt.Errorf("primary network for project %s is ambiguous", p)
	}

	pn := nets[0]
	if len(pn.Prefixes) != 1 {
		return "", fmt.Errorf("multiple prefixes for the primary network are not supported project: %s", p)
	}

	return pn.Prefixes[0], nil
}

func (m *mutator) mutateVPNShootDeployment(ctx context.Context, logger logr.Logger, deployment *appsv1.Deployment) error {
	template := &deployment.Spec.Template
	ps := &template.Spec

	if c := extensionswebhook.ContainerWithName(ps.Containers, "vpn-shoot"); c != nil {
		net, err := m.determineNodeNetwork(ctx, logger)
		if err != nil {
			return err
		}
		nodeNetwork := corev1.EnvVar{
			Name:  "NODE_NETWORK",
			Value: net,
		}
		extensionswebhook.EnsureEnvVarWithName(c.Env, nodeNetwork)
	}

	return nil
}
