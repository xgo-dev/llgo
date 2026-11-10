package build

import (
	stdcontext "context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/lto"
)

func TestNativeTailTransferIntegration(t *testing.T) {
	if runtime.GOARCH != "arm64" || (runtime.GOOS != "linux" && runtime.GOOS != "darwin") {
		t.Skip("native ARM64 bridge")
	}
	if testing.Short() {
		t.Skip("builds and runs native stack-switching assembly")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := filepath.Join(root, "internal/build/testdata/native_tail")
	for _, tc := range []struct {
		mode lto.Mode
		tags string
	}{{lto.Off, ""}, {lto.Full, ""}, {lto.Off, "nogc"}} {
		t.Run(tc.mode.String()+"/"+tc.tags, func(t *testing.T) {
			conf := NewDefaultConf(ModeBuild)
			conf.LTO, conf.Tags = tc.mode, tc.tags
			conf.OutFile = filepath.Join(t.TempDir(), "native-tail")
			if _, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := stdcontext.WithTimeout(t.Context(), 30*time.Second)
			defer cancel()
			out, err := exec.CommandContext(ctx, conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "entry/resume/stack/registers/gc: OK" {
				t.Fatalf("native transfer: %v\n%s", err, out)
			}
		})
	}
}
