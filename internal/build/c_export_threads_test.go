package build

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Use a standalone executable so main-package and dependency exports are both
// exercised through their real C entry points.
func TestCExportForeignThreadsFromExecutableAndDependency(t *testing.T) {
	// Keep optional unwind diagnostics out of the fixture's result protocol.
	t.Setenv("LLGO_DYNUNWIND_DEBUG", "0")
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
	default:
		t.Skip("hosted native C callbacks")
	}
	dir, err := filepath.Abs("../../test/cgo/testdata/foreigncallback")
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "callback")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	// Compile in the host test process; only the generated fixture needs a
	// subprocess to exercise its C entry points and foreign threads.
	t.Chdir(dir)
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = output
	if _, err := Do([]string{"."}, conf); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(output).CombinedOutput()
	const want = "Go-thread reentry: ok\nC-thread reentry: ok\nok"
	if err != nil || strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n")) != want {
		t.Fatalf("native thread C exports: %v\n%s", err, out)
	}
	t.Log(strings.TrimSpace(string(out)))
}
