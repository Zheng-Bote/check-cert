# Makefile for CheckCert
#
#   make          -> build/CheckCert + build/SendMail (static binaries)
#   make test     -> run unit tests
#   make install  -> install to $(PREFIX)/bin

BINARIES   := CheckCert SendMail
PKG        := github.com/Zheng-Bote/check-cert
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo 0.2.0)
VERSION    := $(if $(VERSION),$(VERSION),0.2.0)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w -X $(PKG)/internal/config.Version=$(VERSION) -X $(PKG)/internal/config.BuildDate=$(BUILD_DATE)
PREFIX     ?= /usr/local

.PHONY: all build test vet fmt clean install FORCE

all: build

build: build/CheckCert build/SendMail

build/CheckCert: FORCE
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/checkcert

build/SendMail: FORCE
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $@ ./cmd/sendmail

FORCE:

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

clean:
	rm -rf build

install: build
	for b in $(BINARIES); do install -Dm755 build/$$b $(DESTDIR)$(PREFIX)/bin/$$b; done
