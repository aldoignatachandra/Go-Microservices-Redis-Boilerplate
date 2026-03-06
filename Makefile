.PHONY: all build test lint run clean deps docker-up docker-down wire swagger help

# Variables
GO := go
BINARY_DIR := bin
SERVICES := auth-service user-service product-service
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
all: deps build

# ═══════════════════════════════════════════════════════════════════════════
# BUILD
# ═══════════════════════════════════════════════════════════════════════════

# Build all services
build:
	@echo "Building all services..."
	@mkdir -p $(BINARY_DIR)
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$$service ./cmd/$$service; \
	done
	@echo "Build complete!"

# Build a specific service
build-%:
	@echo "Building $*..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$* ./cmd/$*

# Build for production (optimized)
build-prod:
	@echo "Building for production..."
	@mkdir -p $(BINARY_DIR)
	@for service in $(SERVICES); do \
		echo "Building $$service..."; \
		CGO_ENABLED=0 GOOS=linux $(GO) build -a -installsuffix cgo $(LDFLAGS) -o $(BINARY_DIR)/$$service ./cmd/$$service; \
	done

# ═══════════════════════════════════════════════════════════════════════════
# DEVELOPMENT
# ═══════════════════════════════════════════════════════════════════════════

# Run service (default: auth)
run:
	$(GO) run ./cmd/auth-service

# Run a specific service
run-%:
	$(GO) run ./cmd/$*

# Run with hot reload (requires air)
dev:
	air -c .air.toml

# ═══════════════════════════════════════════════════════════════════════════
# TESTING
# ═══════════════════════════════════════════════════════════════════════════

# Run all tests
test:
	$(GO) test -v -race ./...

# Run tests with coverage (focused on business logic)
# This target filters out mocks, domain structs, and DTOs to give a true reflection of logic coverage.
test-coverage:
	$(GO) test -v -race -coverprofile=internal.out ./internal/...
	@cat internal.out | grep -v "/mocks/" | grep -v "/domain/" | grep -v "/dto/" | grep -v "/common/" > coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out | grep total
	@echo "Coverage report generated: coverage.html (Business Logic Only)"
	@rm -f internal.out

# Run tests for a specific package
test-%:
	$(GO) test -v -race ./$*...

# Run integration tests
test-integration:
	$(GO) test -v -race -tags=integration ./test/integration/...

# Run e2e tests
test-e2e:
	$(GO) test -v -race -tags=e2e ./test/e2e/...

# ═══════════════════════════════════════════════════════════════════════════
# CODE QUALITY
# ═══════════════════════════════════════════════════════════════════════════

# Format code
fmt:
	$(GO) fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not installed"

# Run linters
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Run linters with auto-fix
lint-fix:
	golangci-lint run --fix ./...

# Run go vet
vet:
	$(GO) vet ./...

# Check for security vulnerabilities
security:
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# ═══════════════════════════════════════════════════════════════════════════
# DEPENDENCIES
# ═══════════════════════════════════════════════════════════════════════════

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Update dependencies
update-deps:
	$(GO) get -u ./...
	$(GO) mod tidy

# Verify dependencies
verify-deps:
	$(GO) mod verify

# ═══════════════════════════════════════════════════════════════════════════
# GENERATION
# ═══════════════════════════════════════════════════════════════════════════

# Generate Wire dependency injection
wire:
	@which wire > /dev/null || (echo "Installing wire..." && go install github.com/google/wire/cmd/wire@latest)
	wire gen ./cmd/...

# Generate Swagger documentation
swagger:
	@which swag > /dev/null || (echo "Installing swag..." && go install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/auth-service/main.go -o api/openapi/auth
	swag init -g cmd/user-service/main.go -o api/openapi/user
	swag init -g cmd/product-service/main.go -o api/openapi/product

# Generate mocks for testing
mocks:
	@which mockery > /dev/null || (echo "Installing mockery..." && go install github.com/vektra/mockery/v2@latest)
	mockery --dir=internal/auth/repository --name=UserRepository --output=test/mocks --outpkg=mocks
	mockery --dir=internal/auth/repository --name=SessionRepository --output=test/mocks --outpkg=mocks

# ═══════════════════════════════════════════════════════════════════════════
# DOCKER
# ═══════════════════════════════════════════════════════════════════════════

# Start Docker containers
docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

# Stop Docker containers
docker-down:
	docker-compose -f deployments/docker-compose.yml down

# Restart Docker containers
docker-restart: docker-down docker-up

# View Docker logs
docker-logs:
	docker-compose -f deployments/docker-compose.yml logs -f

# Build Docker images
docker-build:
	docker-compose -f deployments/docker-compose.yml build

# Build production Docker images
docker-build-prod:
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		docker build -f deployments/docker/Dockerfile.$$service -t go-microservices/$$service:$(VERSION) .; \
	done

# Push Docker images
docker-push:
	@for service in $(SERVICES); do \
		echo "Pushing $$service image..."; \
		docker push go-microservices/$$service:$(VERSION); \
	done

# ═══════════════════════════════════════════════════════════════════════════
# DATABASE
# ═══════════════════════════════════════════════════════════════════════════

# Run database migrations up
migrate-up:
	$(GO) run ./cmd/migrate up

# Run database migrations down
migrate-down:
	$(GO) run ./cmd/migrate down

# Create a new migration
migrate-create:
	@read -p "Enter migration name: " name; \
		migrate create -ext sql -dir migrations -seq $$name

# Seed database
seed:
	$(GO) run ./cmd/seed

# ═══════════════════════════════════════════════════════════════════════════
# KUBERNETES
# ═══════════════════════════════════════════════════════════════════════════

# Apply Kubernetes manifests (local)
k8s-apply-local:
	kubectl apply -k deployments/k8s/overlays/local

# Apply Kubernetes manifests (staging)
k8s-apply-staging:
	kubectl apply -k deployments/k8s/overlays/staging

# Apply Kubernetes manifests (production)
k8s-apply-production:
	kubectl apply -k deployments/k8s/overlays/production

# Delete Kubernetes resources
k8s-delete:
	kubectl delete -k deployments/k8s/overlays/local

# ═══════════════════════════════════════════════════════════════════════════
# CLEANUP
# ═══════════════════════════════════════════════════════════════════════════

# Clean build artifacts and coverage files
clean:
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html internal.out clean.out
	@find . -name "*.out" -delete
	@echo "Clean complete!"

# Clean only coverage files
clean-coverage:
	rm -f coverage.out coverage.html internal.out clean.out
	@find . -name "*.out" -delete
	@echo "Coverage files cleaned!"

# Deep clean (including vendor and cache)
deep-clean: clean
	rm -rf vendor
	$(GO) clean -cache -testcache -modcache

# ═══════════════════════════════════════════════════════════════════════════
# CI/CD
# ═══════════════════════════════════════════════════════════════════════════

# Run CI pipeline locally
ci: deps lint test build

# ═══════════════════════════════════════════════════════════════════════════
# HELP
# ═══════════════════════════════════════════════════════════════════════════

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build          - Build all services"
	@echo "  build-<service> - Build a specific service (auth-service, user-service, product-service)"
	@echo "  build-prod     - Build for production (optimized)"
	@echo ""
	@echo "Development:"
	@echo "  run            - Run auth service"
	@echo "  run-<service>  - Run a specific service"
	@echo "  dev            - Run with hot reload (requires air)"
	@echo ""
	@echo "Testing:"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e       - Run end-to-end tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linters"
	@echo "  lint-fix       - Run linters with auto-fix"
	@echo "  vet            - Run go vet"
	@echo "  security       - Check for vulnerabilities"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps           - Download dependencies"
	@echo "  update-deps    - Update dependencies"
	@echo "  verify-deps    - Verify dependencies"
	@echo ""
	@echo "Generation:"
	@echo "  wire           - Generate Wire DI"
	@echo "  swagger        - Generate Swagger docs"
	@echo "  mocks          - Generate mocks for testing"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  docker-build   - Build Docker images"
	@echo "  docker-logs    - View Docker logs"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up     - Run migrations up"
	@echo "  migrate-down   - Run migrations down"
	@echo "  seed           - Seed database"
	@echo ""
	@echo "Kubernetes:"
	@echo "  k8s-apply-local - Apply local K8s manifests"
	@echo "  k8s-apply-staging - Apply staging K8s manifests"
	@echo "  k8s-apply-production - Apply production K8s manifests"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean          - Clean build artifacts"
	@echo "  deep-clean     - Deep clean (including cache)"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci             - Run CI pipeline locally"
