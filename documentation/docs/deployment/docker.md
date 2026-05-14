# Docker

The official image bundles the ONNX Runtime, ships a non-root user, and works on `linux/amd64` and `linux/arm64`.

## Image details

| Property | Value |
| --- | --- |
| Base | Alpine Linux |
| Size | ~200 MB (with ONNX Runtime) |
| User | `stratafs` (uid 1001) |
| Ports | `8080` (REST), `8081` (MCP) |
| Registry | `ghcr.io/neul-labs/stratafs` |

| Tag | Notes |
| --- | --- |
| `latest` | Latest stable release. |
| `v0.2.0` | Pinned version. |
| `main` | Latest build from `main`. |

## Quick start

```bash
docker run -d \
  --name stratafs \
  -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/data:/app/data \
  -v stratafs_config:/app/.stratafs \
  ghcr.io/neul-labs/stratafs:latest
```

Then:

```bash
docker ps
curl http://localhost:8080/health
```

## Docker Compose

A reference `docker-compose.yml` ships in the repo:

```bash
git clone https://github.com/neul-labs/stratafs.git
cd stratafs

docker-compose up -d
docker-compose logs -f stratafs
```

### Profiles

The Compose file defines two optional profiles:

```bash
# Reverse proxy + TLS via Traefik
docker-compose --profile production up -d

# Prometheus + Grafana
docker-compose --profile monitoring up -d
```

The monitoring profile starts Grafana on `:3000` (admin/admin) and Prometheus on `:9090`. The production profile fronts the API and MCP server with Traefik and exposes a dashboard on `:8090`.

## Environment variables

Anything in [Configuration → Environment variables](../user-guide/configuration.md#environment-variables) works under Docker. The most relevant defaults:

| Variable | Default in image |
| --- | --- |
| `STRATAFS_GLOBAL_DIR` | `/app/.stratafs` |
| `STRATAFS_API_PORT` | `8080` |
| `STRATAFS_MCP_PORT` | `8081` |
| `STRATAFS_WORKER_COUNT` | `4` |
| `STRATAFS_LOG_LEVEL` | `info` |

## Volumes

| Mount | Purpose |
| --- | --- |
| `/app/data` | Where your indexed source content lives. Mount your data here. |
| `/app/.stratafs` | StrataFS state — config, queue DB, per-source databases, embedding model cache. **Persist this.** |
| `/app/config.json` (optional, read-only) | Bind-mount a config from the host. |

## Health checks

The image declares a Docker `HEALTHCHECK` that hits `/health` every 30 s. Inspect:

```bash
docker inspect stratafs | jq '.[0].State.Health'
```

## Resource limits

A reasonable starting point for a single-tenant deployment:

```bash
docker run -d \
  --name stratafs \
  --memory=1g --cpus=2 \
  -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/data:/app/data \
  -v stratafs_config:/app/.stratafs \
  ghcr.io/neul-labs/stratafs:latest
```

Embedding model load is the memory peak — budget ~500 MB for BGE Base, ~250 MB for BGE Small.

## Building locally

```bash
docker build -t stratafs:dev .
docker run --rm -p 8080:8080 stratafs:dev
```

The Dockerfile is a multi-stage build: a Go builder image produces the binary, then a minimal Alpine stage assembles the runtime layer with the ONNX Runtime tarball.
