.PHONY: all build test lint run clean deps docker-up docker-down wire swagger install-hooks \
	db-create db-drop db-reset db-migrate db-migrate-up-one db-migrate-down-one db-migrate-down-all db-migrate-create db-setup db-seed \
	mocks mock-clean docker-restart docker-logs docker-build docker-build-prod docker-push \
	fmt vet lint-fix security update-deps verify-deps test-coverage test-race test-integration \
	test-e2e clean-coverage deep-clean ci dev help

# ═══════════════════════════════════════════════════════════════════════════
# VARIABLES
# ═══════════════════════════════════════════════════════════════════════════

GO := go
BINARY_DIR := bin
SERVICES := service-auth service-user service-product
DOCKER_SERVICES := auth user product
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# ═══════════════════════════════════════════════════════════════════════════
# DEFAULT TARGET
# ═══════════════════════════════════════════════════════════════════════════

all: deps wire build ## Default target: install dependencies, generate Wire, and build

# ═══════════════════════════════════════════════════════════════════════════
# BUILD
# ═══════════════════════════════════════════════════════════════════════════

build: ## Build all services
	@echo "Building all services..."
	@mkdir -p $(BINARY_DIR)
	@for service in $(SERVICES); do \
		echo "  → Building $$service..."; \
		$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$$service ./cmd/$$service; \
	done
	@echo "✅ Build complete!"

build-%: ## Build a specific service (e.g., make build-service-auth)
	@echo "Building $*..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$* ./cmd/$*
	@echo "✅ $* built successfully!"

build-prod: ## Build for production (optimized)
	@echo "Building for production..."
	@mkdir -p $(BINARY_DIR)
	@for service in $(SERVICES); do \
		echo "  → Building $$service..."; \
		CGO_ENABLED=0 GOOS=linux $(GO) build -a -installsuffix cgo $(LDFLAGS) -o $(BINARY_DIR)/$$service ./cmd/$$service; \
	done
	@echo "✅ Production build complete!"

# ═══════════════════════════════════════════════════════════════════════════
# DEVELOPMENT
# ═══════════════════════════════════════════════════════════════════════════

run: ## Run auth service (default)
	$(GO) run ./cmd/service-auth

run-%: ## Run a specific service (e.g., make run-service-user)
	$(GO) run ./cmd/$*

dev: ## Run with hot reload (requires air)
	@which air > /dev/null || (echo "Installing air..." && $(GO) install github.com/air-verse/air@latest)
	air -c .air.toml

# ═══════════════════════════════════════════════════════════════════════════
# WIRE (Dependency Injection)
# ═══════════════════════════════════════════════════════════════════════════

wire: ## Generate Wire dependency injection
	@which wire > /dev/null || (echo "Installing wire..." && $(GO) install github.com/google/wire/cmd/wire@latest)
	wire gen ./cmd/...
	@echo "✅ Wire generation complete!"

# ═══════════════════════════════════════════════════════════════════════════
# SWAGGER (API Documentation)
# ═══════════════════════════════════════════════════════════════════════════

swagger: ## Generate Swagger documentation
	@which swag > /dev/null || (echo "Installing swag..." && $(GO) install github.com/swaggo/swag/cmd/swag@latest)
	swag init -g cmd/service-auth/main.go -o cmd/service-auth/docs --parseDependency --parseInternal --exclude "internal/user,internal/product"
	swag init -g cmd/service-user/main.go -o cmd/service-user/docs --parseDependency --parseInternal --exclude "internal/auth,internal/product"
	swag init -g cmd/service-product/main.go -o cmd/service-product/docs --parseDependency --parseInternal --exclude "internal/auth,internal/user"
	@echo "✅ Swagger documentation generated!"

# ═══════════════════════════════════════════════════════════════════════════
# TESTING
# ═══════════════════════════════════════════════════════════════════════════

test: ## Run all tests
	$(GO) test -v -race ./...

test-coverage: ## Run tests with coverage report (focused on business logic)
	$(GO) test -v -race -coverprofile=internal.out ./internal/...
	@cat internal.out | grep -v "/mocks/" | grep -v "/domain/" | grep -v "/dto/" | grep -v "/common/" > coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out | grep total
	@echo "✅ Coverage report generated: coverage.html (Business Logic Only)"
	@rm -f internal.out

test-race: ## Run tests with race detector
	$(GO) test -race ./...

test-%: ## Run tests for a specific package (e.g., make test-auth)
	$(GO) test -v -race ./$*...

test-integration: ## Run integration tests
	$(GO) test -v -race -tags=integration ./test/integration/...

test-e2e: ## Run e2e tests
	$(GO) test -v -race -tags=e2e ./test/e2e/...

# ═══════════════════════════════════════════════════════════════════════════
# MOCKS
# ═══════════════════════════════════════════════════════════════════════════

mocks: ## Generate mock files using mockery
	@which mockery > /dev/null || (echo "Installing mockery..." && $(GO) install github.com/vektra/mockery/v2@latest)
	mockery --dir=internal/auth/repository --name=UserRepository --output=test/mocks --outpkg=mocks
	mockery --dir=internal/auth/repository --name=SessionRepository --output=test/mocks --outpkg=mocks
	@echo "✅ Mocks generated successfully!"

mock-clean: ## Remove generated mock files
	@echo "Cleaning mock files..."
	@find ./test -type d -name "mocks" -exec rm -rf {} + 2>/dev/null || true
	@echo "✅ Mock files cleaned!"

# ═══════════════════════════════════════════════════════════════════════════
# CODE QUALITY
# ═══════════════════════════════════════════════════════════════════════════

fmt: ## Format Go code
	$(GO) fmt ./...
	@which goimports > /dev/null && goimports -w . || echo "goimports not installed"

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

security: ## Check for security vulnerabilities
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && $(GO) install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# ═══════════════════════════════════════════════════════════════════════════
# DEPENDENCIES
# ═══════════════════════════════════════════════════════════════════════════

deps: ## Download and tidy dependencies
	$(GO) mod download
	$(GO) mod tidy

update-deps: ## Update all dependencies
	$(GO) get -u ./...
	$(GO) mod tidy

verify-deps: ## Verify dependencies
	$(GO) mod verify

# ═══════════════════════════════════════════════════════════════════════════
# GIT HOOKS
# ═══════════════════════════════════════════════════════════════════════════

install-hooks: ## Install Git hooks (pre-commit, commit-msg)
	@chmod +x .githooks/pre-commit .githooks/commit-msg
	@git config core.hooksPath .githooks
	@echo "✅ Git hooks installed!"
	@echo "   • pre-commit  - Runs gofmt, go vet, golangci-lint"
	@echo "   • commit-msg  - Validates commit message format"

# ═══════════════════════════════════════════════════════════════════════════
# DOCKER
# ═══════════════════════════════════════════════════════════════════════════

docker-up: ## Start Docker containers
	docker-compose -f deployments/docker-compose.yml up -d

docker-down: ## Stop Docker containers
	docker-compose -f deployments/docker-compose.yml down

docker-restart: docker-down docker-up ## Restart all Docker containers

docker-logs: ## Tail Docker container logs
	docker-compose -f deployments/docker-compose.yml logs -f

docker-build: ## Build Docker images
	docker-compose -f deployments/docker-compose.yml build

docker-build-prod: ## Build production Docker images
	@for service in $(DOCKER_SERVICES); do \
		echo "  → Building $$service image..."; \
		docker build -f deployments/docker/Dockerfile.$$service -t go-microservices/$$service:$(VERSION) .; \
	done
	@echo "✅ Production images built!"

docker-push: ## Push Docker images to registry
	@for service in $(DOCKER_SERVICES); do \
		echo "  → Pushing $$service image..."; \
		docker push go-microservices/$$service:$(VERSION); \
	done
	@echo "✅ Images pushed!"

# ═══════════════════════════════════════════════════════════════════════════
# DATABASE
# ═══════════════════════════════════════════════════════════════════════════

db-create: ## Create database (if not exists)
	$(GO) run ./cmd/db-create

db-drop: ## Drop database
	$(GO) run ./cmd/db-drop
	@sleep 1

db-reset: ## Reset database (drop and create)
	$(MAKE) db-drop
	sleep 2
	$(MAKE) db-create

# Run all SQL file migrations
db-migrate: ## Apply all SQL migrations from migrations/
	$(GO) run ./cmd/db-migrate up-all

# Apply one file migration (up)
db-migrate-up-one: ## Apply exactly one SQL file migration (up)
	$(GO) run ./cmd/db-migrate up-one

# Roll back one file migration (down)
db-migrate-down-one: ## Roll back exactly one SQL file migration (down)
	$(GO) run ./cmd/db-migrate down-one

# Roll back all file migrations
db-migrate-down-all: ## Roll back all SQL file migrations (down all)
	$(GO) run ./cmd/db-migrate down-all

# Create a new sequential migration file (usage: make db-migrate-create name=add_column_to_users)
db-migrate-create: ## Generate a new sequential migration SQL file
	$(GO) run ./cmd/db-migrate-create "$(name)"

# Full setup: create DB + run migrations
db-setup: ## Full setup: create DB + run migrations
	$(MAKE) db-create
	$(MAKE) db-migrate

# Seed database
db-seed: ## Seed database with sample data
	$(GO) run ./cmd/db-seed

# ═══════════════════════════════════════════════════════════════════════════
# KUBERNETES (Future Development)
# ═══════════════════════════════════════════════════════════════════════════
# Note: Kubernetes deployment is planned for future development.
# The following commands will be enabled when K8s support is added.

# # Apply Kubernetes manifests (local)
# k8s-apply-local:
# 	kubectl apply -k deployments/k8s/overlays/local

# # Apply Kubernetes manifests (staging)
# k8s-apply-staging:
# 	kubectl apply -k deployments/k8s/overlays/staging

# # Apply Kubernetes manifests (production)
# k8s-apply-production:
# 	kubectl apply -k deployments/k8s/overlays/production

# # Delete Kubernetes resources
# k8s-delete:
# 	kubectl delete -k deployments/k8s/overlays/local

# ═══════════════════════════════════════════════════════════════════════════
# CLEANUP
# ═══════════════════════════════════════════════════════════════════════════

clean: ## Clean build artifacts and coverage files
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html internal.out clean.out
	@find . -name "*.out" -delete
	@echo "✅ Clean complete!"

clean-coverage: ## Clean only coverage files
	rm -f coverage.out coverage.html internal.out clean.out
	@find . -name "*.out" -delete
	@echo "✅ Coverage files cleaned!"

deep-clean: clean ## Deep clean (including vendor and cache)
	rm -rf vendor
	$(GO) clean -cache -testcache -modcache
	@echo "✅ Deep clean complete!"

# ═══════════════════════════════════════════════════════════════════════════
# CI/CD
# ═══════════════════════════════════════════════════════════════════════════

ci: deps lint test build ## Run CI pipeline locally
	@echo "✅ CI pipeline passed!"

# ═══════════════════════════════════════════════════════════════════════════
# HELP
# ═══════════════════════════════════════════════════════════════════════════

help: ## Show this help message
	@echo "Go Microservices Redis PubSub Boilerplate"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Pattern Targets:"
	@echo "  build-<service>   - Build a specific service (service-auth, service-user, service-product)"
	@echo "  run-<service>     - Run a specific service"
	@echo "  test-<package>    - Run tests for a specific package"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
