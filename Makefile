# chr - Git commit management tool
.PHONY: build test clean install lint fmt help

# Default target
.DEFAULT_GOAL := help

# Build configuration
BINARY_NAME := chr
VERSION := 0.0.4
BUILD_DIR := dist
MAIN_PATH := .

# Go configuration
GO := go
GOFMT := gofmt
GOLINT := golangci-lint

## Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

## Install to system
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "$(BINARY_NAME) installed successfully!"

## Install to local bin
install-local: build
	@echo "Installing $(BINARY_NAME) to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "$(BINARY_NAME) installed to ~/.local/bin"

## Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

## Lint code
lint:
	@echo "Linting code..."
	$(GOLINT) run

## Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## Development build (with race detection)
dev-build:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -race -ldflags "-X main.version=$(VERSION)-dev" -o $(BUILD_DIR)/$(BINARY_NAME)-dev $(MAIN_PATH)

## Run the tool (development)
run:
	$(GO) run $(MAIN_PATH)

## Show version
version:
	@echo "chr version $(VERSION)"

## Create release build for multiple platforms
release:
	@echo "Creating release builds..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux x86_64
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	
	# macOS x86_64
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	@echo "Release builds created in $(BUILD_DIR)/"

## Show help
help:
	@echo "chr - Git commit management tool"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)