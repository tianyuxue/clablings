BINARY=clablings
PKG=github.com/tianyuxue/clablings
GOBUILD=go build
GOTEST=go test
DIST=dist
PREFIX?=$(HOME)/.local
BINDIR=$(DESTDIR)$(PREFIX)/bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-X $(PKG)/internal/cli.Version=$(VERSION)
STATIC_LDFLAGS=$(LDFLAGS) -s -w
PLATFORMS?=linux/amd64 linux/arm64

.PHONY: all build build-static build-dynamic test test-integration vet lint fmt tidy run install uninstall package clean distclean help

all: fmt vet test build

# Default static build (pure Go, no cgo).
build: build-static

build-static:
	CGO_ENABLED=0 $(GOBUILD) -trimpath -tags netgo,osusergo -ldflags '$(STATIC_LDFLAGS)' -o $(BINARY) ./cmd/$(BINARY)/

# Development alternative: dynamic linking, keeps debug info.
build-dynamic:
	$(GOBUILD) -ldflags '$(LDFLAGS)' -o $(BINARY) ./cmd/$(BINARY)/

test:
	$(GOTEST) ./... -count=1

# Requires root + Docker + containerlab.
test-integration:
	$(GOTEST) -tags integration ./... -count=1 -timeout 1800s

vet:
	go vet ./...

# vet + staticcheck (skipped when staticcheck is not installed).
lint: vet
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; else echo "staticcheck not installed, skipping"; fi

fmt:
	go fmt ./...

tidy:
	go mod tidy

# make run ARGS="list --root ../clab-exercises"
run: build
	./$(BINARY) $(ARGS)

install: build
	@mkdir -p $(BINDIR)
	install -m 0755 $(BINARY) $(BINDIR)/$(BINARY)
	@echo "installed $(BINDIR)/$(BINARY)"

uninstall:
	rm -f $(BINDIR)/$(BINARY)
	@echo "removed $(BINDIR)/$(BINARY)"

# Cross-compile and package as tar.gz + checksums.txt.
package: distclean
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; \
		out=$(DIST)/$(BINARY)_$(VERSION)_$${os}_$${arch}; \
		echo "building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GOBUILD) -trimpath -tags netgo,osusergo -ldflags '$(STATIC_LDFLAGS)' -o $$out ./cmd/$(BINARY)/ || exit 1; \
		tar -C $(DIST) -czf $$out.tar.gz $$(basename $$out); \
		rm -f $$out; \
		sha256sum $$out.tar.gz >> $(DIST)/checksums.txt; \
	done
	@echo "packaging complete: $(DIST)/"

clean:
	rm -f $(BINARY)

distclean:
	rm -rf $(DIST)

help:
	@echo "Available targets:"
	@echo "  build            static build (default)"
	@echo "  build-dynamic    development build (dynamic, with debug info)"
	@echo "  test             unit tests"
	@echo "  test-integration integration tests (requires root + Docker + containerlab)"
	@echo "  lint             go vet + staticcheck"
	@echo "  fmt              go fmt"
	@echo "  tidy             go mod tidy"
	@echo "  run              make run ARGS=\"list --root ../clab-exercises\""
	@echo "  install          install to \$$HOME/.local/bin (override PREFIX/DESTDIR)"
	@echo "  uninstall        remove installed binary"
	@echo "  package          cross-compile and package tar.gz + checksums ($(PLATFORMS))"
	@echo "  clean/distclean  clean up"
