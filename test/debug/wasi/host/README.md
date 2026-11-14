# W32 WASI debugger host

This optional developer tool embeds Wasmtime 50.0.1 and the maintained
`wasi-common`/Wiggle 36.0.20 LTS preview1 implementation. It supplies real
`wasi.thread-spawn` threads, shared linear memory, EH, and one process-wide
descriptor table. Only the engine executes guest code; the LTS libraries supply
the preview1 syscall implementation without their old engine dependency.

Build from the repository root with Rust 1.97 or newer:

```sh
rustup toolchain install 1.97.0 --profile minimal
RUSTC="$(rustup which --toolchain 1.97.0 rustc)" \
  rustup run 1.97.0 cargo build --locked --manifest-path test/debug/wasi/host/Cargo.toml
export LLGO_WASI_HOST="$PWD/test/debug/wasi/host/target/debug/llgo-wasi-debug"
llgo debug -target=wasi ./app
```

`--version` reports the host ABI, locked engine/WASI library versions and actual
Rust compiler. Go/LLGo builds do not depend on Rust unless this optional host is
built. Cargo.lock pins the tool's dependencies; `preview1.rs` lists the stable
core signatures and the shared-memory bridge delegates every syscall to the
LTS implementation. Workers clone one `WasiCtx`, retaining its shared table,
and instantiate the same module and memory. A worker trap terminates the
process; `proc_exit` preserves its real exit code.

Source sessions compile with native DWARF, no JIT optimization, and
`debug-builtins` for GDB JIT registration. Persistent compilation caching is
not compiled into this host, so debugger-patched instructions cannot enter a
warm compilation cache. `llgo run -target=wasi` keeps its normal execution host.

Run [the real debugger acceptance](../README.md), which also verifies a worker
created before a file is opened can write and close the process descriptor.
