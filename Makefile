# Makefile for orch-go

# Binary name
BINARY_NAME=orch

# Build directory
BUILD_DIR=build

# Install directory
INSTALL_DIR=$(HOME)/bin

# Stable release channel directory
STABLE_DIR=$(HOME)/.orch/bin

# Go build flags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
SOURCE_DIR ?= $(shell pwd)
GIT_HASH ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.sourceDir=$(SOURCE_DIR) -X main.gitHash=$(GIT_HASH)"

.PHONY: all build smoke release-stable clean test install install-safe install-restart hooks-install cross-compile-linux fmt lint docs version

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/orch/

# Run lightweight smoke checks against the built binary
smoke: build
	@echo "Running smoke checks..."
	@./$(BUILD_DIR)/$(BINARY_NAME) version --json >/dev/null
	@echo "Smoke checks passed."

# Release smoke-tested binary to stable channel
release-stable: smoke
	@echo "Releasing stable binary..."
	@mkdir -p $(STABLE_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(STABLE_DIR)/$(BINARY_NAME)-stable
	@codesign --force --sign - $(STABLE_DIR)/$(BINARY_NAME)-stable
	@$(STABLE_DIR)/$(BINARY_NAME)-stable version --json > $(STABLE_DIR)/$(BINARY_NAME)-stable.metadata.json
	@echo "Stable binary: $(STABLE_DIR)/$(BINARY_NAME)-stable"
	@echo "Version metadata: $(STABLE_DIR)/$(BINARY_NAME)-stable.metadata.json"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install to ~/bin (symlink to build output) with smoke gate
# Keep `install` as the familiar entrypoint, but require smoke first.
install: install-safe

# Safe install path: smoke must pass before binary promotion
install-safe: smoke
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR) (symlink)..."
	@mkdir -p $(INSTALL_DIR)
	@# Codesign the build output (required for macOS)
	@codesign --force --sign - $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME)
	@# Remove existing file/symlink and create new symlink
	@rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@ln -sf $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Linked $(INSTALL_DIR)/$(BINARY_NAME) → $(CURDIR)/$(BUILD_DIR)/$(BINARY_NAME)"
	@echo ""
	@echo "💡 The orch daemon may need restart to pick up the new binary:"
	@echo "   launchctl kickstart -k gui/$$(id -u)/com.orch.daemon"
	@echo ""
	@echo "Or run: make install-restart"

# Install and restart daemon to pick up new binary
install-restart: install
	@echo "Restarting orch daemon..."
	@launchctl kickstart -k gui/$$(id -u)/com.orch.daemon 2>/dev/null || echo "Note: Daemon not running or not installed"
	@echo "Done. Daemon restarted with new binary."

# Install bd git hooks and re-apply project pre-commit guards
hooks-install:
	@echo "Installing bd hooks..."
	@bd hooks install
	@./scripts/install-pre-commit-exec-start-check.sh

# Cross-compile Linux binaries for Docker containers
# Builds bd, orch, and kb for linux/amd64 to ~/.local/bin/linux-amd64/
cross-compile-linux:
	@./scripts/cross-compile-linux.sh --all

# Cross-compile only orch for Linux (faster, for orch-only changes)
cross-compile-linux-orch:
	@./scripts/cross-compile-linux.sh --orch

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f *.test
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint custom
	./bin/custom-gcl run

# Run vet
vet:
	go vet ./...

# Tidy modules
tidy:
	go mod tidy

# Build and run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Generate CLI documentation
docs:
	@echo "Generating CLI documentation..."
	go run ./cmd/gendoc
	@echo "Documentation generated in docs/cli/"

# Show version
version: build
	@./$(BUILD_DIR)/$(BINARY_NAME) version

# Show help
help:
	@echo "Available targets:"
	@echo "  build                  - Build the binary"
	@echo "  smoke                  - Build and run lightweight binary smoke checks"
	@echo "  release-stable         - Promote smoke-tested binary to ~/.orch/bin/orch-stable"
	@echo "  test                   - Run tests"
	@echo "  install                - Run smoke + install to ~/bin"
	@echo "  install-safe           - Run smoke + install to ~/bin"
	@echo "  install-restart        - Install and restart daemon"
	@echo "  hooks-install          - Install bd hooks + project pre-commit guards"
	@echo "  cross-compile-linux    - Build Linux binaries (bd, orch, kb) for Docker"
	@echo "  cross-compile-linux-orch - Build only orch for Linux (faster)"
	@echo "  clean                  - Clean build artifacts"
	@echo "  fmt                    - Format code"
	@echo "  lint                   - Run linter"
	@echo "  vet                    - Run go vet"
	@echo "  tidy                   - Tidy modules"
	@echo "  run                    - Build and run"
	@echo "  docs                   - Generate CLI documentation"
