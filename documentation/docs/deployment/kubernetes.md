# Kubernetes

A reference manifest for running StrataFS in Kubernetes. The image is the same one used for [Docker](docker.md) deployments.

## Quick deploy

A self-contained `Deployment` + `Service` + `PersistentVolumeClaim`:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: stratafs-state
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 20Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: stratafs
spec:
  replicas: 1
  selector:
    matchLabels: { app: stratafs }
  strategy:
    type: Recreate          # single-writer SQLite
  template:
    metadata:
      labels: { app: stratafs }
    spec:
      containers:
      - name: stratafs
        image: ghcr.io/neul-labs/stratafs:latest
        ports:
        - containerPort: 8080
          name: rest
        - containerPort: 8081
          name: mcp
        env:
        - name: STRATAFS_LOG_LEVEL
          value: info
        readinessProbe:
          httpGet: { path: /health, port: rest }
          periodSeconds: 5
        livenessProbe:
          httpGet: { path: /health, port: rest }
          periodSeconds: 30
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "1.5Gi"
        volumeMounts:
        - { name: state, mountPath: /app/.stratafs }
        - { name: data,  mountPath: /app/data }
      volumes:
      - name: state
        persistentVolumeClaim:
          claimName: stratafs-state
      - name: data
        emptyDir: {}        # or your own PVC / hostPath
---
apiVersion: v1
kind: Service
metadata:
  name: stratafs
spec:
  selector: { app: stratafs }
  ports:
  - { name: rest, port: 8080, targetPort: rest }
  - { name: mcp,  port: 8081, targetPort: mcp  }
```

Apply:

```bash
kubectl apply -f stratafs.yaml
kubectl port-forward svc/stratafs 8080:8080 8081:8081
```

## Helm

A Helm chart is published to the project's GitHub Pages:

```bash
helm repo add stratafs https://neul-labs.github.io/stratafs
helm repo update
helm install stratafs stratafs/stratafs
```

To override values:

```bash
helm install stratafs stratafs/stratafs -f values.yaml
```

The full list of values lives in the chart's `values.yaml`; the most-edited ones:

| Value | Default | Description |
| --- | --- | --- |
| `image.tag` | `latest` | Image tag. |
| `replicaCount` | `1` | Must stay `1` — SQLite is single-writer. |
| `persistence.size` | `20Gi` | PVC for `/app/.stratafs`. |
| `resources.requests.memory` | `512Mi` | — |
| `resources.limits.memory` | `1.5Gi` | — |
| `ingress.enabled` | `false` | Ingress for REST + MCP. |
| `env` | `{}` | Extra environment variables. |

## Caveats

- **Single replica.** Per-source SQLite is single-writer. Run StrataFS as a single pod and scale up the embedding worker pool with `STRATAFS_WORKER_COUNT`.
- **State is precious.** The PVC at `/app/.stratafs` holds your index. Back it up with a `VolumeSnapshot` or external sidecar — losing it means reindexing from scratch.
- **Auth.** StrataFS has no built-in auth. Front it with an ingress that does — see the [Production Checklist](production-checklist.md).
- **Logging.** Set `STRATAFS_LOG_LEVEL=info` (or `warn`) in production. `debug` is verbose and noisy at scale.
