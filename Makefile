.PHONY: build test release-snapshot help

GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")
LDFLAGS ?= -s -w -X github.com/Muxcore-Media/muxcorectl-cli/internal/cli.Version=$(VERSION)

build:
	$(GO) build -ldflags="$(LDFLAGS)" -o bin/muxcorectl ./cmd/muxcorectl

test:
	$(GO) test -count=1 -timeout 60s ./...

# Local cross-build preview (linux/darwin × amd64/arm64). Tag releases use GoReleaser on self-hosted CI.
release-snapshot:
	goreleaser release --snapshot --clean

help:
	@echo "muxcorectl-cli targets:"
	@echo "  build             Build bin/muxcorectl"
	@echo "  test              Run unit tests"
	@echo "  release-snapshot  GoReleaser snapshot (no publish)"
