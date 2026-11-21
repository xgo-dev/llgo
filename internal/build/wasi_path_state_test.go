package build

import (
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestOptionalInitializersWASIPathState(t *testing.T) {
	for _, tc := range []struct {
		name, body, symbol string
		wantDrop           bool
		reflect            bool
		replace, with      string
	}{
		{name: "descriptor IO", body: "syscall.Write()", wantDrop: true},
		{name: "getwd", body: "println(syscall.Getwd())"},
		{name: "open", body: "syscall.Open()"},
		{name: "uninvoked method", body: "println(syscall.Box())", wantDrop: true},
		{name: "interface method", body: "println(syscall.Box().(interface{ Path() string }).Path())"},
		{name: "reflective method", body: "println(syscall.Box())", reflect: true},
		{name: "assembly observer", body: "syscall.Write()", symbol: "syscall.preopens"},
		{name: "initializer alias", body: "syscall.Write()", symbol: "syscall.init#1"},
		{name: "unknown path state", body: "syscall.Write()", replace: "preopens", with: "mounts"},
		{name: "unused guard", body: "syscall.Write()", replace: "if llgoWASIPathStateNeeded", with: "if true"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions|ssa.GlobalDebug)
			source := `package syscall
var llgoWASIPathStateNeeded bool
var cwd string
var preopens []string
func init() {
	if llgoWASIPathStateNeeded { cwd = "/"; preopens = []string{"/"} }
}
func Write() {}
func Getwd() string { return cwd }
func Open() { println(len(preopens)) }
type pathReader struct{}
func (pathReader) Path() string { return cwd }
func Box() any { return pathReader{} }
`
			if tc.replace != "" {
				source = strings.ReplaceAll(source, tc.replace, tc.with)
			}
			p := buildFormatTestPackage(t, prog, "syscall", source, nil)
			entry := buildFormatTestPackage(t, prog, "main", "package main\nimport \"syscall\"\nfunc main() {"+tc.body+"}", p.Pkg)
			prog.Build()
			backend := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmReflectBridges: tc.reflect})
			defer backend.Dispose()
			ctx := &context{prog: backend, buildConf: &Config{Verbose: true}}
			roots := []*ssa.Function{entry.Func("init"), entry.Func("main")}
			configureWASISyscallState(ctx, wasiSyscallPackage(prog), roots, nil, func(name string) bool { return name == tc.symbol })
			if got := slices.Contains(ctx.initializerGuardsOff, wasiPathStateGuard); got != tc.wantDrop {
				t.Fatalf("path state dropped = %v, want %v", got, tc.wantDrop)
			}
		})
	}
}

func TestOptionalInitializersWASIPathStateSource(t *testing.T) {
	name := filepath.Join(runtime.GOROOT(), "src", "syscall", "fs_wasip1.go")
	input, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(rewriteWASIPathState(input))
	if strings.Count(updated, "if llgoWASIPathStateNeeded {") != 2 ||
		!strings.Contains(updated, "var llgoWASIPathStateNeeded bool") ||
		!strings.Contains(updated, "fd_prestat_dir_name(preopenFd,") {
		t.Fatal("the rewrite must guard only the state updates and preserve discovery")
	}
	const diagnostic = `panic("fd_prestat_dir_name: "`
	if strings.Count(strings.SplitN(updated, diagnostic, 2)[0], "\n") !=
		strings.Count(strings.SplitN(string(input), diagnostic, 2)[0], "\n") {
		t.Fatal("rewrote the directory error location")
	}
	bad := strings.Replace(string(input), "preopens = append(preopens, opendir{", "preopens = appendPath(preopens, opendir{", 1)
	if rewriteWASIPathState([]byte(bad)) != nil {
		t.Fatal("rewrote an unrecognized preopen state update")
	}
	bad = strings.Replace(string(input), "errno := fd_prestat_get(preopenFd,", "println(cwd)\nerrno := fd_prestat_get(preopenFd,", 1)
	if rewriteWASIPathState([]byte(bad)) != nil {
		t.Fatal("rewrote discovery with an independent access to cwd")
	}
	if rewriteWASIPathState([]byte("package other")) != nil {
		t.Fatal("rewrote a different package")
	}
}
