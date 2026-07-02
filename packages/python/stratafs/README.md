# StrataFS

**StrataFS — a semantic filesystem for AI-era search.** Point it at your directories (local **or** cloud) and StrataFS parses your files into semantic chunks, generates vector embeddings, and exposes everything through a hybrid full-text + semantic search engine. It speaks the [Model Context Protocol](https://modelcontextprotocol.io), so any MCP-aware agent can use your filesystem as a structured knowledge resource. No SaaS. No lock-in. Read-only by design.

[![PyPI](https://img.shields.io/pypi/v/stratafs.svg?logo=pypi&label=pypi)](https://pypi.org/project/stratafs/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/neul-labs/stratafs/blob/main/LICENSE)
[![Documentation](https://img.shields.io/badge/docs-docs.neullabs.com-blue)](https://docs.neullabs.com/stratafs)

**[Website](https://stratafs.neullabs.com) · [Documentation](https://docs.neullabs.com/stratafs) · [GitHub](https://github.com/neul-labs/stratafs)**

This package is the Python distribution of StrataFS. It installs the `stratafs` command and downloads the correct native binary for your platform on first use.

## Why StrataFS?

Stop `grep`-ing. Ask **natural-language questions** across your entire codebase, configs, and docs at once — StrataFS finds related content even when filenames don't match. It runs offline (no API keys, no telemetry), stores everything as plain SQLite on disk, and drops into any agent loop over MCP.

## Install

```bash
pip install stratafs
```

The native binary is downloaded from GitHub releases automatically on first use.

## Usage

```bash
stratafs config init      # writes ~/.stratafs/config.json
stratafs serve            # REST on :8080, MCP on :8081
stratafs search "where do we handle JWT refresh?"
```

Query the same index over the REST API:

```python
import requests

r = requests.get("http://localhost:8080/search",
                 params={"q": "feature flag rollout"})
for hit in r.json()["results"]:
    print(hit["file_path"], hit["relevance_score"])
```

## Links

- **Website**: [stratafs.neullabs.com](https://stratafs.neullabs.com)
- **Documentation**: [docs.neullabs.com/stratafs](https://docs.neullabs.com/stratafs)
- **Repository**: [github.com/neul-labs/stratafs](https://github.com/neul-labs/stratafs)
- **Issues**: [github.com/neul-labs/stratafs/issues](https://github.com/neul-labs/stratafs/issues)

## Part of the Neul Labs toolchain

StrataFS is part of the Neul Labs command-line & filesystem toolchain:

| Project | What it does |
|---------|--------------|
| [stout](https://github.com/neul-labs/stout) | A drop-in replacement for the Homebrew CLI that's 10-100x faster. |
| [recurl](https://github.com/neul-labs/recurl) | curl that just works — drop-in replacement with automatic anti-bot bypass. |
| [rewget](https://github.com/neul-labs/rewget) | wget, but it works everywhere. |

Explore the full toolchain at [neullabs.com](https://www.neullabs.com).

## License

MIT
