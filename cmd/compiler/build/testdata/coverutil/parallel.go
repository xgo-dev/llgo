package coverutil

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// WaitForPeer makes serialized test execution fail deterministically. A shared
// -coverprofile must not stop two test binaries from occupying separate workers.
func WaitForPeer(t *testing.T) {
	t.Helper()
	dir := os.Getenv("LLGO_COVER_BARRIER")
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(os.Getpid())), nil, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		peers, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(peers) >= 2 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("coverage test binaries ran sequentially")
		}
		time.Sleep(time.Millisecond)
	}
}
