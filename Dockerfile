#############      builder-base                             #############
FROM golang:1.13 AS builder

COPY ./hack/install-requirements.sh /install-requirements.sh
COPY ./tools /tools

RUN /install-requirements.sh

WORKDIR /go/src/github.com/metal-pod/gardener-extension-provider-metal
COPY . .

RUN make VERIFY=$VERIFY all

#############      base                                     #############
FROM alpine:3.10 AS base

RUN apk add --update bash curl

WORKDIR /

COPY charts /controllers/provider-metal/charts

COPY --from=builder /go/bin/gardener-extension-provider-metal /gardener-extension-provider-metal

CMD ["/gardener-extension-provider-metal"]
