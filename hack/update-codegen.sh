#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# We need to explicitly pass GO111MODULE=off to k8s.io/code-generator as it is significantly slower otherwise,
# see https://github.com/kubernetes/code-generator/issues/100.
export GO111MODULE=off

rm -f $GOPATH/bin/*-gen

PROJECT_ROOT=$(dirname $0)/..

git config --global --add safe.directory /go/src/github.com/metal-stack/gardener-extension-provider-metal

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "metal:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "metal:v1alpha1" \
  --extra-peer-dirs=github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal,github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client/componentconfig \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client/componentconfig \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "config:v1alpha1" \
  --extra-peer-dirs=github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config,github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${PROJECT_ROOT}/hack/boilerplate.txt"
