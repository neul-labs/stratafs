# StrataFS

[![npm](https://img.shields.io/npm/v/stratafs.svg)](https://www.npmjs.com/package/stratafs)
[![PyPI](https://img.shields.io/pypi/v/stratafs.svg)](https://pypi.org/project/stratafs/)
[![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/neul-labs/stratafs/actions/workflows/ci.yml/badge.svg)](https://github.com/neul-labs/stratafs/actions/workflows/ci.yml)

> **A semantic filesystem that transforms passive file storage into an intelligent, searchable knowledge base.**

StrataFS watches your directories (local or cloud), parses files into semantic chunks, generates vector embeddings, and exposes everything through a hybrid search engine — REST API and Model Context Protocol server included.

## Install

```bash
# npm
npm install -g stratafs

# PyPI
pip install stratafs

# Shell installer (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/neul-labs/stratafs/main/scripts/install.sh | bash
```

Then:

```bash
stratafs config init
stratafs serve                            # REST :8080, MCP :8081
stratafs search "authentication middleware"
```

## Documentation

Full documentation lives in [`documentation/`](documentation/) and is published as a website.

- **Quickstart, installation, and concepts** — [Getting Started](documentation/docs/getting-started/quickstart.md)
- **REST API and MCP integration** — [AI Integration](documentation/docs/ai-integration/mcp.md)
- **Storage backends, configuration, CLI** — [User Guide](documentation/docs/user-guide/configuration.md)
- **Deployment (Docker, systemd, Kubernetes)** — [Deployment](documentation/docs/deployment/docker.md)
- **Architecture and design decisions** — [Architecture](documentation/docs/architecture/overview.md)
- **Contributing** — [Development Guide](documentation/docs/contributing/development.md)

To preview the docs site locally:

```bash
cd documentation
pip install -r requirements.txt
mkdocs serve
```

## License

MIT — see [LICENSE](LICENSE).
