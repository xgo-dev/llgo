# LLGo baseline benchmarks

This suite is the lightweight performance gate for ordinary LLGo changes. It
uses fixed workloads and calibrated benchmarks on Linux and macOS so it
can run on every `main` push and pull request. Branch-only series can be run
explicitly with `workflow_dispatch`, avoiding duplicate push and pull-request
jobs for the same commit. The two native jobs record normalized artifacts; a
trusted `workflow_run` publisher validates and merges both platforms into one
commit, branch, or pull-request series.

The program workloads reuse:

- `benchmark/binary_size/cprintf`: only `lib/c.Printf`;
- `benchmark/binary_size/println`: only the built-in `println`;
- `benchmark/binary_size/fmtprintf`: `fmt.Printf`.

For each workload, the collector performs an unmeasured warm build, then records
the median of six builds and eighteen process runs, file size, executable-code
bytes, allocated non-executable data, and zero-filled data. Workload order is
rotated between rounds to balance runner drift and cache position. On ELF,
read-only constants are included in the data bucket; on Mach-O, `__TEXT`
constants are included in the text bucket. The Go benchmark stream performs one
unrecorded warmup, then records seven one-second samples of compiler helpers and
LLGo-generated core-language operations: direct/interface calls, defer,
channels, `getg`, and global access. Goroutine creation keeps its bounded
100-iteration samples.

The program table also includes standalone memory-profile workloads so the
whole-program no-consumer path is measurable independently of the retained
profile paths. Every process disables BDWGC, warms with two million allocations,
then internally times forty million escaping 16-byte allocations. The reported
duration is the median of eighteen processes. The workloads are
`memprofile-no-consumer`, `memprofile-rate0`, and `memprofile-default`.

For pull requests, each platform job builds the recorded base and current
revisions on one runner, then alternates their measurements within every round.
Compiler and runtime benchmark binaries are built before sampling, so only their
execution is interleaved and each matching base/current sample remains adjacent.
This prevents a phase-wide frequency, thermal, or host load change from being
attributed entirely to one revision. Dependency setup is shared, and Go's build
cache can be reused by unchanged packages; main pushes still run the suite only
once. Very small changes can remain scheduler noise and should be confirmed by
repeated workflow runs. If a workflow does not provide a paired result, the
publisher falls back to the latest matching `main` data.

The trusted publisher commits the current result history and generated site to
the `pages` branch of the configured data repository. Every LLGo repository
defaults to `<owner>/llgo-benchmark-data`:

```text
llgo/baseline/series/main/main
llgo/baseline/series/branch/<safe branch identifier>
llgo/baseline/series/pull/<number>
```

The publisher never executes code from the measured revision and pull request
jobs never receive the benchmark repository token. Pull requests receive one
updated summary comment linking to their long-term trend page. If no matching
`main` history exists yet, the pull-request report is still published and
marks every metric as `new`.

Run the complete local collection with the same script as CI:

```sh
benchmark/baseline/run.sh \
  "$PWD" \
  "$PWD/.benchmark/llgo" \
  "$PWD/.benchmark/results"
```

The normalized artifact is `.benchmark/results/benchmark.txt`.
