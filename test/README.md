# LLGo test version coverage

CI builds the llgo compiler and repository tooling only with the exact Go 1.26
release pinned in `.go-version`. Packages below `test/` are then loaded and
tested with real Go toolchains from Go 1.20 through Go 1.26. The version runner
uses a temporary alternate module file whose `go` directive matches the target
release and sets `GOTOOLCHAIN=local`, so a test cannot silently upgrade or
downgrade to another toolchain.

Go 1.25 and Go 1.26 run all packages on both Linux and macOS. To limit runner
usage, Go 1.20 through Go 1.24 each run a representative package set, alternating
between Linux and macOS. Tests for APIs introduced by a newer Go release belong
in files with standard release tags such as `//go:build go1.24`; the selected Go
toolchain then includes those files automatically. Symbol-coverage checks use
the same toolchain and tags.

The ordinary current-version command remains:

```sh
llgo test ./test/...
```

Use the version runner for an older release or a smaller local package set:

```sh
dev/test_go_version.sh 1.20
dev/test_go_version.sh 1.24 ./test/std/bytes ./test/goroot

# Run the complete local Go 1.20 through Go 1.26 matrix
dev/test_go_versions.sh
```

The complete local matrix is sequential and may take tens of minutes. CI runs
the versions in separate jobs, with the full Go 1.25 and Go 1.26 package sets
sharded on Linux.

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
