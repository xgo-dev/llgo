//go:build go1.24 && !baremetal && !nogc && !wasm

package weakstress

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
	"weak"
)

type referent struct {
	id  int
	pad [256]byte
}

func stressCount(t *testing.T, base int) int {
	t.Helper()
	switch profile := os.Getenv("LLGO_STRESS_PROFILE"); profile {
	case "", "default":
		return base
	case "quick":
		return base / 8
	case "heavy":
		return base * 2
	default:
		t.Fatalf("unknown LLGO_STRESS_PROFILE %q", profile)
		return 0
	}
}

func TestWeakCleanupReentrancy(t *testing.T) {
	if os.Getenv("LLGO_STRESS_WEAK_CHILD") != t.Name() {
		// A GC callback can deadlock the allocating thread, including testing's
		// own timeout machinery. Keep the hard deadline in a separate process.
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, "-test.run=^"+t.Name()+"$", "-test.v", "-test.timeout=45s")
		cmd.Env = append(os.Environ(), "LLGO_STRESS_WEAK_CHILD="+t.Name())
		output, err := cmd.CombinedOutput()
		t.Logf("%s", output)
		if ctx.Err() != nil {
			t.Fatalf("weak cleanup helper did not exit within 60s: %v", ctx.Err())
		}
		if err != nil {
			t.Fatalf("weak cleanup helper failed: %v", err)
		}
		return
	}

	rounds, objects := stressCount(t, 16), stressCount(t, 8192)
	t.Logf("rounds=%d objects=%d GC_MARKERS=%q", rounds, objects, os.Getenv("GC_MARKERS"))
	for round := 0; round < rounds; round++ {
		handles := make(chan []weak.Pointer[referent], 1)
		go func() {
			live := make([]*referent, objects)
			batch := make([]weak.Pointer[referent], objects)
			for i := range live {
				live[i] = &referent{id: i}
				batch[i] = weak.Make(live[i])
			}
			handles <- batch
			runtime.KeepAlive(live)
			// Exit this goroutine so conservative stack scanning does not keep
			// the entire batch alive while another goroutine collects it.
		}()
		batch := <-handles
		var expired int
		for collection := 0; collection < 16; collection++ {
			// Finite collections avoid mixing collector/allocator starvation
			// from an unbounded GC loop into weak-callback reentrancy coverage.
			runtime.GC()
			expired = 0
			for _, h := range batch {
				if h.Value() == nil {
					expired++
				}
			}
			if expired > objects/2 {
				break
			}
			runtime.Gosched()
		}
		if expired <= objects/2 {
			t.Fatalf("round %d: only %d/%d weak handles expired", round, expired, objects)
		}
		t.Logf("round %d: expired %d/%d weak handles", round, expired, objects)
		runtime.KeepAlive(batch)
	}
}
