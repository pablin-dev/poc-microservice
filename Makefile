GOPATH := $(shell go env GOPATH)

.PHONY: build-images up down test-e2e clean all build unit_test lint fmt test_race

# Variables
DOCKER_COMPOSE_FILE = docker-compose.yaml
KAFKA_BOOTSTRAP_SERVERS="localhost:9092,localhost:9094,localhost:9096" 
MOUNTEBANK_BASE_URL="http://127.0.0.1:2525" 
SOAP_SERVICE_URL="http://127.0.0.1:8081/soap" 
KYC_ADMIN_BASE_URL="http://127.0.0.1:8081/admin/v1" 

build:
	@echo "Building Go services locally..."
	go build -buildvcs=false -o services/consumer/consumer-service ./services/consumer
	go build -buildvcs=false -o services/providers/kyc/kyc-service ./services/providers/kyc

unit-test:
	@echo "Running unit tests..."
	go test -v ./services/consumer/...
	go test -v ./services/providers/kyc/...

test-race:
	@echo "Running unit tests with race detector..."
	go test -race -v ./services/consumer/...
	go test -race -v ./services/providers/kyc/...

lint:
	GOFLAGS=-buildvcs=false $(GOBIN)/golangci-lint run ./...

fmt:
	@echo "Formatting Go code..."
	go fmt ./...

build-images:
	@echo "Building Docker images..."
	docker compose -f ${DOCKER_COMPOSE_FILE} build

up: build-images
	@echo "Starting Docker Compose services..."
	docker compose -f ${DOCKER_COMPOSE_FILE} up -d
	sleep 10

down:
	@echo "Stopping and removing Docker Compose services..."
	docker compose -f ${DOCKER_COMPOSE_FILE} down

test-e2e:
	@echo "Running E2E tests locally..."
	ginkgo -v ./tests/e2e/...

tidy:
	go mod tidy

clean:
	go clean -modcache
	@echo "Cleaning up Docker images and build artifacts..."
	docker compose -f ${DOCKER_COMPOSE_FILE} down --rmi local
	# Clean up any locally built Go binaries (if any were built outside Docker)
	rm -f services/consumer/consumer-service services/providers/kyc/kyc-service

all: clean build lint unit_test build-images up test-e2e down
