# Reproducible, override-safe multi-arch build and release
# usage:
#   make build-linux-amd64
#   make build-linux-arm64
#   make release VERSION=v1.2.3

SHELL := /usr/bin/env bash
BINARY ?= cloudflare-dns-failover
DIST   ?= dist

# Reproducible metadata (override if needed)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Reproducible build flags
GOFLAGS  ?= -trimpath -buildvcs=false -mod=readonly
LDFLAGS  ?= -s -w -buildid= -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'
export SOURCE_DATE_EPOCH ?= 0
TARFLAGS ?= --owner=0 --group=0 --numeric-owner --sort=name --mtime=@$(SOURCE_DATE_EPOCH)

.PHONY: all clean release \
        build-linux-amd64 build-linux-arm64 \
        package-linux-amd64 package-linux-arm64 \
        checksums

all: release

clean:
    rm -rf $(DIST)

# ---------- builds ----------
build-linux-amd64:
    mkdir -p $(DIST)/linux_amd64
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/linux_amd64/$(BINARY) .

build-linux-arm64:
    mkdir -p $(DIST)/linux_arm64
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(DIST)/linux_arm64/$(BINARY) .

# ---------- packages ----------
package-linux-amd64: build-linux-amd64
    # include example config if present (no-op if missing)
    @if [ -f config.yml.example ]; then cp -f config.yml.example $(DIST)/linux_amd64/; fi
    tar $(TARFLAGS) -C $(DIST)/linux_amd64 -czf $(DIST)/$(BINARY)_$(VERSION)_linux_amd64.tar.gz .

package-linux-arm64: build-linux-arm64
    @if [ -f config.yml.example ]; then cp -f config.yml.example $(DIST)/linux_arm64/; fi
    tar $(TARFLAGS) -C $(DIST)/linux_arm64 -czf $(DIST)/$(BINARY)_$(VERSION)_linux_arm64.tar.gz .

checksums:
    ( cd $(DIST) && sha256sum *.tar.gz > sha256sums.txt )

release: clean package-linux-amd64 package-linux-arm64 checksums
    @echo "Release artifacts in $(DIST):"
    @ls -1 $(DIST) | sed 's/^/  - /'
