package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestLazyConditionalDeferInit(t *testing.T) {
	const source = `package lazydefer
var Sink int

func cleanup() { Sink++ }

func fastEarlyReturn(fast bool) int {
	if fast {
		return 1
	}
	defer cleanup()
	Sink += 2
	return 2
}

func sharedReturn(cond bool) int {
	res := 1
	if cond {
		defer cleanup()
		res = 2
	}
	return res
}

func loopBehindGuard(fast bool, n int) int {
	if fast {
		return 0
	}
	for i := 0; i < n; i++ {
		defer cleanup()
	}
	return n
}

func alwaysEntry() int {
	defer cleanup()
	return 3
}
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "arm64"},
		{GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileInterfaceBoxIR(t, source, ssa.InstantiateGenerics, &target, target.GOARCH == "wasm")

			entryBlock := func(fnIR string) string {
				start := strings.Index(fnIR, "_llgo_0:")
				if start < 0 {
					t.Fatalf("missing _llgo_0 block:\n%s", fnIR)
				}
				rest := fnIR[start+len("_llgo_0:"):]
				if end := strings.Index(rest, "\n_llgo_"); end >= 0 {
					return rest[:end]
				}
				return rest
			}

			// fastEarlyReturn: entry block must NOT set up a defer frame; all allocas stay in _llgo_0.
			fast := llvmFunction(t, ir, "lazydefer.fastEarlyReturn")
			fastEntry := entryBlock(fast)
			if strings.Contains(fastEntry, "ThreadDeferSlot") {
				t.Fatalf("fastEarlyReturn hoisted ThreadDeferSlot into entry block:\n%s", fast)
			}
			if !strings.Contains(fast, "ThreadDeferSlot") {
				t.Fatalf("fastEarlyReturn missing ThreadDeferSlot on slow path:\n%s", fast)
			}
			inEntry := true
			for _, line := range strings.Split(fast, "\n") {
				if strings.HasPrefix(line, "_llgo_") && !strings.HasPrefix(line, "_llgo_0:") {
					inEntry = false
				}
				if !inEntry && strings.Contains(line, " = alloca ") {
					t.Fatalf("fastEarlyReturn emitted non-entry alloca: %s\n%s", line, fast)
				}
			}

			// sharedReturn: return block is reachable from both fast and defer paths,
			// so defer initialization must fall back to the entry block.
			shared := llvmFunction(t, ir, "lazydefer.sharedReturn")
			if !strings.Contains(entryBlock(shared), "ThreadDeferSlot") {
				t.Fatalf("sharedReturn must initialize defer frame in entry block:\n%s", shared)
			}

			// loopBehindGuard: entry block skips defer init; defer init is hoisted out
			// of the loop into the dominating non-loop block after the guard.
			looped := llvmFunction(t, ir, "lazydefer.loopBehindGuard")
			if strings.Contains(entryBlock(looped), "ThreadDeferSlot") {
				t.Fatalf("loopBehindGuard hoisted ThreadDeferSlot past early return:\n%s", looped)
			}
			if !strings.Contains(looped, "ThreadDeferSlot") {
				t.Fatalf("loopBehindGuard missing ThreadDeferSlot:\n%s", looped)
			}

			// alwaysEntry: unconditional defer stays in the entry block.
			always := llvmFunction(t, ir, "lazydefer.alwaysEntry")
			if !strings.Contains(entryBlock(always), "ThreadDeferSlot") {
				t.Fatalf("alwaysEntry missing ThreadDeferSlot in entry block:\n%s", always)
			}
		})
	}
}
