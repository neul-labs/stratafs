# AgentFS Docker Guide

Complete guide for deploying AgentFS using Docker containers, including single container deployment, Docker Compose, and Kubernetes.

## Overview

AgentFS provides production-ready Docker images with the following features:

- **Multi-stage builds** for optimized image size
- **ONNX Runtime integration** for AI/ML capabilities
- **Health checks** and monitoring support
- **Security hardening** with non-root user
- **Multi-architecture support** (amd64, arm64)

## Quick Start

### Run with Docker

```bash
# Pull and run the latest image
docker run -d \
  --name agentfs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  agentfs:latest

# Check if it's running
docker ps
curl http://localhost:8080/health
```

### Run with Docker Compose

```bash
# Create docker-compose.yml
cat > docker-compose.yml << EOF
version: '3.8'
services:
  agentfs:
    image: agentfs:latest
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - agentfs-data:/data
volumes:
  agentfs-data:
EOF

# Start the service
docker-compose up -d
```

## Docker Image

### Image Details

- **Base Image**: Alpine Linux (minimal, secure)
- **Size**: ~200MB (with ONNX Runtime)
- **Architecture**: linux/amd64, linux/arm64
- **User**: Non-root (agentfs:1001)
- **Ports**: 8080 (API), 8081 (MCP)

### Image Tags

| Tag | Description |
|-----|-------------|
| `latest` | Latest stable release |
| `v0.2.0` | Specific version |
| `main` | Latest development build |
| `alpine` | Alpine-based image |

### Health Checks

The image includes built-in health checks:

```bash
# Check container health
docker inspect agentfs | jq '.[0].State.Health'

# Manual health check
docker exec agentfs curl -f http://localhost:8080/health
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTFS_CONFIG_DIR` | `/app/config` | Configuration directory |
| `AGENTFS_DATA_DIR` | `/data` | Data storage directory |
| `AGENTFS_API_PORT` | `8080` | API server port |
| `AGENTFS_MCP_PORT` | `8081` | MCP server port |
| `AGENTFS_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `AGENTFS_HOST` | `0.0.0.0` | Server bind address |

### Volume Mounts

#### Required Volumes

```bash
# Data volume (persistent storage)
-v agentfs-data:/data

# Configuration volume (optional)
-v /host/config:/app/config

# Logs volume (optional)
-v /host/logs:/app/logs
```

#### Volume Locations

| Container Path | Purpose | Recommended Host Mount |
|----------------|---------|------------------------|
| `/data` | Database and indexed content | Named volume `agentfs-data` |
| `/app/config` | Configuration files | `/etc/agentfs` or named volume |
| `/app/logs` | Application logs | `/var/log/agentfs` |
| `/app/cache` | Embedding model cache | Named volume `agentfs-cache` |

### Port Mapping

```bash
# Default ports
-p 8080:8080  # API server (web interface)
-p 8081:8081  # MCP server (AI agent communication)

# Custom ports
-p 9080:8080  # Map API to port 9080
-p 9081:8081  # Map MCP to port 9081
```

## Deployment Examples

### Basic Development Setup

```bash
docker run -d \
  --name agentfs-dev \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/data:/data \
  -v $(pwd)/config:/app/config \
  -e AGENTFS_LOG_LEVEL=debug \
  agentfs:latest
```

### Production Single Container

```bash
# Create volumes
docker volume create agentfs-data
docker volume create agentfs-config
docker volume create agentfs-logs

# Run container
docker run -d \
  --name agentfs \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  -v agentfs-config:/app/config \
  -v agentfs-logs:/app/logs \
  -e AGENTFS_LOG_LEVEL=info \
  --memory=2g \
  --cpus=2.0 \
  --health-cmd="curl -f http://localhost:8080/health || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  agentfs:latest
```

### Behind Reverse Proxy

```bash
# Run AgentFS on internal network
docker network create agentfs-network

docker run -d \
  --name agentfs \
  --network agentfs-network \
  --restart unless-stopped \
  -v agentfs-data:/data \
  -e AGENTFS_HOST=0.0.0.0 \
  agentfs:latest

# Run nginx proxy
docker run -d \
  --name agentfs-proxy \
  --network agentfs-network \
  -p 80:80 \
  -p 443:443 \
  -v $(pwd)/nginx.conf:/etc/nginx/nginx.conf \
  nginx:alpine
```

### With External Database

```bash
# Run PostgreSQL
docker run -d \
  --name agentfs-db \
  --network agentfs-network \
  -e POSTGRES_DB=agentfs \
  -e POSTGRES_USER=agentfs \
  -e POSTGRES_PASSWORD=secure_password \
  -v agentfs-db:/var/lib/postgresql/data \
  postgres:15-alpine

# Run AgentFS with external DB
docker run -d \
  --name agentfs \
  --network agentfs-network \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  -e DATABASE_URL=postgres://agentfs:secure_password@agentfs-db:5432/agentfs \
  agentfs:latest
```

## Docker Compose

### Basic Setup

```yaml
version: '3.8'

services:
  agentfs:
    image: agentfs:latest
    container_name: agentfs
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - agentfs-data:/data
      - agentfs-config:/app/config
    environment:
      - AGENTFS_LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

volumes:
  agentfs-data:
    driver: local
  agentfs-config:
    driver: local
```

### Production Setup with Monitoring

```yaml
version: '3.8'

services:
  agentfs:
    image: agentfs:latest
    container_name: agentfs
    restart: unless-stopped
    networks:
      - agentfs-network
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - agentfs-data:/data
      - agentfs-config:/app/config
      - agentfs-logs:/app/logs
    environment:
      - AGENTFS_LOG_LEVEL=info
      - AGENTFS_HOST=0.0.0.0
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          memory: 2G
          cpus: '2.0'
        reservations:
          memory: 1G
          cpus: '1.0'
    depends_on:
      - prometheus
      - grafana

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    restart: unless-stopped
    networks:
      - agentfs-network
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    restart: unless-stopped
    networks:
      - agentfs-network
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin

  traefik:
    image: traefik:v2.10
    container_name: traefik
    restart: unless-stopped
    networks:
      - agentfs-network
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"  # Traefik dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik.yml:/traefik.yml:ro
      - traefik-certs:/certs
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.dashboard.rule=Host(`traefik.localhost`)"

networks:
  agentfs-network:
    driver: bridge

volumes:
  agentfs-data:
    driver: local
  agentfs-config:
    driver: local
  agentfs-logs:
    driver: local
  prometheus-data:
    driver: local
  grafana-data:
    driver: local
  traefik-certs:
    driver: local
```

### Development Environment

```yaml
version: '3.8'

services:
  agentfs:
    build: .
    container_name: agentfs-dev
    restart: "no"
    ports:
      - "8080:8080"
      - "8081:8081"
    volumes:
      - ./:/app/src
      - agentfs-dev-data:/data
      - ./config:/app/config
    environment:
      - AGENTFS_LOG_LEVEL=debug
      - AGENTFS_ENV=development
    command: ["./agentfs", "--config-dir=/app/config", "--data-dir=/data", "--log-level=debug"]

volumes:
  agentfs-dev-data:
    driver: local
```

## Building Custom Images

### Build from Source

```bash
# Clone repository
git clone https://github.com/your-repo/agentfs.git
cd agentfs

# Build image
docker build -t agentfs:custom .

# Build with specific version
docker build -t agentfs:0.2.0 --build-arg VERSION=0.2.0 .

# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 -t agentfs:multi-arch .
```

### Custom Dockerfile

```dockerfile
# Start from the official image
FROM agentfs:latest

# Add custom configuration
COPY custom-config.json /app/config/config.json

# Add custom scripts
COPY scripts/ /app/scripts/
RUN chmod +x /app/scripts/*.sh

# Set custom environment
ENV AGENTFS_ENV=production
ENV AGENTFS_LOG_LEVEL=warn

# Custom entrypoint
COPY docker-entrypoint.sh /usr/local/bin/
ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["agentfs"]
```

### Multi-stage Build

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o agentfs ./cmd/agentfs

# Runtime stage
FROM agentfs:base

COPY --from=builder /app/agentfs /usr/local/bin/agentfs
COPY --from=builder /app/config /app/config

USER agentfs
EXPOSE 8080 8081

CMD ["agentfs"]
```

## Kubernetes Integration

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentfs
  namespace: agentfs
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agentfs
  template:
    metadata:
      labels:
        app: agentfs
    spec:
      containers:
      - name: agentfs
        image: agentfs:0.2.0
        ports:
        - containerPort: 8080
          name: api
        - containerPort: 8081
          name: mcp
        env:
        - name: AGENTFS_CONFIG_DIR
          value: "/app/config"
        - name: AGENTFS_LOG_LEVEL
          value: "info"
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /app/config
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          limits:
            memory: "2Gi"
            cpu: "1000m"
          requests:
            memory: "1Gi"
            cpu: "500m"
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: agentfs-data
      - name: config
        configMap:
          name: agentfs-config
```

### Service and Ingress

```yaml
apiVersion: v1
kind: Service
metadata:
  name: agentfs-service
  namespace: agentfs
spec:
  selector:
    app: agentfs
  ports:
  - name: api
    port: 8080
    targetPort: 8080
  - name: mcp
    port: 8081
    targetPort: 8081
  type: ClusterIP

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: agentfs-ingress
  namespace: agentfs
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: agentfs.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: agentfs-service
            port:
              number: 8080
```

## Monitoring and Logging

### Prometheus Metrics

AgentFS exposes metrics at `/metrics`:

```bash
# Scrape metrics
curl http://localhost:8080/metrics

# Example metrics
agentfs_requests_total{method="GET",path="/api/search"} 1024
agentfs_response_duration_seconds_bucket{le="0.1"} 512
agentfs_active_connections 42
```

### Log Aggregation

#### Using Fluentd

```yaml
version: '3.8'

services:
  agentfs:
    image: agentfs:latest
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: agentfs

  fluentd:
    image: fluent/fluentd:latest
    ports:
      - "24224:24224"
    volumes:
      - ./fluent.conf:/fluentd/etc/fluent.conf
```

#### Using ELK Stack

```yaml
version: '3.8'

services:
  agentfs:
    image: agentfs:latest
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.8.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false

  logstash:
    image: docker.elastic.co/logstash/logstash:8.8.0
    volumes:
      - ./logstash.conf:/usr/share/logstash/pipeline/logstash.conf

  kibana:
    image: docker.elastic.co/kibana/kibana:8.8.0
    ports:
      - "5601:5601"
```

## Security

### Security Best Practices

1. **Non-root User**: Containers run as user `agentfs` (UID 1001)
2. **Read-only Root**: Use `--read-only` flag where possible
3. **Resource Limits**: Set memory and CPU limits
4. **Network Security**: Use custom networks, not default bridge
5. **Secrets Management**: Use Docker secrets or external secret managers

### Secure Configuration

```bash
# Run with security options
docker run -d \
  --name agentfs \
  --user 1001:1001 \
  --read-only \
  --tmpfs /tmp \
  --tmpfs /var/cache \
  --memory=2g \
  --cpus=2.0 \
  --cap-drop=ALL \
  --cap-add=NET_BIND_SERVICE \
  --security-opt=no-new-privileges:true \
  -p 8080:8080 \
  -p 8081:8081 \
  -v agentfs-data:/data \
  agentfs:latest
```

### TLS/SSL Configuration

```yaml
version: '3.8'

services:
  agentfs:
    image: agentfs:latest
    volumes:
      - ./certs:/app/certs:ro
    environment:
      - AGENTFS_TLS_CERT=/app/certs/server.crt
      - AGENTFS_TLS_KEY=/app/certs/server.key
      - AGENTFS_TLS_ENABLED=true
    ports:
      - "8443:8443"  # HTTPS port
```

## Troubleshooting

### Common Issues

#### Container Won't Start

```bash
# Check logs
docker logs agentfs

# Check configuration
docker exec agentfs cat /app/config/config.json

# Validate health
docker exec agentfs curl -f http://localhost:8080/health
```

#### Performance Issues

```bash
# Check resource usage
docker stats agentfs

# Check container limits
docker inspect agentfs | jq '.[0].HostConfig.Memory'

# Check disk usage
docker exec agentfs df -h
```

#### Network Issues

```bash
# Check port binding
docker port agentfs

# Test connectivity
docker exec agentfs nc -z localhost 8080

# Check network configuration
docker network ls
docker network inspect bridge
```

### Debugging

#### Enable Debug Mode

```bash
# Environment variable
docker run -e AGENTFS_LOG_LEVEL=debug agentfs:latest

# Or in docker-compose.yml
environment:
  - AGENTFS_LOG_LEVEL=debug
```

#### Interactive Debugging

```bash
# Start with shell
docker run -it --entrypoint=/bin/sh agentfs:latest

# Exec into running container
docker exec -it agentfs /bin/sh

# Check processes
docker exec agentfs ps aux
```

## Performance Optimization

### Resource Tuning

```yaml
deploy:
  resources:
    limits:
      memory: 4G      # Adjust based on data size
      cpus: '4.0'     # Adjust based on workload
    reservations:
      memory: 2G
      cpus: '2.0'
```

### Volume Optimization

```bash
# Use local SSD for data
docker volume create agentfs-data --driver local \
  --opt type=none \
  --opt o=bind \
  --opt device=/mnt/ssd/agentfs

# Use tmpfs for cache
docker run -v agentfs-data:/data --tmpfs /tmp:size=1g agentfs:latest
```

### Networking

```bash
# Use host networking for performance
docker run --network host agentfs:latest

# Or custom bridge with optimized settings
docker network create --driver bridge \
  --opt com.docker.network.bridge.name=agentfs0 \
  --opt com.docker.network.driver.mtu=9000 \
  agentfs-network
```

This comprehensive Docker guide covers all aspects of deploying and managing AgentFS in containerized environments.