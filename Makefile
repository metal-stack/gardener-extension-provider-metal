ENSURE_GARDENER_MOD         := $(shell go get github.com/gardener/gardener@$$(go list -m -f "{{.Version}}" github.com/gardener/gardener))
GARDENER_HACK_DIR    		:= $(shell go list -m -f "{{.Dir}}" github.com/gardener/gardener)/hack
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
GO_VERSION                  := 1.23
GOLANGCI_LINT_VERSION       := v1.61.0

ifeq ($(CI),true)
  DOCKER_TTY_ARG=""
else
  DOCKER_TTY_ARG=t
endif

export GO111MODULE := on

TOOLS_DIR := $(HACK_DIR)/tools
include $(GARDENER_HACK_DIR)/tools.mk

#################################################################
# Rules related to binary build, Docker image build and release #
#################################################################

.PHONY: install
install: tidy $(HELM)
	@LD_FLAGS="-w -X github.com/gardener/$(EXTENSION_PREFIX)-$(NAME)/pkg/version.Version=$(VERSION)" \
	bash $(GARDENER_HACK_DIR)/install.sh ./...

.PHONY: build
build:
	go build -ldflags $(LD_FLAGS) -tags netgo ./cmd/gardener-extension-provider-metal

.PHONY: docker-image
docker-image:
	@docker build --no-cache \
		--build-arg VERIFY=$(VERIFY) \
		--tag $(IMAGE_PREFIX)/gardener-extension-provider-metal:$(IMAGE_TAG) \
		--file Dockerfile --memory 6g .

.PHONY: docker-push
docker-push:
	@docker push $(IMAGE_PREFIX)/gardener-extension-provider-metal:$(IMAGE_TAG)

.PHONY: update-crds
update-crds:
	go mod tidy
	cp -f $(shell go list -mod=mod -m -f '{{.Dir}}' all | grep metal-stack/duros-controller)/config/crd/bases/* charts/internal/crds-storage/templates
	cp -f $(shell go list -mod=mod -m -f '{{.Dir}}' all | grep metal-stack/firewall-controller-manager)/config/crds/* charts/internal/crds-firewall/templates/firewall-controller-manager/
	cp -f $(shell go list -mod=mod -m -f '{{.Dir}}' all | grep metal-stack/firewall-controller/v2)/config/crd/bases/* charts/internal/crds-firewall/templates/firewall-controller/
	cp -f charts/internal/crds-firewall/templates/firewall-controller-manager/*monitors.yaml charts/internal/shoot-control-plane/templates/firewall-controller-manager-crds/
	cp -f charts/internal/crds-firewall/templates/firewall-controller/* charts/internal/shoot-control-plane/templates/firewall-controller-crds/

#####################################################################
# Rules for verification, formatting, linting, testing and cleaning #
#####################################################################

.PHONY: tidy
tidy:
	@GO111MODULE=on go mod tidy
	@mkdir -p $(REPO_ROOT)/.ci/hack && cp $(GARDENER_HACK_DIR)/.ci/* $(REPO_ROOT)/.ci/hack/ && chmod +xw $(REPO_ROOT)/.ci/hack/*

.PHONY: clean
clean:
	@$(shell find ./example -type f -name "controller-registration.yaml" -exec rm '{}' \;)
	@bash $(GARDENER_HACK_DIR)/clean.sh ./cmd/... ./pkg/...

.PHONY: check-generate
check-generate:
	@bash $(GARDENER_HACK_DIR)/check-generate.sh $(REPO_ROOT)

.PHONY: check
check: $(GOIMPORTS) $(GOLANGCI_LINT) $(HELM)
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check.sh --golangci-lint-config=./.golangci.yaml ./cmd/... ./pkg/...
	@REPO_ROOT=$(REPO_ROOT) bash $(GARDENER_HACK_DIR)/check-charts.sh ./charts

.PHONY: generate
generate: $(VGOPATH) $(HELM) $(YQ)
	@REPO_ROOT=$(REPO_ROOT) VGOPATH=$(VGOPATH) GARDENER_HACK_DIR=$(GARDENER_HACK_DIR) bash $(GARDENER_HACK_DIR)/generate-sequential.sh ./charts/... ./cmd/... ./pkg/...

.PHONY: generate-in-docker
generate-in-docker: tidy update-crds $(HELM)
	echo $(shell git describe --abbrev=0 --tags) > VERSION
	docker run --rm -i$(DOCKER_TTY_ARG) \
		--volume $(PWD):/go/src/github.com/metal-stack/gardener-extension-provider-metal golang:$(GO_VERSION) \
			sh -c "cd /go/src/github.com/metal-stack/gardener-extension-provider-metal \
					&& make generate \
					&& chown -R $(shell id -u):$(shell id -g) ."

.PHONY: format
format: $(GOIMPORTS)
	@bash $(GARDENER_HACK_DIR)/format.sh ./cmd ./pkg

.PHONY: test
test:
	@bash $(GARDENER_HACK_DIR)/test.sh ./cmd/... ./pkg/...

.PHONY: test-in-docker
test-in-docker: tidy
	docker run --rm -i$(DOCKER_TTY_ARG) \
		--user $$(id -u):$$(id -g) \
		--mount type=tmpfs,destination=/.cache \
		--volume $(PWD):/go/src/github.com/metal-stack/gardener-extension-provider-metal golang:$(GO_VERSION) \
			sh -c "cd /go/src/github.com/metal-stack/gardener-extension-provider-metal \
					&& make install check test"

.PHONY: test-cov
test-cov:
	@SKIP_FETCH_TOOLS=1 $(GARDENER_HACK_DIR)/test-cover.sh -r ./cmd/... ./pkg/...

.PHONY: test-clean
test-clean:
	$(GARDENER_HACK_DIR)/test-cover-clean.sh

.PHONY: verify
verify: check format test
