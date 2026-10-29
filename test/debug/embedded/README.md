# Embedded debugger transport fixture

This fixture validates the automated embedded path of `llgo debug`. It builds
a host-side Cortex-M ELF with DWARF, starts and stops QEMU, derives unchanged
flash bytes, and runs both transports:

- `gdb-multiarch` through GDB Remote;
- LLDB through `gdb-remote` with an explicit zero slide.

Each backend first uses the production target-configured QEMU stdio server
and its owned loopback relay, with a preloaded image and no writes requested.
It then starts another QEMU **without a kernel image** and exercises the
production `llgo debug -load` path. `qemu_load_proxy.py` translates OpenOCD's
`monitor reset halt` to QEMU's `system_reset`; its TCP server rejects empty
connections, verifying that readiness does not discard the only stub client.
All memory writes, register access,
breakpoints and execution go to the real QEMU GDB stub. The test requires two
resets with real image writes between them, then checks source location,
parameters, aggregate/array/string locals, globals and the backtrace.

This tests the reset/download protocol and the loaded program in a simulator.
It does not test OpenOCD device drivers, `vFlash` erase behavior, or physical
flash/probe hardware.

Run it with QEMU's ARM system emulator, a GDB that supports ARM, LLDB, and the
LLVM 23 tools and Python 3 on `PATH`. `LLGO_GDB`, `LLGO_LLDB`, `QEMU_SYSTEM_ARM`
and `LLGO` can select their executables. Both modes share portable Python
assertions with the physical-probe suite:

```sh
bash test/debug/embedded/runtest.sh
python3 test/debug/embedded/runtest.py --backend lldb
python3 test/debug/embedded/runtest.py --backend gdb
```

The Targets workflow has independent GDB/LLDB jobs on Linux x64, macOS arm64
and Windows x64. Windows uses the separate MSYS2 `gdb-multiarch` package:
ordinary native-only GDB is insufficient for Cortex-M. This is remote target
coverage, independent of native process-launch support on those hosts.

See the [debugging guide](../../../doc/debugging.md) for user-facing session
commands and the [physical-probe acceptance](../hardware/README.md) for opt-in
hardware validation.
