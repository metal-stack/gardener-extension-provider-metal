FROM golang:1.16 AS builder

WORKDIR /go/src/github.com/metal-stack/gardener-extension-provider-metal
COPY . .
RUN apt-get update
# Patch is only required for patching install-requirements.sh; can remove this once fix is in upstream gardener.
RUN apt-get install patch
RUN make install

FROM alpine:3.13
WORKDIR /
COPY charts /controllers/provider-metal/charts
COPY --from=builder /go/bin/gardener-extension-metal-hyper /gardener-extension-metal-hyper
CMD ["/gardener-extension-metal-hyper"]
