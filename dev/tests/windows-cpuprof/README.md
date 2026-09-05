# Windows CPU profiler CONTEXT regression

After the existing Windows dependency setup and target activation, run:

```powershell
./.github/workflows/test_windows_cpuprof.ps1
```

The script uses the activated `LLGO_WINDOWS_ABI` and `LLGO_WINDOWS_ARCH`, does
not download dependencies, and builds a native C executable with optimization
and frame pointers enabled. Each compiler, inspection, and execution process
has a 30-second deadline. Output and diagnostics remain in the reported temporary
directory for investigation.

The harness includes the production `profile_windows.c` in its test translation
unit. A native worker publishes its real
frame pointer, then spins without making calls. The controller waits at most five
seconds for readiness, suspends the worker, invokes the production context helper
and capture function, and resumes the worker before reporting assertions. It
requires the captured frame-pointer register to equal the worker's published
value, checks that capture preserves an interrupted PC, and exercises both
unaligned and unreadable frame-pointer rejection. Thread completion and handle
cleanup are bounded as well.

This checks real `GetThreadContext` FP/PC values and the requested context flags
without depending on the statistical likelihood of sampling a hot loop. In
particular, Windows AMD64 requires `CONTEXT_INTEGER` to retrieve RBP; the harness
also checks those flags because some systems populate unrequested registers.
The harness deliberately does not treat a
Win64 frame-pointer chain as a complete unwind: nonzero unwind frame offsets and
foreign frames still require a separate, safe unwinding design.
