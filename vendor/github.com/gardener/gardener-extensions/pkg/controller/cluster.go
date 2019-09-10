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

package controller

import (
	"context"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Cluster contains the decoded resources of Gardener's extension Cluster resource.
// TODO: Change from `gardenv1beta1` to `gardencorev1alpha1` once we have moved the resources there.
type Cluster struct {
	CloudProfile *gardenv1beta1.CloudProfile
	Seed         *gardenv1beta1.Seed
	Shoot        *gardenv1beta1.Shoot
}

// GetCluster tries to read Gardener's Cluster extension resource in the given namespace.
func GetCluster(ctx context.Context, c client.Client, namespace string) (*Cluster, error) {
	cluster := &extensionsv1alpha1.Cluster{}
	if err := c.Get(ctx, kutil.Key(namespace), cluster); err != nil {
		return nil, err
	}

	cloudProfile, err := CloudProfileFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	seed, err := SeedFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	shoot, err := ShootFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	return &Cluster{cloudProfile, seed, shoot}, nil
}

// CloudProfileFromCluster returns the CloudProfile resource inside the Cluster resource.
func CloudProfileFromCluster(cluster *extensionsv1alpha1.Cluster) (*gardenv1beta1.CloudProfile, error) {
	decoder, err := newGardenDecoder()
	if err != nil {
		return nil, err
	}

	cloudProfile := &gardenv1beta1.CloudProfile{}
	_, _, err = decoder.Decode(cluster.Spec.CloudProfile.Raw, nil, cloudProfile)
	return cloudProfile, err
}

// SeedFromCluster returns the Seed resource inside the Cluster resource.
func SeedFromCluster(cluster *extensionsv1alpha1.Cluster) (*gardenv1beta1.Seed, error) {
	decoder, err := newGardenDecoder()
	if err != nil {
		return nil, err
	}

	seed := &gardenv1beta1.Seed{}
	_, _, err = decoder.Decode(cluster.Spec.Seed.Raw, nil, seed)
	return seed, err
}

// ShootFromCluster returns the Shoot resource inside the Cluster resource.
func ShootFromCluster(cluster *extensionsv1alpha1.Cluster) (*gardenv1beta1.Shoot, error) {
	decoder, err := newGardenDecoder()
	if err != nil {
		return nil, err
	}

	shoot := &gardenv1beta1.Shoot{}
	_, _, err = decoder.Decode(cluster.Spec.Shoot.Raw, nil, shoot)
	return shoot, err
}

func newGardenDecoder() (runtime.Decoder, error) {
	scheme := runtime.NewScheme()
	decoder := serializer.NewCodecFactory(scheme).UniversalDecoder()
	return decoder, gardenv1beta1.AddToScheme(scheme)
}
