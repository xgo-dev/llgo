# Runtime stress tests

These tests exercise runtime correctness under loads that are too expensive for
routine pull-request testing. The leading-underscore directory and nested
module keep them out of the repository's normal `go test ./...` and
`llgo test ./test/...` patterns. No push, pull-request, scheduled, or daily
workflow runs them automatically.

## Placement policy

- Keep deterministic, bounded regression coverage that is suitable for every
  pull request in the normal `test` tree.
- Put a test here when its useful baseline deliberately requires high
  concurrency, large live object counts, many repetitions, long timeouts, or a
  soak profile. A stress test should not be the only regression coverage for a
  bug when a small routine test is practical.
- Group runtime suites under `runtime/<subsystem>`. Use
  `LLGO_STRESS_PROFILE` for scalable counts, give every wait a finite timeout,
  and document platform or compiler restrictions below.
- Adding a suite here does not opt it into CI. Run it explicitly against the
  LLGo revision under test when changing the affected subsystem.

Build LLGo from the revision being tested, then run the suites explicitly:

```sh
repo=$(git rev-parse --show-toplevel)
(cd "$repo" && go build -o /tmp/llgo-runtime-stress ./cmd/llgo)
export LLGO_ROOT="$repo"
cd "$repo/test/_stress"

go test -race -count=3 -timeout=20m ./runtime/timer
/tmp/llgo-runtime-stress test -count=3 -timeout=20m ./runtime/timer
/tmp/llgo-runtime-stress test -count=3 -timeout=30m ./runtime/signal
/tmp/llgo-runtime-stress test -count=3 -timeout=30m ./runtime/cpuprof
/tmp/llgo-runtime-stress test -count=3 -timeout=30m ./runtime/finalizer
/tmp/llgo-runtime-stress test -count=3 -timeout=5m ./runtime/weak
```

The signal suite is LLGo-only and targets Unix hosts. The CPU profile suite is
LLGo-only and targets supported Darwin and Linux hosts. The finalizer suite is
LLGo-only and targets hosted BDWGC builds. The timer suite also runs with the
standard Go compiler so the harness can be checked independently.

`LLGO_STRESS_PROFILE` controls the load while preserving the same assertions:

- `quick`: one eighth of the default counts, for local iteration.
- `default` (or unset): the checked-in baseline.
- `heavy`: twice each default dimension (and potentially more than twice the
  total work), for deliberate soak runs.

For example:

```sh
LLGO_STRESS_PROFILE=heavy /tmp/llgo-runtime-stress test \
  -count=10 -timeout=2h ./runtime/signal
```

The timer suite covers large live heaps, concurrent `Stop`/`Reset` on distinct
and shared timers, callback bursts, and concurrent sleepers. The five signal
tests cover distinct-signal delivery during a lower-number signal storm,
concurrent registration churn, repeated `Notify`/`Stop`/`Reset` barriers,
fatal-signal arbitration in concurrent helper processes while the final
receiver stops, and timer progress while handlers are busy. The flood cases
also exercise the handler under concurrent delivery pressure. The CPU profile
suite changes the logical SIGPROF mode under a continuous signal flood to
regress fatal native-handler replacement windows. The finalizer suite
repeatedly publishes large finalizer batches while many goroutines call
`runtime.GC`, and checks that queued callbacks are neither corrupted nor
delivered twice.

The weak suite creates batches of weak pointers in a goroutine that exits,
then performs bounded collections while retaining only the weak handles. It
regresses cleanup callbacks recursively invoking another cleanup while
allocating under the weak registry lock. Each batch must expire more than half
its handles, and a separate helper process enforces a 60-second deadline even
if the tested runtime deadlocks. It uses only public Go APIs and also runs with
`go test` on Go 1.24 or newer. Use identical profiles before and after the fix;
neither sleeping nor reducing allocation pressure is part of the workload.
