package build

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// The default-path test remains independent. This test deliberately selects
// the patched SDK entry for the main fixture's real Type.Equal function values,
// without manufacturing a gc funcval or changing LLGo's environment transport.
func TestRuntimeEqualityPatchedPublicEntriesExecute(t *testing.T) {
	source, err := os.ReadFile("testdata/runtime-equality/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(source), "func main() {") != 1 {
		t.Fatal("the original 21-check fixture must have exactly one main")
	}
	text := strings.Replace(string(source), "func main() {", "func originalChecks() {", 1)
	text += runtimeEqualityPublicChecks
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}

	conf := NewDefaultConf(ModeBuild)
	conf.OptLevel, conf.LTO = optlevel.O0, lto.Off
	conf.ForceRebuild = true // Observe actual SDK definitions, not cached claims.
	conf.OutFile = filepath.Join(dir, "public-equality")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	const private = llssa.PkgRuntime + ".memequal_varlen"
	const public = "runtime.memequal_varlen"
	redirects, sdkModules := 0, 0
	var declarationType, definitionType llvm.Type
	var declarationEnv, definitionEnv []llvm.Attribute
	conf.ModuleHook = func(pkg Package) {
		module := pkg.LPkg.Module()
		if pkg.PkgPath == "internal/bytealg" {
			sdkModules++
			for _, name := range []string{"runtime.memequal", public} {
				body := module.NamedFunction(name)
				if body.IsNil() || body.IsDeclaration() || body.BasicBlocksCount() == 0 {
					t.Fatal("patched public equality entry lacks its actual body", name)
				}
			}
			body := module.NamedFunction(public)
			definitionType = body.GlobalValueType()
			definitionEnv = body.GetAttributesAtIndex(1)
			if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
				t.Fatal("patched SDK module is invalid", err)
			}
		}
		if !slices.Contains(pkg.GoFiles, mainFile) {
			return // Never redirect the runtime or another dependency's code.
		}
		entry := pkg.LPkg.FuncOf(private)
		declaration := module.NamedFunction(private)
		if entry == nil || !entry.NeedsEnv() || declaration.IsNil() ||
			!declaration.IsDeclaration() || declaration.FirstUse().IsNil() {
			t.Fatal("fixture lacks a used, compiler-produced env-bearing equality declaration")
		}
		if !module.NamedFunction(public).IsNil() {
			t.Fatal("fixture already has a competing public varlen declaration")
		}
		declarationType = declaration.GlobalValueType()
		declarationEnv = declaration.GetAttributesAtIndex(1)
		if declaration.ParamsCount() != 3 || len(declarationEnv) == 0 {
			t.Fatal("native equality declaration lost its real hidden environment ABI")
		}
		callConv := declaration.FunctionCallConv()
		attributes := make([][]llvm.Attribute, declaration.ParamsCount()+2)
		for index := range attributes {
			attributes[index] = declaration.GetAttributesAtIndex(index - 1)
		}
		declaration.SetName(public) // Only select the code entry; keep real Type* captures.
		if declaration.GlobalValueType() != declarationType || declaration.FunctionCallConv() != callConv {
			t.Fatal("public-entry selection changed the actual function type or attributes")
		}
		for index, before := range attributes {
			if !reflect.DeepEqual(declaration.GetAttributesAtIndex(index-1), before) {
				t.Fatal("public-entry selection changed function, return, or parameter attributes")
			}
		}
		redirects++
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal("explicit public-entry fixture is invalid", err)
		}
	}
	if _, err := Do([]string{mainFile}, conf); err != nil {
		t.Fatal("build the real patched-public-entry fixture", err)
	}
	if redirects != 1 || sdkModules != 1 {
		t.Fatalf("observed %d exact main redirects and %d actual SDK body modules, want 1 each", redirects, sdkModules)
	}
	if declarationType != definitionType || !reflect.DeepEqual(declarationEnv, definitionEnv) {
		t.Fatal("the compiler-produced caller and actual SDK body have different environment ABIs")
	}
	var output bytes.Buffer
	err = runRunnerCommand(commandEnv{environ: os.Environ()}, conf.OutFile, nil,
		runnerDetails{phase: "public-equality-owner-test", timeout: 30 * time.Second}, &output, &output)
	if err != nil {
		t.Fatalf("execute actual patched public entries: %v\n%s", err, output.String())
	}
	want := "equality checks 21\npublic equality checks 25"
	if got := strings.TrimSpace(output.String()); got != want {
		t.Fatalf("patched public-entry checks = %q, want %q", got, want)
	}
}

const runtimeEqualityPublicChecks = `

//go:linkname publicMemequal runtime.memequal
func publicMemequal(a, b unsafe.Pointer, size uintptr) bool

func main() {
	// The exact main-module entry redirect makes the original regular-memory
	// Type.Equal calls execute the public varlen body with their real Type*.
	originalChecks()
	left := [3]uint64{1, 2, 3}
	right := left
	a, b := unsafe.Pointer(&left), unsafe.Pointer(&right)
	size := unsafe.Sizeof(left)
	check("public fixed zero size", publicMemequal(nil, nil, 0))
	check("public fixed same address", publicMemequal(a, a, size))
	check("public fixed equal bytes", publicMemequal(a, b, size))
	*(*byte)(unsafe.Add(b, size-1)) ^= 0x80
	check("public fixed last-byte inequality", !publicMemequal(a, b, size))
	println("public equality checks", checks)
}
`
