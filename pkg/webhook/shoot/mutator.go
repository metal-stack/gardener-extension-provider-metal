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
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mutator struct {
	client client.Client
	logger logr.Logger
}

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator(logger logr.Logger) extensionswebhook.Mutator {
	return &mutator{
		logger: logger,
	}
}

func (m *mutator) Mutate(ctx context.Context, obj runtime.Object) error {
	acc, err := meta.Accessor(obj)
	if err != nil {
		return errors.Wrapf(err, "could not create accessor during webhook")
	}
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if acc.GetDeletionTimestamp() != nil {
		return nil
	}

	switch x := obj.(type) {
	case *appsv1.Deployment:
		switch x.Name {
		}
	}
	return nil
}
