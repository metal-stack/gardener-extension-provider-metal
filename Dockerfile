FROM golang:1.16 AS builder

WORKDIR /go/src/github.com/metal-stack/gardener-extension-provider-metal
COPY . .
RUN make install

FROM alpine:3.13
WORKDIR /
COPY charts /charts
COPY --from=builder /go/bin/gardener-extension-metal-hyper /gardener-extension-metal-hyper
CMD ["/gardener-extension-metal-hyper"]
