# LLGo test version coverage

CI builds the llgo compiler and repository tooling only with the exact Go 1.27
release pinned in `.go-version`. Packages below `test/` are then loaded and
tested with real Go toolchains from Go 1.20 through Go 1.27. The version runner
uses a temporary alternate module file whose `go` directive matches the target
release and sets `GOTOOLCHAIN=local`, so a test cannot silently upgrade or
downgrade to another toolchain.

Go 1.27 runs all packages on Linux, macOS, and both Windows ABI profiles. To
limit runner usage, Go 1.20 through Go 1.26 share one sequential Linux job and
each run a representative package set containing their release-specific
checks. Platform behavior is covered by the primary Go 1.27 matrix rather than
forming a Cartesian product of every Go version and platform. Tests for APIs
introduced by a newer Go release belong
in files with standard release tags such as `//go:build go1.24`; the selected Go
toolchain then includes those files automatically. Symbol-coverage checks use
the same toolchain and tags.

The ordinary current-version command remains:

```sh
llgo test ./test/...
```

Native C-export and nested callback execution checks are ordinary test cases in
`cgo/export_threads_test.go`, run by both `go test ./test/cgo` and the native
LLGo test suite. They build and execute `cgo/testdata/foreigncallback` to cover
both main-package and dependency exports. The LLGo build helper uses
`LLGO_TEST_LLGO` (set by the version runner), falling back to `llgo` on `PATH`.
These cases use the normal build cache and run on the existing native CI
matrix: Linux amd64, macOS arm64, and Windows amd64/arm64/386 with both MSVC
and MinGW ABI profiles.
The program reports Go-thread and C-thread reentry separately, after checking
allocation/GC, retained outer values, and all nested defers. Compiler IR and
symbol checks remain in `cl/_test*` and `internal/build`.

Use the version runner for an older release or a smaller local package set:

```sh
dev/test_go_version.sh 1.20
dev/test_go_version.sh 1.24 ./test/std/bytes ./test/goroot

# Run the complete local Go 1.20 through Go 1.27 matrix
dev/test_go_versions.sh
```

The complete local matrix is sequential and may take tens of minutes. CI also
runs Go 1.20 through Go 1.26 sequentially in one compatibility job, while the
full Go 1.27 package set is sharded on Linux and runs once on each other native
toolchain profile.

The runner downloads an exact toolchain when needed, builds llgo itself with
the `.go-version` toolchain, and leaves the working tree unchanged. Set `LLGO`
to reuse an existing compiler binary.

The wasm runtime lanes use the same model and are locally reproducible with
`dev/test_wasm_runtime_go_version.sh 1.24`; run both CI endpoints with
`dev/test_wasm_runtime_go_versions.sh`. The wasm clite syscall implementation
uses `structs.HostLayout`, so this subrange explicitly requires Go 1.24.

The `runtime` and `_demo` modules retain a Go 1.20 compatibility floor unless a
submodule explicitly needs a newer language or standard-library feature. The
native runtime floor is reproducible with `dev/test_runtime_go_version.sh 1.20`.
