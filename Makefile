GOPATH := $(shell go env GOPATH)

.PHONY: build-images up down test-e2e clean all build unit_test lint fmt test_race

# Variables
DOCKER_COMPOSE_FILE = docker-compose.yaml

build:
	@echo "Building Go services locally..."
	go build -o services/consumer/consumer-service ./services/consumer
	go build -o services/providers/kyc/kyc-service ./services/providers/kyc

unit-test:
	@echo "Running unit tests..."
	go test -v ./services/consumer/...
	go test -v ./services/providers/kyc/...

test-race:
	@echo "Running unit tests with race detector..."
	go test -race -v ./services/consumer/...
	go test -race -v ./services/providers/kyc/...

lint:
	@echo "Running golangci-lint..."
	$(GOPATH)/bin/golangci-lint run ./...

fmt:
	@echo "Formatting Go code..."
	go fmt ./...

build-images:
	@echo "Building Docker images..."
	docker compose -f ${DOCKER_COMPOSE_FILE} build

up: build-images
	@echo "Starting Docker Compose services..."
	docker compose -f ${DOCKER_COMPOSE_FILE} up -d

down:
	@echo "Stopping and removing Docker Compose services..."
	docker compose -f ${DOCKER_COMPOSE_FILE} down

test-e2e:
	@echo "Running E2E tests..."
	docker compose -f ${DOCKER_COMPOSE_FILE} run --rm e2e-tests run -v ./tests/e2e || (echo "E2E tests failed. Printing consumer-service and kyc-service logs:" && docker compose -f ${DOCKER_COMPOSE_FILE} logs consumer-service kyc-service && exit 1)
	# After tests run, bring down the services.
	docker compose -f ${DOCKER_COMPOSE_FILE} down

clean:
	@echo "Cleaning up Docker images and build artifacts..."
	docker compose -f ${DOCKER_COMPOSE_FILE} down --rmi local
	# Clean up any locally built Go binaries (if any were built outside Docker)
	rm -f services/consumer/consumer-service services/providers/kyc/kyc-service

all: clean build lint unit_test build-images up test-e2e down
