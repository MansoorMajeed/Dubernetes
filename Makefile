# Dubernetes Makefile

.PHONY: build test test-integration test-coverage test-race clean run-orchestrator run-cli

# Build commands
build: build-orchestrator build-cli

build-orchestrator:
	go build -o bin/orchestrator ./cmd/orchestrator

build-cli:
	go build -o bin/dubectl ./cmd/dubectl

# Test commands
test:
	go test ./pkg/... -v

test-integration:
	go test ./test/integration/... -v

test-coverage:
	go test ./pkg/... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

test-race:
	go test ./pkg/... -race

# Run commands
run-orchestrator:
	go run ./cmd/orchestrator

run-cli:
	go run ./cmd/dubectl

# Development
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Dependencies
deps:
	go mod tidy
	go mod download

# Format and lint
fmt:
	go fmt ./...

vet:
	go vet ./...

# All checks
check: fmt vet test

.DEFAULT_GOAL := build