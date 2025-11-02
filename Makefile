LISTEN_ADDR ?= "127.0.0.1\:5000"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary names and paths
BINARY_NAME=gobinrepo
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./cmd/gobinrepo
BUILD_DIR=./bin

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH) -X main.buildDate=$(BUILD_DATE)"

.PHONY: all build clean test coverage deps lint fmt vet run dev install uninstall help

# Default target
all: fmt vet lint test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(MAIN_PATH)

# Build for Linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) verify

# Tidy up dependencies
tidy:
	@echo "Tidying up dependencies..."
	$(GOMOD) tidy

# Update dependencies
update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run; \
	else \
		echo "golangci-lint not found. Install it with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2"; \
	fi

# Vet code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Run the application in development mode

run: build
	@echo "Running $(BINARY_NAME) on $(LISTEN_ADDR)..."
	$(GOCMD) run $(MAIN_PATH) --http-listen-addr="$(LISTEN_ADDR)" $(ARGS)

# Run the application in development mode with live reload (requires air)
dev:
	@echo "Starting development server with live reload..."
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "air not found. Install it with: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		$(MAKE) run; \
	fi

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(LDFLAGS) $(MAIN_PATH)

# Uninstall the binary from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)

# Build and run
build-run: build
	@echo "Running built binary on $(LISTEN_ADDR)..."
	./$(BUILD_DIR)/$(BINARY_NAME) --http-listen-addr="$(LISTEN_ADDR)" $(ARGS)


# Check if dependencies are up to date
check-deps:
	@echo "Checking for outdated dependencies..."
	$(GOCMD) list -u -m all

# Generate code (if you have go:generate directives)
generate:
	@echo "Generating code..."
	$(GOCMD) generate ./...

# Create a release build
release: clean fmt vet lint test
	@echo "Creating release build..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Format, vet, lint, test, and build"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Build the binary for Linux"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy up dependencies"
	@echo "  update       - Update dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code (requires golangci-lint)"
	@echo "  vet          - Vet code"
	@echo "  run          - Run the application (use ARGS='--flag value' for arguments)"
	@echo "  dev          - Run with live reload (requires air)"
	@echo "  install      - Install binary to GOPATH/bin"
	@echo "  uninstall    - Remove binary from GOPATH/bin"
	@echo "  build-run    - Build and run the binary"
	@echo "  check-deps   - Check for outdated dependencies"
	@echo "  generate     - Run go generate"
	@echo "  release      - Create a release build"
	@echo "  help         - Show this help message"
