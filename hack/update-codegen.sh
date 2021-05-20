#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

rm -f $GOPATH/bin/*-gen

PROJECT_ROOT=$(dirname $0)/..

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "metal:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "metal:v1alpha1" \
  --extra-peer-dirs=github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal,github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  deepcopy,defaulter \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client/componentconfig \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "config:v1alpha1" \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"

bash "${PROJECT_ROOT}"/vendor/k8s.io/code-generator/generate-internal-groups.sh \
  conversion \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/client/componentconfig \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  github.com/metal-stack/gardener-extension-provider-metal/pkg/apis \
  "config:v1alpha1" \
  --extra-peer-dirs=github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config,github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/v1alpha1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/conversion,k8s.io/apimachinery/pkg/runtime,github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config/v1alpha1 \
  --go-header-file "${PROJECT_ROOT}/vendor/github.com/gardener/gardener/hack/LICENSE_BOILERPLATE.txt"
