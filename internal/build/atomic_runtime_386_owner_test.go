//go:build go1.27

package build

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
	"github.com/xgo-dev/llvm"
)

// Independent 386 source roles: five other entries already have SDK Go bodies,
// and ARM64's LoadAcq64/StoreRel64 do not belong to this architecture's API.
func runtimeAtomic386Rows() []runtimeAtomicRow {
	var rows []runtimeAtomicRow
	add := func(family, value, delta, names string) {
		for _, name := range strings.Fields(names) {
			rows = append(rows, runtimeAtomicRow{name, family, value, delta})
		}
	}
	add("load", "uint64", "", "Load64")
	add("load", "int32", "", "Loadint32")
	add("load", "int64", "", "Loadint64")
	add("load", "uintptr", "", "Loaduintptr")
	add("load", "uint", "", "Loaduint")
	add("store", "uint8", "", "Store8")
	add("store", "uint32", "", "Store StoreRel")
	add("store", "uint64", "", "Store64")
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
	add("store-pointer", "unsafe.Pointer", "", "StorepNoWB")
	add("cas-pointer", "unsafe.Pointer", "", "Casp1")
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	return rows
}

func TestRuntimeAtomic386CompleteSDKAndBodyOwners(t *testing.T) {
	dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic")
	assembly, err := os.ReadFile(filepath.Join(dir, "atomic_386.s"))
	if err != nil {
		t.Fatal(err)
	}
	rows := runtimeAtomic386Rows()
	if len(rows) != 42 {
		t.Fatal("the independent complete 386 owner denominator changed", len(rows))
	}
	var names []string
	for _, row := range rows {
		names = append(names, "·"+row.name)
	}
	actual := runtimeEqualityPublicTexts(string(assembly))
	for i := range actual {
		actual[i] = strings.ReplaceAll(actual[i], ".", "·")
	}
	sort.Strings(names)
	sort.Strings(actual)
	if !reflect.DeepEqual(actual, names) {
		t.Fatal("the complete actual 386 TEXT inventory differs", actual, names)
	}
	declarations := make(map[string][]byte)
	for _, name := range []string{"stubs.go", "atomic_386.go"} {
		declarations[name], err = os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
	}
	originals, err := atomicSourcePatchFunctions([][]byte{declarations["stubs.go"], declarations["atomic_386.go"]})
	if err != nil {
		t.Fatal(err)
	}
	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), sourcePatchBuildContext{
		goos: "linux", goarch: "386", goversion: runtime.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var generated [][]byte
	for path, source := range overlay {
		if filepath.Dir(path) == dir && strings.HasPrefix(filepath.Base(path), "z_llgo_patch_") {
			generated = append(generated, source)
		}
	}
	bodies, err := atomicSourcePatchFunctions(generated)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			fixture := runtimeAtomicFunctions(t, []byte("package atomic\nimport \"unsafe\"\n"+row.declaration()))[row.name]
			original, body := originals[row.name], bodies[row.name]
			if original == nil || original.Body != nil || runtimeAtomicSignature(t, original) != runtimeAtomicSignature(t, fixture) {
				t.Fatal("actual SDK declaration differs from the independent 386 role", row.name)
			}
			if body == nil || body.Body == nil || len(body.Body.List) == 0 ||
				runtimeAtomicSignature(t, body) != runtimeAtomicSignature(t, original) {
				t.Fatal("the complete native 386 owner lacks one real, exactly typed Go body", row.name)
			}
		})
	}
	if string(overlay[filepath.Join(dir, "atomic_386.s")]) != "// replaced by LLGo source patch\n" {
		t.Error("the complete native 386 assembly has no runtime body owner")
	}
	selected := declarations["atomic_386.go"]
	if source, exists := overlay[filepath.Join(dir, "atomic_386.go")]; exists {
		selected = source
	}
	retained := runtimeAtomicFunctions(t, selected)
	var originalGo []string
	for name, original := range originals {
		if original.Body == nil {
			continue
		}
		originalGo = append(originalGo, name)
		current := retained[name]
		var before, after bytes.Buffer
		if err := format.Node(&before, token.NewFileSet(), original); err != nil {
			t.Fatal(err)
		}
		if current == nil || current.Body == nil {
			t.Fatal("an existing actual SDK Go body was discarded", name)
		}
		if err := format.Node(&after, token.NewFileSet(), current); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before.Bytes(), after.Bytes()) || bodies[name] != nil {
			t.Fatal("the 386 assembly owner replaced or duplicated an existing SDK Go definition", name)
		}
	}
	sort.Strings(originalGo)
	if !reflect.DeepEqual(originalGo, []string{"Load", "Load8", "LoadAcq", "LoadAcquintptr", "Loadp"}) {
		t.Fatal("the independently retained original Go body inventory changed", originalGo)
	}
	for _, name := range []string{"atomic_arm64.s", "atomic_amd64.s", "atomic_arm.s"} {
		if _, replaced := overlay[filepath.Join(dir, name)]; replaced {
			t.Fatal("the 386 owner swallowed another architecture's assembly", name)
		}
	}
}

// The operation oracle is shared, but uintptr/uint are genuinely i32 here.
// Actual Go signatures and pointer roles are not rewritten by this test.
func runtimeAtomic386RequireOperations(t *testing.T, module llvm.Module) {
	t.Helper()
	for _, row := range runtimeAtomic386Rows() {
		entry := module.NamedFunction("internal/runtime/atomic." + row.name)
		if entry.IsNil() || entry.IsDeclaration() {
			t.Fatal("missing actual 386 atomic operation entry", row.name)
		}
		if row.value == "uintptr" || row.value == "uint" {
			row.value = "uint32"
		}
		runtimeAtomicRequireOperation(t, module, entry, row)
	}
}

func TestRuntimeAtomic386RejectsIncompleteOwner(t *testing.T) {
	var assembly, patch, declarations strings.Builder
	patch.WriteString("//go:build 386 && go1.27\n\npackage atomic\nimport \"unsafe\"\n")
	declarations.WriteString("package atomic\nimport \"unsafe\"\n")
	for _, row := range runtimeAtomic386Rows() {
		fmt.Fprintf(&assembly, "TEXT ·%s(SB),4,$0\nRET\n", row.name)
		fmt.Fprintf(&patch, "%s { panic(\"syntax-only owner fixture\") }\n", row.declaration())
		fmt.Fprintln(&declarations, row.declaration())
	}
	for _, mutation := range []string{
		"valid", "missing-body", "wrong-uintptr-width", "new-public-text", "missing-text", "duplicate-text",
		"unowned-public-body", "replace-existing-Go-body", "skip-all", "overlay-new-text", "wrong-sdk-declaration",
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
			case "wrong-uintptr-width":
				body = strings.Replace(body, "func Loaduintptr(ptr *uintptr) uintptr", "func Loaduintptr(ptr *uint64) uint64", 1)
			case "new-public-text":
				original += "TEXT ·FutureAtomic(SB),4,$0\nRET\n"
			case "missing-text":
				original = strings.Replace(original, "TEXT ·And32(SB),4,$0\nRET\n", "", 1)
			case "duplicate-text":
				original += "TEXT ·And32(SB),4,$0\nRET\n"
			case "unowned-public-body":
				body += "\nfunc Unowned386() {}\n"
			case "replace-existing-Go-body":
				body += "\nfunc Load(ptr *uint32) uint32 { return 0 }\n"
			case "skip-all":
				body += "\n//llgo:skipall\n"
			}
			asmFile := assemblyLanguageTestWrite(t, dir, "atomic_386.s", original)
			assemblyLanguageTestWrite(t, dir, "stubs.go", declarations.String())
			assemblyLanguageTestWrite(t, dir, "atomic_386.go", "package atomic\nfunc Load(ptr *uint32) uint32 { return *ptr }\n")
			other := assemblyLanguageTestWrite(t, dir, "other_386.s", "TEXT ·Other(SB),4,$0\nRET\n")
			assemblyLanguageTestWrite(t, patchDir, "atomic_386.go", body)
			base := make(map[string][]byte)
			if mutation == "overlay-new-text" {
				base[asmFile] = []byte(original + "TEXT ·FutureAtomic(SB),4,$0\nRET\n")
			}
			if mutation == "wrong-sdk-declaration" {
				base[filepath.Join(dir, "stubs.go")] = []byte(strings.Replace(declarations.String(), "func Loaduintptr(ptr *uintptr) uintptr", "func Loaduintptr(ptr *uint64) uint64", 1))
			}
			ctx := sourcePatchBuildContext{goos: "linux", goarch: "386", goversion: "go1.27.2"}
			_, overlay, _, err := applySourcePatchForPkg(base, nil, runtimeDir, goroot, "internal/runtime/atomic", ctx)
			if mutation != "valid" {
				if err == nil {
					t.Fatal("incomplete/changed whole-file 386 owner was silently accepted", mutation)
				}
				return
			}
			if err != nil || string(overlay[asmFile]) != "// replaced by LLGo source patch\n" {
				t.Fatal("complete 386 owner did not replace exactly atomic_386.s", err)
			}
			if _, changed := overlay[other]; changed {
				t.Fatal("386 ownership suppressed unrelated assembly")
			}
			_, repeated, _, err := applySourcePatchForPkg(overlay, nil, runtimeDir, goroot, "internal/runtime/atomic", ctx)
			if err != nil || !reflect.DeepEqual(repeated, overlay) {
				t.Fatal("intact reused 386 ownership changed", err)
			}
			for _, name := range []string{"stubs.go", "z_llgo_patch_atomic_386.go"} {
				changed := cloneOverlay(repeated)
				path := filepath.Join(dir, name)
				changed[path] = append(changed[path], []byte("\nvar changedSelected386Source int\n")...)
				if _, _, _, err := applySourcePatchForPkg(changed, nil, runtimeDir, goroot, "internal/runtime/atomic", ctx); err == nil {
					t.Fatal("changed reused 386 source borrowed its former owner marker", name)
				}
			}
		})
	}
}

func TestRuntimeAtomic386RetainsOtherSelections(t *testing.T) {
	for _, ctx := range []sourcePatchBuildContext{
		{goos: "linux", goarch: "amd64", goversion: runtime.Version()},
		{goos: "linux", goarch: "arm64", goversion: runtime.Version()},
		{goos: "linux", goarch: "arm", goversion: runtime.Version()},
		{goos: "js", goarch: "wasm", goversion: runtime.Version()},
		{goos: "linux", goarch: "386", goversion: "go1.26.10"},
	} {
		t.Run(ctx.goarch+"/"+ctx.goversion, func(t *testing.T) {
			overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), ctx)
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(runtime.GOROOT(), "src", "internal", "runtime", "atomic")
			for _, name := range []string{"atomic_386.s", "atomic_386.go", "z_llgo_patch_atomic_386.go"} {
				if _, changed := overlay[filepath.Join(dir, name)]; changed {
					t.Fatal("the Go1.27 386 owner changed an unrelated selection", ctx, name)
				}
			}
		})
	}
}
