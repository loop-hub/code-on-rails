.PHONY: build test install clean lint fmt help

# Build variables
BINARY_NAME=cr
BUILD_DIR=bin
CMD_DIR=cmd/cr

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

install: ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install ./$(CMD_DIR)
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -s -w .
	go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

run-init: build ## Run init command
	./$(BUILD_DIR)/$(BINARY_NAME) init

run-check: build ## Run check command
	./$(BUILD_DIR)/$(BINARY_NAME) check

run-learn: build ## Run learn command
	./$(BUILD_DIR)/$(BINARY_NAME) learn

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

example: ## Run example on this codebase
	@echo "Running Code on Rails on itself..."
	go run ./$(CMD_DIR) init || true
	go run ./$(CMD_DIR) check || true

.DEFAULT_GOAL := help
