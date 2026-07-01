# Runtime Funcinfo Benchmark

This benchmark keeps runtime funcinfo measurements comparable across branches by
generating the same probe programs and rebuilding them with each compiler/root
pair in one run.

It covers:

- hot runtime metadata calls: `Caller`, `Callers`, `CallersFrames`,
  `FuncForPC`, and `Func.FileLine`.
- deep stacks through direct calls, interface calls, and closures.
- many packages and methods, generated from configurable package/method counts.
- cold first-use runtime metadata paths, including lazy table initialization.
- a stdlib-heavy program with `encoding/json`, `text/template`, `regexp`,
  `go/parser`, `go/token`, and `net/netip` imports.

Generated modules use `example.com/llgo-bench/...` import paths. This is
intentional: LLGo does not enable caller-frame tracking for stdlib-shaped paths
without a dot, and that would benchmark the fallback path instead of normal
third-party package behavior.

Example:

```sh
go run ./benchmark/runtime_funcinfo \
  -runs=11 \
  -llgo-opt=2 \
  -variant go=go \
  -variant main=llgo,/path/to/llgo-main,/path/to/llgo-main-root \
  -variant 2002=llgo,/path/to/llgo-2002,/path/to/llgo-2002-root \
  -variant 2009=llgo,/path/to/llgo-2009,/path/to/llgo-2009-root \
  -variant 2010=llgo,/path/to/llgo-2010,/path/to/llgo-2010-root
```

Add `-include-lto` to build an additional `+lto` variant for every LLGo
compiler. LLGo builds use `-O2` by default; pass `-llgo-opt=` to omit the
optimization flag. Add `-scales=6x6,12x12,24x24` to generate separate
`multipkg_*` and `cold_*` scenarios for several package/function counts in one
run. Output is written to `benchmark/runtime_funcinfo/out` by default:

- `summary.md`: markdown performance and size tables.
- `results.json`: raw build and run data.
- `work/`: generated probe modules.
- `bin/`: generated executables.

Performance cells are `best/trimmed avg` from process-level runs. The trimmed
average drops one minimum and one maximum when at least three runs are present.
`-iters` is a base iteration count: `hot` uses the full count, `deep` uses a
quarter, and `multipkg`/`stdlib` use one twentieth because each operation does
substantially more work.

`multipkg.FuncForPCMany` and `multipkg.FileLineMany` are batch metrics over all
generated target functions (`-packages * -methods`, 144 targets with the default
settings), not single-lookup timings.

`cold.First*` metrics are single measurements from a fresh process and include
lazy runtime initialization that has not already happened in that process.
`cold.WarmFuncForPCMany` and `cold.WarmFileLineMany` use the same batch target
count as `multipkg`.
