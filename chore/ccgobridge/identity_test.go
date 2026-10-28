package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const identityExporter = `
func __ccgo_fp(f interface{}) uintptr {
	type iface [2]uintptr
	return (*iface)(unsafe.Pointer(&f))[1]
}
`

func TestRejectCrossPackageExportIdentity(t *testing.T) {
	for _, externalOwner := range []bool{false, true} {
		name := "local-and-imported"
		owner := `package a
import "unsafe"
func F() {}
func Export() uintptr { return __ccgo_fp(F) }
` + identityExporter
		if externalOwner {
			name = "two-imported"
			owner = `package a
import (
	"unsafe"
	"example.com/ccgo/target"
)
func Export() uintptr { return __ccgo_fp(target.F) }
` + identityExporter
		}
		t.Run(name, func(t *testing.T) {
			bTarget := "a"
			if externalOwner {
				bTarget = "target"
			}
			app := identityFixture(t, map[string]string{
				"dependency/a/a.go": owner,
				"dependency/b/b.go": `package b
import (
	"unsafe"
	"example.com/ccgo/` + bTarget + `"
)
func Export() uintptr { return __ccgo_fp(` + bTarget + `.F) }
` + identityExporter,
				"app/main.go": `package main
import (
	"example.com/ccgo/a"
	"example.com/ccgo/b"
)
func main() {
	if a.Export() != b.Export() { panic("cross-package function identity changed") }
}
`,
			})
			if output, err := identityRun(app, ""); err != nil {
				t.Fatalf("original Go function identity: %v\n%s", err, output)
			}
			out := filepath.Join(filepath.Dir(app), "staged")
			err := run(app, out, []string{"example.com/ccgo/a", "example.com/ccgo/b"})
			if err == nil {
				// Preserve an executable counterexample if the preflight guard
				// regresses: independently allocated boxes compare unequal.
				output, runErr := identityRun(app, filepath.Join(out, "build.mod"))
				t.Fatalf("accepted cross-package export; adapted execution: %v\n%s", runErr, output)
			}
			for _, want := range []string{"cross-package", "example.com/ccgo/a", "example.com/ccgo/b", ".F"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("missing %q in diagnostic: %v", want, err)
				}
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("identity failure created output snapshot: %v", err)
			}
		})
	}
}

func TestSinglePackageImportedExportIdentity(t *testing.T) {
	app := identityFixture(t, map[string]string{
		"dependency/a/a.go": `package a
import (
	"unsafe"
	"example.com/ccgo/target"
)
func Export() uintptr { return __ccgo_fp(target.F) }
func Again() uintptr { return __ccgo_fp(target.F) }
` + identityExporter,
		"app/main.go": `package main
import "example.com/ccgo/a"
func main() {
	if a.Export() != a.Again() { panic("same-package function identity changed") }
}
`,
	})
	out := filepath.Join(filepath.Dir(app), "staged")
	if err := run(app, out, []string{"example.com/ccgo/a"}); err != nil {
		t.Fatal(err)
	}
	if output, err := identityRun(app, filepath.Join(out, "build.mod")); err != nil {
		t.Fatalf("single-owner imported function: %v\n%s", err, output)
	}
}

func identityFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	files["app/go.mod"] = "module example.com/app\n\ngo 1.27\nrequire example.com/ccgo v0.0.0\nreplace example.com/ccgo => ../dependency\n"
	files["dependency/go.mod"] = "module example.com/ccgo\n\ngo 1.27\n"
	files["dependency/target/target.go"] = "package target\nfunc F() {}\n"
	for name, contents := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	return filepath.Join(root, "app")
}

func identityRun(dir, modfile string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	args := []string{"run", "-mod=readonly"}
	if modfile != "" {
		args = append(args, "-modfile="+modfile)
	}
	command := exec.CommandContext(ctx, "go", append(args, ".")...)
	command.Dir = dir
	return command.CombinedOutput()
}
