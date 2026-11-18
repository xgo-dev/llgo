//go:build go1.25

package runtime_test

import (
	"runtime"
	"testing"
)

func TestSetDefaultGOMAXPROCS(t *testing.T) {
	original := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(original)
	runtime.SetDefaultGOMAXPROCS()
	if got := runtime.GOMAXPROCS(0); got < 1 {
		t.Fatalf("SetDefaultGOMAXPROCS selected %d", got)
	}
	// A restored default remains a real setting that a subsequent setter
	// returns, rather than a separate fallback used only by queries.
	previous := runtime.GOMAXPROCS(0)
	if got := runtime.GOMAXPROCS(1); got != previous {
		t.Fatalf("GOMAXPROCS returned %d after reset, want %d", got, previous)
	}
}
