IMAGE_TAG                   := $(or ${GITHUB_TAG_NAME}, latest)
REGISTRY                    := ghcr.io/metal-stack
IMAGE_PREFIX                := $(REGISTRY)
REPO_ROOT                   := $(shell dirname "$(realpath $(lastword $(MAKEFILE_LIST)))")
HACK_DIR                    := $(REPO_ROOT)/hack
HOSTNAME                    := $(shell hostname)
LD_FLAGS                    := "-w -X github.com/metal-stack/gardener-extension-provider-metal/pkg/version.Version=$(IMAGE_TAG)"
VERIFY                      := true
LEADER_ELECTION             := false
IGNORE_OPERATION_ANNOTATION := false
WEBHOOK_CONFIG_URL          := localhost

GOLANGCI_LINT_VERSION := v1.48.0

ifeq ($(CI),true)
  DOCKER_TTY_ARG=""
else
  DOCKER_TTY_ARG=t
endif

export GO111MODULE := on

TOOLS_DIR := hack/tools
-include vendor/github.com/gardener/gardener/hack/tools.mk

#########################################
# Rules for local development scenarios #
#########################################

.PHONY: build
build:
	go build -ldflags $(LD_FLAGS) -tags netgo ./cmd/gardener-extension-provider-metal

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

.PHONY: start-admission-metal
start-admission-metal:
	@LEADER_ELECTION_NAMESPACE=garden go run \
		-ldflags $(LD_FLAGS) \
		-tags netgo \
		./cmd/gardener-extension-admission-metal \
		--webhook-config-server-host=0.0.0.0 \
		--webhook-config-server-port=9443 \
		--webhook-config-cert-dir=./example/admission-metal-certs

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: build
build:
	go build -ldflags $(LD_FLAGS) -tags netgo ./cmd/gardener-extension-provider-metal

.PHONY: install
install: revendor $(HELM)
	@LD_FLAGS="-w -X github.com/gardener/$(EXTENSION_PREFIX)-$(NAME)/pkg/version.Version=$(VERSION)" \
	$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/install.sh ./...

.PHONY: docker-image
docker-image:
	@docker build --no-cache \
		--build-arg VERIFY=$(VERIFY) \
		--tag $(IMAGE_PREFIX)/gardener-extension-provider-metal:$(IMAGE_TAG) \
		--file Dockerfile --memory 6g .

.PHONY: docker-push
docker-push:
	@docker push $(IMAGE_PREFIX)/gardener-extension-provider-metal:$(IMAGE_TAG)

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: revendor
revendor:
	@GO111MODULE=on go mod vendor
	@GO111MODULE=on go mod tidy
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/*
	@chmod +x $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/.ci/*
	@$(REPO_ROOT)/hack/update-github-templates.sh

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/clean.sh ./cmd/... ./pkg/...

.PHONY: check-generate
check-generate:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/...
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/check-charts.sh ./charts

.PHONY: generate
generate: $(HELM)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/generate.sh ./charts/... ./cmd/... ./pkg/...

.PHONY: generate-in-docker
generate-in-docker: revendor $(HELM)
	echo $(shell git describe --abbrev=0 --tags) > VERSION
	docker run --rm -i$(DOCKER_TTY_ARG) -v $(PWD):/go/src/github.com/metal-stack/gardener-extension-provider-metal golang:1.19.4 \
		sh -c "cd /go/src/github.com/metal-stack/gardener-extension-provider-metal \
				&& make generate \
				&& chown -R $(shell id -u):$(shell id -g) ."

.PHONY: format
format: $(GOIMPORTS)
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/format.sh ./cmd ./pkg

.PHONY: test
test:
	@SKIP_FETCH_TOOLS=1 $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test.sh ./cmd/... ./pkg/...

.PHONY: test-in-docker
test-in-docker: revendor
	docker run --rm -i$(DOCKER_TTY_ARG) -v $(PWD):/go/src/github.com/metal-stack/gardener-extension-provider-metal golang:1.19.4 \
		sh -c "cd /go/src/github.com/metal-stack/gardener-extension-provider-metal \
				&& make install check test"

.PHONY: test-cov
test-cov:
	@SKIP_FETCH_TOOLS=1 $(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover.sh -r ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	@$(REPO_ROOT)/vendor/github.com/gardener/gardener/hack/test-cover-clean.sh

.PHONY: verify
verify: check format test
