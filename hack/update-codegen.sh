#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# The code generators are pinned in the go.mod of the dedicated gardener tools module,
# hence it must be used to resolve the k8s.io/code-generator dependency.
GARDENER_HACK_TOOLS_DIR="$(GOWORK=off go list -m -f '{{.Dir}}' github.com/gardener/gardener/hack/tools)"
GARDENER_HACK_TOOLS_MODFILE="${GARDENER_HACK_TOOLS_DIR}/go.mod"
# k8s.io/code-generator is not part of this module's build, so it might not be present in the module
# cache yet. Download it upfront, otherwise 'go list -m' cannot report its directory.
GOWORK=off go mod download -modfile "${GARDENER_HACK_TOOLS_MODFILE}" k8s.io/code-generator
CODE_GEN_DIR="$(GOWORK=off go list -modfile "${GARDENER_HACK_TOOLS_MODFILE}" -m -f '{{.Dir}}' k8s.io/code-generator)"
source "${CODE_GEN_DIR}/kube_codegen.sh"

rm -f "$(go env GOPATH)"/bin/*-gen

CURRENT_DIR=$(dirname "$0")
PROJECT_ROOT="${CURRENT_DIR}"/..

kube::codegen::gen_helpers \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.txt" \
  "${PROJECT_ROOT}/pkg/apis/metal"

kube::codegen::gen_helpers \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.txt" \
  "${PROJECT_ROOT}/pkg/apis/config"
