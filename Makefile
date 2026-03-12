# Makefile for orch-go

# Binary name
BINARY_NAME=orch

# Build directory
BUILD_DIR=build

# Install directory
INSTALL_DIR=$(HOME)/bin

# Go build flags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
SOURCE_DIR ?= $(shell pwd)
GIT_HASH ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.sourceDir=$(SOURCE_DIR) -X main.gitHash=$(GIT_HASH)"

.PHONY: all build clean test install install-restart fmt lint docs version

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/orch/

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Install to ~/bin (symlink to build output)
# This makes `make build` automatically update the human-accessible CLI
install: build
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

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	go clean

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

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
	@echo "  build           - Build the binary"
	@echo "  test            - Run tests"
	@echo "  install         - Install to ~/bin (symlink to build output)"
	@echo "  install-restart - Install and restart daemon"
	@echo "  clean           - Clean build artifacts"
	@echo "  fmt             - Format code"
	@echo "  lint            - Run linter"
	@echo "  vet             - Run go vet"
	@echo "  tidy            - Tidy modules"
	@echo "  run             - Build and run"
	@echo "  docs            - Generate CLI documentation"
