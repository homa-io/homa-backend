# Homa Makefile
# Simplifies Docker and development operations

# Variables
DOCKER_IMAGE = homa
DOCKER_TAG = latest
COMMIT_HASH = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_NUMBER = $(shell date +%s)
AUTHOR = $(shell git config user.name 2>/dev/null || echo "unknown")
BUILD_TIMESTAMP = $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Default target
.PHONY: help
help: ## Show this help message
	@echo "Homa Development Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development commands
.PHONY: dev
dev: ## Start development environment with hot reload
	go run main.go -c config.dev.yml --swagger

.PHONY: build
build: ## Build the binary locally
	go build -o homa main.go

.PHONY: run
run: build ## Build and run locally
	./homa -c config.dev.yml --swagger

.PHONY: admin
admin: ## Create admin user (interactive)
	@echo "Creating admin user..."
	@read -p "Email: " email; \
	read -p "Name: " name; \
	read -p "Last Name: " lastname; \
	read -s -p "Password: " password; \
	echo ""; \
	go run main.go -c config.dev.yml --create-admin -email "$$email" -name "$$name" -lastname "$$lastname" -password "$$password"

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		--build-arg BUILDNUMBER=$(BUILD_NUMBER) \
		--build-arg AUTHOR="$(AUTHOR)" \
		--build-arg BUILD_TIMESTAMP=$(BUILD_TIMESTAMP) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):$(COMMIT_HASH) \
		.

.PHONY: docker-dev
docker-dev: ## Start development environment in Docker
	COMMIT_HASH=$(COMMIT_HASH) BUILDNUMBER=$(BUILD_NUMBER) AUTHOR="$(AUTHOR)" BUILD_TIMESTAMP=$(BUILD_TIMESTAMP) \
	docker-compose -f docker-compose.dev.yml up --build

.PHONY: docker-dev-down
docker-dev-down: ## Stop development Docker environment
	docker-compose -f docker-compose.dev.yml down

.PHONY: docker-prod
docker-prod: ## Start production environment in Docker
	COMMIT_HASH=$(COMMIT_HASH) BUILDNUMBER=$(BUILD_NUMBER) AUTHOR="$(AUTHOR)" BUILD_TIMESTAMP=$(BUILD_TIMESTAMP) \
	docker-compose up --build -d

.PHONY: docker-prod-down
docker-prod-down: ## Stop production Docker environment
	docker-compose down

.PHONY: docker-logs
docker-logs: ## Show Docker logs
	docker-compose logs -f homa

.PHONY: docker-logs-dev
docker-logs-dev: ## Show development Docker logs
	docker-compose -f docker-compose.dev.yml logs -f homa-dev

# Database commands
.PHONY: migrate
migrate: ## Run database migrations
	go run main.go -c config.dev.yml --migration-do

.PHONY: docker-migrate
docker-migrate: ## Run database migrations in Docker
	docker-compose exec homa ./homa -c config.yml --migration-do

# Utility commands
.PHONY: clean
clean: ## Clean up build artifacts and Docker resources
	rm -f homa
	docker system prune -f
	docker volume prune -f

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: tidy
tidy: ## Tidy Go modules
	go mod tidy

.PHONY: swagger
swagger: ## View Swagger documentation info
	@echo "Swagger UI will be available at: http://localhost:8000/swagger"
	@echo "OpenAPI spec will be available at: http://localhost:8000/swagger/openapi.json"
	@echo ""
	@echo "Start the application with --swagger flag to enable Swagger UI"

# Health checks
.PHONY: health
health: ## Check application health
	curl -f http://localhost:8000/health || echo "Application not running or unhealthy"

.PHONY: status
status: ## Show Docker container status
	docker-compose ps

.PHONY: status-dev
status-dev: ## Show development Docker container status
	docker-compose -f docker-compose.dev.yml ps