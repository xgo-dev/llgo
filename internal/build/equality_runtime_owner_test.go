package build

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/internal/env"
	llruntime "github.com/xgo-dev/llgo/runtime"
)

func TestRuntimeEqualitySourcePatchOwnsOnlyCompleteEqualFile(t *testing.T) {
	for _, arch := range []string{"386", "amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			goroot := runtime.GOROOT()
			dir := filepath.Join(goroot, "src", "internal", "bytealg")
			equal := filepath.Join(dir, "equal_"+arch+".s")
			original, err := os.ReadFile(equal)
			if err != nil {
				t.Fatal(err)
			}
			if got := runtimeEqualityPublicTexts(string(original)); !reflect.DeepEqual(got, []string{"runtime.memequal", "runtime.memequal_varlen"}) {
				t.Fatal("selected original whole equality owner changed", got)
			}
			overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), goroot, sourcePatchBuildContext{
				goos: "linux", goarch: arch, goversion: runtime.Version(),
			})
			if err != nil {
				t.Fatal(err)
			}
			if string(overlay[equal]) != "// replaced by LLGo source patch\n" {
				t.Fatalf("whole selected equality assembly has no actual runtime body owner: %s", equal)
			}
			if llruntime.SourcePatchReplacesAsmForGOARCH("internal/bytealg", arch) {
				t.Fatal("a two-entry equality bridge must not replace every bytealg assembly file")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			otherSources := 0
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".s" {
					continue
				}
				path := filepath.Join(dir, entry.Name())
				if path == equal {
					continue
				}
				otherSources++
				if _, replaced := overlay[path]; replaced {
					t.Fatal("equality source patch swallowed an unrelated whole assembly owner", path)
				}
			}
			if otherSources == 0 {
				t.Fatal("original bytealg inventory has no other assembly sources")
			}
			runtimeEqualityRequireBodies(t, overlay, dir)
		})
	}
}

func runtimeEqualityPublicTexts(source string) []string {
	var names []string
	for _, match := range regexp.MustCompile(`(?m)^\s*TEXT\s+([^\s(,]+)`).FindAllStringSubmatch(source, -1) {
		name := match[1]
		if strings.Contains(name, "<>") {
			continue
		}
		name = strings.TrimSuffix(strings.TrimSuffix(name, "<ABIInternal>"), "<ABI0>")
		names = append(names, strings.ReplaceAll(name, "·", "."))
	}
	sort.Strings(names)
	return names
}

func runtimeEqualityRequireBodies(t *testing.T, overlay map[string][]byte, dir string) {
	t.Helper()
	owners := make(map[string]bool)
	for path, source := range overlay {
		if filepath.Dir(path) != dir || !strings.HasPrefix(filepath.Base(path), "z_llgo_patch_") || filepath.Ext(path) != ".go" {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, declaration := range file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Doc == nil {
				continue
			}
			name, hasEnv := "", false
			for _, comment := range fn.Doc.List {
				words := strings.Fields(comment.Text)
				if len(words) == 3 && words[0] == "//go:linkname" && words[1] == fn.Name.Name {
					name = words[2]
				}
				hasEnv = hasEnv || comment.Text == "//llgo:env" || comment.Text == "// llgo:env"
			}
			if name != "runtime.memequal" && name != "runtime.memequal_varlen" {
				continue
			}
			wantEnv := name == "runtime.memequal_varlen"
			if owners[name] || fn.Body == nil || len(fn.Body.List) == 0 || hasEnv != wantEnv {
				t.Fatal("assembly replacement lacks one real body with its explicit LLGo environment contract", name)
			}
			owners[name] = true
		}
	}
	if !owners["runtime.memequal"] || !owners["runtime.memequal_varlen"] {
		t.Fatal("whole equality source was suppressed without both actual public Go bodies", owners)
	}
}

// This is a syntax-only owner fixture, not an equality implementation or an
// execution certificate. The real runtime bodies are tested separately below.
const runtimeEqualityOwnerPatchFixture = `package bytealg
import "unsafe"
//llgo:skip abigen_runtime_memequal abigen_runtime_memequal_varlen
//go:linkname abigen_runtime_memequal runtime.memequal
func abigen_runtime_memequal(a, b unsafe.Pointer, size uintptr) bool { return a == b }
//go:linkname abigen_runtime_memequal_varlen runtime.memequal_varlen
//llgo:env
func abigen_runtime_memequal_varlen(a, b unsafe.Pointer) bool { return a == b }
`

func TestRuntimeEqualitySourcePatchRejectsIncompleteOwners(t *testing.T) {
	const asm = "TEXT runtime·memequal<ABIInternal>(SB),4,$0-25\nRET\n" +
		"TEXT runtime·memequal_varlen<ABIInternal>(SB),4,$0-17\nRET\n" +
		"TEXT memeqbody<>(SB),4,$0-0\nRET\n"
	for _, mutation := range []string{
		"valid", "missing-body", "wrong-signature", "missing-env", "new-public-text",
		"missing-public-text", "duplicate-text", "overlay-new-public-text", "overlay-missing-body", "unowned-marker",
		"skip-all", "semicolon-text", "duplicate-go-body",
	} {
		t.Run(mutation, func(t *testing.T) {
			goroot, runtimeDir := t.TempDir(), t.TempDir()
			dir := filepath.Join(goroot, "src", "internal", "bytealg")
			patchDir := filepath.Join(runtimeDir, "_patch", "internal", "bytealg")
			for _, path := range []string{dir, patchDir} {
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			patch, original := runtimeEqualityOwnerPatchFixture, asm
			switch mutation {
			case "missing-body":
				patch = strings.Replace(patch, "bool { return a == b }", "bool", 1)
			case "wrong-signature":
				patch = strings.Replace(patch, "a, b unsafe.Pointer) bool", "a, b unsafe.Pointer, size uintptr) bool", 1)
			case "missing-env":
				patch = strings.Replace(patch, "//llgo:env\n", "", 1)
			case "new-public-text":
				original += "TEXT runtime·newEqualityEntry(SB),4,$0-0\nRET\n"
			case "missing-public-text":
				original = strings.Replace(original, "TEXT runtime·memequal_varlen<ABIInternal>(SB),4,$0-17\nRET\n", "", 1)
			case "duplicate-text":
				original += "TEXT runtime·memequal<ABIInternal>(SB),4,$0-25\nRET\n"
			case "skip-all":
				patch += "\n//llgo:skipall\n"
			case "semicolon-text":
				original = strings.Replace(original, "$0-25\n", "$0-25; TEXT runtime·newEqualityEntry(SB),4,$0-0\n", 1)
			case "duplicate-go-body":
				patch += "\n//go:linkname duplicate runtime.memequal\n" +
					"func duplicate(a, b unsafe.Pointer, size uintptr) bool { return a == b }\n"
			}
			equal := assemblyLanguageTestWrite(t, dir, "equal_amd64.s", original)
			other := assemblyLanguageTestWrite(t, dir, "compare_amd64.s", "TEXT ·Compare(SB),4,$0-0\nRET\n")
			patchFile := assemblyLanguageTestWrite(t, patchDir, "equal_native.go", patch)
			base := make(map[string][]byte)
			if mutation == "overlay-new-public-text" {
				base[equal] = []byte(asm + "TEXT runtime·newEqualityEntry(SB),4,$0-0\nRET\n")
			}
			if mutation == "overlay-missing-body" {
				base[patchFile] = []byte(strings.Replace(patch, "bool { return a == b }", "bool", 1))
			}
			if mutation == "unowned-marker" {
				base[equal] = []byte("// replaced by LLGo source patch\n")
			}
			_, overlay, _, err := applySourcePatchForPkg(base, nil, runtimeDir, goroot, "internal/bytealg", sourcePatchBuildContext{
				goos: "linux", goarch: "amd64", goversion: "go1.27.2",
			})
			if mutation != "valid" {
				if err == nil {
					t.Fatal("incomplete/new equality ownership must fail, not silently suppress a whole source", mutation)
				}
				return
			}
			if err != nil || string(overlay[filepath.Join(dir, "equal_amd64.s")]) != "// replaced by LLGo source patch\n" {
				t.Fatal("complete equality owner did not replace exactly its source file", err)
			}
			if _, replaced := overlay[other]; replaced {
				t.Fatal("complete equality owner also suppressed compare assembly")
			}
			_, repeated, _, err := applySourcePatchForPkg(overlay, nil, runtimeDir, goroot, "internal/bytealg", sourcePatchBuildContext{
				goos: "linux", goarch: "amd64", goversion: "go1.27.2",
			})
			if err != nil || !reflect.DeepEqual(repeated, overlay) {
				t.Fatal("reusing a complete selected source view must remain stable", err)
			}
		})
	}
}

func TestRuntimeEqualitySourcePatchRetainsOtherTargets(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{{"linux", "arm"}, {"js", "wasm"}} {
		t.Run(target.goos+"/"+target.goarch, func(t *testing.T) {
			goroot := runtime.GOROOT()
			overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), goroot, sourcePatchBuildContext{
				goos: target.goos, goarch: target.goarch, goversion: runtime.Version(),
			})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(goroot, "src", "internal", "bytealg")
			for path := range overlay {
				if filepath.Dir(path) == dir {
					t.Fatal("native equality bridge changed an out-of-scope target", path)
				}
			}
		})
	}
}

func TestRuntimeEqualityActualClosuresMatchOriginalGo(t *testing.T) {
	dir := t.TempDir()
	source, err := filepath.Abs("testdata/runtime-equality/main.go")
	if err != nil {
		t.Fatal(err)
	}
	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	reference := filepath.Join(dir, "original-go"+extension)
	run := func(binary string, args []string, budget time.Duration) string {
		t.Helper()
		var output bytes.Buffer
		err := runRunnerCommand(commandEnv{environ: os.Environ()}, binary, args,
			runnerDetails{phase: "equality-owner-test", timeout: budget}, &output, &output)
		if err != nil {
			t.Fatalf("actual equality owner command: %v\n%s", err, output.String())
		}
		return strings.TrimSpace(output.String())
	}
	goCommand := filepath.Join(runtime.GOROOT(), "bin", "go"+extension)
	run(goCommand, []string{"build", "-o", reference, source}, 5*time.Minute)
	want := run(reference, nil, 30*time.Second)
	if want != "equality checks 21" {
		t.Fatal("independent original-Go denominator changed", want)
	}
	conf := NewDefaultConf(ModeBuild)
	conf.OutFile = filepath.Join(dir, "llgo"+extension)
	if _, err := Do([]string{source}, conf); err != nil {
		t.Fatal("build real equality closure owner program", err)
	}
	if got := run(conf.OutFile, nil, 30*time.Second); got != want {
		t.Fatalf("LLVM equality closure result = %q, original Go = %q", got, want)
	}
}
