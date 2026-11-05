package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/goescape"
)

func TestEscapeVersionInvalidatesArchives(t *testing.T) {
	old, current := newManifestBuilder(), newManifestBuilder()
	current.common.EscapeAnalysisVersion = goescape.Version
	if current.common.empty() || old.Fingerprint() == current.Fingerprint() {
		t.Fatal("escape analysis did not invalidate pre-analysis archives")
	}
	decoded, err := decodeManifest(current.Build())
	if err != nil || decoded.Common == nil || decoded.Common.EscapeAnalysisVersion != goescape.Version {
		t.Fatalf("escape version round trip: %+v, %v", decoded, err)
	}
}

// Run phases in fresh processes: a warm archive must not rely on summaries
// accidentally left in memory by the process that generated it.
func TestEscapeCrossPackageCacheAndWorkers(t *testing.T) {
	const prefix = "LLGO_TEST_ESCAPE_"
	if dir := os.Getenv(prefix + "DIR"); dir != "" {
		cacheRootFunc = func() string { return os.Getenv(prefix + "CACHE") }
		t.Setenv(llgoBuildCache, "1")
		root, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			t.Fatal(err)
		}
		t.Setenv("LLGO_ROOT", root)
		t.Setenv("LLGO_ESCAPE", "example.com/escape")
		conf := NewDefaultConf(ModeBuild)
		conf.BuildParallelism, _ = strconv.Atoi(os.Getenv(prefix + "WORKERS"))
		conf.OutFile = filepath.Join(t.TempDir(), "probe")
		if runtime.GOOS == "windows" {
			conf.OutFile += ".exe"
		}
		pkgs, err := Build(Invocation{Args: []string{"."}, Config: conf, Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		phase := os.Getenv(prefix + "PHASE")
		depHit, rootHit := false, false
		for _, pkg := range pkgs {
			if pkg.PkgPath == "example.com/escape/dep" {
				depHit = pkg.CacheHit
			}
			if pkg.PkgPath == "example.com/escape" {
				rootHit = pkg.CacheHit
			}
		}
		// Executable main packages are deliberately never archive-cache hits.
		if depHit != (phase != "cold") || rootHit {
			t.Fatalf("phase=%s dependency hit=%v root hit=%v", phase, depHit, rootHit)
		}
		out, err := exec.Command(conf.OutFile).CombinedOutput()
		if err != nil || strings.TrimSpace(string(out)) != "escape-ok" {
			t.Fatalf("run: %v\n%s", err, out)
		}
		return
	}
	dir, cache := t.TempDir(), t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/escape\n\ngo 1.24\n",
		"dep/dep.go": `package dep
type Payload [16]uint64
//go:noinline
func Read(p *Payload) uint64 { return p[0] + p[15] }
//go:noinline
func Call(v any) uint64 { return v.(func() uint64)() }
`,
		"main.go": `package main
import ("runtime"; "example.com/escape/dep")
//go:noinline
func borrow(p dep.Payload) uint64 {return dep.Read(&p)}
//go:noinline
func box(f func()uint64)uint64{return dep.Call(f)}
func main() {
 var p dep.Payload; p[0]=3;p[15]=7
 f:=func()uint64{return 11}
 for i:=0;i<100;i++ {borrow(p);box(f)}
 var before,after runtime.MemStats
 runtime.ReadMemStats(&before)
 var sum uint64
 for i:=0;i<10000;i++ {sum+=borrow(p)+box(f)}
 runtime.ReadMemStats(&after)
 if sum!=210000 {panic("result corrupted")}
 if after.TotalAlloc-before.TotalAlloc>65536 {panic("borrow or interface box still allocates per call")}
 println("escape-ok")
}
`,
	}
	for name, src := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for i, phase := range []string{"cold", "hot", "rebuild", "hot"} {
		workers := 1
		if i >= 2 {
			workers = 8
		}
		if phase == "rebuild" {
			// Recompile only the caller while the analyzed dependency comes
			// from its archive. A changed optimization proof must not be lost.
			if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(files["main.go"]+"\n// caller changed\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestEscapeCrossPackageCacheAndWorkers$", "-test.count=1")
		cmd.Env = append(os.Environ(), prefix+"DIR="+dir, prefix+"CACHE="+cache, prefix+"PHASE="+phase, fmt.Sprintf("%sWORKERS=%d", prefix, workers))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s p%d: %v\n%s", phase, workers, err, out)
		}
		for _, proof := range []string{"example.com/escape.borrow: Alloc stack", "example.com/escape.box: MakeInterface stack"} {
			if !strings.Contains(string(out), proof) {
				t.Fatalf("%s p%d lost %q:\n%s", phase, workers, proof, out)
			}
		}
	}
}
