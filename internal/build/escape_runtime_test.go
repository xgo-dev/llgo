package build

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestAppendUint32CachedPrivateMethod(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the cached AppendUint32 regression")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv(llgoBuildCache, "1")
	t.Setenv(llgoEscapeCache, "0")
	oldCacheRoot := cacheRootFunc
	cacheDir := t.TempDir()
	cacheRootFunc = func() string { return cacheDir }
	t.Cleanup(func() { cacheRootFunc = oldCacheRoot })
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module appendprobe\n\ngo 1.24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	const probe = `package main
import ("encoding/binary"; "runtime")
var stats runtime.MemStats
var keep []byte
//go:noinline
func word(b []byte, v uint32) []byte { return binary.LittleEndian.AppendUint32(b, v) }
func main() {
 b := make([]byte, 0, 16)
 best := ^uint64(0)
 // Counters are process-wide; allow retries for unrelated runtime activity.
 for attempt := 0; attempt < 4 && best != 0; attempt++ {
  runtime.ReadMemStats(&stats)
  before := stats.Mallocs
  for i := 0; i < 10000; i++ {
   b = word(b[:0], uint32(i))
   if binary.LittleEndian.Uint32(b) != uint32(i) { panic("append checksum") }
  }
  runtime.ReadMemStats(&stats)
  best = min(best, stats.Mallocs-before)
 }
 if best != 0 { panic("AppendUint32 allocated with spare capacity") }
 keep = b
 println("append-cache-ok")
}`
	for phase, source := range []string{
		// Populate the dependency archive without a reference to AppendUint32.
		`package main; import "encoding/binary"; func main() { if binary.Size(uint32(1)) != 4 { panic("size") }; println("append-cache-ok") }`,
		probe,
	} {
		if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		conf := NewDefaultConf(ModeBuild)
		conf.BuildParallelism = 2
		conf.OutFile = filepath.Join(t.TempDir(), "probe")
		if runtime.GOOS == "windows" {
			conf.OutFile += ".exe"
		}
		pkgs, err := Build(Invocation{Args: []string{"."}, Dir: dir, Config: conf})
		if err != nil {
			t.Fatalf("phase %d: %v", phase, err)
		}
		found := false
		for _, pkg := range pkgs {
			if pkg.PkgPath == "encoding/binary" {
				found = true
				if pkg.CacheHit != (phase == 1) {
					t.Fatalf("phase %d: binary CacheHit=%v", phase, pkg.CacheHit)
				}
			}
		}
		if !found {
			t.Fatal("encoding/binary dependency missing")
		}
		out, err := exec.Command(conf.OutFile).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "append-cache-ok" {
			t.Fatalf("phase %d runtime: %v\n%s", phase, err, out)
		}
	}
}

func TestRuntimeEscapeContractsAvoidObserverAndMapKeyAllocation(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	for _, test := range []struct {
		path  string
		names []string
	}{
		{"runtime", []string{"runtime.ReadMemStats"}},
		{llssa.PkgRuntime, []string{llssa.PkgRuntime + ".mapaccess1_faststr", llssa.PkgRuntime + ".mapaccess2_faststr"}},
	} {
		t.Run(test.path, func(t *testing.T) {
			conf := NewDefaultConf(ModeGen)
			pkgs, err := Build(Invocation{Args: []string{test.path}, Config: conf})
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 || pkgs[0].LPkg == nil {
				t.Fatal("missing compiled runtime package")
			}
			defer pkgs[0].LPkg.Prog.Dispose()
			for _, name := range test.names {
				fn := pkgs[0].LPkg.Module().NamedFunction(name)
				if fn.IsNil() || fn.FirstBasicBlock().IsNil() {
					t.Fatalf("missing %s definition", name)
				}
				if body := fn.String(); strings.Contains(body, "runtime.Alloc") {
					t.Errorf("%s retains temporary heap storage:\n%s", name, body)
				}
			}
		})
	}
}
