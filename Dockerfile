# ==============================================================================
# Multi-Stage Dockerfile for godp - DigitalPersona Biometric Engine
# ==============================================================================

# ── Stage 1: Build ───────────────────────────────────────────────────────────
FROM ubuntu:22.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Go
ENV GO_VERSION=1.23.2
RUN curl -fsSL https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz | tar -xz -C /usr/local
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /build

COPY go.mod ./
COPY lib/ ./lib/
COPY pkg/ ./pkg/
COPY internal/ ./internal/
COPY cmd/ ./cmd/

# Build static-linked binary with embedded rpath for bundled libraries
RUN mkdir -p bin && \
    go build -ldflags="-s -w" -o bin/godp-server ./cmd/godp-server

# ── Stage 2: Runtime ──────────────────────────────────────────────────────────
FROM ubuntu:22.04 AS runtime

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libusb-1.0-0 \
    && rm -rf /var/lib/apt/lists/*

# Create unprivileged service user
RUN groupadd -g 1001 godp && useradd -u 1001 -g godp -m -s /bin/bash godp

WORKDIR /app

# Copy binary and proprietary libraries
COPY --from=builder /build/bin/godp-server /app/godp-server
COPY --from=builder /build/lib/x64/ /app/lib/x64/

# Configure linker cache for bundled libraries
RUN echo "/app/lib/x64" > /etc/ld.so.conf.d/godp.conf && ldconfig

USER godp:godp

ENV PORT=5050
ENV HOST=0.0.0.0
EXPOSE 5050

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:5050/health || exit 1

ENTRYPOINT ["/app/godp-server"]
