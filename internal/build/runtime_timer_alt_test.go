package build

import (
	stdcontext "context"
	"go/ast"
	gobuild "go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	llruntime "github.com/xgo-dev/llgo/runtime"
)

func TestNativeTimerSchedulerParkedDeadline(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a native timer probe")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)

	// Read the wait state under the scheduler's mutex. Acquiring it after
	// timerWaitUntil is set proves the timed wait has released the mutex.
	file := filepath.Join(root, "runtime", "internal", "lib", "runtime", "time_heap_llgo.go")
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	const observer = `
//go:linkname timerSchedulerProbe timerprobe.wait
func timerSchedulerProbe() (wakeAt, headWhen int64) {
	timerSchedulerMu.Lock()
	if timerSchedulerWait == timerWaitUntil {
		wakeAt = timerSchedulerWakeAt
		if len(timerSchedulerHeap) > 0 {
			headWhen = timerSchedulerHeap[0].r.when
		}
	}
	timerSchedulerMu.Unlock()
	return
}
`
	probe := filepath.Join(t.TempDir(), "probe.go")
	const program = `package main

import (
	"runtime"
	"time"
	_ "unsafe"
)

//go:linkname timerWait timerprobe.wait
func timerWait() (wakeAt, headWhen int64)

func main() {
	head := time.NewTimer(48*time.Hour)
	var wakeAt, headWhen int64
	for headWhen == 0 {
		wakeAt, headWhen = timerWait()
		runtime.Gosched()
	}
	if remaining := time.Duration(headWhen - wakeAt); remaining < 23*time.Hour || remaining > 24*time.Hour {
		panic("scheduler did not record the capped wait deadline")
	}

	later := time.NewTimer(36*time.Hour)
	if got, _ := timerWait(); got != wakeAt {
		panic("a deadline after the capped wake time interrupted the wait")
	}
	if !head.Stop() || !later.Stop() {
		panic("long timers fired early")
	}
	if got, deadline := timerWait(); got != wakeAt || deadline != 0 {
		panic("stopping the timers changed the parked deadline")
	}

	next := time.NewTimer(0)
	<-next.C
}
`
	if err := os.WriteFile(probe, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.Overlay = map[string][]byte{file: append(src, observer...)}
	conf.OutFile = filepath.Join(filepath.Dir(probe), "timer-probe")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	if _, err := Do([]string{probe}, conf); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := stdcontext.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, conf.OutFile).CombinedOutput(); err != nil {
		t.Fatalf("timer scheduler probe: %v (context: %v)\n%s", err, ctx.Err(), out)
	}
}

func TestNativeRuntimeTimerRemainsReplaceAltPackage(t *testing.T) {
	if !llruntime.HasAltPkg("runtime") {
		t.Fatal("runtime must remain an LLGo alt package")
	}
	if llruntime.HasAdditiveAltPkg("runtime") {
		t.Fatal("runtime must remain replace-mode; additive mode would pull in the Go runtime sources")
	}

	runtimeDir := filepath.Join(env.LLGoRuntimeDir(), "internal", "lib", "runtime")
	runtimePatchDir := filepath.Join(env.LLGoRuntimeDir(), "_patch", "runtime")
	for _, target := range []struct {
		goos   string
		goarch string
	}{
		{goos: "darwin", goarch: "arm64"},
		{goos: "linux", goarch: "amd64"},
		{goos: "windows", goarch: "amd64"},
	} {
		t.Run(target.goos+"_"+target.goarch, func(t *testing.T) {
			ctx := gobuild.Default
			ctx.GOOS = target.goos
			ctx.GOARCH = target.goarch
			ctx.BuildTags = []string{"llgo"}
			pkg, err := ctx.ImportDir(runtimeDir, 0)
			if err != nil {
				t.Fatal(err)
			}

			selected := make(map[string]bool, len(pkg.GoFiles)+len(pkg.CgoFiles))
			for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
				selected[name] = true
				file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(runtimeDir, name), nil, parser.ImportsOnly)
				if err != nil {
					t.Fatal(err)
				}
				for _, spec := range file.Imports {
					path, err := strconv.Unquote(spec.Path.Value)
					if err != nil {
						t.Fatal(err)
					}
					if path == "github.com/xgo-dev/llgo/runtime/internal/clite/libuv" {
						t.Fatalf("native runtime selected %s, which imports libuv", name)
					}
				}
			}
			if !selected["time_heap_llgo.go"] {
				t.Fatal("native runtime did not select the LLGo timer heap backend")
			}

			patchFiles, err := filepath.Glob(filepath.Join(runtimePatchDir, "*.go"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range patchFiles {
				name := filepath.Base(path)
				match, err := ctx.MatchFile(runtimePatchDir, name)
				if err != nil {
					t.Fatal(err)
				}
				if match {
					// Constants must reach the frontend before it checks importers.
					// Runtime implementations, including timers, still come from
					// the replace-mode alternative package.
					file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
					if err != nil {
						t.Fatal(err)
					}
					for _, decl := range file.Decls {
						gen, ok := decl.(*ast.GenDecl)
						if !ok || gen.Tok != token.CONST {
							t.Fatalf("native runtime source patch %s contains implementation declarations; timer reuse must stay in the replacement runtime", name)
						}
					}
				}
			}
		})
	}
}
