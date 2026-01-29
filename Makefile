.PHONY: build run test lint clean migrate-up migrate-down swagger docker-build docker-up docker-down seed

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=chat-server
MAIN_PATH=./cmd/server

# Docker
DOCKER_COMPOSE=docker-compose

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Run the application
run:
	$(GOCMD) run $(MAIN_PATH)/main.go

# Run tests
test:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Database migrations
migrate-up:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/chat?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/chat?sslmode=disable" down

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# Generate Swagger documentation
swagger:
	swag init -g cmd/server/main.go -o docs

# Docker commands
docker-build:
	docker build -t chat-server .

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

# Run seed data
seed:
	$(GOCMD) run ./scripts/seed.go

# Development setup
dev-setup: deps docker-up migrate-up
	@echo "Development environment is ready!"

# Full CI pipeline locally
ci: lint test build
	@echo "CI pipeline completed successfully!"

# Help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the application"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make lint           - Run linter"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make deps           - Download dependencies"
	@echo "  make migrate-up     - Run database migrations"
	@echo "  make migrate-down   - Rollback database migrations"
	@echo "  make migrate-create - Create a new migration"
	@echo "  make swagger        - Generate Swagger docs"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-up      - Start Docker containers"
	@echo "  make docker-down    - Stop Docker containers"
	@echo "  make seed           - Run seed data"
	@echo "  make dev-setup      - Setup development environment"
	@echo "  make ci             - Run full CI pipeline locally"
