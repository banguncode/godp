# godp — Go DigitalPersona Biometric Engine

[![Language: Go](https://img.shields.io/badge/Language-Go%201.22+-00ADD8.svg?logo=go)](https://go.dev/)
[![Platform: Linux x64](https://img.shields.io/badge/Platform-Linux%20x64-blue.svg)](https://ubuntu.com/)
[![SDK: Crossmatch / DigitalPersona](https://img.shields.io/badge/SDK-U.are.U%203.x-orange.svg)](https://www.crossmatch.com)
[![Docker: Ready](https://img.shields.io/badge/Docker-Ready-2496ED.svg?logo=docker)](https://www.docker.com/)
[![License: Proprietary](https://img.shields.io/badge/License-Proprietary-lightgrey.svg)](#license)

`godp` is an enterprise-grade, high-performance **server-side biometric processing microservice** written in Go, wrapping the official DigitalPersona / Crossmatch U.are.U C SDK (`libdpfj`).

Designed specifically for headless server deployments, `godp` receives raw fingerprint scans or candidate templates from client workstations (Windows, Linux, Web, or Mobile) and handles **multi-sample enrollment, ISO/ANSI minutiae extraction, 1:1 verification, and 1:N biometric identification** at cloud scale.

---

## Architecture Overview

```
[ PC Klien Windows ]
  │  • USB Fingerprint Scanner (DigitalPersona U.are.U 4500/5100)
  │  • Capture via DigitalPersona Web SDK / Windows Service
  │  • User scans finger 4 times during enrollment
  │
  ▼ HTTP POST (Kirim candidate samples base64)
[ Backend Utama (Laravel / Node.js / Go) ]
  │
  ▼ HTTP POST http://godp-server:5050/api/v1/fmd/enroll
[ godp Biometric Engine (Linux Server) ]
  │  • Headless Cgo Wrapper (libdpfj.so)
  │  • Multi-sample Minutiae Extraction (ISO/IEC 19794-2)
  │  • Algoritma Fusi Master Enrollment FMD
  │
  ▼ Return Master Enrolled FMD Base64
[ Database PostgreSQL / MySQL / Oracle ]
```

---

## Key Capabilities

* **Server-Side Enrollment Engine:** Aggregates multiple raw samples (3–4 scans) into one unified master enrollment FMD template using `dpfj_create_enrollment_fmd`.
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

## Directory Structure

Following the standard Go project layout:

```
godp/
├── cmd/
│   └── godp-server/
│       └── main.go                  # Main entrypoint with signal handling
├── internal/
│   ├── api/
│   │   ├── dto/                     # Request and Response transfer objects
│   │   ├── handler/                 # FMD and Health handlers
│   │   ├── middleware/              # Security, CORS, Logging, Recovery
│   │   └── router/                  # HTTP route registry
│   └── config/                      # Environment-based configuration
├── pkg/
│   └── dp/                          # Pure Go wrapper around libdpfj C SDK
│       ├── dpfj.go                  # Core biometric algorithms
│       ├── dpfj_test.go             # Unit test suite
│       ├── types.go                 # Enums, constants, and data structures
│       └── errors.go                # Strongly-typed error mapping
├── lib/
│   ├── include/                     # Vendor C headers (dpfj.h, dpfpdd.h, etc.)
│   └── x64/                         # Vendor x64 shared objects (.so)
├── docs/
│   └── API.md                       # Comprehensive REST API specifications
├── Makefile                         # Automation commands
└── Dockerfile                       # Multi-stage production container
```

---

## Quick Start

### 1. Build and Run Locally

Prerequisites: Linux x86_64, GCC, Go 1.22+.

```bash
# Clone and enter directory
cd /home/rum/godp

# Run automated tests
make test

# Build and run server (default port 5050)
make run
```

### 2. Verify Server Status

```bash
curl -s http://localhost:5050/health | python3 -m json.tool
```

### 3. Test Biometric Verification (1:1 Compare)

```bash
curl -s -X POST http://localhost:5050/api/v1/fmd/compare \
  -H "Content-Type: application/json" \
  -d '{
    "probe_fmd_base64": "<BASE64_FMD_1>",
    "enrolled_fmd_base64": "<BASE64_FMD_2>",
    "format": "ISO_19794_2_2005"
  }'
```

---

## Configuration

`godp` is configured via environment variables:

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

## Docker Deployment

Build and run in an isolated Linux container:

```bash
# Build production image
make docker-build

# Run container on port 5050
make docker-run
```

---

## Complete API Reference

Detailed endpoint payloads, sample responses, and status codes are documented in [docs/API.md](docs/API.md).

---

## License

DigitalPersona U.are.U shared libraries (`lib/x64`) and headers (`lib/include`) are proprietary software copyright © Crossmatch / DigitalPersona, Inc.
The Go wrapper implementation is proprietary and confidential.
