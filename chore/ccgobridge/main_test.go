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

func TestStagedModuleRoundTrip(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"app/go.mod": "module example.com/app\n\ngo 1.27\nrequire example.com/ccgo v0.0.0\nreplace example.com/ccgo => ../dependency\n",
		"app/main.go": `package main
import (
 "example.com/ccgo"
 "example.com/ccgo/extra"
)
func main() { if ccgo.Call(extra.Input) != 42 { panic("callback") } }
`,
		"dependency/go.mod":         "module example.com/ccgo\n\ngo 1.27\n",
		"dependency/extra/extra.go": "package extra\nconst Input = 21\n",
		"dependency/other.go": `package ccgo
import "unsafe"
// Another file's consumer must remain valid after its unsafe cast is adapted.
func Other(p uintptr) func() {
 return *(*func())(unsafe.Pointer(&struct{ uintptr }{p}))
}
`,
		"dependency/ccgo.go": `package ccgo
import "unsafe"
func __ccgo_fp(f interface{}) uintptr {
 type iface [2]uintptr
 return (*iface)(unsafe.Pointer(&f))[1]
}
func twice(n int) int { return n*2 }
func Call(n int) int {
 p := __ccgo_fp(twice)
 return (*(*func(int) int)(unsafe.Pointer(&struct{ uintptr }{p})))(n)
}
`,
	}
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
	app := filepath.Join(root, "app")
	out := filepath.Join(root, "staged")
	if err := run(app, out, []string{"example.com/ccgo"}); err != nil {
		t.Fatal(err)
	}
	if err := run(app, out, []string{"example.com/ccgo"}); err == nil {
		t.Fatal("existing output was overwritten")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "go", "run", "-mod=readonly", "-modfile="+filepath.Join(out, "build.mod"), ".")
	command.Dir = app
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("staged callback: %v\n%s", err, output)
	}
	for name, original := range files {
		actual, err := os.ReadFile(filepath.Join(root, name))
		if err != nil || string(actual) != original {
			t.Fatalf("input %s was modified: %v", name, err)
		}
	}
	manifest, err := os.ReadFile(filepath.Join(out, "adaptations.json"))
	if err != nil || !strings.Contains(string(manifest), `"StaticBoxes": 1`) {
		t.Fatalf("missing adaptation receipt: %v\n%s", err, manifest)
	}
}

func TestRequiresExplicitSelectionAndNewOutput(t *testing.T) {
	for _, test := range []struct {
		out      string
		patterns []string
	}{
		{"", []string{"example.com/ccgo"}},
		{filepath.Join(t.TempDir(), "out"), nil},
		{t.TempDir(), []string{"example.com/ccgo"}},
	} {
		if err := run(".", test.out, test.patterns); err == nil {
			t.Fatal("invalid command accepted")
		}
	}
}
