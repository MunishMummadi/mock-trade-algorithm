# Variables
APP_NAME=mock-trade-algorithm
DOCKER_IMAGE=mock-trade-algorithm
GO_VERSION=1.24
BUILD_DIR=build
DATA_DIR=data
CONFIG_DIR=config

# Default target
.DEFAULT_GOAL := help

# Build the application
.PHONY: build
build: ## Build the application
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for production (with optimizations)
.PHONY: build-prod
build-prod: ## Build for production with optimizations
	@echo "Building $(APP_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Production build complete: $(BUILD_DIR)/$(APP_NAME)"

# Run the application
.PHONY: run
run: ## Run the application
	@echo "Running $(APP_NAME)..."
	@mkdir -p $(DATA_DIR)
	go run .

# Run with live reload for development
.PHONY: dev
dev: ## Run with live reload (requires air)
	@echo "Starting development server with live reload..."
	@mkdir -p $(DATA_DIR)
	air

# Test the application
.PHONY: test
test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint the code
.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run

# Format the code
.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Tidy dependencies
.PHONY: tidy
tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean

# Setup development environment
.PHONY: setup
setup: ## Setup development environment
	@echo "Setting up development environment..."
	@mkdir -p $(DATA_DIR) $(CONFIG_DIR)
	@if [ ! -f $(CONFIG_DIR)/.env ]; then \
		cp $(CONFIG_DIR)/.env.example $(CONFIG_DIR)/.env; \
		echo "Created $(CONFIG_DIR)/.env from template"; \
		echo "Please edit $(CONFIG_DIR)/.env with your Alpaca API credentials"; \
	fi
	go mod download
	@echo "Development environment setup complete"

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):latest .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@mkdir -p $(DATA_DIR)
	docker run --rm -it \
		-v $(PWD)/$(DATA_DIR):/app/data \
		-v $(PWD)/$(CONFIG_DIR)/.env:/app/config/.env \
		-p 8080:8080 \
		$(DOCKER_IMAGE):latest

.PHONY: docker-compose-up
docker-compose-up: ## Start with Docker Compose
	@echo "Starting with Docker Compose..."
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop Docker Compose
	@echo "Stopping Docker Compose..."
	docker-compose down

.PHONY: docker-compose-logs
docker-compose-logs: ## View Docker Compose logs
	docker-compose logs -f

# Database targets
.PHONY: db-reset
db-reset: ## Reset database (WARNING: deletes all data)
	@echo "Resetting database..."
	@read -p "Are you sure you want to delete all trading data? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		rm -f $(DATA_DIR)/trades.db; \
		echo "Database reset complete"; \
	else \
		echo "Database reset cancelled"; \
	fi

.PHONY: db-backup
db-backup: ## Backup database
	@echo "Backing up database..."
	@mkdir -p backups
	@cp $(DATA_DIR)/trades.db backups/trades_backup_$(shell date +%Y%m%d_%H%M%S).db
	@echo "Database backup complete"

# Install development tools
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Security scan
.PHONY: security-scan
security-scan: ## Run security scan
	@echo "Running security scan..."
	gosec ./...

# Generate documentation
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060
	@echo "Documentation server started at http://localhost:6060"

# CI/CD targets
.PHONY: ci
ci: lint test build ## Run CI pipeline
	@echo "CI pipeline completed successfully"

# Help target
.PHONY: help
help: ## Show this help message
	@echo "$(APP_NAME) - Mock Trading Algorithm"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST) 