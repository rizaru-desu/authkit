.PHONY: build test lint fmt run clean deps sqlc migrate-up migrate-down help

BINARY_NAME=mns-backend
BUILD_DIR=bin
GO=go

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

run:
	$(GO) run ./cmd/server

test:
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test
	$(GO) tool cover -html=coverage.out

lint:
	golangci-lint run ./...

fmt:
	$(GO) fmt ./...

clean:
	rm -rf $(BUILD_DIR) coverage.out

deps:
	$(GO) mod download
	$(GO) mod tidy

sqlc:
	sqlc generate

migrate-up:
	migrate -path migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path migrations -database "$$DATABASE_URL" down

docker-build:
	docker build -t $(BINARY_NAME):latest .

help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  run           - Run the server"
	@echo "  test          - Run tests with race detector"
	@echo "  test-coverage - Run tests and open HTML coverage"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format code"
	@echo "  clean         - Remove build artifacts"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  sqlc          - Generate type-safe Go from SQL queries"
	@echo "  migrate-up    - Apply all migrations"
	@echo "  migrate-down  - Rollback last migration"
