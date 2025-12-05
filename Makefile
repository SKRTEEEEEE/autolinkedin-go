.PHONY: help build run test clean deps deps-update deps-clean vendor docker-dev docker-test docker-stop lint fmt

# Default target
help:
	@echo "LinkGen AI - Available Make Targets:"
	@echo "  make build         - Build the application binary"
	@echo "  make run           - Run the application locally"
	@echo "  make test          - Run all tests"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Install dependencies"
	@echo "  make deps-update   - Update dependencies"
	@echo "  make deps-clean    - Clean module cache"
	@echo "  make vendor        - Vendor dependencies"
	@echo "  make docker-dev    - Start development environment with Docker"
	@echo "  make docker-test   - Run tests in isolated Docker environment"
	@echo "  make docker-stop   - Stop all Docker containers"
	@echo "  make lint          - Run linters"
	@echo "  make fmt           - Format code"

# Build the application
build:
	@echo "Building LinkGen AI..."
	cd src && go build -o ../bin/linkgenai main.go

# Run the application locally
run:
	@echo "Running LinkGen AI..."
	cd src && go run main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./test/...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	cd src && go clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	cd src && go mod download
	cd src && go mod tidy

# Update dependencies
deps-update:
	@echo "Updating dependencies..."
	cd src && go get -u ./...
	cd src && go mod tidy

# Clean module cache
deps-clean:
	@echo "Cleaning module cache..."
	go clean -modcache

# Vendor dependencies (optional)
vendor:
	@echo "Vendoring dependencies..."
	cd src && go mod vendor

# Start development environment
docker-dev:
	@echo "Starting development environment..."
	docker-compose up -d

# Run tests in Docker
docker-test:
	@echo "Running tests in Docker..."
	docker-compose -f docker-compose.test.yml up --abort-on-container-exit --exit-code-from app
	docker-compose -f docker-compose.test.yml down -v

# Stop Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down
	docker-compose -f docker-compose.test.yml down -v

# Run linters
lint:
	@echo "Running linters..."
	cd src && golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	cd src && go fmt ./...
	cd test && go fmt ./...
