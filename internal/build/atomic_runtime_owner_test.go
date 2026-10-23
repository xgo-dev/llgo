package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
	"github.com/xgo-dev/llvm"
)

type runtimeAtomicRow struct {
	name, family, value, delta string
}

// This independent inventory describes the complete public atomic_arm64.s
// owner, not only the two stores which exposed its native ABI0 boundary.
func runtimeAtomicRows() []runtimeAtomicRow {
	var rows []runtimeAtomicRow
	add := func(family, value, delta, names string) {
		for _, name := range strings.Fields(names) {
			rows = append(rows, runtimeAtomicRow{name, family, value, delta})
		}
	}
	add("load", "uint8", "", "Load8")
	add("load", "uint32", "", "Load LoadAcq")
	add("load", "uint64", "", "Load64 LoadAcq64")
	add("load", "int32", "", "Loadint32")
	add("load", "int64", "", "Loadint64")
	add("load", "uintptr", "", "Loaduintptr LoadAcquintptr")
	add("load", "uint", "", "Loaduint")
	add("store", "uint8", "", "Store8")
	add("store", "uint32", "", "Store StoreRel")
	add("store", "uint64", "", "Store64 StoreRel64")
	add("store", "int32", "", "Storeint32")
	add("store", "int64", "", "Storeint64")
	add("store", "uintptr", "", "Storeuintptr StoreReluintptr")
	add("cas", "uint32", "", "Cas CasRel")
	add("cas", "uint64", "", "Cas64")
	add("cas", "int32", "", "Casint32")
	add("cas", "int64", "", "Casint64")
	add("cas", "uintptr", "", "Casuintptr")
	add("add", "uint32", "int32", "Xadd")
	add("add", "uint64", "int64", "Xadd64")
	add("add", "int32", "int32", "Xaddint32")
	add("add", "int64", "int64", "Xaddint64")
	add("add", "uintptr", "uintptr", "Xadduintptr")
	add("xchg", "uint8", "", "Xchg8")
	add("xchg", "uint32", "", "Xchg")
	add("xchg", "uint64", "", "Xchg64")
	add("xchg", "int32", "", "Xchgint32")
	add("xchg", "int64", "", "Xchgint64")
	add("xchg", "uintptr", "", "Xchguintptr")
	add("and-void", "uint8", "", "And8")
	add("or-void", "uint8", "", "Or8")
	add("and-void", "uint32", "", "And")
	add("or-void", "uint32", "", "Or")
	add("and", "uint32", "", "And32")
	add("or", "uint32", "", "Or32")
	add("and", "uint64", "", "And64")
	add("or", "uint64", "", "Or64")
	add("and", "uintptr", "", "Anduintptr")
	add("or", "uintptr", "", "Oruintptr")
	add("load-pointer", "unsafe.Pointer", "", "Loadp")
	add("store-pointer", "unsafe.Pointer", "", "StorepNoWB")
	add("cas-pointer", "unsafe.Pointer", "", "Casp1")
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	return rows
}

func (r runtimeAtomicRow) declaration() string {
	parameters, result := "ptr *"+r.value, r.value
	switch r.family {
	case "load-pointer":
		parameters = "ptr unsafe.Pointer"
	case "store-pointer":
		parameters, result = "ptr unsafe.Pointer, value unsafe.Pointer", ""
	case "store", "and-void", "or-void":
		parameters, result = parameters+", value "+r.value, ""
	case "cas", "cas-pointer":
		parameters, result = parameters+", old, next "+r.value, "bool"
	case "add":
		parameters += ", delta " + r.delta
	case "xchg", "and", "or":
		parameters += ", value " + r.value
	}
	return "func " + r.name + "(" + parameters + ") " + result
}

func runtimeAtomicSignature(t *testing.T, function *ast.FuncDecl) string {
	t.Helper()
	fields := func(list *ast.FieldList) string {
		var types []string
		if list != nil {
			for _, field := range list.List {
				var text bytes.Buffer
				if err := format.Node(&text, token.NewFileSet(), field.Type); err != nil {
					t.Fatal(err)
				}
				count := len(field.Names)
				if count == 0 {
					count = 1
				}
				for i := 0; i < count; i++ {
					types = append(types, text.String())
				}
			}
		}
		return strings.Join(types, ",")
	}
	return "(" + fields(function.Type.Params) + ")->(" + fields(function.Type.Results) + ")"
}

func runtimeAtomicFunctions(t *testing.T, source []byte) map[string]*ast.FuncDecl {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "owner.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	functions := make(map[string]*ast.FuncDecl)
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
			if functions[function.Name.Name] != nil {
				t.Fatal("duplicate actual source function", function.Name.Name)
			}
			functions[function.Name.Name] = function
		}
	}
	return functions
}

func TestRuntimeAtomicARM64CompleteSDKAndBodyOwners(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic")
	assembly, err := os.ReadFile(filepath.Join(dir, "atomic_arm64.s"))
	if err != nil {
		t.Fatal(err)
	}
	rows := runtimeAtomicRows()
	if len(rows) != 49 {
		t.Fatal("the independent ARM64 public owner denominator changed", len(rows))
	}
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = "·" + row.name
	}
	actual := runtimeEqualityPublicTexts(string(assembly))
	for i := range actual {
		actual[i] = strings.ReplaceAll(actual[i], ".", "·")
	}
	sort.Strings(names)
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, names) {
		t.Fatal("the complete actual ARM64 TEXT inventory differs", actual, names)
	}
	declarations := make(map[string]*ast.FuncDecl)
	for _, name := range []string{"stubs.go", "atomic_arm64.go"} {
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		for name, function := range runtimeAtomicFunctions(t, source) {
			declarations[name] = function
		}
	}
	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), sourcePatchBuildContext{
		goos: "linux", goarch: "arm64", goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	bodies := make(map[string]*ast.FuncDecl)
	for path, source := range overlay {
		if filepath.Dir(path) == dir && strings.HasPrefix(filepath.Base(path), "z_llgo_patch_") {
			for name, function := range runtimeAtomicFunctions(t, source) {
				bodies[name] = function
			}
		}
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			fixture := runtimeAtomicFunctions(t, []byte("package atomic\nimport \"unsafe\"\n"+row.declaration()))[row.name]
			original, body := declarations[row.name], bodies[row.name]
			if original == nil || runtimeAtomicSignature(t, original) != runtimeAtomicSignature(t, fixture) {
				t.Fatal("the actual SDK declaration disagrees with the independent typed role", row.name)
			}
			if body == nil || body.Body == nil || len(body.Body.List) == 0 ||
				runtimeAtomicSignature(t, body) != runtimeAtomicSignature(t, original) {
				t.Fatal("a complete native owner needs one real, exactly typed Go body", row.name)
			}
		})
	}
	if string(overlay[filepath.Join(dir, "atomic_arm64.s")]) != "// replaced by LLGo source patch\n" {
		t.Error("the complete native ARM64 file has no runtime body owner")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".s") && entry.Name() != "atomic_arm64.s" {
			if _, replaced := overlay[filepath.Join(dir, entry.Name())]; replaced {
				t.Fatal("the ARM64 owner swallowed another assembly file", entry.Name())
			}
		}
	}
}

func TestRuntimeAtomicARM64RejectsIncompleteOwner(t *testing.T) {
	var assembly, patch, declarations strings.Builder
	patch.WriteString("package atomic\nimport \"unsafe\"\n")
	declarations.WriteString("package atomic\nimport \"unsafe\"\n")
	for _, row := range runtimeAtomicRows() {
		fmt.Fprintf(&assembly, "TEXT ·%s(SB),4,$0\nRET\n", row.name)
		fmt.Fprintf(&patch, "%s { panic(\"syntax-only owner fixture\") }\n", row.declaration())
		fmt.Fprintln(&declarations, row.declaration())
	}
	for _, mutation := range []string{
		"valid", "missing-body", "wrong-width", "new-public-text", "missing-text",
		"duplicate-text", "skip-all", "overlay-new-text", "wrong-sdk-declaration",
	} {
		t.Run(mutation, func(t *testing.T) {
			goroot, runtimeDir := t.TempDir(), t.TempDir()
			dir := filepath.Join(goroot, "src", "internal", "runtime", "atomic")
			patchDir := filepath.Join(runtimeDir, "_patch", "internal", "runtime", "atomic")
			for _, path := range []string{dir, patchDir} {
				if err := os.MkdirAll(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			body, original := patch.String(), assembly.String()
			switch mutation {
			case "missing-body":
				body = strings.Replace(body, "func And32(ptr *uint32, value uint32) uint32 { panic(\"syntax-only owner fixture\") }", "func And32(ptr *uint32, value uint32) uint32", 1)
			case "wrong-width":
				body = strings.Replace(body, "func Load8(ptr *uint8)", "func Load8(ptr *uint64)", 1)
			case "new-public-text":
				original += "TEXT ·FutureAtomic(SB),4,$0\nRET\n"
			case "missing-text":
				original = strings.Replace(original, "TEXT ·And32(SB),4,$0\nRET\n", "", 1)
			case "duplicate-text":
				original += "TEXT ·And32(SB),4,$0\nRET\n"
			case "skip-all":
				body += "\n//llgo:skipall\n"
			}
			asmFile := assemblyLanguageTestWrite(t, dir, "atomic_arm64.s", original)
			assemblyLanguageTestWrite(t, dir, "stubs.go", declarations.String())
			assemblyLanguageTestWrite(t, dir, "atomic_arm64.go", "package atomic\n")
			other := assemblyLanguageTestWrite(t, dir, "other_arm64.s", "TEXT ·Other(SB),4,$0\nRET\n")
			assemblyLanguageTestWrite(t, patchDir, "atomic_arm64.go", body)
			base := make(map[string][]byte)
			if mutation == "overlay-new-text" {
				base[asmFile] = []byte(original + "TEXT ·FutureAtomic(SB),4,$0\nRET\n")
			}
			if mutation == "wrong-sdk-declaration" {
				base[filepath.Join(dir, "stubs.go")] = []byte(strings.Replace(declarations.String(), "func Load8(ptr *uint8)", "func Load8(ptr *uint64)", 1))
			}
			_, overlay, _, err := applySourcePatchForPkg(base, nil, runtimeDir, goroot, "internal/runtime/atomic", sourcePatchBuildContext{
				goos: "linux", goarch: "arm64", goversion: "go1.27.2",
			})
			if mutation != "valid" {
				if err == nil {
					t.Fatal("incomplete/changed whole-file owner was silently accepted", mutation)
				}
				return
			}
			if err != nil || string(overlay[asmFile]) != "// replaced by LLGo source patch\n" {
				t.Fatal("complete owner did not replace exactly atomic_arm64.s", err)
			}
			if _, changed := overlay[other]; changed {
				t.Fatal("whole-package assembly suppression replaced an unrelated file")
			}
			_, repeated, _, err := applySourcePatchForPkg(overlay, nil, runtimeDir, goroot, "internal/runtime/atomic", sourcePatchBuildContext{
				goos: "linux", goarch: "arm64", goversion: "go1.27.2",
			})
			if err != nil || !reflect.DeepEqual(repeated, overlay) {
				t.Fatal("an intact reused atomic source view changed ownership", err)
			}
			repeated[filepath.Join(dir, "stubs.go")] = append(repeated[filepath.Join(dir, "stubs.go")], []byte("\nvar changedSelectedSource int\n")...)
			if _, _, _, err := applySourcePatchForPkg(repeated, nil, runtimeDir, goroot, "internal/runtime/atomic", sourcePatchBuildContext{
				goos: "linux", goarch: "arm64", goversion: "go1.27.2",
			}); err == nil {
				t.Fatal("a changed reused declaration view borrowed the previous owner marker")
			}
		})
	}
}

func TestRuntimeAtomicARM64RetainsOtherArchitectures(t *testing.T) {
	// Isolate this actual ARM64 owner: the runtime also has an independently
	// validated 386 owner, which may legitimately change the 386 selection.
	runtimeDir := t.TempDir()
	patchDir := filepath.Join(runtimeDir, "_patch", "internal", "runtime", "atomic")
	if err := os.MkdirAll(patchDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"atomic_arm64.go", "atomic_rmw_arm64.go", "atomic_xchg8_arm64.go"} {
		source, err := os.ReadFile(filepath.Join(env.LLGoRuntimeDir(), "_patch", "internal", "runtime", "atomic", name))
		if err != nil {
			t.Fatal(err)
		}
		assemblyLanguageTestWrite(t, patchDir, name, string(source))
	}
	for _, target := range []struct{ goos, goarch string }{
		{"linux", "amd64"}, {"linux", "386"}, {"linux", "arm"}, {"js", "wasm"},
	} {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
			overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, runtimeDir, runtime.GOROOT(), sourcePatchBuildContext{
				goos: target.goos, goarch: target.goarch, goversion: runtime.Version(),
			})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic")
			for path := range overlay {
				if filepath.Dir(path) == dir {
					t.Fatal("ARM64 body ownership changed another architecture's selection", path)
				}
			}
		})
	}
}

func TestRuntimeAtomicARM64RejectsMismatchedSDKProfile(t *testing.T) {
	// Go1.23 has only 48 entries. Applying that profile to the newer actual
	// SDK must not silently suppress a source group with a different inventory.
	changed, _, _, err := applySourcePatchForPkg(nil, nil, env.LLGoRuntimeDir(), runtime.GOROOT(), "internal/runtime/atomic", sourcePatchBuildContext{
		goos: "linux", goarch: "arm64", goversion: "go1.23.12",
	})
	if err == nil || changed {
		t.Fatal("a mismatched selected SDK profile borrowed an atomic source owner", err)
	}
}

func runtimeAtomicCommand(t *testing.T, binary string, args []string, budget time.Duration) string {
	t.Helper()
	var output bytes.Buffer
	if err := runRunnerCommand(commandEnv{environ: os.Environ()}, binary, args,
		runnerDetails{phase: "native-atomic-owner", timeout: budget}, &output, &output); err != nil {
		t.Fatalf("actual atomic owner command %q: %v\n%s", append([]string{binary}, args...), err, output.String())
	}
	return strings.TrimSpace(output.String())
}

func TestRuntimeAtomicARM64ActualWrappersMatchOriginalGo(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("ARM64 native execution; the independent SDK/body owner test is portable")
	}
	dir := t.TempDir()
	source := runtimeAtomicProgram(t)
	mainFile := assemblyLanguageTestWrite(t, dir, "main.go", source)
	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	original := filepath.Join(dir, "original"+extension)
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go"+extension)
	// Only this reference permits explicit calls to all original SDK entries.
	// It changes no compiler/runtime semantics or global linkname policy.
	runtimeAtomicCommand(t, goTool, []string{"build", "-ldflags=-checklinkname=0", "-o", original, mainFile}, 5*time.Minute)
	want := runtimeAtomicCommand(t, original, nil, 5*time.Minute)
	if want != "atomic wrappers 49 checks 441" {
		t.Fatal("independent original-Go execution denominator changed", want)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OptLevel, conf.LTO = optlevel.O0, lto.Off
	conf.ForceRebuild = true
	conf.OutFile = filepath.Join(dir, "llgo"+extension)
	modules := 0
	conf.ModuleHook = func(pkg Package) {
		if pkg.PkgPath != "internal/runtime/atomic" {
			return
		}
		modules++
		module := pkg.LPkg.Module()
		for _, row := range runtimeAtomicRows() {
			function := module.NamedFunction("internal/runtime/atomic." + row.name)
			if function.IsNil() || function.IsDeclaration() {
				t.Fatal("actual call target lacks its complete Go wrapper body", row.name)
			}
			runtimeAtomicRequireOperation(t, module, function, row)
		}
		if err := llvm.VerifyModule(module, llvm.ReturnStatusAction); err != nil {
			t.Fatal("actual atomic wrapper module is invalid", err)
		}
	}
	if _, err := Do([]string{mainFile}, conf); err != nil {
		t.Fatal("compile actual native atomic wrappers", err)
	}
	if modules != 1 {
		t.Fatal("expected exactly one actual SDK atomic body module", modules)
	}
	if got := runtimeAtomicCommand(t, conf.OutFile, nil, 5*time.Minute); got != want {
		t.Fatalf("LLGo atomic result %q differs from actual original Go %q", got, want)
	}
}

func runtimeAtomicRequireOperation(t *testing.T, module llvm.Module, entry llvm.Value, row runtimeAtomicRow) {
	t.Helper()
	width := "i64"
	switch row.value {
	case "uint8":
		width = "i8"
	case "uint32", "int32":
		width = "i32"
	case "unsafe.Pointer":
		width = "ptr"
	}
	op := strings.Split(row.family, "-")[0]
	wanted := map[string]string{"load": "load atomic " + width, "store": "store atomic " + width,
		"cas": "cmpxchg ", "add": "atomicrmw add ", "xchg": "atomicrmw xchg ",
		"and": "atomicrmw and ", "or": "atomicrmw or "}[op]
	seen, found := make(map[string]bool), 0
	queue := []llvm.Value{entry}
	for len(queue) != 0 {
		function := queue[0]
		queue = queue[1:]
		if seen[function.Name()] {
			continue
		}
		seen[function.Name()] = true
		if len(seen) > 64 {
			t.Fatal("atomic wrapper call graph exceeds its bounded owner")
		}
		for block := function.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
			for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
				text := instruction.String()
				if strings.Contains(text, wanted) && strings.Contains(text, "seq_cst") {
					if (op == "cas" || op == "add" || op == "xchg" || op == "and" || op == "or") &&
						!strings.Contains(text, ", "+width+" ") {
						t.Fatal("atomic wrapper widened its actual native access", row.name, text)
					}
					found++
				}
				if !instruction.IsACallInst().IsNil() {
					callee := module.NamedFunction(instruction.CalledValue().Name())
					if !callee.IsNil() && !callee.IsDeclaration() {
						queue = append(queue, callee)
					}
				}
			}
		}
	}
	if found != 1 {
		t.Fatalf("%s has %d actual %s %s operations, want one sequentially consistent operation", row.name, found, op, width)
	}
}

func TestRuntimeAtomicARM64OriginalSDKTests(t *testing.T) {
	if runtime.GOARCH != "arm64" {
		t.Skip("actual original-Go/LLGo ARM64 SDK test binaries require an ARM64 host")
	}
	const selector = "^Test(Xadduintptr|XadduintptrOnUint64|Unaligned64|And8|And|Or8|Or|BitwiseContended8|BitwiseContended|CasRel|StorepNoWB|And32|And64|Or32|Or64|Xchg8)$"
	// There are sixteen selected roots: fifteen native roots plus the SDK's
	// explicitly 32-bit-only alignment test, whose ARM64 SKIP remains visible.
	sdkDir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic")
	var actual []string
	for _, name := range []string{"atomic_test.go", "atomic_andor_test.go", "xchg8_test.go"} {
		source, err := os.ReadFile(filepath.Join(sdkDir, name))
		if err != nil {
			t.Fatal(err)
		}
		for name := range runtimeAtomicFunctions(t, source) {
			if strings.HasPrefix(name, "Test") {
				actual = append(actual, name)
			}
		}
	}
	sort.Strings(actual)
	wantRoots := runtimeAtomicSDKRoots()
	sort.Strings(wantRoots)
	if !reflect.DeepEqual(actual, wantRoots) {
		t.Fatal("the complete original SDK test inventory changed", actual, wantRoots)
	}
	dir := t.TempDir()
	goTool := filepath.Join(runtime.GOROOT(), "bin", "go")
	original := filepath.Join(dir, "original-sdk.test")
	runtimeAtomicCommand(t, goTool, []string{"test", "-c", "-o", original, "internal/runtime/atomic"}, 5*time.Minute)
	runtimeAtomicCheckSDKTerminals(t, runtimeAtomicCommand(t, original,
		[]string{"-test.run=" + selector, "-test.v", "-test.count=1", "-test.timeout=5m"}, 5*time.Minute))
	conf := NewDefaultConf(ModeTest)
	conf.CompileOnly, conf.ForceRebuild = true, true
	conf.OutFile = filepath.Join(dir, "llgo-sdk.test")
	if _, err := Do([]string{"internal/runtime/atomic"}, conf); err != nil {
		t.Fatal("compile the actual selected SDK atomic tests with LLGo", err)
	}
	runtimeAtomicCheckSDKTerminals(t, runtimeAtomicCommand(t, conf.OutFile,
		[]string{"-test.run=" + selector, "-test.v", "-test.count=1", "-test.timeout=5m"}, 5*time.Minute))
}

func runtimeAtomicSDKRoots() []string {
	return strings.Fields("TestXadduintptr TestXadduintptrOnUint64 TestUnaligned64 TestAnd8 TestAnd TestOr8 TestOr TestBitwiseContended8 TestBitwiseContended TestCasRel TestStorepNoWB TestAnd32 TestAnd64 TestOr32 TestOr64 TestXchg8")
}

func runtimeAtomicCheckSDKTerminals(t *testing.T, output string) {
	t.Helper()
	for _, name := range runtimeAtomicSDKRoots() {
		terminal := "--- PASS: " + name + " ("
		if name == "TestUnaligned64" {
			terminal = "--- SKIP: " + name + " ("
		}
		if strings.Count(output, "=== RUN   "+name+"\n") != 1 || strings.Count(output, terminal) != 1 {
			t.Fatalf("actual SDK root lacks its unique original RUN/terminal: %s\n%s", name, output)
		}
	}
	if strings.Contains(output, "--- FAIL:") || !strings.HasSuffix(output, "PASS") {
		t.Fatal("the actual selected SDK suite did not succeed", output)
	}
	t.Log("actual SDK atomic roots: 15 PASS, one original 32-bit-only alignment SKIP (not execution credit)")
}
