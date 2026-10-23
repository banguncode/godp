# godp — Go DigitalPersona Biometric Engine

[![Language: Go](https://img.shields.io/badge/Language-Go%201.22+-00ADD8.svg?logo=go)](https://go.dev/)
[![Platform: Linux x64](https://img.shields.io/badge/Platform-Linux%20x64-blue.svg)](https://ubuntu.com/)
[![SDK: Crossmatch / DigitalPersona](https://img.shields.io/badge/SDK-U.are.U%203.x-orange.svg)](https://www.crossmatch.com)
[![Docker: Ready](https://img.shields.io/badge/Docker-Ready-2496ED.svg?logo=docker)](https://www.docker.com/)
[![Systemd: Service](https://img.shields.io/badge/Systemd-Production--Ready-red.svg)](systemd/godp.service)
[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-lightgrey.svg)](#license)

`godp` is an enterprise-grade, high-performance **server-side biometric processing microservice** written in Go, wrapping the official DigitalPersona / Crossmatch U.are.U C SDK (`libdpfj`).

Designed specifically for headless server deployments, `godp` receives raw fingerprint scans or candidate templates from client workstations (Windows, Linux, Web, or Mobile) and handles **multi-sample enrollment, ISO/ANSI minutiae extraction, 1:1 verification, and 1:N biometric identification** at cloud scale.

---

## Table of Contents

1. [Key Capabilities](#key-capabilities)
2. [Quick Start (Local Development)](#quick-start-local-development)
3. [Deployment Options](#deployment-options)
   - [Option A: Systemd Service (Recommended for Linux Server / VM)](#option-a-systemd-service-recommended-for-linux-server--vm)
   - [Option B: Docker & Docker Compose (Container Deployment)](#option-b-docker--docker-compose-container-deployment)
4. [Configuration Reference](#configuration-reference)
5. [API Reference Summary](#api-reference-summary)
6. [License](#license)

---

## Key Capabilities

* **Server-Side Enrollment Engine:** Fuses 2 or more raw captures (3–4 recommended) of the same finger into one DigitalPersona registration template (`DP_REG_FEATURES`) using `dpfj_start_enrollment` / `dpfj_add_to_enrollment` / `dpfj_create_enrollment_fmd`. Returns a JPEG preview of the last capture alongside the template.
* **Zero System Dependencies:** Pre-compiled DigitalPersona libraries (`lib/x64`) and C headers (`lib/include`) are bundled directly within the repository; embedded `$ORIGIN` runtime paths allow immediate execution without manual `LD_LIBRARY_PATH`.
* **Standard Compliant:** Native support for international biometric interchange formats:
  - **ISO/IEC 19794-2:2005**
  - **ANSI INCITS 378-2004**
* **High-Throughput Verification (1:1 & 1:N):** Sub-millisecond matching using hardware-accelerated minutiae comparators with configurable False Accept Rate (FAR) thresholds.
* **NIST NFIQ Quality Assessment:** Evaluates fingerprint clarity (score 1 to 5) before enrolling.
* **Production-Grade Microservice:**
  - Built-in graceful shutdown (`SIGINT`/`SIGTERM`)
  - Structured JSON logging (`log/slog`)
  - Configurable request size limiting (`MaxBytesReader`)
  - Optional API Key authentication (`X-API-Key` or `Bearer`)
  - CORS pre-flight handling

---

## Quick Start (Local Development)

Prerequisites: Linux x86_64, GCC, Go 1.22+.

```bash
# Clone repository and enter directory
git clone https://github.com/banguncode/godp.git
cd godp

# 1. Run automated test suite with race detector
make test

# 2. Build and run server (default port 5050)
make run
```

Test the health probe:
```bash
curl -s http://localhost:5050/health | python3 -m json.tool
```

---

## Deployment Options

`godp` can be deployed either natively as a **Systemd background service** or containerized via **Docker / Docker Compose**.

---

### Option A: Systemd Service (Recommended for Linux Server / VM)

Running as a native systemd daemon offers the lowest possible latency and minimal memory footprint (~0.25 MB).

#### Automated Installation (Using Makefile)

```bash
# Compile and install service automatically
make install-systemd
```

#### Manual Step-by-Step Installation

1. **Build the production binary:**
   ```bash
   make build
   ```

2. **Deploy binary and shared libraries to `/opt/godp`:**
   ```bash
   sudo mkdir -p /opt/godp/bin /opt/godp/lib
   sudo cp bin/godp-server /opt/godp/bin/
   sudo cp -r lib/x64 /opt/godp/lib/
   ```

3. **Create unprivileged system user:**
   ```bash
   sudo useradd --system --no-create-home --shell /usr/sbin/nologin godp
   sudo chown -R godp:godp /opt/godp
   sudo chmod -R 755 /opt/godp
   ```

4. **Install and activate the systemd unit:**
   ```bash
   sudo cp systemd/godp.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable godp.service
   sudo systemctl start godp.service
   ```

5. **Verify service status and stream live logs:**
   ```bash
   # Check service status
   sudo systemctl status godp.service

   # Stream live JSON structured logs
   sudo journalctl -u godp.service -f
   ```

#### Managing the Service

```bash
sudo systemctl restart godp.service   # Restart
sudo systemctl stop godp.service      # Stop
sudo systemctl disable godp.service   # Disable autostart
```

---

### Option B: Docker & Docker Compose (Container Deployment)

Deploying via Docker isolates the application and bundled SDK dependencies inside a container.

#### 1. Using Docker Compose (Quickest)

Start the service in detached mode:
```bash
docker compose up -d --build
```

Check logs:
```bash
docker compose logs -f
```

Stop the service:
```bash
docker compose down
```

#### 2. Using Standalone Docker CLI

Build and run container:
```bash
# Build multi-stage image
make docker-build

# Run container exposing port 5050
make docker-run
```

Or execute directly:
```bash
docker build -t godp:latest .
docker run -d --name godp-app -p 5050:5050 --restart unless-stopped godp:latest
```

---

## Configuration Reference

`godp` is configured via environment variables (or lines in `godp.service` / `docker-compose.yml`):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `5050` | HTTP service listening port |
| `HOST` | `0.0.0.0` | Network interface binding |
| `API_KEY` | *(empty)* | Optional API key required in `X-API-Key` or `Authorization: Bearer` |
| `MAX_REQUEST_SIZE_MB` | `15` | Maximum incoming request body limit |
| `CORS_ALLOW_ORIGIN` | `*` | Allowed CORS origins |
| `READ_TIMEOUT` | `15s` | Maximum duration for reading the entire request |
| `WRITE_TIMEOUT` | `30s` | Maximum duration before timing out writes |

---

## API Reference Summary

All endpoints follow the uniform `{ "success": bool, "data"/"error": {...} }` schema:

| Endpoint | Method | Description |
|---|:---:|---|
| `/health` | `GET` | Health check & resource metrics |
| `/api/v1/fmd/extract` | `POST` | Extract ISO/ANSI FMD from raw 8bpp grayscale image bytes |
| `/api/v1/fmd/enroll` | `POST` | **Enrollment**: Fuses 2+ raw captures into 1 `DP_REG_FEATURES` template + JPEG preview of the last capture |
| `/api/v1/fmd/compare` | `POST` | **1:1 Verification**: Compare probe FMD vs enrolled FMD |
| `/api/v1/fmd/identify` | `POST` | **1:N Search**: Search probe FMD across candidate list |
| `/api/v1/fmd/convert` | `POST` | Format conversion (ANSI 378 ⇄ ISO 19794-2) |
| `/api/v1/quality/nfiq` | `POST` | NIST NFIQ fingerprint quality assessment (1–5) |

> **Enrollment note:** `/api/v1/fmd/enroll` takes raw captures (DigitalPersona WebSDK capture JSON, or plain base64 8bpp pixels with `width`/`height`), not PNG/JPEG images or FMDs. ISO/ANSI templates cannot be fused. The output is a DigitalPersona proprietary template, so it is not interchangeable with ISO/ANSI FMDs.

Detailed request payloads, schemas, and curl examples are documented in [docs/API.md](docs/API.md).

---

## License

DigitalPersona U.are.U shared libraries (`lib/x64`) and headers (`lib/include`) are proprietary software copyright © Crossmatch / DigitalPersona, Inc.
The Go wrapper implementation is proprietary and confidential.
