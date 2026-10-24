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
3. [SDK Library Installation](#sdk-library-installation)
   - [How library loading works](#how-library-loading-works)
   - [Verify the libraries](#verify-the-libraries)
   - [Option 1: Bundled libraries (default)](#option-1-bundled-libraries-default)
   - [Option 2: System-wide install (ldconfig)](#option-2-system-wide-install-ldconfig)
   - [Systemd and Docker](#systemd-and-docker)
   - [Troubleshooting](#troubleshooting)
4. [Deployment Options](#deployment-options)
   - [Option A: Systemd Service (Recommended for Linux Server / VM)](#option-a-systemd-service-recommended-for-linux-server--vm)
   - [Option B: Docker & Docker Compose (Container Deployment)](#option-b-docker--docker-compose-container-deployment)
5. [Configuration Reference](#configuration-reference)
6. [API Reference Summary](#api-reference-summary)
7. [License](#license)

---

## Key Capabilities

* **Server-Side Enrollment Engine:** Fuses 2 or more raw captures (3–4 recommended) of the same finger into one DigitalPersona registration template (`DP_REG_FEATURES`) using `dpfj_start_enrollment` / `dpfj_add_to_enrollment` / `dpfj_create_enrollment_fmd`. Returns a JPEG preview of the last capture alongside the template.
* **Bundled SDK, No Separate Installer:** Pre-compiled DigitalPersona libraries (`lib/x64`) and C headers (`lib/include`) ship inside the repository. The built binary locates them through an embedded `$ORIGIN` runtime path, so no manual `LD_LIBRARY_PATH` is needed as long as the `lib/x64` directory stays next to the binary layout described in [SDK Library Installation](#sdk-library-installation).
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

Prerequisites: Linux x86_64, GCC, Go 1.22+. The DigitalPersona SDK libraries are already included in `lib/x64`; there is nothing extra to install for local development.

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

If `make test` or `make run` fails with `error while loading shared libraries`, see [Troubleshooting](#troubleshooting).

---

## SDK Library Installation

`godp` depends on the native DigitalPersona U.are.U shared libraries (`libdpfj.so`, `libdpfpdd.so.3`, and related files). They are **bundled in `lib/x64`**, so you normally do not need to install the SDK separately. This section explains how they are found at runtime and what to do when they are not.

### How library loading works

The Linux dynamic loader must be able to find `libdpfj.so` and its dependencies when the process starts. `godp` supports three ways, in order of preference:

| Method | Used by | Notes |
|---|---|---|
| Embedded `$ORIGIN` rpath | `make build` / `make run` binary, systemd | Resolved relative to the binary's own location. No configuration needed. |
| Absolute rpath + `LD_LIBRARY_PATH` | `make test` | `go test` builds its binary in a temporary directory (`/tmp/go-build.../`), so `$ORIGIN` does not point at `lib/x64`. The Makefile passes an absolute path instead. |
| System-wide (`ldconfig`) | Any process on the host | Optional. See [Option 2](#option-2-system-wide-install-ldconfig). |

### Verify the libraries

Run these from the repository root to confirm the bundled SDK is complete:

```bash
# The required libraries should be present
ls -l lib/x64/ | grep -E 'dpfj|dpfpdd'

# No line should say "not found"
ldd lib/x64/libdpfj.so | grep -i "not found"

# After building, check the binary resolves everything
make build
ldd bin/godp-server | grep -Ei "dpfj|dpfpdd|not found"
```

Expected: `libdpfpdd.so.3 -> libdpfpdd.so.3.0.0` is present in `lib/x64`, and `ldd` reports no `not found` entries.

> **Keep symlinks intact.** The `.so` files are versioned symlinks (for example `libdpfpdd.so -> libdpfpdd.so.3 -> libdpfpdd.so.3.0.0`). When copying, use `cp -a` (or `cp -r`) rather than copying single files, otherwise the `.so.3` name the loader asks for will be missing.

### Option 1: Bundled libraries (default)

Nothing to install. Keep the directory layout so `$ORIGIN` resolves correctly:

```
godp/
├── bin/godp-server        # binary built by `make build`
└── lib/x64/               # DigitalPersona libraries
```

The same layout is used by the systemd deployment:

```
/opt/godp/
├── bin/godp-server
└── lib/x64/
```

Both `make install-systemd` and the manual steps below produce this layout.

### Option 2: System-wide install (ldconfig)

Use this when **other applications on the server** (not only `godp`) must load the SDK, or when a binary lives in a directory without a sibling `lib/x64`. Once registered with `ldconfig`, the libraries are found by every process without `LD_LIBRARY_PATH` or rpath.

**Step 0: Check for leftover copies.** An older manual install (for example in `/lib` or `/usr/lib`) can conflict with this one. The output should be empty or say "clean":

```bash
ldconfig -p | grep -E 'dpfj|dpfpdd' || echo "clean"
```

If old entries show up, find out where they came from before removing anything:

```bash
ls -l /lib/libdpf* /usr/lib/libdpf* 2>/dev/null
dpkg -S /lib/libdpfj.so.3 2>&1
```

If `dpkg -S` reports `no path found matching pattern`, the files were copied by hand and can be removed:

```bash
sudo rm -f /lib/libdpfj.so* /usr/lib/libdpfj.so* /usr/lib/libdpfpdd*
sudo ldconfig
```

If `dpkg -S` names a package, remove it with `sudo apt remove <package>` instead.

**Step 1: Install.** Run from the repository root:

```bash
# Copy libraries and headers, preserving symlinks
sudo mkdir -p /usr/local/lib/godp /usr/local/include/godp
sudo cp -a lib/x64/. /usr/local/lib/godp/
sudo cp -a lib/include/. /usr/local/include/godp/

# Register the library directory with the dynamic loader
echo "/usr/local/lib/godp" | sudo tee /etc/ld.so.conf.d/godp.conf

# Let every user (www-data, service accounts, ...) read the files
sudo chmod -R a+rX /usr/local/lib/godp /usr/local/include/godp

# Refresh the loader cache
sudo ldconfig
```

**Step 2: Verify.** First confirm the loader knows the libraries:

```bash
ldconfig -p | grep -E 'dpfj|dpfpdd'
```

Every entry should point to `/usr/local/lib/godp/...`. Then prove that a binary *outside* the repository can use them. A plain `ldd ~/godp/bin/godp-server` is not enough, because it still resolves through the `$ORIGIN` rpath to `lib/x64` first:

```bash
mkdir -p /tmp/uji/bin
cp ~/godp/bin/godp-server /tmp/uji/bin/
cd /tmp
env -u LD_LIBRARY_PATH ldd /tmp/uji/bin/godp-server | grep -Ei "dpfj|dpfpdd|not found"
rm -rf /tmp/uji
```

Expected: `libdpfj.so.3` and `libdpfpdd.so.3` resolve to `/usr/local/lib/godp/...` and nothing says `not found`.

**Step 3: Use it from other applications.**

| Application type | What to do |
|---|---|
| Prebuilt binaries and systemd services | Nothing. The loader finds the libraries through the `ldconfig` cache. |
| C / C++ | `gcc app.c -I/usr/local/include/godp -L/usr/local/lib/godp -ldpfj` |
| Go (cgo) | `CGO_CFLAGS="-I/usr/local/include/godp" CGO_LDFLAGS="-L/usr/local/lib/godp -ldpfj" go build` |
| Python (`ctypes`), PHP (FFI), etc. | Load by soname, no path needed: `ctypes.CDLL("libdpfj.so.3")` |
| Docker containers | **Not automatic**, containers are isolated from the host. Copy the libraries into the image, or mount them: `-v /usr/local/lib/godp:/usr/local/lib/godp:ro` and run `ldconfig` inside the container (or set `LD_LIBRARY_PATH=/usr/local/lib/godp`). |

> Linking with `-ldpfj` needs the unversioned symlink `libdpfj.so`. Check with `ls -l /usr/local/lib/godp/libdpfj*`. If only `libdpfj.so.3` exists, create it: `sudo ln -s libdpfj.so.3 /usr/local/lib/godp/libdpfj.so && sudo ldconfig`.

**Updating the SDK.** After replacing files in `lib/x64`, repeat Step 1 (the `cp -a` lines and `sudo ldconfig`). Existing files are overwritten and the cache is refreshed.

**Uninstalling.**

```bash
sudo rm -f /etc/ld.so.conf.d/godp.conf
sudo rm -rf /usr/local/lib/godp /usr/local/include/godp
sudo ldconfig

# Verify: nothing should be registered anymore
ldconfig -p | grep -E 'dpfj|dpfpdd' || echo "clean"
```

This does not touch `lib/x64` inside the repository or `/opt/godp` (systemd deployment).

### Systemd and Docker

**Systemd.** `make install-systemd` copies `lib/x64` to `/opt/godp/lib/x64`, next to `/opt/godp/bin/godp-server`, so the `$ORIGIN` rpath works as-is. If the service fails with `cannot open shared object file`, add this line to the `[Service]` section of `/etc/systemd/system/godp.service`, then run `sudo systemctl daemon-reload && sudo systemctl restart godp.service`:

```ini
Environment=LD_LIBRARY_PATH=/opt/godp/lib/x64
```

**Docker.** The provided `Dockerfile` copies the bundled libraries into the image, so no SDK needs to be installed on the host. If you write your own Dockerfile, make sure the final stage includes the libraries and points the loader at them:

```dockerfile
COPY lib/x64 /opt/godp/lib/x64
COPY --from=build /src/bin/godp-server /opt/godp/bin/godp-server
ENV LD_LIBRARY_PATH=/opt/godp/lib/x64
```

### Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `libdpfpdd.so.3: cannot open shared object file` when running `go test` directly | Test binary is built in `/tmp`, so `$ORIGIN` does not resolve to `lib/x64` | Use `make test`, or run `LD_LIBRARY_PATH=$(pwd)/lib/x64 go test -v -race ./pkg/dp/...` |
| Same error when running `bin/godp-server` | Binary was moved away from its `lib/x64` sibling directory | Restore the layout from [Option 1](#option-1-bundled-libraries-default), or use [Option 2](#option-2-system-wide-install-ldconfig) |
| Same error under systemd | Libraries missing from `/opt/godp/lib/x64`, or wrong ownership/permissions | Re-run `make install-systemd`, or set `Environment=LD_LIBRARY_PATH=/opt/godp/lib/x64` |
| `ldd` shows `libxxx => not found` for a library that is not in `lib/x64` | A system package is missing (for example `libstdc++`, `libusb`) | Install it with `apt install <package>` and re-check with `ldd` |
| `.so.3` missing but `.so.3.0.0` exists | Symlinks were lost when copying | Re-copy with `cp -a`, or recreate: `ln -s libdpfpdd.so.3.0.0 libdpfpdd.so.3` |
| `ldconfig -p` lists `dpfj`/`dpfpdd` under `/lib` or `/usr/lib` | Leftover manual copy from an earlier install, may shadow or duplicate the new one | Follow Step 0 in [Option 2](#option-2-system-wide-install-ldconfig) |
| Works as your user, fails as `www-data` or a service user | Library files are not world-readable | `sudo chmod -R a+rX /usr/local/lib/godp && sudo ldconfig` |
| Works on the host, fails inside a container | Containers do not see the host's `ldconfig` cache | Copy the libraries into the image or mount `/usr/local/lib/godp` |
| `wrong ELF class` or `cannot execute binary file` | Architecture mismatch (not x86_64) | The bundled libraries are Linux x86_64 only |

Useful diagnostics:

```bash
# See exactly which libraries a binary resolves
ldd bin/godp-server

# See which paths the loader searches
LD_DEBUG=libs bin/godp-server 2>&1 | head -40
```

---

## Deployment Options

`godp` can be deployed either natively as a **Systemd background service** or containerized via **Docker / Docker Compose**. In both cases the SDK libraries must travel with the application; see [SDK Library Installation](#sdk-library-installation).

---

### Option A: Systemd Service (Recommended for Linux Server / VM)

Running as a native systemd daemon offers the lowest possible latency and minimal memory footprint (~0.25 MB).

#### Automated Installation (Using Makefile)

```bash
# Compile and install service automatically
# (also deploys lib/x64 to /opt/godp/lib/x64)
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
   sudo cp -a lib/x64 /opt/godp/lib/
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

5. **Verify the libraries resolve, then check service status and stream live logs:**
   ```bash
   # No "not found" lines expected
   ldd /opt/godp/bin/godp-server | grep -i "not found"

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

Deploying via Docker isolates the application and bundled SDK dependencies inside a container. No SDK installation is required on the host.

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
| `LD_LIBRARY_PATH` | *(unset)* | Optional. Extra search path for the SDK libraries; only needed if the `$ORIGIN` layout cannot be used |

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
