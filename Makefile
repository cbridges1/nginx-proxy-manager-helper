# Makefile for nginx-proxy-manager-helper

# Variables
APP_NAME = nginx-proxy-manager-helper
BUILD_DIR = ./cmd/$(APP_NAME)
BINARY_NAME = $(APP_NAME)

# Go related variables
GOBASE = $(shell pwd)
GOBIN = $(GOBASE)/bin
GOFILES = $(wildcard *.go)

# Default target
.PHONY: all
all: clean test build

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@cd $(BUILD_DIR) && go build -o ../../$(BINARY_NAME)

# Run all tests
.PHONY: test
test:
	@echo "Running all tests..."
	@go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run only unit tests (exclude integration tests)
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	@go test -v ./internal/...

# Run only integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@go test -v ./test/...

# Run tests in short mode
.PHONY: test-short
test-short:
	@echo "Running tests in short mode..."
	@go test -short -v ./...

# Run specific package tests
.PHONY: test-config
test-config:
	@echo "Running config tests..."
	@go test -v ./internal/config/

.PHONY: test-models
test-models:
	@echo "Running models tests..."
	@go test -v ./internal/models/

.PHONY: test-utils
test-utils:
	@echo "Running utils tests..."
	@go test -v ./internal/utils/

.PHONY: test-database
test-database:
	@echo "Running database tests..."
	@go test -v ./internal/database/

.PHONY: test-npm
test-npm:
	@echo "Running NPM tests..."
	@go test -v ./internal/npm/

.PHONY: test-cloudflare
test-cloudflare:
	@echo "Running Cloudflare tests..."
	@go test -v ./internal/cloudflare/

.PHONY: test-reconciler
test-reconciler:
	@echo "Running reconciler tests..."
	@go test -v ./internal/reconciler/

# Benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@go clean

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	@go vet ./...

# Run linter (requires golangci-lint to be installed)
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

# Tidy go modules
.PHONY: tidy
tidy:
	@echo "Tidying go modules..."
	@go mod tidy

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download

# Run the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	@./$(BINARY_NAME)

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p dist
	@cd $(BUILD_DIR) && GOOS=linux GOARCH=amd64 go build -o ../../dist/$(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && GOOS=darwin GOARCH=amd64 go build -o ../../dist/$(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && GOOS=darwin GOARCH=arm64 go build -o ../../dist/$(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && GOOS=windows GOARCH=amd64 go build -o ../../dist/$(BINARY_NAME)-windows-amd64.exe

# Development workflow
.PHONY: dev
dev: clean fmt vet test build

# CI workflow
.PHONY: ci
ci: clean fmt vet test-coverage

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all            - Clean, test, and build"
	@echo "  build          - Build the application"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-unit      - Run only unit tests"
	@echo "  test-integration - Run only integration tests"
	@echo "  test-short     - Run tests in short mode"
	@echo "  test-<package> - Run tests for specific package"
	@echo "  benchmark      - Run benchmark tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code"
	@echo "  vet            - Vet code"
	@echo "  lint           - Run linter"
	@echo "  tidy           - Tidy go modules"
	@echo "  deps           - Install dependencies"
	@echo "  run            - Build and run the application"
	@echo "  build-all      - Build for multiple platforms"
	@echo "  dev            - Development workflow (clean, fmt, vet, test, build)"
	@echo "  ci             - CI workflow (clean, fmt, vet, test with coverage)"
	@echo "  help           - Show this help message"