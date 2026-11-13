# CI placement

Compiler, runtime, ABI and firmware correctness remain PR gates. Mise and Pixi
host provisioning/build checks run daily in llgo-compat/ci's **Development
environments** workflow (companion PR: https://github.com/llgo-compat/ci/pull/84).
Both tools cover Linux amd64/arm64, macOS amd64/arm64 and Windows amd64. The
existing Mise/Pixi gate names remain during migration; they check preparation,
while the environment builds live in the daily repository. Embedded debugger
checks retain the Linux PR gate and their daily native-host coverage.

**Baseline benchmarks** run only on PRs. Base and head are measured sequentially
in the same runner job, with the exact merge-base recorded in both native and
Wasm artifacts. A trusted `workflow_run` publisher validates source/configuration
and updates one PR comment per suite. Its contents permission is read-only; no
history branch or Pages site is updated. Publishing a release still dispatches
the external release compatibility snapshot. Main pushes no longer dispatch
external benchmark runs.

Linux work uses the existing centralized Qiniu runner policy, with hosted
fallback for forks. Native macOS and Windows jobs retain their hosted images.
The daily repository can opt into its own accessible Qiniu Linux group; it does
not assume xgo-dev's runner access extends to llgo-compat.

## Queue observations, 2026-11-13

Sample: three recent completed runs per workflow, plus active runs, across Go,
LLGo, benchmarks, Mise, Pixi, targets and release-build. The table includes only
completed jobs with a runner acquired and known dependency mapping. Waiting
starts at the later of job creation and completion of its dependencies;
execution time and cancelled jobs that never acquired a runner are excluded.
These mixed-workload observations are not a controlled runner speed comparison.

| Runner | Completed queue samples | Median wait (min) | P90 wait (min) |
| --- | ---: | ---: | ---: |
| Qiniu Linux | 133 | 0.32 | 5.80 |
| GitHub Linux amd64 | 43 | 0.32 | 17.33 |
| GitHub Linux arm64 | 12 | 1.98 | 17.49 |
| GitHub Windows amd64/386 host | 38 | 16.98 | 85.61 |
| GitHub Windows arm64 | 15 | 11.68 | 46.71 |
| GitHub macOS arm64 | 24 | 31.65 | 116.52 |
| GitHub macOS amd64 | 10 | 24.20 | 63.71 |

Representative sampled runs: [LLGo](https://github.com/xgo-dev/llgo/actions/runs/43309387008),
[Go](https://github.com/xgo-dev/llgo/actions/runs/43309386985),
[benchmarks](https://github.com/xgo-dev/llgo/actions/runs/43309387017).

This supports reducing repeated host provisioning on PRs and keeping Linux on
Qiniu. The successful [Qiniu Windows probe](https://github.com/xgo-dev/llgo/actions/runs/41011157382)
waited 64 seconds but took 29m17s, including installing VS Build Tools and 7-Zip.
It qualified Pixi, not the complete MSVC/MinGW amd64/386 benchmark matrix. A
prepared image and full ABI qualification are needed before claiming a Windows
latency improvement. There is no verified Qiniu macOS runner in this sample.
Runner inventory permissions were unavailable, so these numbers do not establish
total capacity or the cause of hosted concurrency limits.
