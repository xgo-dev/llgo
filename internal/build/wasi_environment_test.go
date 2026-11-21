package build

import (
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/ssa"
)

func TestOptionalInitializersWASIEnvironmentSnapshot(t *testing.T) {
	const source = `package syscall
var llgoMutableEnvironment, llgoEnvironmentNeeded, llgoWASIPathStateNeeded bool
var envs = llgoLoadIndexedEnvironment()
var llgoEnvironmentSlots []uint32
var copyenv = llgoEnvironmentCopier()
var cwd string
var preopens []string
func llgoLoadEnvironment() []string { return nil }
func llgoLoadIndexedEnvironment() []string {
	entries := llgoLoadEnvironment()
	llgoEnvironmentSlots = []uint32{1}
	return entries
}
func llgoEnvironmentCopier() func() {
	if llgoEnvironmentNeeded && llgoMutableEnvironment { return llgoCopyEnvironmentIndex }
	return nil
}
func llgoCopyEnvironmentIndex() { println(len(envs)) }
func init() { if llgoWASIPathStateNeeded { cwd = Getenv("PWD") } }
func Getenv(key string) string { return envs[0] }
func Getwd() string { return cwd }
func Setenv() { envs = nil }
func Environ() []string { return envs }
func Write() {}
`
	for _, tc := range []struct {
		name, body, symbol, extra string
		wantDrop, mayReflect      bool
	}{
		{name: "unused", body: "syscall.Write()", wantDrop: true},
		{name: "getter", body: `println(syscall.Getenv("KEY"))`},
		{name: "setter", body: "syscall.Setenv()"},
		{name: "enumeration", body: "println(len(syscall.Environ()))"},
		{name: "startup getter", body: "syscall.Write()", extra: `func init() { println(Getenv("PWD")) }`},
		{name: "cwd", body: "println(syscall.Getwd())"},
		{name: "assembly observer", body: "syscall.Write()", symbol: "syscall.envs"},
		{name: "index observer", body: "syscall.Write()", symbol: "syscall.llgoEnvironmentSlots"},
		{name: "linked getter", body: "syscall.Write()", symbol: "syscall.Getenv"},
		{name: "initializer root", body: "syscall.Write()", symbol: "syscall.llgoLoadEnvironment"},
		{name: "possible reflection", body: "syscall.Write()", mayReflect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions|ssa.GlobalDebug)
			p := buildFormatTestPackage(t, prog, "syscall", source+tc.extra, nil)
			entry := buildFormatTestPackage(t, prog, "main", "package main\nimport \"syscall\"\nfunc main() {"+tc.body+"}", p.Pkg)
			prog.Build()
			backend := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmReflectBridges: tc.mayReflect})
			defer backend.Dispose()
			ctx := &context{prog: backend, buildConf: &Config{Verbose: true}}
			roots := []*ssa.Function{entry.Func("init"), entry.Func("main")}
			configureWASISyscallState(ctx, p, roots, nil, func(name string) bool { return name == tc.symbol })
			if got := slices.Contains(ctx.initializerGuardsOff, wasiEnvironmentNeededGuard); got != tc.wantDrop {
				t.Fatalf("snapshot omitted = %v, want %v", got, tc.wantDrop)
			}
		})
	}
}

func TestOptionalInitializersWASIEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name, body, symbol string
		want               bool
	}{
		{name: "lookup", body: `syscall.Getenv("key")`, want: true},
		{name: "unused", body: `println(1)`},
		{name: "set", body: `syscall.Getenv("key"); syscall.Setenv("key", "value")`},
		{name: "unset", body: `syscall.Getenv("key"); syscall.Unsetenv("key")`},
		{name: "clear", body: `syscall.Getenv("key"); syscall.Clearenv()`},
		{name: "enumerate", body: `syscall.Getenv("key"); syscall.Environ()`},
		{name: "assembly observation", body: `syscall.Getenv("key")`, symbol: "syscall.envs"},
		{name: "linked writer", body: `syscall.Getenv("key")`, symbol: "syscall.Setenv"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			p := buildFormatTestPackage(t, prog, "syscall", `package syscall
var llgoMutableEnvironment bool
func Getenv(string) (string, bool) { return "", false }
func Setenv(string, string) error { return nil }
func Unsetenv(string) error { return nil }
func Clearenv() {}
func Environ() []string { return nil }
`, nil)
			entry := buildFormatTestPackage(t, prog, "main", `package main
import "syscall"
var _ = syscall.Getenv
func main() {`+tc.body+`}`, p.Pkg)
			prog.Build()
			ctx := &context{buildConf: &Config{Verbose: true}}
			result := rta.Analyze([]*ssa.Function{entry.Func("main")}, false)
			configureWASIEnvironment(ctx, wasiSyscallPackage(prog), result, func(name string) bool { return name == tc.symbol })
			got := len(ctx.initializerGuardsOff) == 1 && ctx.initializerGuardsOff[0] == wasiEnvironmentGuard
			if got != tc.want {
				t.Fatalf("specialized = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOptionalInitializersWASIEnvironmentSource(t *testing.T) {
	srcDir := filepath.Join(runtime.GOROOT(), "src", "syscall")
	name := filepath.Join(srcDir, "env_unix.go")
	source, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(rewriteWASIGetenv(source))
	for _, fragment := range []string{"llgoLoadIndexedEnvironment()", "return llgoGetenvIndexed(key)", "func llgoGetenvWithLock(", "envLock.RLock()", "defer envLock.RUnlock()"} {
		if !strings.Contains(updated, fragment) {
			t.Fatalf("rewritten Getenv missing %q", fragment)
		}
	}
	if strings.Contains(string(source), "var copyenv = sync.OnceFunc(func() {") {
		for _, fragment := range []string{
			"var copyenv = llgoEnvironmentCopier()",
			"if !llgoEnvironmentNeeded || !llgoMutableEnvironment",
			"return sync.OnceFunc(llgoCopyEnvironmentIndex)",
			"func llgoGetenvWithLock(key string) (value string, found bool) {\n\tcopyenv()",
		} {
			if !strings.Contains(updated, fragment) {
				t.Fatalf("rewritten OnceFunc environment missing %q", fragment)
			}
		}
		for _, replacement := range []string{"existingCopier", "anotherOnce(func() {})", "sync.OnceFunc(copyEnvironment)"} {
			input := strings.Replace(string(source), "var copyenv = sync.OnceFunc(func() {", "var copyenv = "+replacement+"\n\nvar unrelated = sync.OnceFunc(func() {", 1)
			patched := string(rewriteWASIGetenv([]byte(input)))
			if !strings.Contains(patched, "return llgoGetenvWithoutLock(key)") || strings.Contains(patched, "llgoEnvironmentCopier") {
				t.Errorf("unexpected initializer rewrite with %s", replacement)
			}
		}
		input := strings.Replace(string(source), "func Getenv(key string) (value string, found bool) {\n\tcopyenv()",
			"func Getenv(key string) (value string, found bool) {\n\tif len(key) == 0 { return \"\", false }\n\tcopyenv()", 1)
		if patched := string(rewriteWASIGetenv([]byte(input))); !strings.Contains(patched, "return llgoGetenvWithoutLock(key)") || strings.Contains(patched, "llgoEnvironmentCopier") {
			t.Fatal("moved the environment initialization past a different Getenv entry")
		}
	}
	if strings.Contains(string(source), "envOnce.Do(copyenv)") &&
		!strings.Contains(updated, "func llgoGetenvWithLock(key string) (value string, found bool) {\n\tenvOnce.Do(copyenv)") {
		t.Fatal("legacy locked Getenv lost its lazy initialization")
	}
	for _, unsupported := range []string{"other package", "map[string]string", "differentLock.RLock"} {
		input := "package invalid"
		switch unsupported {
		case "map[string]string":
			input = strings.Replace(string(source), "map[string]int", unsupported, 1)
		case "differentLock.RLock":
			input = strings.Replace(string(source), "envLock.RLock", unsupported, 1)
		}
		if rewriteWASIGetenv([]byte(input)) != nil {
			t.Errorf("rewrote unsupported environment source %q", unsupported)
		}
	}
	for _, tc := range []struct {
		name string
		base map[string][]byte
		want bool
	}{
		{name: "size profile", want: true},
		{name: "user overlay", base: map[string][]byte{name: source}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, overlay, _, err := applySourcePatchForPkg(tc.base, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), "syscall", sourcePatchBuildContext{
				goos: "wasip1", goarch: "wasm", goversion: runtime.Version(), targetTags: []string{"llgo", "llgo.size.wasi"},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, installed := overlay[filepath.Join(srcDir, "z_llgo_patch_"+wasiEnvironmentPatch)]
			if installed != tc.want || strings.Contains(string(overlay[name]), "func llgoGetenvWithLock(") != tc.want {
				t.Fatalf("environment patch installed = %v, want %v", installed, tc.want)
			}
		})
	}
}

func TestOptionalInitializersWASIEnvironmentLegacySource(t *testing.T) {
	const source = `package syscall
import "sync"
var envs []string = runtime_envs()
var env map[string]int
var envOnce sync.Once
var envLock sync.RWMutex
func copyenv() {}
func Getenv(key string) (string, bool) {
	envOnce.Do(copyenv)
	envLock.RLock()
	defer envLock.RUnlock()
	return "", false
}`
	indexed := string(rewriteWASIGetenv([]byte(source)))
	if !strings.Contains(indexed, "envs []string = llgoLoadIndexedEnvironment()") ||
		!strings.Contains(indexed, "return llgoGetenvIndexed(key)") ||
		!strings.Contains(indexed, "func llgoGetenvWithLock(key string) (string, bool) {\n\tenvOnce.Do(copyenv)") {
		t.Fatalf("legacy indexed lookup lost its mutable fallback:\n%s", indexed)
	}
	other := strings.Replace(source, "envOnce.Do(copyenv)", "envOnce.Do(other)", 1)
	fallback := string(rewriteWASIGetenv([]byte(other)))
	if !strings.Contains(fallback, "envs []string = llgoLoadEnvironment()") ||
		!strings.Contains(fallback, "return llgoGetenvWithoutLock(key)") ||
		!strings.Contains(fallback, "envOnce.Do(other)") {
		t.Fatalf("unrecognized initialization changed:\n%s", fallback)
	}
}
