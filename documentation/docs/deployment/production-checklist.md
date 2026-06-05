# Production Checklist

StrataFS is designed for local-first use. Putting it on the network requires a few extra moves.

## Network

- [ ] Put the API behind a reverse proxy (nginx, Caddy, Traefik).
- [ ] Terminate TLS at the proxy. StrataFS speaks plain HTTP.
- [ ] Add basic auth, mTLS, OAuth2-proxy, or your provider's IAM in front.
- [ ] Restrict the MCP port (`:8081`) to your agent network. It is intended to be trusted.
- [ ] Don't expose StrataFS directly to the open internet.

## Storage

- [ ] Mount `/app/.stratafs` (or `~/.stratafs/` for bare-metal) on a persistent volume.
- [ ] Schedule snapshots (volume snapshot, `restic`, `rsync` to S3). The state directory is the source of truth — losing it means reindexing from scratch.
- [ ] Size the volume for **2× the size of your raw data**. Embeddings + indexes add roughly 1.5–2×, compression buys back ~40%.
- [ ] Use SSDs. SQLite query latency depends heavily on storage IOPS.

## Resources

- [ ] Budget ~500 MB RAM for BGE Base, ~250 MB for BGE Small, plus ~200 MB base process overhead.
- [ ] CPU: 1 core per ~25 files/sec of expected indexing throughput.
- [ ] Disk: see "Storage" above.

## Configuration

- [ ] Pin a specific image tag (e.g. `ghcr.io/neul-labs/stratafs:v0.2.0`) — don't deploy `latest`.
- [ ] Set `worker.scan_interval` (or `STRATAFS_SCAN_INTERVAL`) to a value that respects your cloud-storage rate limits.
- [ ] Bound `filters.max_file_size` to keep one giant file from stalling the queue.
- [ ] Tune `STRATAFS_WORKERS` to match the available CPU; default is 4.

## Secrets

- [ ] Cloud credentials (S3 access keys, GCS service-account JSON, Azure account keys) live in `config.json` today. Mount them via a secret-management tool (Kubernetes `Secret`, Docker `secrets`, Vault Agent) rather than baking them into the image.
- [ ] Don't commit a populated `config.json` to git.
- [ ] Rotate cloud credentials on a schedule.

## Observability

- [ ] Poll `/health` from your monitoring system. Alert on non-200.
- [ ] Scrape `/queue/stats` — alert if `pending_jobs` grows unboundedly (indicates worker starvation).
- [ ] Aggregate stdout/stderr logs (Loki, CloudWatch, etc.). Per-source health endpoints are tracked on the [Roadmap](../contributing/roadmap.md).

## Backups and disaster recovery

- [ ] Daily snapshot of the state volume.
- [ ] Test restore quarterly. Bring up a parallel pod against the snapshot and run a representative search.
- [ ] Document where source content lives. If you lose the volume *and* lose the source content, you have nothing to reindex from.

## Upgrades

- [ ] Tag images and pin them in your deploy manifests.
- [ ] Test upgrades on a staging copy of the state volume first — schema migrations run on start.
- [ ] Keep the previous image tag around for at least one release cycle.

## Limits

StrataFS is **not** built for:

- Multi-tenant deployments with isolation per user. There is no RBAC layer (yet — see the [Roadmap](../contributing/roadmap.md)).
- Write workloads against source storage. The architecture is strictly read-only.
- Synchronous indexing-on-write semantics. Indexing is asynchronous; queries see eventual consistency.
- Horizontal scaling. SQLite is single-writer per source.

If you need any of the above, open an issue describing your workload before standing up a production deployment — there are known patterns that work, but they aren't the default.
