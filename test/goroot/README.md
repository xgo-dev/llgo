# GOROOT Runner

This package runs selected upstream `GOROOT/test` cases against `llgo` without
copying the upstream source files into this repository.

Current scope:

- Source of truth is an external `GOROOT`
- `legacy` mode runs `// run` cases only
- CI mode runs `// run`, `// runoutput`, and `// buildrun` cases
- Cases are compared as `go run <file>` vs `llgo run <file>`
- Upstream files stay read-only; the runner uses a temporary symlinked work tree

Basic usage:

```bash
go test ./test/goroot -count=1 -args \
  -goroot "$(go env GOROOT)" \
  -dirs . \
  -case '^helloworld\.go$'
```

Multiple toolchains:

```bash
bash ./dev/test_goroot.sh /path/to/go1.23 /path/to/go1.24 -- -dirs . -case '^helloworld\.go$'
```

Useful flags:

- `-goroot`: upstream Go toolchain root to read tests from
- `-go`: baseline `go` binary; defaults to `<goroot>/bin/go`
- `-llgo`: existing `llgo` binary to use; otherwise one is built from the current checkout
- `-dirs`: comma-separated `GOROOT/test` subdirectories to scan
- `-case`: regexp filter on the relative case path
- `-directive-mode`: case discovery mode: `legacy`, `ci`, or `runlike`
- `-limit`: stop after N matching cases
- `-shard-index`: 0-based shard index used to partition matching cases
- `-shard-total`: total number of shards used to partition matching cases
- `-keepwork`: keep the temporary symlink work tree for debugging
- `-run-timeout`: timeout for each child `go`/`llgo` process; `0` disables the timeout
- `-xfail`: xfail YAML file, relative to repo root by default
