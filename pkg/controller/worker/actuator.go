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

package worker

import (
	"context"

	"github.com/gardener/gardener-extensions/pkg/util"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	"github.com/gardener/gardener-extensions/pkg/controller/worker/genericactuator"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardener "github.com/gardener/gardener/pkg/client/kubernetes"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

type delegateFactory struct {
	logger logr.Logger

	restConfig *rest.Config

	client  client.Client
	scheme  *runtime.Scheme
	decoder runtime.Decoder

	machineImageMapping []config.MachineImage
}

// NewActuator creates a new Actuator that updates the status of the handled WorkerPoolConfigs.
func NewActuator(machineImages []config.MachineImage) worker.Actuator {
	delegateFactory := &delegateFactory{
		logger:              log.Log.WithName("worker-actuator"),
		machineImageMapping: machineImages,
	}
	return genericactuator.NewActuator(
		log.Log.WithName("metal-worker-actuator"),
		delegateFactory,
		metal.MachineControllerManagerName,
		mcmChart,
		mcmShootChart,
		imagevector.ImageVector(),
		extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
	)
}

func (d *delegateFactory) InjectScheme(scheme *runtime.Scheme) error {
	d.scheme = scheme
	d.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (d *delegateFactory) InjectConfig(restConfig *rest.Config) error {
	d.restConfig = restConfig
	return nil
}

func (d *delegateFactory) InjectClient(client client.Client) error {
	d.client = client
	return nil
}

func (d *delegateFactory) WorkerDelegate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) (genericactuator.WorkerDelegate, error) {
	clientset, err := kubernetes.NewForConfig(d.restConfig)
	if err != nil {
		return nil, err
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	seedChartApplier, err := gardener.NewChartApplierForConfig(d.restConfig)
	if err != nil {
		return nil, err
	}

	return NewWorkerDelegate(
		d.client,
		d.scheme,
		d.decoder,

		d.machineImageMapping,
		seedChartApplier,
		serverVersion.GitVersion,

		worker,
		cluster,
	), nil
}

type workerDelegate struct {
	client  client.Client
	scheme  *runtime.Scheme
	decoder runtime.Decoder

	machineImageMapping []config.MachineImage
	seedChartApplier    gardener.ChartApplier
	serverVersion       string

	cluster *extensionscontroller.Cluster
	worker  *extensionsv1alpha1.Worker

	machineClasses     []map[string]interface{}
	machineDeployments worker.MachineDeployments
	machineImages      []apismetal.MachineImage
}

// NewWorkerDelegate creates a new context for a worker reconciliation.
func NewWorkerDelegate(
	client client.Client,
	scheme *runtime.Scheme,
	decoder runtime.Decoder,

	machineImageMapping []config.MachineImage,
	seedChartApplier gardener.ChartApplier,
	serverVersion string,

	worker *extensionsv1alpha1.Worker,
	cluster *extensionscontroller.Cluster,
) genericactuator.WorkerDelegate {
	return &workerDelegate{
		client:  client,
		scheme:  scheme,
		decoder: decoder,

		machineImageMapping: machineImageMapping,
		seedChartApplier:    seedChartApplier,
		serverVersion:       serverVersion,

		cluster: cluster,
		worker:  worker,
	}
}
