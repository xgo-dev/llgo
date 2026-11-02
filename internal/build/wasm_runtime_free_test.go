package build

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func TestWasmRuntimeFreeFunctions(t *testing.T) {
	for _, test := range []struct {
		name, source string
		want         bool
	}{
		{"empty", `func main() {}`, true},
		{"text", `func main() { println("hello", true, 42, uint64(99)) }`, true},
		{"scalar state", `var n int; func next() int { n++; return n }; func main() { println(next()) }`, true},
		{"loop", `func main() { for i:=0; i<3; i++ { println(i) } }`, true},
		{"repeated direct call", `func next(n int) int { return n+1 }; func main(){ println(next(1),next(2)) }`, true},
		{"unary scalars", `var n int; var b bool; func main(){ println(-n, ^n, !b) }`, true},
		{"named scalar", `type number int; func change(n int) number { return number(n) }; func main(){ println(change(42)) }`, true},
		{"named pointer", `type pointer *int; func change(p *int) pointer { return pointer(p) }; func main(){ change(nil) }`, false},
		{"indirect parameter", `func invoke(f func()) { f() }; func main(){ invoke(nil) }`, false},
		{"receive parameter", `func receive(c chan int) { <-c }; func main(){ receive(nil) }`, false},
		{"other builtin", `func size(s string) int { return len(s) }; func main(){ println(size("x")) }`, false},
		{"unreachable allocation", `func unused() any { return new(int) }; func main() { println("ok") }`, true},
		{"panic", `func main() { panic("bad") }`, false},
		{"defer", `func main() { defer println("bye") }`, false},
		{"go", `func main() { go func(){ println("worker") }() }`, false},
		{"allocation", `var p *int; func main() { p = new(int) }`, false},
		{"interface", `var a any; func main() { a = 1 }`, false},
		{"string addition", `var s = "hi"; func main() { s += "!"; println(s) }`, false},
		{"float formatting", `func main() { println(1.25) }`, false},
		{"division", `var n=1; func main() { println(42/n) }`, false},
		{"shift", `var n=1; func main() { println(42<<n) }`, false},
		{"nil dereference", `var p *int; func main() { println(*p) }`, false},
		{"call through variable", `var f=func(){ }; func main() { f() }`, false},
		{"recursion", `func main() { main() }`, false},
		{"unknown external", `func external(); func main() { external() }`, false},
		{"receive", `var c chan int; func main() { println(<-c) }`, false},
		{"bounds", `var s []int; func main() { println(s[1]) }`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, "package main; "+test.source)
			roots := []*ssa.Function{pkg.Func("init"), pkg.Func("main")}
			if got := wasmRuntimeFreeFunctions(roots, func(*ssa.Function) string { return "" }, func(*ssa.Global) bool { return true }); got != test.want {
				t.Fatalf("runtime-free = %v, want %v", got, test.want)
			}
		})
	}
}

func TestWasmRuntimeFreeNativeBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, source, symbol string
		want                 bool
	}{
		{"literal C string", `func external(string) *byte; func main() { external("x") }`, "llgo.cstr", true},
		{"dynamic C string", `var s="x"; func external(string) *byte; func main() { external(s) }`, "llgo.cstr", false},
		{"C printf", `func external(*byte, ...any) int32; func main() { external(nil) }`, "C.printf", true},
		{"C printf unknown arguments", `func external(*byte, ...any) int32; func invoke(args []any){ external(nil,args...) }; func main(){ invoke(nil) }`, "C.printf", false},
		{"C printf arguments", `func external(*byte, ...any) int32; func main() { external(nil, 42) }`, "C.printf", false},
		{"C callback", `func external(func()); func main() { external(func(){}) }`, "C.foreign", false},
		{"Go body with C name", `func external(*byte, ...any) int32 { panic("bad") }; func main() { external(nil) }`, "C.printf", false},
		{"wrong printf declaration", `func external(); func main() { external() }`, "C.printf", false},
		{"wrong cstr declaration", `func external(); func main() { external() }`, "llgo.cstr", false},
		{"wrong cstr argument", `func external(int); func main() { external(1) }`, "llgo.cstr", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := buildWasmReflectTestProgram(t, "package main; "+test.source)
			linkname := func(fn *ssa.Function) string {
				if fn.Name() == "external" {
					return test.symbol
				}
				return ""
			}
			got := wasmRuntimeFreeFunctions([]*ssa.Function{pkg.Func("main")}, linkname, func(*ssa.Global) bool { return true })
			if got != test.want {
				t.Fatalf("runtime-free = %v, want %v", got, test.want)
			}
		})
	}
	if wasmRuntimeFreeFunctions(nil, nil, nil) {
		t.Fatal("missing roots accepted")
	}
	pkg := buildWasmReflectTestProgram(t, `package main; var n int; func main(){ println(n) }`)
	if wasmRuntimeFreeFunctions([]*ssa.Function{pkg.Func("main")}, nil, func(*ssa.Global) bool { return false }) {
		t.Fatal("foreign or owner-local global accepted")
	}
}

func TestConfigureWasmRuntimeFree(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	t.Setenv("LDFLAGS", "")
	for _, test := range []struct {
		name   string
		change func(*testing.T, *context, *aPackage)
		want   bool
	}{
		{"plain", func(*testing.T, *context, *aPackage) {}, true},
		{"verbose", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.Verbose = true }, true},
		{"native", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.Goarch = "amd64" }, false},
		{"no optimization", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.OptLevel = optlevel.O0 }, false},
		{"library", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.BuildMode = BuildModeCArchive }, false},
		{"generation", func(_ *testing.T, c *context, _ *aPackage) { c.mode = ModeGen }, false},
		{"coverage hooks", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.Coverage = &CoverageConfig{} }, false},
		{"multiple roots", func(_ *testing.T, c *context, _ *aPackage) { c.initial = append(c.initial, c.initial[0]) }, false},
		{"debugger", func(_ *testing.T, c *context, _ *aPackage) { c.frontendOptions.DebuggerFrames = true }, false},
		{"named WASI target", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.Target = "wasi" }, true},
		{"custom target", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.Target = "custom-wasi.json" }, false},
		{"target native input", func(_ *testing.T, c *context, _ *aPackage) { c.crossCompile.ExtraFiles = []string{"entry.c"} }, false},
		{"browser", func(_ *testing.T, c *context, _ *aPackage) { c.prog.Target().WasmProvider = "emscripten" }, false},
		{"asyncify", func(_ *testing.T, c *context, _ *aPackage) { c.crossCompile.WasmPostLink.Asyncify = true }, false},
		{"linker flags", func(_ *testing.T, c *context, _ *aPackage) { c.buildConf.LinkOptions.ExternalLinkerFlags = "-lforeign" }, false},
		{"environment linker flags", func(_ *testing.T, c *context, _ *aPackage) { c.commands.environ = []string{"LDFLAGS=-lforeign"} }, false},
		{"diagnostic linker flags", func(t *testing.T, _ *context, _ *aPackage) { t.Setenv("LDFLAGS", "-Wl,-Map=diagnostic.map") }, false},
		{"response linker flags", func(_ *testing.T, c *context, _ *aPackage) { c.commands.environ = []string{"LDFLAGS=@local.rsp"} }, false},
		{"process linker flags", func(t *testing.T, _ *context, _ *aPackage) { t.Setenv("LDFLAGS", "-lforeign") }, false},
		{"missing source directory", func(_ *testing.T, _ *context, p *aPackage) { p.Dir = filepath.Join(p.Dir, "missing") }, false},
		{"native source", func(_ *testing.T, _ *context, p *aPackage) { p.OtherFiles = []string{"entry.c"} }, false},
		{"embedded data", func(_ *testing.T, _ *context, p *aPackage) { p.EmbedFiles = []string{"data.txt"} }, false},
		{"native wrapper", func(t *testing.T, _ *context, p *aPackage) {
			if err := os.Mkdir(filepath.Join(p.Dir, "_wrap"), 0o700); err != nil {
				t.Fatal(err)
			}
		}, false},
		{"export", func(_ *testing.T, _ *context, p *aPackage) {
			p.Syntax[0].Comments = []*ast.CommentGroup{{List: []*ast.Comment{{Text: "//export\tcallback"}}}}
		}, false},
		{"linked Go entry", func(_ *testing.T, c *context, p *aPackage) {
			c.prog.SetLinkname(llssa.FullName(p.Types, "main"), "C.callback")
		}, false},
		{"aliased global", func(_ *testing.T, c *context, p *aPackage) {
			c.prog.SetLinkname(llssa.FullName(p.Types, "n"), "C.number")
		}, false},
		{"thread local global", func(_ *testing.T, c *context, p *aPackage) {
			c.prog.SetLocalityInfoFor(p.Types, llssa.FullName(p.Types, "n"), llssa.LocalityInfo{Locality: llssa.ThreadLocal})
		}, false},
		{"cgo linker directive", func(_ *testing.T, _ *context, p *aPackage) {
			p.Syntax[0].Comments = []*ast.CommentGroup{{List: []*ast.Comment{{Text: `//go:cgo_ldflag "-lforeign"`}}}}
		}, false},
		{"compiler runtime input", func(_ *testing.T, _ *context, p *aPackage) { p.Dir = filepath.Join(root, "runtime") }, true},
		{"runtime thread local global", func(_ *testing.T, c *context, p *aPackage) {
			p.Dir = filepath.Join(root, "runtime")
			c.prog.SetLocalityInfoFor(p.Types, llssa.FullName(p.Types, "n"), llssa.LocalityInfo{Locality: llssa.ThreadLocal})
		}, false},
		{"runtime goroutine local global", func(_ *testing.T, c *context, p *aPackage) {
			p.Dir = filepath.Join(root, "runtime")
			c.prog.SetLocalityInfoFor(p.Types, llssa.FullName(p.Types, "n"), llssa.LocalityInfo{Locality: llssa.GoroutineLocal})
		}, false},
		{"runtime aliased global", func(_ *testing.T, c *context, p *aPackage) {
			p.Dir = filepath.Join(root, "runtime")
			c.prog.SetLinkname(llssa.FullName(p.Types, "n"), "C.number")
		}, false},
		{"C import", func(_ *testing.T, _ *context, p *aPackage) {
			p.Syntax[0].Imports = []*ast.ImportSpec{{Path: &ast.BasicLit{Kind: token.STRING, Value: `"C"`}}}
		}, false},
		{"native package", func(_ *testing.T, _ *context, p *aPackage) {
			p.Types.Scope().Insert(types.NewConst(0, p.Types, "LLGoPackage", types.Typ[types.String], constant.MakeString("link -lforeign")))
		}, false},
		{"native files", func(_ *testing.T, _ *context, p *aPackage) {
			p.Types.Scope().Insert(types.NewConst(0, p.Types, "LLGoFiles", types.Typ[types.String], constant.MakeString("foreign.c")))
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			const source = "package main; var n = 42; func next(n int) int { return n+1 }; func main() { println(next(n)) }"
			ssaPkg := buildWasmReflectTestProgram(t, source)
			file, err := parser.ParseFile(token.NewFileSet(), "main.go", source, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := &aPackage{Package: &packages.Package{ID: "main", PkgPath: "main", Types: ssaPkg.Pkg, Dir: t.TempDir(), Syntax: []*ast.File{file}}, SSA: ssaPkg}
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProvider: "wasi"})
			defer prog.Dispose()
			prog.EnableGCRoots(true)
			prog.EnableCooperativeSafepoints(true)
			ctx := &context{prog: prog, progSSA: ssaPkg.Prog, initial: []*packages.Package{pkg.Package}, pkgByID: map[string]Package{"main": pkg}, mode: ModeBuild,
				buildConf: &Config{Goos: "wasip1", Goarch: "wasm", BuildMode: BuildModeExe, OptLevel: optlevel.Os}}
			ctx.frontendOptions.ShadowStack = true
			test.change(t, ctx, pkg)
			configureWasmRuntimeFree(ctx)
			if ctx.wasmRuntimeFree != test.want {
				t.Fatalf("runtime-free = %v, want %v", ctx.wasmRuntimeFree, test.want)
			}
			if ctx.frontendOptions.ShadowStack == test.want || prog.GCRootsEnabled() == test.want || prog.CooperativeSafepointsEnabled() == test.want {
				t.Fatalf("instrumentation changed inconsistently: shadow=%v GC=%v polls=%v", ctx.frontendOptions.ShadowStack, prog.GCRootsEnabled(), prog.CooperativeSafepointsEnabled())
			}
		})
	}
}

func TestWasmRuntimeFreeFingerprint(t *testing.T) {
	prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
	defer prog.Dispose()
	ctx := &context{prog: prog, buildConf: &Config{Goos: "wasip1", Goarch: "wasm"}}
	regular := newManifestBuilder()
	ctx.collectCommonInputs(regular)
	ctx.wasmRuntimeFree = true
	minimal := newManifestBuilder()
	ctx.collectCommonInputs(minimal)
	if !minimal.common.WasmRuntimeFree || regular.Fingerprint() == minimal.Fingerprint() {
		t.Fatal("runtime-free package code must have a distinct cache key")
	}
	if (&commonSection{WasmRuntimeFree: true}).empty() {
		t.Fatal("runtime-free cache field omitted")
	}
}
