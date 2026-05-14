# Building

Practical detail for getting a working binary out of a checkout.

## Hard requirements

- Go 1.24+
- A C compiler (`gcc`, `clang`, or `mingw-w64` on Windows)
- SQLite headers (usually bundled with the C toolchain)
- ONNX Runtime — see below

## The `fts5` build tag

Every build must pass `-tags fts5`. The Makefile does this for you; if you `go build` by hand, do it explicitly:

```bash
go build -tags fts5 -o build/stratafs ./cmd/stratafs
```

## ONNX Runtime

The embedding pipeline calls into the ONNX Runtime via CGO. For local development:

```bash
make fetch-onnx
make build
```

`fetch-onnx` downloads the runtime for your host OS / arch into `build/onnx/<os>-<arch>/`. To use a system install instead:

```bash
export ONNX_PATH=/path/to/onnxruntime/lib
make build
```

At run time the embedding library needs the runtime on its search path. Setting one of these works on macOS / Linux:

```bash
LD_LIBRARY_PATH=$(pwd)/build/onnx/darwin-arm64/lib ./build/stratafs serve
# or:
DYLD_LIBRARY_PATH=$(pwd)/build/onnx/darwin-arm64/lib ./build/stratafs serve
```

The release builds bundle the runtime alongside the binary so this is invisible to end users.

## Cross-compilation

```bash
make build-all
```

Produces binaries for every supported `GOOS/GOARCH` under `build/<os>-<arch>/`. Override what you build with:

```bash
ONNX_VERSION=1.17.0 TARGET_OS=linux TARGET_ARCH=amd64 make build-all
```

## Release builds

```bash
make release
# or:
VERSION=0.3.0 ONNX_VERSION=1.17.0 make release
```

Produces archives under `build/release/` containing the binary plus the matching ONNX Runtime libraries. CI runs the same target for tagged commits.

## Desktop app (Wails)

```bash
cd desktop/stratafs-ui
wails build
```

Output binaries land in `desktop/stratafs-ui/build/bin/`. The build script picks the right Wails target per platform — see `desktop/stratafs-ui/build.sh` for the flags.

## Docker

```bash
docker build -t stratafs:dev .
```

The Dockerfile is multi-stage: a Go builder produces the binary, then a small Alpine runtime layer adds the ONNX libraries and a non-root user.

## Common build failures

**`undefined reference to sqlite3_fts5_*`**
: The `fts5` tag is missing. Re-run with `-tags fts5`.

**`ld: library not found for -lonnxruntime`**
: ONNX Runtime isn't on the linker path. `make fetch-onnx` or set `ONNX_PATH`.

**`sqlite-vec: build constraints exclude all Go files`**
: You're cross-compiling without CGO. The vector extension needs CGO; build natively or use a Linux builder image.

**`signal: killed` during `wails build`**
: The frontend build ran out of memory. Bump Node memory: `NODE_OPTIONS=--max-old-space-size=4096 wails build`.
