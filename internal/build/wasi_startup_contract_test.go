package build

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWasiSizePruningTestlogStartupSource(t *testing.T) {
	path := filepath.Join(runtime.GOROOT(), "src", "internal", "testlog", "log.go")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rewritten := rewriteWASIStartupTestlog(source)
	if _, err := parser.ParseFile(token.NewFileSet(), path, rewritten, parser.AllErrors); err != nil {
		t.Fatal(err)
	}
	text := string(rewritten)
	for _, fragment := range []string{
		"if !llgoWASIUseOrdinaryStartup { return nil }",
		"llgoRequireWASIStartup();",
		"C.__llgo_wasi_requires_managed_startup",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("missing %q", fragment)
		}
	}
	legacy := strings.Replace(string(source), "return *impl\n", "return *impl.(*Interface)\n", 1)
	if got := rewriteWASIStartupTestlog([]byte(legacy)); got == nil {
		t.Fatal("did not rewrite the older atomic.Value logger")
	}
	changed := strings.Replace(string(source), "impl := logger.Load()", "impl := readLogger()", 1)
	if got := rewriteWASIStartupTestlog([]byte(changed)); got != nil {
		t.Fatal("rewrote an unrecognized logger implementation")
	}
}
