# Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

REGISTRY                    := metalpod
IMAGE_PREFIX                := $(REGISTRY)
REPO_ROOT                   := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
HACK_DIR                    := $(REPO_ROOT)/hack
HOSTNAME                    := $(shell hostname)
VERSION                     := $(shell bash -c 'source $(HACK_DIR)/common.sh && echo $$VERSION')
LD_FLAGS                    := "-w -X github.com/gardener/gardener-extensions/pkg/version.Version=$(IMAGE_TAG)"
VERIFY                      := true
LEADER_ELECTION             := false
IGNORE_OPERATION_ANNOTATION := false
WEBHOOK_CONFIG_URL          := localhost


### Build commands

.PHONY: format
format:
	@./hack/format.sh

.PHONY: clean
clean:
	@./hack/clean.sh

.PHONY: generate
generate:
	@./hack/generate.sh

.PHONY: check
check:
	@./hack/check.sh

.PHONY: test
test:
	@./hack/test.sh

.PHONY: verify
verify: check generate test format

.PHONY: install
install:
	@./hack/install.sh

.PHONY: all
ifeq ($(VERIFY),true)
all: verify generate install
else
all: generate install
endif

### Docker commands

.PHONY: docker-image
docker-image:
	@docker build --no-cache --build-arg VERIFY=$(VERIFY) -t $(IMAGE_PREFIX)/gardener-extension-provider-metal:$(VERSION) -t $(IMAGE_PREFIX)/gardener-extension-provider-metal:latest -f Dockerfile -m 6g .

### Debug / Development commands

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy

.PHONY: start-provider-metal
start-provider-metal:
	@LEADER_ELECTION_NAMESPACE=garden go run \
		-ldflags $(LD_FLAGS) \
		-tags netgo \
		./cmd/gardener-extension-provider-metal \
		--config-file=./example/00-componentconfig.yaml \
		--ignore-operation-annotation=$(IGNORE_OPERATION_ANNOTATION) \
		--leader-election=$(LEADER_ELECTION) \
		--webhook-config-server-host=$(HOSTNAME) \
		--webhook-config-server-port=8443 \
		--webhook-config-mode=url \
		--webhook-config-url=$(WEBHOOK_CONFIG_URL)