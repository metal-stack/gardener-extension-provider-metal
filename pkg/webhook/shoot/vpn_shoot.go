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

	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func (m *mutator) mutateVPNShootDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	template := &deployment.Spec.Template
	ps := &template.Spec

	if c := extensionswebhook.ContainerWithName(ps.Containers, "vpn-shoot"); c != nil {
		nodeNetwork := corev1.EnvVar{
			Name:  "NODE_NETWORK",
			Value: "10.3.20.0/22",
		}
		extensionswebhook.EnsureEnvVarWithName(c.Env, nodeNetwork)
	}

	return nil
}
