# Dockerfile for AgentFS

# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git build-base

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with FTS5 support
RUN go build -o build/agentfs -tags "fts5" cmd/agentfs/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh agentfs

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/build/agentfs ./agentfs

# Change ownership to non-root user
RUN chown -R agentfs:agentfs /app

# Switch to non-root user
USER agentfs

# Expose ports
EXPOSE 8080 8081

# Default command
ENTRYPOINT ["./agentfs"]