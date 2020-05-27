#############      builder-base                             #############
FROM golang:1.13 AS builder

WORKDIR /go/src/github.com/metal-stack/gardener-extension-provider-metal
COPY . .
RUN hack/install-requirements.sh \
    && make VERIFY=$VERIFY all

#############      base                                     #############
FROM alpine:3.11
RUN apk add --update bash curl
WORKDIR /
COPY charts /controllers/provider-metal/charts
COPY --from=builder /go/bin/gardener-extension-metal-hyper /gardener-extension-metal-hyper
CMD ["/gardener-extension-metal-hyper"]
