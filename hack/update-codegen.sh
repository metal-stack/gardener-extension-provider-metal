#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

# setup virtual GOPATH
source "$GARDENER_HACK_DIR"/vgopath-setup.sh

CODE_GEN_DIR=$(go list -m -f '{{.Dir}}' k8s.io/code-generator)
source "${CODE_GEN_DIR}/kube_codegen.sh"

rm -f $GOPATH/bin/*-gen

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

kube::codegen::gen_helpers \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.txt" \
  "${PROJECT_ROOT}/pkg/apis/metal"

kube::codegen::gen_helpers \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.txt" \
  "${PROJECT_ROOT}/pkg/apis/config"