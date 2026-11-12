package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
)

func TestEmscriptenLTOLateStdioLibcall(t *testing.T) {
	if testing.Short() {
		t.Skip("links and executes Emscripten LTO binaries")
	}
	for _, tool := range []string{"emcc", "node"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not installed", tool)
		}
	}
	dir := t.TempDir()
	source := filepath.Join(dir, "stdio.c")
	if err := os.WriteFile(source, []byte(`#include <stdio.h>
__attribute__((noinline)) void byte(FILE *f) { fwrite("0", 1, 1, f); }
int main(void) { byte(stderr); return 0; }
`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		profile crosscompile.WasmProfile
		width   string
	}{{crosscompile.WasmProfileJ32, "-m32"}, {crosscompile.WasmProfileJ64, "-m64"}} {
		for _, mode := range []lto.Mode{lto.Thin, lto.Full} {
			t.Run(string(target.profile)+"/"+mode.String(), func(t *testing.T) {
				ctx := &context{buildConf: &Config{LTO: mode}, crossCompile: crosscompile.Export{WasmProfile: target.profile}}
				out := filepath.Join(dir, string(target.profile)+"-"+mode.String()+".mjs")
				// Leave fwrite intact until link-time optimization, as happens
				// when LLVM specializes LLGo's runtime print buffers during LTO.
				args := []string{"-O2", "-flto=" + mode.String(), target.width, "-sEXIT_RUNTIME=1", "-sNODE_HOST_ENV=1", "-Xclang", "-disable-llvm-passes", source, "-o", out}
				args = append(args, emscriptenLTOLibcallArgs(ctx)...)
				if output, err := exec.CommandContext(t.Context(), "emcc", args...).CombinedOutput(); err != nil {
					t.Fatalf("link late stdio libcall: %v\n%s", err, output)
				}
				runner := `import {pathToFileURL} from 'node:url'; const m = await import(pathToFileURL(process.argv[1]).href); await m.default();`
				if output, err := exec.CommandContext(t.Context(), "node", "--input-type=module", "-e", runner, out).CombinedOutput(); err != nil || strings.TrimSpace(string(output)) != "0" {
					t.Fatalf("execute late stdio libcall: %v\n%s", err, output)
				}
			})
		}
	}
}
