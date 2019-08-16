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

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GetNetworkPolicyMeta returns the network policy object with filled meta data.
func GetNetworkPolicyMeta(namespace, providerName string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{ObjectMeta: kutil.ObjectMeta(namespace, "gardener-extension-"+providerName)}
}

// EnsureNetworkPolicy ensures that the required network policy that allows the kube-apiserver running in the given namespace
// to talk to the extension webhook is installed.
func EnsureNetworkPolicy(ctx context.Context, c client.Client, namespace, providerName string, port int) error {
	networkPolicy := GetNetworkPolicyMeta(namespace, providerName)

	policyPort := intstr.FromInt(port)
	policyProtocol := corev1.ProtocolTCP

	_, err := controllerutil.CreateOrUpdate(ctx, c, networkPolicy, func() error {
		networkPolicy.Spec = networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &policyPort,
							Protocol: &policyProtocol,
						},
					},
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									gardencorev1alpha1.LabelControllerRegistrationName: providerName,
									gardencorev1alpha1.GardenRole:                      gardencorev1alpha1.GardenRoleExtension,
								},
							},
							PodSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"app.kubernetes.io/name": "gardener-extension-" + providerName,
								},
							},
						},
					},
				},
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					gardencorev1alpha1.LabelApp:  gardencorev1alpha1.LabelKubernetes,
					gardencorev1alpha1.LabelRole: gardencorev1alpha1.LabelAPIServer,
				},
			},
		}
		return nil
	})
	return err
}
