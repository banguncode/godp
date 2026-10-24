# ==============================================================================
# Makefile for godp - DigitalPersona Biometric Engine in Go
# ==============================================================================

APP_NAME     = godp-server
MAIN_PATH    = ./cmd/godp-server
BIN_DIR      = bin
BINARY       = $(BIN_DIR)/$(APP_NAME)
INSTALL_DIR  = /opt/godp
ROOT         = $(CURDIR)

# Auto-detect Go binary: check PATH first, then fallback to ~/.local/go/bin/go, /usr/local/go/bin/go
GO ?= $(shell which go 2>/dev/null || \
      (test -x $(HOME)/.local/go/bin/go && echo $(HOME)/.local/go/bin/go) || \
      (test -x /usr/local/go/bin/go && echo /usr/local/go/bin/go) || \
      echo go)

PORT        ?= 5050
DOCKER_IMAGE = godp:latest

.PHONY: all build run test clean docker-build docker-run compose-up compose-down install-systemd uninstall-systemd help

all: test build

build: ## Compile godp server binary with embedded rpath
	@echo "==> Building $(BINARY) using $(GO)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags="-s -w" -o $(BINARY) $(MAIN_PATH)
	@echo "==> Build complete: $(BINARY)"

run: build ## Build and run the server locally
	@echo "==> Running $(BINARY) on port $(PORT)..."
	PORT=$(PORT) ./$(BINARY)

test: ## Execute unit and integration tests
	@echo "==> Running test suite using $(GO)..."
	LD_LIBRARY_PATH=$(ROOT)/lib/x64 \
	CGO_LDFLAGS="-L$(ROOT)/lib/x64 -Wl,-rpath,$(ROOT)/lib/x64" \
	$(GO) test -v -race ./pkg/dp/...

clean: ## Remove build artifacts
	@echo "==> Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "==> Clean complete."

# ── Docker Targets ────────────────────────────────────────────────────────────

docker-build: ## Build container image
	@echo "==> Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .

docker-run: ## Run container on port 5050
	@echo "==> Running $(DOCKER_IMAGE)..."
	docker run --rm -p $(PORT):5050 --name godp-app $(DOCKER_IMAGE)

compose-up: ## Start service using Docker Compose in background
	@echo "==> Starting godp via Docker Compose..."
	docker compose up -d --build

compose-down: ## Stop Docker Compose service
	@echo "==> Stopping godp Docker Compose..."
	docker compose down

# ── Systemd Deployment Targets ────────────────────────────────────────────────

install-systemd: build ## Deploy binary to /opt/godp and install systemd service unit
	@echo "==> Installing godp to $(INSTALL_DIR)..."
	sudo mkdir -p $(INSTALL_DIR)/bin $(INSTALL_DIR)/lib
	sudo cp $(BINARY) $(INSTALL_DIR)/bin/
	sudo cp -r lib/x64 $(INSTALL_DIR)/lib/
	@if ! id "godp" >/dev/null 2>&1; then \
		echo "==> Creating godp system user..."; \
		sudo useradd --system --no-create-home --shell /usr/sbin/nologin godp; \
	fi
	sudo chown -R godp:godp $(INSTALL_DIR)
	sudo chmod -R 755 $(INSTALL_DIR)
	@echo "==> Installing systemd unit file..."
	sudo cp systemd/godp.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable godp.service
	sudo systemctl restart godp.service
	@echo "==> godp service installed and started successfully!"
	@echo "    Verify status: sudo systemctl status godp.service"

uninstall-systemd: ## Stop, disable, and remove systemd service unit
	@echo "==> Removing godp service..."
	sudo systemctl stop godp.service 2>/dev/null || true
	sudo systemctl disable godp.service 2>/dev/null || true
	sudo rm -f /etc/systemd/system/godp.service
	sudo systemctl daemon-reload
	@echo "==> godp service uninstalled."

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
