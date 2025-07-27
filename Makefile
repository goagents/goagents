# GoAgents Makefile

# Variables
BINARY_NAME=goagents
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse HEAD)
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-w -s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build targets
.PHONY: all build clean test coverage lint fmt vet deps help
.PHONY: build-linux build-darwin build-windows build-all
.PHONY: docker docker-build docker-push
.PHONY: deploy deploy-k8s deploy-docker

# Default target
all: clean deps test build

# Build the binary
build:
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) cmd/goagents/main.go

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 cmd/goagents/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 cmd/goagents/main.go

# Build for macOS
build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 cmd/goagents/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 cmd/goagents/main.go

# Build for Windows
build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe cmd/goagents/main.go

# Build for all platforms
build-all: build-linux build-darwin build-windows

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*

# Run tests
test:
	$(GOTEST) -v -race -timeout 30s ./...

# Run tests with coverage
coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run vet
vet:
	$(GOCMD) vet ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Security scan
security:
	gosec ./...

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) -f deployments/Dockerfile .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# Push Docker image
docker-push:
	docker push $(BINARY_NAME):$(VERSION)
	docker push $(BINARY_NAME):latest

# Docker compose up
docker-up:
	docker-compose -f deployments/docker-compose.yaml up -d

# Docker compose down
docker-down:
	docker-compose -f deployments/docker-compose.yaml down

# Deploy to Kubernetes
deploy-k8s:
	kubectl apply -f deployments/kubernetes.yaml

# Remove Kubernetes deployment
undeploy-k8s:
	kubectl delete -f deployments/kubernetes.yaml

# Install locally
install: build
	cp $(BINARY_NAME) /usr/local/bin/

# Uninstall
uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)

# Generate documentation
docs:
	@echo "Generating documentation..."
	$(GOCMD) doc -all ./... > docs/api.md

# Run the server with example config
run:
	./$(BINARY_NAME) run --config examples/config.yaml

# Run with cluster deployment
run-cluster:
	./$(BINARY_NAME) run --config examples/config.yaml --cluster examples/customer-support.yaml

# Check status
status:
	./$(BINARY_NAME) status

# Deploy example cluster
deploy-example:
	./$(BINARY_NAME) deploy --cluster examples/customer-support.yaml

# Development setup
dev-setup: deps
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Pre-commit checks
pre-commit: fmt vet lint test security

# Release preparation
release-prep: clean deps pre-commit build-all
	@echo "Release $(VERSION) prepared"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-all     - Build for all platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  coverage      - Run tests with coverage"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run vet"
	@echo "  deps          - Download dependencies"
	@echo "  security      - Run security scan"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-push   - Push Docker image"
	@echo "  docker-up     - Start with Docker Compose"
	@echo "  docker-down   - Stop Docker Compose"
	@echo "  deploy-k8s    - Deploy to Kubernetes"
	@echo "  undeploy-k8s  - Remove Kubernetes deployment"
	@echo "  install       - Install binary locally"
	@echo "  uninstall     - Uninstall binary"
	@echo "  run           - Run server with example config"
	@echo "  run-cluster   - Run with cluster deployment"
	@echo "  status        - Check status"
	@echo "  deploy-example- Deploy example cluster"
	@echo "  dev-setup     - Setup development environment"
	@echo "  pre-commit    - Run pre-commit checks"
	@echo "  release-prep  - Prepare release"
	@echo "  help          - Show this help"