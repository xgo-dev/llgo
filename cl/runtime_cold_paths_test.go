package cl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestRuntimeFatalNoInlineAndUnreachable(t *testing.T) {
	runtimeDir := filepath.Join("..", "runtime", "internal", "runtime")
	for _, file := range []string{"fatal_default.go", "fatal_emscripten.go", "stubs.go"} {
		data, err := os.ReadFile(filepath.Join(runtimeDir, file))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, "//go:noinline") {
			t.Fatalf("%s missing //go:noinline on fatal/throw", file)
		}
	}
	// Verify evacuated compiles without injecting an AssertNilDeref check.
	const src = `package mapcheck
import "unsafe"
type bmap struct { tophash [8]uint8 }
func evacuated(b *bmap) bool {
	h := *(*uint8)(unsafe.Pointer(b))
	return h > 1 && h < 5
}
`
	ssaPkg, _, files := buildGoSSAPkg(t, src)
	prog := newLLSSAProgForTarget(t, &llssa.Target{GOOS: "darwin", GOARCH: "arm64"})
	defer prog.Dispose()
	pkg, err := NewPackage(prog, ssaPkg, files)
	if err != nil {
		t.Fatal(err)
	}
	defer pkg.Module().Dispose()
	body := mustNamedFunction(t, pkg.Module(), "mapcheck.evacuated").String()
	if strings.Contains(body, "AssertNilDeref") {
		t.Fatalf("evacuated emitted unexpected AssertNilDeref:\n%s", body)
	}
}
