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
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build clean test install fmt lint docs version

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

# Install to ~/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	@mkdir -p $(INSTALL_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY_NAME)"

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
	@echo "  build    - Build the binary"
	@echo "  test     - Run tests"
	@echo "  install  - Install to ~/bin"
	@echo "  clean    - Clean build artifacts"
	@echo "  fmt      - Format code"
	@echo "  lint     - Run linter"
	@echo "  vet      - Run go vet"
	@echo "  tidy     - Tidy modules"
	@echo "  run      - Build and run"
	@echo "  docs     - Generate CLI documentation"
