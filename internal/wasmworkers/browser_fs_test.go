package wasmworkers

import (
	"os/exec"
	"testing"
)

func TestBrowserFSProxyContract(t *testing.T) {
	testBrowserFS(t, "browser_fs_test.mjs")
}

func TestBrowserFSSnapshotContract(t *testing.T) {
	testBrowserFS(t, "browser_fs_snapshot_test.mjs")
}

func testBrowserFS(t *testing.T, script string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	if out, err := exec.Command(node, script).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v\n%s", script, err, out)
	}
}
