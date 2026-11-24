package build

import (
	"os"
	"path/filepath"
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
				crossCompile: crosscompile.Export{Libc: "newlib-esp32", LDFLAGS: []string{"--gc-sections", espPrintfFloatRoot}},
			}
			if test.custom {
				ctx.buildConf.LinkOptions.ExternalLinkerFlags = "-lfmt"
			}
			archive := &aPackage{Package: pkg, SSA: ssaPkg}
			if test.name == "alternate archive source" {
				archive.SSA = nil
				archive.AltPkg = &packages.Cached{Package: pkg, Types: ssaPkg.Pkg}
			}
			configureEmbeddedPrintf(ctx, []*aPackage{archive}, false, nil)
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

func TestEmbeddedPrintfFloatRuntimeReference(t *testing.T) {
	for _, key := range []string{"LDFLAGS", "CFLAGS", "CCFLAGS", "CGO_CFLAGS", "CGO_LDFLAGS"} {
		t.Setenv(key, "")
	}
	for _, tc := range []struct {
		name, reference string
		args            []string
		runtimeFree     bool
		keep, probe     bool
	}{
		{name: "float reference", reference: "C." + espPrintfFloatMarker, args: []string{"--wrap=malloc"}, keep: true, probe: true},
		{name: "older runtime", keep: true},
		{name: "different symbol", reference: "C.other", keep: true},
		{name: "allocator only", runtimeFree: true, args: []string{"--wrap=calloc"}},
		{name: "extra runtime library", reference: "C." + espPrintfFloatMarker, args: []string{"-lextra"}, keep: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			main := buildWasmReflectTestProgram(t, "package main; func main() {}")
			prog := llssa.NewProgram(&llssa.Target{GOARCH: "riscv32"})
			defer prog.Dispose()
			if tc.reference != "" {
				prog.SetLinkname(llssa.PkgRuntime+".requirePrintfFloat", tc.reference)
			}
			pkg := &packages.Package{Name: "main", PkgPath: "main", Types: main.Pkg}
			ctx := &context{
				prog: prog, progSSA: main.Prog, initial: []*packages.Package{pkg},
				buildConf:           &Config{Target: "esp32c3-basic", BuildMode: BuildModeExe, OptLevel: optlevel.Oz, LTO: lto.Full},
				crossCompile:        crosscompile.Export{Libc: "newlib-esp32", LDFLAGS: []string{espPrintfFloatRoot}},
				embeddedRuntimeFree: tc.runtimeFree,
			}
			if probe := configureEmbeddedPrintf(ctx, []*aPackage{{Package: pkg, SSA: main}}, true, tc.args); probe != tc.probe {
				t.Fatalf("try float-free link = %v, want %v", probe, tc.probe)
			}
			if got := slices.Contains(ctx.crossCompile.LDFLAGS, espPrintfFloatRoot); got != tc.keep {
				t.Fatalf("keep float root = %v, want %v", got, tc.keep)
			}
		})
	}
}

func TestEmbeddedPrintfFloatSpeculativeLink(t *testing.T) {
	for _, test := range []struct {
		name, mode     string
		linked, failed bool
	}{
		{name: "publish reduced link", mode: "write", linked: true},
		{name: "runtime marker selects fallback", mode: "undefined:" + embeddedStartupMarker},
		{name: "float marker selects fallback", mode: "undefined:" + espPrintfFloatMarker},
		{name: "unexpected linker failure", mode: "fail", failed: true},
		{name: "failed artifact publication", mode: "missing-output", failed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LLGO_TEST_LINKER_HELPER", test.mode)
			dir := t.TempDir()
			output := filepath.Join(dir, "firmware.elf")
			if err := os.WriteFile(output, []byte("previous firmware"), 0o644); err != nil {
				t.Fatal(err)
			}
			plan := &mainLinkPlan{
				embeddedStartupEntry:      "compact-entry.o",
				embeddedPrintfFloatMarker: "float-marker.o",
				linkInputs:                []string{"normal-entry.o", "packages.a"},
			}
			ctx := &context{
				buildConf:    &Config{Mode: ModeBuild, BuildMode: BuildModeExe, Goos: "linux", Goarch: "arm"},
				crossCompile: crosscompile.Export{Linker: os.Args[0], LDFLAGS: []string{"retain", espPrintfFloatRoot}},
			}
			linked, err := linkCompactEmbeddedEntry(ctx, plan, output, true)
			if linked != test.linked || (err != nil) != test.failed {
				t.Fatalf("link result = (%t, %v), want linked=%t failed=%t", linked, err, test.linked, test.failed)
			}
			wantOutput := "previous firmware"
			if linked {
				wantOutput = "linked"
			}
			if got, err := os.ReadFile(output); err != nil || string(got) != wantOutput {
				t.Fatalf("firmware output = (%q, %v), want %q", got, err, wantOutput)
			}
			if forceFloat := slices.Contains(ctx.crossCompile.LDFLAGS, espPrintfFloatRoot); forceFloat == linked {
				t.Fatalf("float root retained=%t, linked=%t", forceFloat, linked)
			}
			files, err := os.ReadDir(dir)
			if err != nil || len(files) != 1 {
				t.Fatalf("link probe directory not removed: %v, %v", files, err)
			}
		})
	}
}
