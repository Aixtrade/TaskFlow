.PHONY: all build build-api build-server run-api run-server test lint clean docker-build docker-up docker-down deps tidy proto-gen

GO=go
GOFLAGS=-ldflags="-s -w"
BIN_DIR=bin
API_BINARY=$(BIN_DIR)/api
SERVER_BINARY=$(BIN_DIR)/server

all: build

deps:
	$(GO) mod download

tidy:
	$(GO) mod tidy

build: build-api build-server

build-api:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(API_BINARY) ./cmd/api

build-server:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(SERVER_BINARY) ./cmd/server

run-api:
	$(GO) run ./cmd/api -config configs/config.dev.yaml

run-server:
	$(GO) run ./cmd/server -config configs/config.dev.yaml

test:
	$(GO) test -v -race -cover ./...

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

# Docker commands
docker-build:
	docker build -t taskflow-api -f deployments/docker/Dockerfile.api .
	docker build -t taskflow-server -f deployments/docker/Dockerfile.server .

docker-up:
	docker-compose -f deployments/docker/docker-compose.yaml up -d

docker-down:
	docker-compose -f deployments/docker/docker-compose.yaml down

docker-logs:
	docker-compose -f deployments/docker/docker-compose.yaml logs -f

# Development helpers
redis-up:
	docker run -d --name taskflow-redis -p 6379:6379 redis:7-alpine

redis-down:
	docker stop taskflow-redis && docker rm taskflow-redis

asynqmon:
	@echo "Starting Asynqmon UI at http://localhost:8081"
	docker run -d --name asynqmon -p 8081:8080 hibiken/asynqmon --redis-addr=host.docker.internal:6379

# Proto generation
proto-gen:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/grpc_task/v1/task.proto
	@echo "Proto generation complete"

proto-clean:
	rm -f api/proto/grpc_task/v1/*.pb.go
