# Build stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libsqlite3-dev \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Install ONNX Runtime
RUN mkdir -p /opt/onnxruntime && \
    curl -fsSL https://github.com/microsoft/onnxruntime/releases/download/v1.16.3/onnxruntime-linux-x64-1.16.3.tgz | \
    tar -xz -C /opt/onnxruntime --strip-components=1

# Build the application
ENV CGO_CFLAGS="-I/opt/onnxruntime/include"
ENV CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime"
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "fts5" \
    -ldflags "-w -s" \
    -o stratafs \
    ./cmd/stratafs

# Runtime stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
    tzdata \
    wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Create non-root user
RUN groupadd -g 1001 stratafs && \
    useradd -m -s /bin/sh -u 1001 -g stratafs stratafs

# Copy ONNX Runtime libraries
COPY --from=builder /opt/onnxruntime/lib/* /usr/local/lib/

# Copy binary
COPY --from=builder /app/stratafs /usr/local/bin/stratafs

# Update library cache
RUN ldconfig

# Create directories with proper permissions
RUN mkdir -p /app/data /app/config /app/.stratafs && \
    chown -R stratafs:stratafs /app

# Set library path
ENV LD_LIBRARY_PATH="/usr/local/lib"

# Switch to non-root user
USER stratafs

# Set working directory
WORKDIR /app

# Expose ports
EXPOSE 8080 8081

# Add volume for persistent data
VOLUME ["/app/data", "/app/.stratafs"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["stratafs"]
