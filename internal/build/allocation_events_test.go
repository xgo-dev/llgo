package build

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
)

func TestNativeAllocationEventsExactIntegration(t *testing.T) {
	if testing.Short() || (runtime.GOOS != "darwin" && runtime.GOOS != "linux" && runtime.GOOS != "windows") {
		t.Skip("builds and runs a native BDWGC allocation probe")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, mode := range []lto.Mode{lto.Off, lto.Full} {
		t.Run(mode.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.LTO = mode
			conf.OutFile = filepath.Join(t.TempDir(), "allocation-events")
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: filepath.Join(root, "internal/build/testdata/allocation_events")}); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 10; i++ {
				out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
				if err != nil || strings.TrimSpace(string(out)) != "allocation events: OK" {
					t.Fatalf("exact allocation events: %v\n%s", err, out)
				}
			}
		})
	}
}
