//go:build go1.25

package synctest_test

import (
	"testing"
	"testing/synctest"
)

func TestCallbackExecution(t *testing.T) {
	calls := 0
	cleanupRan := false
	synctest.Test(t, func(bubbleT *testing.T) {
		calls++
		if bubbleT.Name() != t.Name() {
			bubbleT.Fatalf("callback test name = %q, want %q", bubbleT.Name(), t.Name())
		}
		bubbleT.Cleanup(func() { cleanupRan = true })
		synctest.Wait()
		if calls != 1 {
			bubbleT.Fatalf("Wait resumed with callback count %d", calls)
		}
	})
	if calls != 1 {
		t.Fatalf("synctest callback ran %d times, want 1", calls)
	}
	if !cleanupRan {
		t.Fatal("synctest callback cleanup did not run before Test returned")
	}
}
