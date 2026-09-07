# WebAssembly standard-library behavior and reference hosts

## W3 full package audit

`-full` discovers every test package below `test/` independently of build tags, executes one deterministic shard, continues after individual failures, and preserves JSON accounting plus per-package logs. Source exclusions and host-only suites must be classified explicitly; an unknown exclusion remains a failure rather than becoming a compatibility pass.

```sh
go run ./dev/wasmstdlib -full -profile J32-GoJS -shard 0 -shards 2 -llgo /path/to/llgo -report /tmp/full.json
```

The full audit requires GNU `timeout`. Each package has a five-minute build/run budget and a separate 60-second guest deadline, extended only for reviewed finite slow packages. LLGo package compilation is cached within the job, but `-count=1` keeps execution mandatory. The normally pattern-excluded `_stress/runtime/timer` package is named explicitly and runs with `LLGO_STRESS_PROFILE=quick`. `other-shard`, `not-run`, and interrupted `incomplete` results are never counted as passes. W3 separately runs the target-aware GOROOT corpus and browser acceptance.

Reviewed exclusions are reported as `not-applicable` with a profile-specific reason. Native-only signal, CPU-profiler and BDWGC stress suites; OS-only plugin, syscall and Windows suites; and the unsupported Go `cgo` frontend are not counted as passes. Dedicated target tests continue to cover LLGo's C ABI and host boundaries.

## W2 focused standard-library slice

This bounded W2 acceptance slice runs the complete repository test packages for `errors`, `sort`, `encoding/binary`, `fmt`, `strconv`, and `io`. It exercises error wrapping and assertion, reflection-based sorting, byte-order interfaces, structured encoding, varints, fixed- and native-width integer boundaries, formatting and scanning interfaces, readers and writers, and pipe goroutine/timer coordination.

No test-name filter or blanket skip is used. The driver clears inherited `GOFLAGS` and sets `GOENV=off`, so command-line or saved Go configuration cannot silently narrow the suite. Explicit process environment such as `GOPROXY` remains available.

| Path | Compiler and execution contract |
| --- | --- |
| J32-GoJS | LLGo Memory32 with Go-compatible `js/wasm` source/API and the Emscripten-based JavaScript adapter, Node |
| J32-Emscripten | LLGo Memory32 with the Emscripten JavaScript provider, Node |
| J64-Emscripten | LLGo Memory64 with the Emscripten JavaScript provider, Node |
| W32-WASI | LLGo Memory32 with WASI Preview 1, Wasmtime |
| GoJS-reference | Official Go compiler and the selected GOROOT's `go_js_wasm_exec` |
| GoWASI-reference | Official Go compiler and the selected GOROOT's `go_wasip1_wasm_exec`, Wasmtime |

The reference rows execute Go compiler output, not LLGo output. They establish the expected source behavior but do not substitute for any of the four LLGo paths. CI uses the repository-selected Go version and records it in every report.

From the repository root:

```sh
go test ./dev/wasmstdlib
go run ./dev/wasmstdlib -profile J64-Emscripten -llgo /path/to/llgo -report /tmp/j64.json
go run ./dev/wasmstdlib -profile GoJS-reference -report /tmp/go-js.json
```

Each package must exit successfully, print exactly one terminal `PASS`, execute its expected witness, and contain no failed or top-level skipped test record. Intentional skipped subtests remain valid only when their enclosing top-level test passes. A failure stops that profile; completed results and every unvalidated inventory entry remain in the JSON report.

Test binaries have a 60-second test deadline. CI bounds each profile to 25 minutes, or 40 minutes for W32-WASI's six Binaryen/Asyncify links. LLGo compilation cache is shared among the six package builds, while `-count=1` ensures that every test binary executes. Official Go reference commands force `GOMAXPROCS=1` because the current Go wasm runtimes do not create operating-system threads.

The inventory walks `test/std` independently of profile build constraints and then records source selection from `go list`. `pass` and `fail` are actual execution results; `not-run` remains unvalidated; `source-excluded` requires later review or replacement coverage and is not treated as a compatibility pass. This focused slice is not a completeness claim. W3 runs and classifies the full applicable `test/**` and GOROOT corpus.
