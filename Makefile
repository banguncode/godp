# ==============================================================================
# Makefile for godp - DigitalPersona Biometric Engine in Go
# ==============================================================================

APP_NAME     = godp-server
MAIN_PATH    = ./cmd/godp-server
BIN_DIR      = bin
BINARY       = $(BIN_DIR)/$(APP_NAME)

GO          ?= go
PORT        ?= 5050
DOCKER_IMAGE = godp:latest

.PHONY: all build run test clean lint docker-build docker-run help

all: test build

build: ## Compile godp server binary with embedded rpath
	@echo "==> Building $(BINARY)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags="-s -w" -o $(BINARY) $(MAIN_PATH)
	@echo "==> Build complete: $(BINARY)"

run: build ## Build and run the server locally
	@echo "==> Running $(BINARY) on port $(PORT)..."
	PORT=$(PORT) ./$(BINARY)

test: ## Execute unit and integration tests
	@echo "==> Running test suite..."
	$(GO) test -v -race ./pkg/dp/...

clean: ## Remove build artifacts
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "==> Clean complete."

docker-build: ## Build container image
	@echo "==> Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run container on port 5050
	@echo "==> Running $(DOCKER_IMAGE)..."
	docker run --rm -p $(PORT):5050 --name godp-app $(DOCKER_IMAGE)

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
