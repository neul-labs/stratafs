# Deployment Guide

This guide covers various deployment options for AgentFS, from simple desktop installations to production-ready containerized deployments.

## Quick Installation

### Linux & macOS
```bash
curl -fsSL https://raw.githubusercontent.com/yourusername/agentfs/main/scripts/install.sh | bash
```

### Windows (PowerShell)
```powershell
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/yourusername/agentfs/main/scripts/install.ps1" -OutFile "install.ps1"; .\install.ps1
```

### Manual Installation
1. Download the appropriate binary from [releases](https://github.com/yourusername/agentfs/releases)
2. Extract the archive
3. Copy `agentfs` to a directory in your PATH
4. Run `agentfs config init` to initialize

## Docker Deployment

### Simple Docker Run
```bash
# Basic deployment
docker run -d \
  --name agentfs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/data:/app/data \
  -v agentfs_config:/app/.agentfs \
  ghcr.io/yourusername/agentfs:latest

# With custom configuration
docker run -d \
  --name agentfs \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/config.json:/app/config.json:ro \
  -v agentfs_config:/app/.agentfs \
  ghcr.io/yourusername/agentfs:latest
```

### Docker Compose

#### Basic Setup
```bash
# Clone repository
git clone https://github.com/yourusername/agentfs.git
cd agentfs

# Start AgentFS
docker-compose up -d

# View logs
docker-compose logs -f agentfs

# Stop
docker-compose down
```

#### Production Setup with Reverse Proxy
```bash
# Start with Traefik reverse proxy
docker-compose --profile production up -d

# Access via:
# - API: http://agentfs.local (add to /etc/hosts)
# - MCP: http://agentfs-mcp.local
# - Traefik Dashboard: http://localhost:8090
```

#### With Monitoring
```bash
# Start with monitoring stack
docker-compose --profile monitoring up -d

# Access:
# - Grafana: http://localhost:3000 (admin/admin)
# - Prometheus: http://localhost:9090
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AGENTFS_GLOBAL_DIR` | `/app/.agentfs` | Global configuration directory |
| `AGENTFS_API_PORT` | `8080` | REST API port |
| `AGENTFS_MCP_PORT` | `8081` | MCP server port |
| `AGENTFS_WORKER_COUNT` | `4` | Number of worker processes |
| `AGENTFS_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `AGENTFS_CHUNK_SIZE` | `1000` | Default chunk size |
| `AGENTFS_CHUNK_STRATEGY` | `simple` | Default chunking strategy |

## Kubernetes Deployment

### Prerequisites
- Kubernetes cluster (1.20+)
- Helm 3.x
- Persistent volume support

### Helm Installation

#### Add Helm Repository
```bash
helm repo add agentfs https://yourusername.github.io/agentfs
helm repo update
```

#### Quick Install
```bash
# Install with default values
helm install agentfs agentfs/agentfs

# Install with custom values
helm install agentfs agentfs/agentfs -f values.yaml

# Install from local chart
helm install agentfs ./helm/agentfs
```

#### Configuration Examples

**Basic Configuration** (`values.yaml`):
```yaml
replicaCount: 1

image:
  repository: ghcr.io/yourusername/agentfs
  tag: "latest"

service:
  type: ClusterIP

persistence:
  enabled: true
  size: 20Gi

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 500m
    memory: 1Gi
```

**Production Configuration**:
```yaml
replicaCount: 2

image:
  repository: ghcr.io/yourusername/agentfs
  tag: "v0.2.0"

service:
  type: LoadBalancer

ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: agentfs.yourdomain.com
      paths:
        - path: /
          pathType: Prefix
          port: 8080
  tls:
    - secretName: agentfs-tls
      hosts:
        - agentfs.yourdomain.com

persistence:
  enabled: true
  storageClassName: "fast-ssd"
  size: 100Gi

resources:
  limits:
    cpu: 4000m
    memory: 8Gi
  requests:
    cpu: 1000m
    memory: 2Gi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true

agentfs:
  config:
    worker_count: 8
    log_level: "warn"

  embedding:
    batch_size: 64
    max_concurrency: 8
```

#### Upgrading
```bash
# Update repository
helm repo update

# Upgrade release
helm upgrade agentfs agentfs/agentfs

# Upgrade with new values
helm upgrade agentfs agentfs/agentfs -f new-values.yaml
```

#### Uninstalling
```bash
# Uninstall release (keeps PVCs)
helm uninstall agentfs

# Delete persistent volumes
kubectl delete pvc -l app.kubernetes.io/instance=agentfs
```

### Manual Kubernetes Deployment

If you prefer not to use Helm, you can deploy using kubectl:

```bash
# Apply all manifests
kubectl apply -f k8s/

# Or apply individually
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/pvc.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

### Monitoring in Kubernetes

#### Prometheus Operator
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: agentfs
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: agentfs
  endpoints:
  - port: api
    path: /metrics
    interval: 30s
```

#### Grafana Dashboard
```bash
# Import dashboard from ConfigMap
kubectl create configmap agentfs-dashboard \
  --from-file=dashboard.json \
  -n monitoring

# Label for auto-discovery
kubectl label configmap agentfs-dashboard \
  grafana_dashboard=1 \
  -n monitoring
```

## Cloud Platform Deployment

### AWS

#### ECS with Fargate
```json
{
  "family": "agentfs",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/agentfsTaskRole",
  "containerDefinitions": [
    {
      "name": "agentfs",
      "image": "ghcr.io/yourusername/agentfs:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        },
        {
          "containerPort": 8081,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "AGENTFS_WORKER_COUNT",
          "value": "4"
        }
      ],
      "mountPoints": [
        {
          "sourceVolume": "agentfs-data",
          "containerPath": "/app/data"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/agentfs",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ],
  "volumes": [
    {
      "name": "agentfs-data",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-123456789"
      }
    }
  ]
}
```

#### EKS
```bash
# Create EKS cluster
eksctl create cluster --name agentfs-cluster --region us-west-2

# Deploy AgentFS
helm install agentfs agentfs/agentfs --set persistence.storageClassName=gp2
```

### Google Cloud Platform

#### Cloud Run
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: agentfs
  annotations:
    run.googleapis.com/ingress: all
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/cpu-throttling: "false"
    spec:
      containerConcurrency: 80
      containers:
      - image: ghcr.io/yourusername/agentfs:latest
        ports:
        - containerPort: 8080
        env:
        - name: AGENTFS_WORKER_COUNT
          value: "2"
        resources:
          limits:
            cpu: "2"
            memory: "4Gi"
        volumeMounts:
        - name: data
          mountPath: /app/data
      volumes:
      - name: data
        persistentVolumeClaim:
          claimName: agentfs-data
```

#### GKE
```bash
# Create GKE cluster
gcloud container clusters create agentfs-cluster \
  --zone us-central1-a \
  --num-nodes 3

# Deploy AgentFS
helm install agentfs agentfs/agentfs --set persistence.storageClassName=standard
```

### Azure

#### Container Instances
```yaml
apiVersion: 2019-12-01
location: eastus
name: agentfs
properties:
  containers:
  - name: agentfs
    properties:
      image: ghcr.io/yourusername/agentfs:latest
      ports:
      - port: 8080
        protocol: TCP
      - port: 8081
        protocol: TCP
      resources:
        requests:
          cpu: 1.0
          memoryInGB: 2.0
      environmentVariables:
      - name: AGENTFS_WORKER_COUNT
        value: "2"
      volumeMounts:
      - name: data
        mountPath: /app/data
  volumes:
  - name: data
    azureFile:
      shareName: agentfs-data
      storageAccountName: mystorageaccount
      storageAccountKey: "..."
  osType: Linux
  restartPolicy: Always
  ipAddress:
    type: Public
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 8081
```

#### AKS
```bash
# Create AKS cluster
az aks create \
  --resource-group myResourceGroup \
  --name agentfs-cluster \
  --node-count 3 \
  --enable-addons monitoring \
  --generate-ssh-keys

# Deploy AgentFS
helm install agentfs agentfs/agentfs --set persistence.storageClassName=default
```

## Production Considerations

### Security

#### Network Security
- Use TLS/SSL certificates for HTTPS
- Implement network policies in Kubernetes
- Use private networks/VPCs
- Configure firewall rules

#### Authentication & Authorization
- Place behind reverse proxy with auth (nginx, Traefik)
- Use API gateways (Kong, Ambassador)
- Implement rate limiting
- Monitor access logs

#### Container Security
```dockerfile
# Use non-root user
USER 1001

# Read-only root filesystem
RUN chown -R 1001:1001 /app
USER 1001
```

### Performance Tuning

#### Resource Allocation
```yaml
resources:
  limits:
    cpu: "4"
    memory: "8Gi"
  requests:
    cpu: "1"
    memory: "2Gi"
```

#### Configuration Optimization
```json
{
  "worker": {
    "count": 8,
    "batch_size": 50
  },
  "embedding": {
    "batch_size": 64,
    "max_concurrency": 8,
    "cache_results": true
  },
  "database": {
    "compression_enabled": true,
    "maintenance_interval": "12h"
  }
}
```

### Monitoring & Observability

#### Metrics Collection
- Prometheus metrics endpoint: `/metrics`
- Key metrics: request rate, response time, queue depth, memory usage
- Custom dashboards in Grafana

#### Logging
```yaml
env:
- name: AGENTFS_LOG_LEVEL
  value: "info"
- name: AGENTFS_LOG_FORMAT
  value: "json"
```

#### Health Checks
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Backup & Recovery

#### Data Backup
```bash
# Backup configuration
kubectl create backup agentfs-config \
  --include-resources configmaps,secrets \
  --selector app.kubernetes.io/name=agentfs

# Backup persistent volumes
kubectl create backup agentfs-data \
  --include-resources persistentvolumeclaims,persistentvolumes \
  --selector app.kubernetes.io/name=agentfs
```

#### Disaster Recovery
- Regular automated backups
- Cross-region replication
- Documented recovery procedures
- Regular disaster recovery testing

## Troubleshooting

### Common Issues

#### Port Conflicts
```bash
# Check port usage
netstat -tlnp | grep :8080

# Change ports via environment
export AGENTFS_API_PORT=8090
export AGENTFS_MCP_PORT=8091
```

#### Permission Issues
```bash
# Fix permissions
chown -R 1001:1001 /app/data
chmod -R 755 /app/data
```

#### Memory Issues
```bash
# Monitor memory usage
docker stats agentfs

# Increase memory limits
docker run --memory=4g ghcr.io/yourusername/agentfs:latest
```

### Debug Mode
```bash
# Enable debug logging
export AGENTFS_LOG_LEVEL=debug

# Run with verbose output
docker run -e AGENTFS_LOG_LEVEL=debug ghcr.io/yourusername/agentfs:latest
```

### Support

For deployment issues:
1. Check the [troubleshooting guide](troubleshooting.md)
2. Review [GitHub issues](https://github.com/yourusername/agentfs/issues)
3. Join our [Discord community](https://discord.gg/agentfs)
4. Contact support: support@agentfs.dev