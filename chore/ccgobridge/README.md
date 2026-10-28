# Explicit source ABI adaptation

This experimental tool adapts **selected dependencies**, not ordinary Go
`unsafe` conversions. It leaves LLGo's two-word function representation and
the module cache unchanged. It is separate from Plan9 instruction translation.

Build the tool with the host Go compiler, then select the target and packages
explicitly. For the reported SQLite case, the application's module graph must
pin `modernc.org/sqlite v1.58.0` and `modernc.org/libc v1.75.7`:

```sh
go build -o ./_out/ccgobridge ./chore/ccgobridge
GOWORK=off GOOS=linux GOARCH=amd64 CGO_ENABLED=0 ./_out/ccgobridge \
  -dir /path/to/application -out ./_out/sqlite-adapted \
  modernc.org/libc modernc.org/sqlite/lib modernc.org/sqlite
```

The output directory must be new. It contains only files selected for this
target, a private `build.mod`/`build.sum`, and `adaptations.json`. Build from the
application directory with the printed **absolute** `-modfile` path, using Go
and then LLGo/LLVM 22. Do not reuse the snapshot for another target, source
version or build-tag selection. Do not treat adaptation counts as test passes.

The tool validates the exporter bodies and their typed call sites. Only
top-level function declarations acquire static, typed boxes. Lazy `sync.Once`
initialization preserves pointer identity without introducing initialization
cycles for self-references. Function variables, method values, closures and
escaped exporter helpers are rejected. Consumers load the full function value
from its box, not from a one-word temporary and not from native machine code.

Each static function may have only one exporting package in the selected
graph. Repeated exports within that package share a box, including imported
functions. If two selected packages export the same declaration, preflight
rejects the graph before writing a snapshot: separate package-local boxes
would change function-pointer equality. This does not certify identity with
unadapted exporters outside the selected graph.

The audited musl callback adapters write full typed functions into their TLS
slots. Typed roots retain environments until the corresponding TLS frame is
freed; reuse and `Close` release them. Unsupported slot/lifetime shapes fail.
ABI0 code addresses and C function pointers are not function-box addresses.

Tests:

```sh
go test ./internal/ccgobridge ./chore/ccgobridge -count=1
go test -race ./internal/ccgobridge ./chore/ccgobridge -count=1
```

Real acceptance additionally requires final linking and execution: SQLite
queries, rollback, UDF callbacks with distinct captured environments and GC,
and the ABI0 `Yqsort`/TLS-slot path. Compilation or unit tests alone do not
establish those results. This tool does not make the original, unadapted
CCGo representation portable.

## purego v0.9.1

The separate, hash-checked purego profile is Linux/amd64 with cgo enabled only.
It replaces the Go-runtime trampoline backend with a qualified C ABI transport
and the dedicated LLGo callback bridge. Original package initializers still
run. It does not enable `runtime.cgocall` or `crosscall2` globally.

```sh
GOWORK=off GOOS=linux GOARCH=amd64 CGO_ENABLED=1 ./_out/ccgobridge \
  -dir /path/to/application -out ./_out/purego-adapted \
  -profile purego-v0.9.1 -runtime ./runtime github.com/ebitengine/purego
```

Build this stage with LLGo, using the printed `-modfile`. Compare its runtime
results against Go running the original application, not the LLGo-only staged
backend. Other versions, platforms, fakecgo, and changed backend source hashes
are rejected. The emitted `environment.json` records the target and profile.
See [the native and callback contracts](../../internal/puregobridge/README.md).

## End-to-end regression gate

On Linux/amd64 with Go 1.27, a compiler built against LLVM 22, and the matching
LLVM 22 tools selected, run:

```sh
LLGO=/path/to/built/llgo bash dev/test_source_abi.sh
```

Optional case arguments are `sqlite`, `purego`, `turso`, `libccallback`, and
`crc32`. Each has its own pinned module graph under `testdata/`. The gate builds,
links and executes original Go and adapted LLGo programs; CCGo cases also run
with Go after adaptation. CRC32 uses its original, unadapted sources and requires
the SSE4.1/SSE4.2/PCLMUL CPU paths; it does not certify native AVX512 execution.
Turso executes SQL rather than relying on a lazy `sql.Open`.

The libc callback fixture explicitly bridges a `uintptr` callback signature to
a pointer-typed assembly implementation. This does not qualify arbitrary
`uintptr` dereferences or interchange physical Go ABI0 and C function pointers.
It tests each compiler's matching translated producer/consumer pair.

Reports retain per-case failures, tool configuration, compiler/binary hashes and
build/run logs under `_out/source-abi.*`. Environment-reported LLVM versions are
tool configuration, not proof of an arbitrary compiler binary's linked LLVM;
CI builds its own compiler with the selected LLVM 22 toolchain. Temporary source
snapshots and executables are removed on exit. Do not edit a running snapshot.
The separate callback and `reflectzeroabi` fixtures run in CI as well; neither
translation counts nor a successful source rewrite substitutes for execution.
