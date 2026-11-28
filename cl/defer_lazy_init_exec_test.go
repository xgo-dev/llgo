package cl_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestLazyConditionalDeferInitExecution(t *testing.T) {
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	dir := t.TempDir()
	const source = `package main

import "fmt"

var log []string

func record(s string) { log = append(log, s) }

//go:noinline
func earlyReturn(fast bool, tag string) string {
	if fast {
		return "fast:" + tag
	}
	defer record("early:" + tag)
	return "slow:" + tag
}

//go:noinline
func midFunctionPanicRecover(fast bool, base int) (out int) {
	live := base + 10
	if fast {
		return live
	}
	defer func() {
		if r := recover(); r != nil {
			record(fmt.Sprintf("recovered:%v:%d", r, live))
			out = live + 100
		}
	}()
	live += 5
	panic("boom")
}

//go:noinline
func loopDeferBehindGuard(fast bool, n int) int {
	if fast {
		return -1
	}
	for i := 0; i < n; i++ {
		v := i
		defer record(fmt.Sprintf("loop:%d", v))
	}
	return n
}

func main() {
	if got := earlyReturn(true, "a"); got != "fast:a" || len(log) != 0 {
		panic("earlyReturn fast path failed: " + got)
	}
	if got := earlyReturn(false, "b"); got != "slow:b" {
		panic("earlyReturn slow path failed: " + got)
	}
	if got := midFunctionPanicRecover(true, 7); got != 17 {
		panic("midFunctionPanicRecover fast path failed")
	}
	if got := midFunctionPanicRecover(false, 7); got != 122 {
		panic(fmt.Sprintf("midFunctionPanicRecover recover path got %d, want 122", got))
	}
	if got := loopDeferBehindGuard(true, 3); got != -1 {
		panic("loopDeferBehindGuard fast path failed")
	}
	if got := loopDeferBehindGuard(false, 3); got != 3 {
		panic("loopDeferBehindGuard slow path failed")
	}
	want := "early:b,recovered:boom:22,loop:2,loop:1,loop:0"
	got := ""
	for i, s := range log {
		if i > 0 {
			got += ","
		}
		got += s
	}
	if got != want {
		panic("unexpected defer log: got " + got + ", want " + want)
	}
	println("lazy-defer-ok")
}
`
	for name, text := range map[string]string{
		"go.mod": "module lazydeferexec\n\ngo 1.27.0\n", "main.go": source,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, opt := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		t.Run(opt.String(), func(t *testing.T) {
			conf := build.NewDefaultConf(build.ModeBuild)
			conf.OptLevel = opt
			conf.OutFile = filepath.Join(dir, "lazydefer-"+opt.String())
			if runtime.GOOS == "windows" {
				conf.OutFile += ".exe"
			}
			if _, err := build.Build(build.Invocation{Args: []string{"."}, Config: conf, Dir: dir}); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), conf.OutFile).CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) != "lazy-defer-ok" {
				t.Fatalf("lazy defer execution (%s): %v\n%s", opt, err, out)
			}
		})
	}
}
