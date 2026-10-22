package plan9asm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	llpackages "github.com/xgo-dev/llgo/internal/packages"
	xpackages "golang.org/x/tools/go/packages"
)

func TestTranslatePreprocessActiveInclude(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/include", "package include\nfunc Included()\n")
	pkg.Dir = t.TempDir()
	header := filepath.Join(pkg.Dir, "implementation.h")
	if err := os.WriteFile(header, []byte("TEXT ·Included(SB),4,$0-0\nRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tr, err := TranslateSourceModuleForPkg(pkg, filepath.Join(pkg.Dir, "entry.s"),
		[]byte("#include \"implementation.h\"\n"), "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Module.Dispose()
	if len(tr.Functions) != 1 || tr.Functions[0].ResolvedSymbol != "example.com/include.Included" {
		t.Fatalf("active include was not translated: %+v", tr.Functions)
	}
}

func TestTranslatePreprocessMissingActiveInclude(t *testing.T) {
	pkg := mustTestPackage(t, "example.com/include", "package include\nfunc Included()\n")
	pkg.Dir = t.TempDir()
	tr, err := TranslateSourceModuleForPkg(pkg, filepath.Join(pkg.Dir, "entry.s"),
		[]byte("#include \"missing-active.h\"\nTEXT ·Included(SB),4,$0-0\nRET\n"), "linux", "amd64")
	if err == nil {
		tr.Module.Dispose()
		t.Fatal("missing active include was silently discarded")
	}
}

func writePreprocessFile(t *testing.T, file, source string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func preprocessFixture(t *testing.T) (*llpackages.Package, string, TranslateOptions) {
	t.Helper()
	dir := t.TempDir()
	pkg := mustTestPackage(t, "example.com/include", `package include
const Count = 17
type Item struct { A byte; B uintptr }
func Included() int32
`)
	pkg.Dir = filepath.Join(dir, "module", "pkg")
	pkg.Module = &xpackages.Module{Dir: filepath.Join(dir, "module")}
	if err := os.MkdirAll(pkg.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "selected-sdk")
	writePreprocessFile(t, filepath.Join(root, "VERSION"), "go1.27.2\n")
	return pkg, filepath.Join(pkg.Dir, "entry.s"), TranslateOptions{GOROOT: root}
}

func TestPreprocessFixedPackageSearchAndSelectedSDK(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	writePreprocessFile(t, filepath.Join(pkg.Dir, "nested", "first.h"), "#include \"choice.h\"\n")
	writePreprocessFile(t, filepath.Join(pkg.Dir, "nested", "choice.h"), "#define CHOICE 91\n")
	writePreprocessFile(t, filepath.Join(pkg.Module.Dir, "choice.h"), "#define CHOICE 92\n")
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "pkg", "include", "choice.h"), "#define CHOICE 73\n")
	source := []byte("#include \"nested/first.h\"\nTEXT ·Included(SB),4,$0-4\nMOVL $CHOICE,ret+0(FP)\nRET\n")
	for _, test := range []struct{ packageHeader, want string }{
		{"#define CHOICE 42\n", "$42"}, {"", "$73"},
	} {
		header := filepath.Join(pkg.Dir, "choice.h")
		if test.packageHeader != "" {
			writePreprocessFile(t, header, test.packageHeader)
		} else if err := os.Remove(header); err != nil {
			t.Fatal(err)
		}
		got, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
		if err != nil || !bytes.Contains(got, []byte(test.want)) {
			t.Fatalf("fixed search output = %q, %v; want %s", got, err, test.want)
		}
	}
	if err := os.Remove(filepath.Join(opt.GOROOT, "pkg", "include", "choice.h")); err != nil {
		t.Fatal(err)
	}
	if _, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt); err == nil {
		t.Fatal("nested header parent or Module.Dir silently became an include search directory")
	}
	// Module.Dir authorizes an explicit relative path, never an implicit search.
	got, err := PreprocessSourceForPkg(pkg, file, []byte("#include \"../choice.h\"\nCHOICE\n"), "linux", "amd64", opt)
	if err != nil || !bytes.Contains(got, []byte("92")) {
		t.Fatalf("explicit module-local header = %q, %v", got, err)
	}
}

func TestPreprocessOverlayDeletionErrorsAndInactiveIncludes(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	header := filepath.Join(pkg.Dir, "choice.h")
	writePreprocessFile(t, header, "#define CHOICE 41\n")
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "pkg", "include", "choice.h"), "#define CHOICE 99\n")
	opt.Overlay = map[string][]byte{header: []byte("#define CHOICE 43\n")}
	source := []byte("#include \"choice.h\"\nCHOICE\n")
	got, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
	if err != nil || !bytes.Contains(got, []byte("43")) {
		t.Fatalf("selected overlay = %q, %v", got, err)
	}
	opt.Overlay[header] = nil
	if _, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt); err == nil {
		t.Fatal("deleted active overlay fell back to disk or the SDK")
	}
	opt.Overlay[file] = nil
	if _, err := PreprocessFileForPkg(pkg, file, "linux", "amd64", opt); err == nil {
		t.Fatal("deleted assembly overlay was accepted")
	}
	// The shared parser must not invoke even a failing resolver for this branch.
	inactive := []byte("#ifdef NEVER_DEFINED\n#include \"choice.h\"\n#include malformed inactive operand\n#endif\nRET\n")
	if _, err := PreprocessSourceForPkg(pkg, file, inactive, "linux", "amd64", opt); err != nil {
		t.Fatalf("inactive include was read or validated: %v", err)
	}
	delete(opt.Overlay, header)
	if err := os.Remove(header); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(header, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt); err == nil {
		t.Fatal("active non-regular header silently fell back to the SDK")
	}
}

func TestPreprocessGoAssemblyHeaderSelectedTypesAndActualPriority(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	source := []byte("#include \"go_asm.h\"\nMOVL $Item_B,AX\nMOVL $const_Count,BX\n")
	for _, test := range []struct{ arch, offset string }{{"386", "$4"}, {"amd64", "$8"}} {
		got, err := PreprocessSourceForPkg(pkg, file, source, "linux", test.arch, opt)
		if err != nil || !bytes.Contains(got, []byte(test.offset)) || !bytes.Contains(got, []byte("$17")) {
			t.Fatalf("gc/%s generated header = %q, %v", test.arch, got, err)
		}
	}
	actual := filepath.Join(pkg.Dir, "go_asm.h")
	writePreprocessFile(t, actual, "#define Item_B 81\n#define const_Count 82\n")
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "pkg", "include", "go_asm.h"), "#define Item_B 91\n#define const_Count 92\n")
	got, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
	if err != nil || !bytes.Contains(got, []byte("$81")) || !bytes.Contains(got, []byte("$82")) {
		t.Fatalf("actual package go_asm.h priority = %q, %v", got, err)
	}
	opt.Overlay = map[string][]byte{actual: nil}
	if _, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt); err == nil {
		t.Fatal("deleted actual go_asm.h was replaced with generated declarations")
	}
	opt.Overlay = nil
	if err := os.Remove(actual); err != nil {
		t.Fatal(err)
	}
	got, err = PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
	if err != nil || !bytes.Contains(got, []byte("$17")) || bytes.Contains(got, []byte("$91")) {
		t.Fatalf("generated go_asm.h must precede conflicting SDK header: %q, %v", got, err)
	}
	pkg.Types = nil
	if _, err := PreprocessSourceForPkg(pkg, file, []byte("RET\n"), "linux", "amd64", opt); err != nil {
		t.Fatalf("ordinary preprocessing unexpectedly required package types: %v", err)
	}
	if err := os.Remove(filepath.Join(opt.GOROOT, "pkg", "include", "go_asm.h")); err != nil {
		t.Fatal(err)
	}
	if _, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt); err == nil {
		t.Fatal("go_asm.h was invented without selected types")
	}
}

func TestPreprocessExplicitTargetFeatures(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	for _, name := range []string{"GOAMD64", "GO386", "GOARM", "GOARM64"} {
		t.Setenv(name, "invalid-ambient-value")
	}
	for _, test := range []struct {
		arch, macro string
		options     TranslateOptions
	}{
		{"386", "GO386_softfloat", TranslateOptions{GO386: "softfloat"}},
		{"amd64", "GOAMD64_v3", TranslateOptions{GOAMD64: "v3"}},
		{"arm", "GOARM_6", TranslateOptions{GOARM: "6,softfloat"}},
		{"arm64", "GOARM64_LSE", TranslateOptions{GOARM64: "v8.1"}},
	} {
		test.options.GOROOT = opt.GOROOT
		source := []byte("#ifdef " + test.macro + "\nSELECTED\n#else\nWRONG\n#endif\n")
		got, err := PreprocessSourceForPkg(pkg, file, source, "linux", test.arch, test.options)
		if err != nil || string(got) != "SELECTED\n" {
			t.Fatalf("explicit %s = %q, %v", test.arch, got, err)
		}
	}
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "VERSION"), "go1.20.14\n")
	got, err := PreprocessSourceForPkg(pkg, file, []byte("#ifdef GOARM_7\nWRONG\n#else\nOLD_SDK\n#endif\n"), "linux", "arm", opt)
	if err != nil || string(got) != "OLD_SDK\n" {
		t.Fatalf("selected old SDK macro semantics = %q, %v", got, err)
	}
}

func TestPreprocessSelectedSDKExperimentPackageRole(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	opt.ToolTags = []string{"amd64.v1", "goexperiment.jsonv2"}
	main := `package main
func main() {
 if objabi.LookupPkgSpecial(ctxt.Pkgpath).AllowAsmABI {
  for _, exp := range buildcfg.Experiment.Enabled() { flags.D = append(flags.D, "GOEXPERIMENT_"+exp) }
 }
}`
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "src/cmd/asm/main.go"), main)
	roles := filepath.Join(opt.GOROOT, "src/cmd/internal/objabi/pkgspecial.go")
	writePreprocessFile(t, roles, "package objabi\nvar allowAsmABIPkgs = []string{\"runtime\", \"reflect\"}\n")
	source := []byte("#ifdef GOEXPERIMENT_jsonv2\nENABLED\n#else\nORDINARY\n#endif\n")
	for _, test := range []struct{ path, want string }{{"runtime", "ENABLED\n"}, {"example.com/user", "ORDINARY\n"}} {
		pkg.PkgPath = test.path
		got, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
		if err != nil || string(got) != test.want {
			t.Fatalf("SDK role %s = %q, %v", test.path, got, err)
		}
	}
	for _, declaration := range []string{"var other = []string{}", "var allowAsmABIPkgs = computed()", "var allowAsmABIPkgs = []string{variable}"} {
		writePreprocessFile(t, roles, "package objabi\n"+declaration+"\n")
		if _, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt); err == nil {
			t.Fatalf("missing/nonliteral package-role registration accepted: %s", declaration)
		}
	}
	writePreprocessFile(t, filepath.Join(opt.GOROOT, "src/cmd/asm/main.go"), "package main\nfunc main() {}\n")
	got, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
	if err != nil || string(got) != "ORDINARY\n" {
		t.Fatalf("legacy SDK invented experiment macro = %q, %v", got, err)
	}
}

func TestPreprocessSharedDepthAndResourceBounds(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	writePreprocessFile(t, filepath.Join(pkg.Dir, "cycle.h"), "#include \"cycle.h\"\n")
	if _, err := PreprocessSourceForPkg(pkg, file, []byte("#include \"cycle.h\"\n"), "linux", "amd64", opt); err == nil || !strings.Contains(err.Error(), "32-level") {
		t.Fatalf("shared include depth bound = %v", err)
	}
	if _, err := PreprocessSourceForPkg(pkg, file, bytes.Repeat([]byte(" "), maxAssemblySourceBytes+1), "linux", "amd64", opt); err == nil {
		t.Fatal("oversized root source was accepted")
	}
	opt.Overlay = map[string][]byte{filepath.Join(pkg.Dir, "large.h"): bytes.Repeat([]byte(" "), maxAssemblySourceBytes+1)}
	if _, err := PreprocessSourceForPkg(pkg, file, []byte("#include \"large.h\"\n"), "linux", "amd64", opt); err == nil {
		t.Fatal("oversized active overlay was accepted")
	}
	var macros strings.Builder
	macros.WriteString("#define M0 RET\n")
	for i := 1; i <= 20; i++ {
		fmt.Fprintf(&macros, "#define M%d M%d M%d\n", i, i-1, i-1)
	}
	macros.WriteString("M20\n")
	if _, err := PreprocessSourceForPkg(pkg, file, []byte(macros.String()), "linux", "amd64", opt); err == nil || !strings.Contains(err.Error(), "bound") {
		t.Fatalf("shared live expansion budget = %v", err)
	}
}

func TestPreprocessExpandedSourceReuseRetainsValidation(t *testing.T) {
	pkg, file, opt := preprocessFixture(t)
	header := filepath.Join(pkg.Dir, "implementation.h")
	writePreprocessFile(t, header, "TEXT ·Included(SB),4,$0-4\nMOVL $42,ret+0(FP)\nRET\n")
	source := []byte("#include \"implementation.h\"\n")
	expanded, err := PreprocessSourceForPkg(pkg, file, source, "linux", "amd64", opt)
	if err != nil || !HasTextAsmSource(expanded) {
		t.Fatalf("expanded TEXT inspection = %q, %v", expanded, err)
	}
	if err := os.Remove(header); err != nil {
		t.Fatal(err)
	}
	opt.SourcePreprocessed = true
	tr, err := TranslateSourceModuleForPkgWithOptions(pkg, file, expanded, "linux", "amd64", opt)
	if err != nil {
		t.Fatal(err)
	}
	tr.Module.Dispose()
	opt.GOAMD64 = "v99"
	if tr, err := TranslateSourceModuleForPkgWithOptions(pkg, file, expanded, "linux", "amd64", opt); err == nil {
		tr.Module.Dispose()
		t.Fatal("preprocessed source bypassed target validation")
	}
	opt.GOAMD64 = ""
	pkg.Types = nil
	if tr, err := TranslateSourceModuleForPkgWithOptions(pkg, file, expanded, "linux", "amd64", opt); err == nil {
		tr.Module.Dispose()
		t.Fatal("preprocessed source bypassed selected signature/type validation")
	}
}
