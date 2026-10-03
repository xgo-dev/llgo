# Size-build correctness regressions

Run from the repository root:

```sh
python3 test/sizebuild/runtest.py --target=wasi
python3 test/sizebuild/runtest.py --target=esp32c3
```

The runner builds its own development LLGo. The WASI full-LTO cases exercise
Go-metadata DCE, while the separate ESP32-C3 lane uses `-deadcodedrop`; the two
paths are not enabled together. The WASI lane requires WAMR (`iwasm`, or
`IWASM=/path/to/iwasm`) with threads and legacy exception handling. It builds and
runs println with full/ThinLTO and fmt.Printf with full LTO, first forcing
compilation and then permitting package-cache reuse. The existing worker-defer
fixture checks Go panic/recover, Goexit, and C setjmp/longjmp with both full and
ThinLTO. A full-LTO build of the threaded-GC fixture also checks goroutines,
panic/recover during collection, C blocking, and arena growth.

WAMR runs with `--max-threads=128 --stack-size=1048576 --heap-size=0`.

The ESP32-C3 lane checks C-only DCE firmware compilation before and after cache
reuse. It does not claim hardware or emulator execution. Both lanes use LLGo's
normal target-toolchain setup. These are correctness checks, not size budgets.

The runner's failure diagnostics have a short host-only regression:

```sh
python3 -m unittest discover -s test/sizebuild -p '*_test.py'
```
