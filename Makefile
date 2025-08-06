# Argus Collector Makefile

# Build variables
BINARY_NAME=argus-collector
READER_NAME=argus-reader
PROCESSOR_NAME=argus-processor
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME)_windows.exe
BINARY_DARWIN=$(BINARY_NAME)_darwin
READER_UNIX=$(READER_NAME)_unix
READER_WINDOWS=$(READER_NAME)_windows.exe
READER_DARWIN=$(READER_NAME)_darwin
PROCESSOR_UNIX=$(PROCESSOR_NAME)_unix
PROCESSOR_WINDOWS=$(PROCESSOR_NAME)_windows.exe
PROCESSOR_DARWIN=$(PROCESSOR_NAME)_darwin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
GIT_COMMIT := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_USER := $(shell whoami)

# Build flags
BUILD_FLAGS=-tags rtlsdr
LDFLAGS=-ldflags "-s -w \
	-X 'argus-collector/internal/version.Version=$(VERSION)' \
	-X 'argus-collector/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'argus-collector/internal/version.GitBranch=$(GIT_BRANCH)' \
	-X 'argus-collector/internal/version.BuildDate=$(BUILD_DATE)' \
	-X 'argus-collector/internal/version.BuildUser=$(BUILD_USER)'"

# Default target
.PHONY: all
all: clean deps build

# Build the binary with RTL-SDR support
.PHONY: build
build:
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) .
	$(GOBUILD) $(LDFLAGS) -o $(READER_NAME) ./cmd/argus-reader
	$(GOBUILD) $(LDFLAGS) -o $(PROCESSOR_NAME) ./cmd/argus-processor

# Build the collector utility
.PHONY: build-collector
build-collector:
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) .

# Build the reader utility
.PHONY: build-reader
build-reader:
	$(GOBUILD) $(LDFLAGS) -o $(READER_NAME) ./cmd/argus-reader

# Build the processor utility
.PHONY: build-processor
build-processor:
	$(GOBUILD) $(LDFLAGS) -o $(PROCESSOR_NAME) ./cmd/argus-processor

# Build all tools (collector, reader, processor)
.PHONY: build-all-tools
build-all-tools: build build-reader build-processor

# Build without RTL-SDR support (for testing)
.PHONY: build-stub
build-stub:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all: build-linux build-windows build-darwin

# Build for Linux
.PHONY: build-linux
build-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_UNIX) .

# Build for Windows (requires cross-compilation setup)
.PHONY: build-windows
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_WINDOWS) .

# Build for macOS (requires cross-compilation setup)
.PHONY: build-darwin
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_DARWIN) .

# Install dependencies
.PHONY: deps
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
.PHONY: clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(READER_NAME)
	rm -f $(PROCESSOR_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_WINDOWS)
	rm -f $(BINARY_DARWIN)
	rm -f $(READER_UNIX)
	rm -f $(READER_WINDOWS)
	rm -f $(READER_DARWIN)
	rm -f $(PROCESSOR_UNIX)
	rm -f $(PROCESSOR_WINDOWS)
	rm -f $(PROCESSOR_DARWIN)
	rm -f coverage.out
	rm -f coverage.html

# Install the binary to $GOPATH/bin
.PHONY: install
install: build
	cp $(BINARY_NAME) $$GOPATH/bin/

# Development build with debug info
.PHONY: build-dev
build-dev:
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) .

# Check for security vulnerabilities
.PHONY: security
security:
	$(GOCMD) list -json -m all | nancy sleuth

# Format code
.PHONY: fmt
fmt:
	$(GOCMD) fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Vet code
.PHONY: vet
vet:
	$(GOCMD) vet ./...

# Run all quality checks
.PHONY: check
check: fmt vet lint test

# Create release packages
.PHONY: package
package: build-all
	mkdir -p dist
	tar -czf dist/$(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_UNIX) config.yaml README.md
	zip -r dist/$(BINARY_NAME)-windows-amd64.zip $(BINARY_WINDOWS) config.yaml README.md
	tar -czf dist/$(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_DARWIN) config.yaml README.md

# Quick test run with short duration
.PHONY: test-run
test-run: build
	./$(BINARY_NAME) --duration 2s --gps-port /dev/ttyACM0

# Display help
.PHONY: help
help:
	@echo "Argus Collector Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build           - Build all tools with RTL-SDR support"
	@echo "  build-collector - Build collector binary only"
	@echo "  build-reader    - Build reader utility only"
	@echo "  build-processor - Build TDOA processor utility only"
	@echo "  build-all-tools - Build all tools (collector, reader, processor)"
	@echo "  build-stub      - Build binary without RTL-SDR (testing)"
	@echo "  build-all       - Build for all platforms"
	@echo "  build-linux     - Build for Linux"
	@echo "  build-windows   - Build for Windows"
	@echo "  build-darwin    - Build for macOS"
	@echo "  build-dev       - Development build with debug info"
	@echo ""
	@echo "  deps          - Download and tidy dependencies"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-run      - Quick test run (2 seconds)"
	@echo ""
	@echo "  clean         - Remove build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  package       - Create release packages"
	@echo ""
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code (requires golangci-lint)"
	@echo "  vet           - Vet code"
	@echo "  security      - Security scan (requires nancy)"
	@echo "  check         - Run all quality checks"
	@echo ""
	@echo "  help          - Show this help message"
	@echo ""
	@echo "  version       - Display version information"
	@echo "  show-version  - Show version string only"
	@echo ""
	@echo "Prerequisites for RTL-SDR build:"
	@echo "  sudo apt-get install librtlsdr-dev  # Ubuntu/Debian"
	@echo "  sudo dnf install rtl-sdr-devel      # Fedora/RHEL"

# Display version information
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Git Branch: $(GIT_BRANCH)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Build User: $(BUILD_USER)"

# Show version that will be embedded in binaries
.PHONY: show-version
show-version:
	@echo $(VERSION)

# Default help target
.DEFAULT_GOAL := help
