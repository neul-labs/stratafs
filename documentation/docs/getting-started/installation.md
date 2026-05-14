# Installation

StrataFS ships in several forms. Pick whichever fits your environment — they all install the same `stratafs` binary and use the same config layout (`~/.stratafs/`).

## Package managers

=== "npm"

    The npm package downloads the matching prebuilt binary on install.

    ```bash
    npm install -g stratafs
    ```

=== "PyPI"

    The Python package wraps the binary and adds a console entry point.

    ```bash
    pip install stratafs
    ```

=== "Homebrew"

    ```bash
    brew tap neul-labs/stratafs
    brew install stratafs
    ```

## Shell installer

The shell installer detects your OS/arch, downloads the latest release, and places `stratafs` on your `PATH`.

=== "macOS / Linux"

    ```bash
    curl -fsSL https://raw.githubusercontent.com/neul-labs/stratafs/main/scripts/install.sh | bash
    ```

=== "Windows (PowerShell)"

    ```powershell
    Invoke-WebRequest `
      -Uri "https://raw.githubusercontent.com/neul-labs/stratafs/main/scripts/install.ps1" `
      -OutFile install.ps1
    .\install.ps1
    ```

## Native installers

For end-user desktop installs you can grab a platform-native installer from the [releases page](https://github.com/neul-labs/stratafs/releases):

| Platform | Artifact | Notes |
| --- | --- | --- |
| Windows | `StrataFS-Setup.exe` | NSIS installer. Optional Windows Service + tray app. |
| macOS | `StrataFS-{version}.pkg` | Includes signed `.app` bundle and LaunchAgent. |
| Ubuntu / Debian | `stratafs_{version}_amd64.deb` | Installs `stratafs` plus a `systemd --user` unit. |
| Any Linux | `StrataFS-{version}-x86_64.AppImage` | Portable, no install needed. |

## Docker

The official image bundles the ONNX Runtime, so no host setup is required.

```bash
docker run -d \
  --name stratafs \
  -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/data:/app/data \
  -v stratafs_config:/app/.stratafs \
  ghcr.io/neul-labs/stratafs:latest
```

See [Deployment → Docker](../deployment/docker.md) for Compose, environment variables, and production hardening.

## Build from source

```bash
git clone https://github.com/neul-labs/stratafs.git
cd stratafs
make build
```

Requires Go 1.24+, a C compiler (for the SQLite extension), and the ONNX Runtime. See [Contributing → Building](../contributing/building.md) for the long version.

## Verify

```bash
stratafs --version
stratafs config init
stratafs serve
```

`stratafs serve` binds the REST API to `:8080` and the MCP server to `:8081`. Hit the health endpoint to confirm everything is alive:

```bash
curl http://localhost:8080/health
```

Continue with the [Quickstart](quickstart.md) to add your first source and run a search.
