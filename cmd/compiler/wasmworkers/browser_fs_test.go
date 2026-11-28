package wasmworkers

import (
	"os/exec"
	"testing"
)

func TestBrowserFSProxyContract(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	if out, err := exec.Command(node, "browser_fs_test.mjs").CombinedOutput(); err != nil {
		t.Fatalf("browser filesystem proxy: %v\n%s", err, out)
	}
}
