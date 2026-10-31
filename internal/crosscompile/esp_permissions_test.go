package crosscompile

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUnreadableBundledESPClangDoesNotFallBackToDownload(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("requires POSIX directory permissions enforced for a non-root user")
	}
	bundle, cached, requests := bundledESPFixture(t)
	directory := filepath.Dir(bundle)
	if err := os.Chmod(directory, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(directory, 0755); err != nil {
			t.Error(err)
		}
	})
	if _, err := os.Stat(bundle); !errors.Is(err, os.ErrPermission) {
		t.Skip("filesystem does not enforce the expected permission failure")
	}
	root, err := getESPClangRoot(true)
	if root != "" || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("unreadable bundle = %q, %v; want permission error", root, err)
	}
	if requests.Load() != 0 {
		t.Fatal("an inaccessible local archive must fail without a network fallback")
	}
	if _, err := os.Stat(cached); !os.IsNotExist(err) {
		t.Fatalf("failed installation left a cache at %s: %v", cached, err)
	}
}
