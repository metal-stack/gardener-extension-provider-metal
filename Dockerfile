FROM golang:1.15 AS builder

WORKDIR /go/src/github.com/metal-stack/gardener-extension-provider-metal
COPY . .
RUN make install-requirements check test install

FROM alpine:3.12
RUN apk add --update bash curl
WORKDIR /
COPY charts /controllers/provider-metal/charts
COPY --from=builder /go/bin/gardener-extension-metal-hyper /gardener-extension-metal-hyper
CMD ["/gardener-extension-metal-hyper"]
