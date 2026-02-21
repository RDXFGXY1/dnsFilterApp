.PHONY: build run clean test install deps build-all

# Build variables
APP_NAME=dns-filter
VERSION=1.0.0
BUILD_DIR=build
GO=go
GOFLAGS=-ldflags="-s -w"

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/server
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 ./cmd/server
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/server
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/server
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/server
	
	@echo "Build complete for all platforms!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Run the application
run: build
	@echo "Starting $(APP_NAME)..."
	sudo $(BUILD_DIR)/$(APP_NAME)

# Run in development mode
dev:
	@echo "Starting $(APP_NAME) in development mode..."
	sudo $(GO) run ./cmd/server/main.go --dev

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf data/*.db
	rm -rf data/logs/*

# Install as system service (Linux)
install-service:
	@echo "Installing systemd service..."
	sudo cp scripts/dns-filter.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Service installed. Enable with: sudo systemctl enable dns-filter"

# Create necessary directories
setup:
	@echo "Setting up directories..."
	mkdir -p data/logs
	mkdir -p web/static/css
	mkdir -p web/static/js
	mkdir -p web/templates
	@echo "Setup complete!"

# Show help
help:
	@echo "DNS Filter - Makefile commands:"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build         - Build the application"
	@echo "  make build-all     - Build for all platforms (Linux, Windows, macOS)"
	@echo "  make clean         - Remove build artifacts"
	@echo ""
	@echo "Dependency & Setup Commands:"
	@echo "  make deps          - Install Go dependencies"
	@echo "  make setup         - Create necessary directories"
	@echo ""
	@echo "Running Commands:"
	@echo "  make run           - Build and run the application"
	@echo "  make dev           - Run in development mode"
	@echo ""
	@echo "Testing Commands:"
	@echo "  make test          - Run tests"
	@echo "  make test-cli      - Test CLI functionality"
	@echo ""
	@echo "Service Management:"
	@echo "  make install-service - Install as systemd service"
	@echo "  make install-cli   - Install CLI tool system-wide"
	@echo "  make uninstall-cli - Uninstall CLI tool"
	@echo ""
	@echo "CLI Quick Commands:"
	@echo "  make block DOMAIN=example.com  - Block a domain"
	@echo "  make unblock DOMAIN=example.com - Unblock a domain"
	@echo "  make reload        - Reload custom blocklists"
	@echo "  make status        - Show DNS Filter status"
	@echo ""

# CLI Management
install-cli: ## Install CLI tool system-wide
	sudo bash scripts/install-cli.sh

uninstall-cli: ## Remove CLI tool
	sudo rm -f /usr/local/bin/dns-filter-cli
	@echo "CLI tool uninstalled"

test-cli: ## Test CLI functionality
	@echo "Testing CLI..."
	./dns-cli.sh connect
	./dns-cli.sh status
	./dns-cli.sh test pornhub.com
	@echo "CLI tests done!"

# Quick commands
block: ## Block a domain: make block DOMAIN=example.com
	./dns-cli.sh block $(DOMAIN)

unblock: ## Unblock a domain: make unblock DOMAIN=example.com
	./dns-cli.sh unblock $(DOMAIN)

reload: ## Reload custom blocklists
	./dns-cli.sh reload

status: ## Show DNS Filter status
	./dns-cli.sh status
