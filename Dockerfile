FROM golang:1.15 AS builder

WORKDIR /go/src/github.com/metal-stack/gardener-extension-provider-metal
COPY . .
RUN apt-get update
RUN apt-get install patch
RUN make install

FROM alpine:3.12
WORKDIR /
COPY charts /controllers/provider-metal/charts
COPY --from=builder /go/bin/gardener-extension-metal-hyper /gardener-extension-metal-hyper
CMD ["/gardener-extension-metal-hyper"]
