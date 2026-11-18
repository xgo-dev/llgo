package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestExportObjectWithClangCleansTemporaryFiles(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Fatal(err)
	}
	const ir = "define i32 @answer() { ret i32 42 }\n"
	for _, tc := range []struct {
		name            string
		genLL           bool
		invalid         bool
		missingCompiler bool
	}{
		{name: "success"},
		{name: "retain explicit IR", genLL: true},
		{name: "compile error", invalid: true},
		{name: "compiler unavailable", missingCompiler: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(key, tmp)
			}
			exportFile := filepath.Join(t.TempDir(), "package")
			ctx := &context{
				buildConf:    &Config{GenLL: tc.genLL},
				crossCompile: crosscompile.Export{CC: clang},
				commands:     commandEnv{environ: os.Environ()},
			}
			if tc.missingCompiler {
				ctx.crossCompile.CC = filepath.Join(tmp, "missing-clang")
			}
			data := []byte(ir)
			if tc.invalid {
				data = []byte("invalid LLVM IR\n")
			}
			object, err := exportObjectWithClang(ctx, "example.com/p", exportFile, data)
			if tc.invalid || tc.missingCompiler {
				if err == nil {
					t.Fatal("compilation succeeded unexpectedly")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				// The caller still owns the successful object until archiving/linking.
				info, err := os.Stat(object)
				if err != nil || info.Size() == 0 {
					t.Fatalf("compiled object unavailable: %v", err)
				}
				if err := os.Remove(object); err != nil {
					t.Fatal(err)
				}
			}
			entries, err := os.ReadDir(tmp)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Fatalf("temporary compilation files leaked: %v", entries)
			}
			if tc.genLL {
				got, err := os.ReadFile(exportFile + ".ll")
				if err != nil || string(got) != ir {
					t.Fatalf("explicit IR output = %q, error %v", got, err)
				}
			}
		})
	}
}
