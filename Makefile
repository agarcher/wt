.PHONY: build build-all install clean test lint release

# Version information - prefer VERSION file, fall back to git describe
VERSION ?= $(shell cat VERSION 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/agarcher/wt/internal/commands.Version=$(VERSION)"

# Binary name
BINARY := wt

# Build directories
BUILD_DIR := build

# Default target
all: build

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/wt

# Build for all platforms
build-all: build-darwin-amd64 build-darwin-arm64 build-linux-amd64 build-linux-arm64

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/wt

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/wt

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/wt

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/wt

# Install to /usr/local/bin
install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	go vet ./...
	@which golangci-lint > /dev/null && golangci-lint run || echo "golangci-lint not installed, skipping"

# Run go mod tidy
tidy:
	go mod tidy

# Development: build and run
run: build
	./$(BUILD_DIR)/$(BINARY)

# Create a release (bump version, tag, push)
# Usage: make release patch "Fix bug"
#        make release minor "Add feature"
#        make release major "Breaking change"
release:
	@./scripts/release.sh $(filter-out $@,$(MAKECMDGOALS))

# Catch-all to allow positional arguments to release target
%:
	@:

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms (darwin/linux, amd64/arm64)"
	@echo "  install     - Install to /usr/local/bin"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linters"
	@echo "  tidy        - Run go mod tidy"
	@echo "  release     - Create a release: make release <major|minor|patch> \"notes\""
	@echo "  help        - Show this help"
