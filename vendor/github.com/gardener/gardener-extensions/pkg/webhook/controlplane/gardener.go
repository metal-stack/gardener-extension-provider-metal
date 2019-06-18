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

package controlplane

import (
	"context"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetLoadBalancerIngress takes a context, a client, a namespace and a service name. It queries for a load balancer's technical name
// (ip address or hostname). It returns the value of the technical name whereby it always prefers the IP address (if given)
// over the hostname. It also returns the list of all load balancer ingresses.
// TODO This function is copy / pasted from Gardener, remove it once dependency to Gardener is updated
func GetLoadBalancerIngress(ctx context.Context, client client.Client, namespace, name string) (string, error) {
	service := &corev1.Service{}
	if err := client.Get(ctx, kutil.Key(namespace, name), service); err != nil {
		return "", err
	}

	var (
		serviceStatusIngress = service.Status.LoadBalancer.Ingress
		length               = len(serviceStatusIngress)
	)

	switch {
	case length == 0:
		return "", errors.New("`.status.loadBalancer.ingress[]` has no elements yet, i.e. external load balancer has not been created (is your quota limit exceeded/reached?)")
	case serviceStatusIngress[length-1].IP != "":
		return serviceStatusIngress[length-1].IP, nil
	case serviceStatusIngress[length-1].Hostname != "":
		return serviceStatusIngress[length-1].Hostname, nil
	}

	return "", errors.New("`.status.loadBalancer.ingress[]` has an element which does neither contain `.ip` nor `.hostname`")
}
