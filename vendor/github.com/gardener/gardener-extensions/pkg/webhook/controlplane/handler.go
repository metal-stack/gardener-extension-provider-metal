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
	"net/http"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// newHandler creates a new handler for the given types, using the given mutator, and logger.
func newHandler(mgr manager.Manager, types []runtime.Object, mutator Mutator, logger logr.Logger) (*handler, error) {
	// Build a map of the given types keyed by their GVKs
	typesMap, err := buildTypesMap(mgr, types)
	if err != nil {
		return nil, err
	}

	// Create and return a handler
	return &handler{
		typesMap: typesMap,
		mutator:  mutator,
		logger:   logger.WithName("handler"),
	}, nil
}

type handler struct {
	typesMap map[metav1.GroupVersionKind]runtime.Object
	mutator  Mutator
	decoder  types.Decoder
	logger   logr.Logger
}

// InjectDecoder injects the given decoder into the handler.
func (h *handler) InjectDecoder(d types.Decoder) error {
	h.decoder = d
	return nil
}

// InjectClient injects the given client into the mutator.
// TODO Replace this with the more generic InjectFunc when controller runtime supports it
func (h *handler) InjectClient(client client.Client) error {
	if _, err := inject.ClientInto(client, h.mutator); err != nil {
		return errors.Wrap(err, "could not inject the client into the mutator")
	}
	return nil
}

// Handle handles the given admission request.
func (h *handler) Handle(ctx context.Context, req types.Request) types.Response {
	ar := req.AdmissionRequest

	// Decode object
	t, ok := h.typesMap[ar.Kind]
	if !ok {
		return admission.ErrorResponse(http.StatusBadRequest, errors.Errorf("unexpected request kind %s", ar.Kind.String()))
	}
	obj := t.DeepCopyObject()
	err := h.decoder.Decode(req, obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, errors.Wrapf(err, "could not decode request %v", ar))
	}

	// Get object accessor
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, errors.Wrapf(err, "could not get accessor for %v", obj))
	}

	// Mutate the resource
	h.logger.Info("Mutating resource", "kind", ar.Kind.String(), "namespace", accessor.GetNamespace(),
		"name", accessor.GetName(), "operation", ar.Operation)
	newObj := obj.DeepCopyObject()
	err = h.mutator.Mutate(ctx, newObj)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError,
			errors.Wrapf(err, "could not mutate %s %s/%s", ar.Kind.Kind, accessor.GetNamespace(), accessor.GetName()))
	}

	// Return a patch response if the resource should be changed
	if !equality.Semantic.DeepEqual(obj, newObj) {
		return admission.PatchResponse(obj, newObj)
	}

	// Return a validation response if the resource should not be changed
	return admission.ValidationResponse(true, "")
}

// buildTypesMap builds a map of the given types keyed by their GroupVersionKind, using the scheme from the given Manager.
func buildTypesMap(mgr manager.Manager, types []runtime.Object) (map[metav1.GroupVersionKind]runtime.Object, error) {
	typesMap := make(map[metav1.GroupVersionKind]runtime.Object)
	for _, t := range types {
		// Get GVK from the type
		gvk, err := apiutil.GVKForObject(t, mgr.GetScheme())
		if err != nil {
			return nil, errors.Wrapf(err, "could not get GroupVersionKind from object %v", t)
		}

		// Add the type to the types map
		typesMap[metav1.GroupVersionKind(gvk)] = t
	}
	return typesMap, nil
}
