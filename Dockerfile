FROM golang:1.22.1-alpine3.19 AS builder

ENV GOARCH $TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /build
COPY *.go go.* ./

RUN go build -buildvcs=false -ldflags="-w -s" -o ./cloudflare-dns-failover

FROM scratch

COPY --from=builder /build/cloudflare-dns-failover /bin/cloudflare-dns-failover

ENTRYPOINT ["/bin/cloudflare-dns-failover"]
