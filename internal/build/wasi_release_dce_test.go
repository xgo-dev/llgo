package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestWasiSizePruningAssembly(t *testing.T) {
	goroot := t.TempDir()
	dir := filepath.Join(goroot, "src", "internal", "bytealg")
	path := filepath.Join(dir, "compare_wasm.s")
	direct := parseWasmAggregateIR(t, `
@input = external global i32
@unused = external global i32
declare i32 @callGo(i32)
declare void @unusedGo()
define i32 @asmEntry() {
  %input = load i32, ptr @input
  %result = call i32 @callGo(i32 %input)
  ret i32 %result
}
`)
	indirect := parseWasmAggregateIR(t, `
define void @asmEntry(ptr %fn) {
  call void %fn()
  ret void
}
`)
	standard := &packages.Package{PkgPath: "internal/bytealg", Dir: dir}
	thirdParty := &packages.Package{PkgPath: "example.com/bytealg", Dir: t.TempDir()}

	known := &wasiSizePruningInputs{goroot: goroot}
	known.collectAsmReferences(standard, path, &Config{}, direct)
	if known.opaqueAsmPkg != "" || !slices.Equal(known.roots, []string{"callGo", "input"}) {
		t.Fatalf("standard assembly roots: %+v", known)
	}
	assembly := []struct {
		name string
		pkg  *packages.Package
		conf *Config
	}{
		{"outside GOROOT", thirdParty, &Config{}},
		{"overlaid", standard, &Config{Overlay: map[string][]byte{path: []byte("replacement")}}},
	}
	for _, test := range assembly {
		t.Run(test.name, func(t *testing.T) {
			in := &wasiSizePruningInputs{goroot: goroot}
			in.collectAsmReferences(test.pkg, path, test.conf, direct)
			if in.opaqueAsmPkg != test.pkg.PkgPath || len(in.roots) != 0 {
				t.Fatalf("opaque assembly modeled as ordinary roots: %+v", in)
			}
		})
	}
	in := &wasiSizePruningInputs{goroot: goroot}
	in.collectAsmReferences(standard, path, &Config{}, indirect)
	if in.opaqueAsmPkg != standard.PkgPath {
		t.Fatalf("indirect assembly call treated as closed: %+v", in)
	}
}

func TestWasiSizePruningEligibility(t *testing.T) {
	t.Setenv("LDFLAGS", "")
	src := "package main\nfunc main() {}"
	ssaPkg := buildWasmReflectTestProgram(t, src)
	for _, test := range []struct {
		name  string
		flags string
		decl  string
		arch  crosscompile.WasmProfile
		proof bool
	}{
		{name: "closed", arch: crosscompile.WasmProfileW32, proof: true},
		{name: "known C export", decl: "//export main\n", arch: crosscompile.WasmProfileW32, proof: true},
		{name: "unknown export", decl: "//go:wasmexport hostMain\n", arch: crosscompile.WasmProfileW32},
		{name: "explicit linker root", flags: "-Wl,--export=hostMain", arch: crosscompile.WasmProfileW32},
		{name: "JS", arch: crosscompile.WasmProfileJ32},
	} {
		t.Run(test.name, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "wasip1", GOARCH: "wasm"})
			defer prog.Dispose()
			file, err := parser.ParseFile(token.NewFileSet(), "main.go",
				"package main\n"+test.decl+"func main() {}", parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			pkg := &packages.Package{PkgPath: "main", Name: "main", Types: ssaPkg.Pkg, Syntax: []*ast.File{file}}
			conf := &Config{
				Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "wasip1", Goarch: "wasm",
				OptLevel: optlevel.Oz, WasmOpt: WasmOptSize, PrintCommands: true,
				LinkOptions: LinkOptions{ExternalLinkerFlags: test.flags},
			}
			ctx := &context{
				mode: ModeBuild, prog: prog, progSSA: ssaPkg.Prog, buildConf: conf,
				initial: []*packages.Package{pkg},
				pkgByID: map[string]*aPackage{"main": {Package: pkg}},
				crossCompile: crosscompile.Export{
					WasmProfile: test.arch, WasmProvider: crosscompile.WasmProviderWASI,
				},
			}
			inputs := ctx.wasiSizePruningInputs(t.TempDir())
			configureUnusedGlobalInits(ctx, inputs)
			configureWasiReleaseDCE(ctx, inputs)
			if got := conf.deadcodeDropEnabled(); got != test.proof {
				t.Fatalf("WASI -Oz pruning = %v, want %v", got, test.proof)
			}
		})
	}
}
