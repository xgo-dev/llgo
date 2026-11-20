package build

import (
	"slices"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestEmbeddedPrintfFloatRoot(t *testing.T) {
	for _, key := range []string{"LDFLAGS", "CFLAGS", "CCFLAGS", "CGO_CFLAGS", "CGO_LDFLAGS"} {
		t.Setenv(key, "")
	}
	const prefix = `package main
func cstr(string) *byte
func printf(*byte, ...any)
func fprintf(int, *byte, ...any)
func warn(*byte, ...any)
type Format byte
func (*Format) Printf(...any) {}
`
	for _, test := range []struct {
		name, code string
		custom     bool
		keep       bool
	}{
		{"integer", `func main() { printf(cstr("%% %08x %.*s\n"), 42, 1, cstr("ok")) }`, false, false},
		{"stream", `func main() { fprintf(2, cstr("%lld"), 42) }`, false, false},
		{"float", `func main() { printf(cstr("%2$*1$.*3$Le"), 8, 1.5, 2) }`, false, true},
		{"dynamic", `func main() { output(cstr("%g")) }; func output(s *byte) { printf(s, 1.25) }`, false, true},
		{"bare C symbol", `func main() { printf(cstr("%f"), 1.25) }`, false, true},
		{"function value", `var f = printf; func main() { f(cstr("%f"), 1.25) }`, false, true},
		{"foreign alias", `func main() { warn(cstr("%f"), 1.25) }`, false, true},
		{"method", `func main() { (*Format)(nil).Printf(1.25) }`, false, true},
		{"alternate archive source", `func main() { printf(cstr("%f"), 1.25) }`, false, true},
		{"link override", `func main() {}`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ssaPkg := buildWasmReflectTestProgram(t, prefix+test.code)
			prog := llssa.NewProgram(&llssa.Target{GOOS: "none", GOARCH: "riscv32"})
			defer prog.Dispose()
			for name, link := range map[string]string{"cstr": "llgo.cstr", "printf": "C.printf", "fprintf": "C.fprintf", "warn": "C.warn"} {
				prog.SetLinkname(llssa.FullName(ssaPkg.Pkg, name), link)
			}
			if test.name == "bare C symbol" {
				prog.SetLinkname(llssa.FullName(ssaPkg.Pkg, "printf"), "printf")
			}
			if test.name == "method" {
				prog.SetLinkname("main.(*Format).Printf", "C.printf")
			}
			pkg := &packages.Package{Name: "main", PkgPath: "main", Types: ssaPkg.Pkg}
			ctx := &context{
				prog: prog, progSSA: ssaPkg.Prog, initial: []*packages.Package{pkg},
				buildConf:    &Config{Target: "esp32c3-basic", BuildMode: BuildModeExe, OptLevel: optlevel.Oz, LTO: lto.Full, PrintCommands: true},
				crossCompile: crosscompile.Export{LDFLAGS: []string{"--gc-sections", espPrintfFloatRoot}},
			}
			if test.custom {
				ctx.buildConf.LinkOptions.ExternalLinkerFlags = "-lfmt"
			}
			archive := &aPackage{Package: pkg, SSA: ssaPkg}
			if test.name == "alternate archive source" {
				archive.SSA = nil
				archive.AltPkg = &packages.Cached{Package: pkg, Types: ssaPkg.Pkg}
			}
			configureEmbeddedPrintf(ctx, []*aPackage{archive})
			if got := slices.Contains(ctx.crossCompile.LDFLAGS, espPrintfFloatRoot); got != test.keep {
				t.Fatalf("keep float root = %v, want %v", got, test.keep)
			}
			if !slices.Contains(ctx.crossCompile.LDFLAGS, "--gc-sections") {
				t.Fatal("printf proof removed a different linker flag")
			}
		})
	}
}

func TestIntegerPrintfFormat(t *testing.T) {
	for _, format := range []string{"hello", "%%", "%08ld %.*s", "%x\x00%f", "%zu", "%p"} {
		if !integerPrintfFormat(format) {
			t.Errorf("integer format %q rejected", format)
		}
	}
	for _, format := range []string{"%", "%lf", "%+A", "%-e", "%g", "%20F", "%q"} {
		if integerPrintfFormat(format) {
			t.Errorf("unknown or floating format %q accepted", format)
		}
	}
}
