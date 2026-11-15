//go:build !llgo

package wasmdecoder

import (
	"os/exec"
	"testing"
)

func TestRuntimeDecoderJavaScript(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is unavailable")
	}
	command := exec.Command(node, "--test", "plugin_test.js", "goroutines_test.js")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("runtime decoder tests: %v\n%s", err, output)
	}
}
