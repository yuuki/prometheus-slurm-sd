# Makefile for prometheus-slurm-sd

# Environment variables
APPNAME := prometheus-slurm-sd
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Go commands
GO := go
GOTEST := $(GO) test
GOBUILD := $(GO) build

# Build settings
BINDIR := bin
BINARY := $(BINDIR)/$(APPNAME)

# Files and directories
GOFILES := $(shell find . -type f -name "*.go" -not -path "./vendor/*")
GOPACKAGES := $(shell go list ./... | grep -v /vendor/)

# Color settings
BLUE := \033[0;34m
NC := \033[0m  # No Color

.PHONY: all build test clean run fmt lint help vet

all: build test

$(BINDIR):
	@mkdir -p $(BINDIR)

# Build the application
build: $(BINDIR)
	@echo "${BLUE}Building $(APPNAME)...${NC}"
	$(GOBUILD) $(LDFLAGS) -o $(BINARY)

# Run all tests
test:
	@echo "${BLUE}Running tests...${NC}"
	$(GOTEST) -v ./...

# Run tests with coverage report
test-coverage:
	@echo "${BLUE}Running tests with coverage...${NC}"
	$(GOTEST) -v -cover ./...

# Clean build artifacts
clean:
	@echo "${BLUE}Cleaning...${NC}"
	@rm -rf $(BINDIR)
	@go clean -cache

# Format code
fmt:
	@echo "${BLUE}Formatting code...${NC}"
	@gofmt -s -w $(GOFILES)

# Run linter
lint:
	@echo "${BLUE}Linting code...${NC}"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed"; \
		exit 1; \
	fi

# Run the application
run: build
	@echo "${BLUE}Running $(APPNAME)...${NC}"
	@$(BINARY)

# Verify code
vet:
	@echo "${BLUE}Vetting code...${NC}"
	@$(GO) vet ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  ${BLUE}build${NC}          - Build the application"
	@echo "  ${BLUE}test${NC}           - Run tests"
	@echo "  ${BLUE}test-coverage${NC}  - Run tests with coverage report"
	@echo "  ${BLUE}clean${NC}          - Remove build artifacts"
	@echo "  ${BLUE}fmt${NC}            - Format code using gofmt"
	@echo "  ${BLUE}lint${NC}           - Run linter"
	@echo "  ${BLUE}run${NC}            - Build and run the application"
	@echo "  ${BLUE}vet${NC}            - Run go vet"
	@echo "  ${BLUE}help${NC}           - Show this help"
