# Performance

Numbers measured on consumer hardware (M-series Mac, NVMe SSD, 16 GB RAM) with the default BGE Base EN v1.5 model.

## Headline numbers

| Metric | Typical value |
| --- | --- |
| Indexing throughput | ~50–100 files/sec |
| Search latency | < 100 ms for 10 k files |
| Embedding model | BGE Base EN v1.5 (768d) |
| Disk overhead | ~1.5–2× original text size (with compression) |
| Memory baseline | ~200 MB + model cache |

## What slows indexing

In rough order of impact:

1. **Embedding model**. BGE Base takes ~5× longer per chunk than BGE Small. If you're CPU-bound, switch to `bge-small-en-v1.5` — quality drop is small, throughput gain is large.
2. **File size**. PDF and DOCX parsing scales with page count, not file size. A 50 MB PDF can take seconds; a 50 MB CSV is faster.
3. **Worker count**. `worker.count = 4` is a reasonable default. On a high-core-count machine, bump it to `cpu_count - 1`.
4. **Cloud latency**. Remote sources spend most of their time on `LIST` and `GET` calls. Tighten `filters.include_patterns` and lengthen `worker.scan_interval`.
5. **Storage IOPS**. SQLite is sensitive to write latency. Run on SSD if possible.

## What slows search

1. **Result count**. Asking for `limit=100` is meaningfully slower than `limit=10`. Page if you need more.
2. **Database size**. FTS5 BM25 scales sub-linearly; vector search is `O(k log n)` with an HNSW-ish backing. Above ~500 k chunks per source, expect single-source queries in the 100–300 ms range.
3. **Embedding generation for the query**. Every vector / hybrid query embeds the query string. With BGE Base this is ~10–20 ms.
4. **Cross-source queries**. The engine runs one query per source in parallel and merges. More sources → more parallel queries → eventually you exhaust workers.

## Tuning knobs

For indexing-heavy workloads:

```json
{
  "worker": {
    "count": 8,
    "batch_size": 20
  },
  "embedding": {
    "model": "bge-small-en-v1.5",
    "dimension": 384,
    "performance": { "batch_size": 64, "max_concurrency": 4 }
  }
}
```

For search-heavy workloads:

```json
{
  "worker": {
    "count": 2
  },
  "database": {
    "compression_enabled": true,
    "maintenance_interval": "6h"
  }
}
```

Chunk size / overlap are baked into the queue processor today; pick a smaller embedding model for the biggest wins on indexing throughput, and let `database.compression_enabled` stay on for the search-side disk savings.

## Memory

| Component | Footprint |
| --- | --- |
| Base process | ~200 MB |
| BGE Base model loaded | ~500 MB |
| BGE Small model loaded | ~250 MB |
| Per-source SQLite cache | ~50–100 MB |
| Streaming chunker | ~10 MB regardless of file size |

The embedding model is the dominant cost. If you have multiple sources, the model is shared across all of them.

## Disk

A rough rule of thumb: budget **2×** your raw text size for the indexed footprint, before compression. With compression enabled (the default), expect ~1.5×.

The vector index dominates the disk cost. A 768-dimension embedding is 3 KB per chunk (float32). Switching to BGE Small (384d) cuts that in half.

## Profiling

The daemon writes pipeline progress to stdout. Pair it with `/queue/stats` to see whether the queue is building up:

```bash
stratafs serve 2>&1 | tee stratafs.log &
watch -n 5 'curl -s http://localhost:8080/queue/stats | jq'
```

`processing_jobs` should stay above zero; a growing `pending_jobs` count means the embedder is the bottleneck — either drop to BGE Small, lift `worker.count`, or add CPU.

## Benchmarks

Reproducible benchmarks live under `research/benchmarks/` in the repo and are exercised in CI. They cover:

- **Indexing throughput** across file types.
- **Search quality** (precision/recall against curated queries).
- **Latency** distributions for hybrid / FTS / vector modes.
- **Ablation** studies for individual ranking signals.

See `research/benchmarks/README.md` for how to run them locally.
