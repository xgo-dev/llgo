# Dev tooling

This directory contains scripts for running LLGo locally and inside reusable Linux dev containers.

## Prerequisites

- Docker installed and running
- Docker Compose v2 (`docker compose`, not `docker-compose`)

## 1) Start a Linux container, then run `dev/llgo.sh` / `dev/llgo_wasm.sh`

Start an interactive shell (pick one):

```bash
./dev/docker.sh amd64
./dev/docker.sh arm64
./dev/docker.sh i386
```

Notes:
- `amd64` uses the `pydeps` image target (includes extra Python demo deps like `numpy`/`torch`).
- `arm64` and `i386` use the smaller `base` target (no extra Python ML deps).

Inside the container, run tests/builds using the repo scripts:

```bash
./dev/llgo.sh test ./...
./dev/llgo.sh test ./test

# WASI/WASM (wasip1/wasm)
./dev/llgo_wasm.sh build ./...
```

Notes:
- `dev/docker.sh` starts in the same repo subdirectory you launched it from.
- `dev/llgo.sh` and `dev/llgo_wasm.sh` must be run from within `LLGO_ROOT` (the repo) and will error otherwise.

## 2) Start a Linux container, run one command, then exit

```bash
./dev/docker.sh amd64 bash -lc './dev/llgo.sh test ./test'
```

## 3) Run on the host (no container)

From anywhere inside the repo:

```bash
./dev/llgo.sh test ./test
./dev/llgo_wasm.sh build ./...
```

## 4) Run local CI (covers most checks)

```bash
./dev/local_ci.sh
```

This script creates a temporary workspace, runs formatting/build/tests, runs `llgo test`, and then runs demo checks.
You can control demo parallelism via `LLGO_DEMO_JOBS` (defaults to up to 4 jobs).

## 5) `dev/docker.sh` (composition-friendly)

`dev/docker.sh` is a thin wrapper around `docker compose`:

```bash
./dev/docker.sh <arch> [command...]
```

- `<arch>` must be `amd64`, `arm64`, or `i386`.
- If `[command...]` is omitted, it starts an interactive `bash`.
- If `[command...]` is provided, it runs that command and exits.
- You must run it from within the repo (within `LLGO_ROOT`), and it will start in the matching repo subdirectory inside the container.

## 6) Refresh test goldens

LLGo currently has two different golden-test refresh flows:

- `gentests` for directory-based golden files such as `out.ll` and `expect.txt`
- `litgen` for source-embedded `// LITTEST` FileCheck directives

### `gentests`

Run:

```bash
go run ./chore/gentests
```

Behavior:

- Refreshes `out.ll` for the built-in test suites under `cl/_testlibc`, `cl/_testlibgo`, `cl/_testrt`, `cl/_testgo`, `cl/_testpy`, and `cl/_testdata`.
- Refreshes `expect.txt` for the same directories using the existing runtime execution flow.
- Preserves the existing skip convention where `out.ll` or `expect.txt` containing only `;` means "do not refresh".
- New behavior: if a test case directory contains a non-test Go source file whose first line is exactly `// LITTEST`, `gentests` skips `llgen` for that directory and does not regenerate `out.ll` there.

Use `gentests` when the test still stores LLVM IR in `out.ll`.

### `litgen`

Run on a single marked file:

```bash
go run ./chore/litgen path/to/in.go
```

Run on a directory tree:

```bash
go run ./chore/litgen cl/_testrt/litdemo
go run ./chore/litgen cl/_testdata
```

Behavior:

- Accepts one or more paths.
- If the path is a `.go` file, it refreshes only that file. The file must start with `// LITTEST`.
- If the path is a directory, it walks that directory recursively, finds marked source files, and refreshes each marked test in place.
- Rewrites embedded `CHECK-LABEL`, `CHECK-NEXT`, `CHECK-EMPTY`, and referenced constant `CHECK-LINE` directives from the current generated IR.
- Does not update `expect.txt` and does not write `out.ll`.

Use `litgen` when the test case stores its IR expectations directly in the Go source instead of `out.ll`.

### Marker convention

Source-embedded IR checks are enabled by putting this marker on the first line of the source file:

```go
// LITTEST
```

The generated directives are consumed by the existing `littest`/FileCheck path in the compiler tests.

Example:

- [cl/_testrt/litdemo/in.go](../cl/_testrt/litdemo/in.go) is a minimal `_testrt` case that demonstrates `litgen` output.
