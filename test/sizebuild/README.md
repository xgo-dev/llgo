# Size-build correctness regressions

Run from the repository root:

```sh
python3 test/sizebuild/runtest.py --target=wasi
python3 test/sizebuild/runtest.py --target=esp32c3
```

The runner builds its own development LLGo. The WASI full-LTO cases exercise
Go-metadata DCE, while the separate ESP32-C3 lane uses `-deadcodedrop`; the two
paths are not enabled together. The WASI lane requires Wasmer (`wasmer`, or
`WASMER=/path/to/wasmer`) with threads and standard exception handling. It builds and
runs C printf and println with off/full/ThinLTO and fmt.Printf with full LTO, first forcing
compilation and then permitting package-cache reuse. The existing worker-defer
fixture checks Go panic/recover, Goexit, and C setjmp/longjmp with both full and
ThinLTO. A full-LTO build of the threaded-GC fixture also checks goroutines,
panic/recover during collection, C blocking, and arena growth.

Before that matrix, a shared helper is built in a pure program, a panicking
program, and the pure program again. This verifies that cached packages retain
the correct runtime policy and panic source locations. The pure case also checks
NUL/Unicode text, booleans, signed/unsigned integer limits and stderr routing.

Wasmer runs with `--enable-exceptions --enable-simd --stack-size=1048576`
and the same source and `/tmp` volumes as the main WASI acceptance runner.

The ESP32-C3 lane checks C-only DCE firmware compilation before and after cache
reuse. It does not claim hardware or emulator execution. Both lanes use LLGo's
normal target-toolchain setup. These are correctness checks, not size budgets.

The runner's failure diagnostics have a short host-only regression:

```sh
python3 -m unittest discover -s test/sizebuild -p '*_test.py'
```
