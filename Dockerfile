# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    sqlite-dev \
    curl \
    tar \
    wget

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
    curl -L https://github.com/microsoft/onnxruntime/releases/download/v1.16.3/onnxruntime-linux-x64-1.16.3.tgz | \
    tar -xz -C /opt/onnxruntime --strip-components=1

# Set CGO flags for ONNX Runtime
ENV CGO_CFLAGS="-I/opt/onnxruntime/include"
ENV CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime"
ENV LD_LIBRARY_PATH="/opt/onnxruntime/lib:$LD_LIBRARY_PATH"

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build \
    -tags "fts5" \
    -ldflags "-w -s" \
    -o agentfs \
    ./cmd/agentfs

# Runtime stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    sqlite \
    tzdata \
    wget \
    && update-ca-certificates

# Create non-root user
RUN addgroup -g 1001 agentfs && \
    adduser -D -s /bin/sh -u 1001 -G agentfs agentfs

# Copy ONNX Runtime libraries
COPY --from=builder /opt/onnxruntime/lib/* /usr/local/lib/

# Copy binary
COPY --from=builder /app/agentfs /usr/local/bin/agentfs

# Create directories with proper permissions
RUN mkdir -p /app/data /app/config /app/.agentfs && \
    chown -R agentfs:agentfs /app

# Set library path
ENV LD_LIBRARY_PATH="/usr/local/lib:$LD_LIBRARY_PATH"

# Switch to non-root user
USER agentfs

# Set working directory
WORKDIR /app

# Expose ports
EXPOSE 8080 8081

# Add volume for persistent data
VOLUME ["/app/data", "/app/.agentfs"]

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["agentfs"]