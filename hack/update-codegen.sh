#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

CODE_GEN_DIR=$(go list -m -f '{{.Dir}}' k8s.io/code-generator)
source "${CODE_GEN_DIR}/kube_codegen.sh"

CURRENT_DIR=$(dirname $0)
PROJECT_ROOT="${CURRENT_DIR}"/..

kube::codegen::gen_helpers \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.txt" \
  "${PROJECT_ROOT}/pkg/apis/metal"

kube::codegen::gen_helpers \
  --boilerplate "${PROJECT_ROOT}/hack/boilerplate.txt" \
  "${PROJECT_ROOT}/pkg/apis/config"
