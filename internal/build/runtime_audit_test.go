package build

import (
	"bytes"
	"encoding/json"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/env"
)

func TestRuntimeOwnership(t *testing.T) {
	for _, target := range []struct{ os, arch string }{{"darwin", "arm64"}, {"linux", "amd64"}, {"windows", "amd64"}, {"windows", "arm64"}, {"windows", "386"}, {"js", "wasm"}, {"wasip1", "wasm"}} {
		t.Run(target.os+"-"+target.arch, func(t *testing.T) {
			var tags []string
			if target.arch == "wasm" {
				tags = []string{"llgo.wasm.gc.linear"}
				if target.os == "js" {
					tags = append(tags, "llgo.wasm.emscripten")
				}
				if target.os == "wasip1" {
					tags = append(tags, "llgo.wasi_threads", "llgo.wasm.workers")
				}
			}
			report, err := AuditRuntimeSources(RuntimeAuditConfig{RuntimeDir: env.LLGoRuntimeDir(), GOOS: target.os, GOARCH: target.arch, Tags: tags})
			if err != nil {
				t.Fatal(err)
			}
			for _, tag := range []string{"llgo", "purego", "math_big_pure_go"} {
				if !slices.Contains(report.Tags, tag) {
					t.Errorf("source selection lost default tag %q", tag)
				}
			}
			t.Logf("%d symbols, %d requirements, %d missing", len(report.Symbols), len(report.Required), len(report.Missing))
			if dir := os.Getenv("LLGO_RUNTIME_AUDIT_DIR"); dir != "" {
				data, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, target.os+"-"+target.arch+".json"), append(data, '\n'), 0644); err != nil {
					t.Fatal(err)
				}
			}
			for _, use := range report.Missing {
				t.Errorf("missing %s (%s)", use.Symbol, use.Source)
			}
			for _, use := range report.DynamicCompilerUses {
				t.Errorf("unbounded runtime lookup %s (%s)", use.Symbol, use.Source)
			}
			for _, use := range report.UnresolvedBoundaries {
				t.Errorf("unresolved runtime boundary %s (%s)", use.Symbol, use.Source)
			}
		})
	}
}

func runtimeAuditFixture(t *testing.T) *runtimeAuditor {
	t.Helper()
	root := t.TempDir()
	match := build.Default
	match.GOOS, match.GOARCH = "linux", "amd64"
	match.BuildTags = []string{"llgo"}
	return &runtimeAuditor{runtimeDir: filepath.Join(root, "runtime"), goroot: filepath.Join(root, "go"),
		match: match, overlay: make(map[string][]byte), fset: token.NewFileSet(),
		definitions: make(map[string]string), implementations: make(map[string]string)}
}

func auditFile(t *testing.T, a *runtimeAuditor, path, source string) runtimeAuditFile {
	t.Helper()
	file, err := parser.ParseFile(a.fset, path, source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	return runtimeAuditFile{path, file}
}

func TestRuntimeAuditOwnershipAndPackageLinknames(t *testing.T) {
	a := runtimeAuditFixture(t)
	files := []runtimeAuditFile{
		auditFile(t, a, filepath.Join(a.runtimeDir, "internal/lib/runtime/body.go"), `package runtime
func hook(x int) int { return x }
func external()
func intrinsic()
var state uintptr
`),
		auditFile(t, a, filepath.Join(a.runtimeDir, "internal/lib/runtime/links.go"), `package runtime
import _ "unsafe"
//go:linkname hook runtime.Hook
//go:linkname external C.clock
//go:linkname intrinsic llgo.clock
`),
		auditFile(t, a, filepath.Join(a.runtimeDir, "internal/runtime/imported.go"), "package runtime\n//line "+filepath.ToSlash(filepath.Join(a.goroot, "src/runtime/leaf.go"))+":17\nfunc leaf() {}\n"),
	}
	var report RuntimeOwnership
	if err := a.inventory(&report, "runtime", "standard-runtime", files); err != nil {
		t.Fatal(err)
	}
	byName := make(map[string]RuntimeSymbol)
	for _, symbol := range report.Symbols {
		byName[symbol.Name] = symbol
	}
	if hook := byName["runtime.hook"]; hook.Linkname != "runtime.Hook" || hook.Signature != "func(x int) int" || !hook.Body {
		t.Fatalf("hook: %+v", hook)
	}
	if byName["runtime.external"].Owner != "external-boundary" || byName["runtime.intrinsic"].Owner != "compiler-intrinsic" {
		t.Fatal("lost boundary ownership")
	}
	if leaf := byName["runtime.leaf"]; leaf.Owner != "goroot" || leaf.Source != "$GOROOT/src/runtime/leaf.go:17" {
		t.Fatalf("source provenance: %+v", leaf)
	}
	consumer := auditFile(t, a, filepath.Join(a.goroot, "src/example/use.go"), `package example
import (
 rt "runtime"
 _ "unsafe"
)
//go:linkname get runtime.Hook
func get(int) int
//go:linkname future runtime.futureHook
func future()
func f() { rt.Hook(1); rt := struct { Shadow int }{}; _ = rt.Shadow }
`)
	links, err := runtimeAuditLinknames([]runtimeAuditFile{consumer})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.requirements(&report, consumer, links); err != nil {
		t.Fatal(err)
	}
	a.resolve(&report)
	if len(report.Missing) != 1 || report.Missing[0].Symbol != "runtime.futureHook" {
		t.Fatalf("new hook not diagnosed accurately: %+v", report.Missing)
	}
}

func TestRuntimeAuditRejectsDuplicateOwners(t *testing.T) {
	for _, test := range []struct{ name, first, second, want string }{
		{"declaration", "func hook() {}", "func hook() {}", "duplicate runtime declaration"},
		{"alias", "//go:linkname one runtime.Hook\nfunc one() {}", "//go:linkname two runtime.Hook\nfunc two() {}", "duplicate runtime implementation"},
		{"conflict", "//go:linkname hook runtime.One\nfunc hook()", "//go:linkname hook runtime.Two", "conflicting linkname"},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := runtimeAuditFixture(t)
			files := []runtimeAuditFile{auditFile(t, a, "a.go", "package p\nimport _ \"unsafe\"\n"+test.first), auditFile(t, a, "b.go", "package p\nimport _ \"unsafe\"\n"+test.second)}
			err := a.inventory(&RuntimeOwnership{}, "runtime", "standard-runtime", files)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestRuntimeAuditVariableAliasIsRequirement(t *testing.T) {
	a := runtimeAuditFixture(t)
	file := auditFile(t, a, "vars.go", `package example
import _ "unsafe"
//go:linkname declared runtime.present
var declared uintptr
//go:linkname missing runtime.absent
var missing uintptr
`)
	links, err := runtimeAuditLinknames([]runtimeAuditFile{file})
	if err != nil {
		t.Fatal(err)
	}
	report := RuntimeOwnership{Symbols: []RuntimeSymbol{{Name: "github.com/xgo-dev/llgo/runtime/internal/runtime.present", Kind: "var"}}}
	if err := a.requirements(&report, file, links); err != nil {
		t.Fatal(err)
	}
	a.resolve(&report)
	if len(report.Missing) != 1 || report.Missing[0].Symbol != "runtime.absent" {
		t.Fatalf("variable aliases incorrectly provide storage: %+v", report.Missing)
	}
}

func TestRuntimeAuditUsesEffectiveSourceSelection(t *testing.T) {
	a := runtimeAuditFixture(t)
	write := func(path, contents string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	upstream := filepath.Join(a.goroot, "src/internal/abi")
	alternate := filepath.Join(a.runtimeDir, "internal/lib/internal/abi")
	write(filepath.Join(upstream, "base.go"), "package abi\nfunc Kept() {}\nfunc Replaced() {}\n")
	write(filepath.Join(upstream, "unused_windows.go"), "package abi\nfunc Kept() {}\n")
	write(filepath.Join(upstream, "ignored_test.go"), "package abi\nfunc Kept() {}\n")
	write(filepath.Join(alternate, "replace.go"), "package abi\nfunc Replaced() {}\n")
	write(filepath.Join(upstream, "overlay.go"), "package abi\nfunc Old() {}\n")
	a.overlay[filepath.Join(upstream, "overlay.go")] = []byte("package abi\nfunc Updated() {}\n")
	a.overlay[filepath.Join(upstream, "generated.go")] = []byte("package abi\nfunc Added() {}\n")
	a.match.OpenFile = func(path string) (io.ReadCloser, error) {
		if src, ok := a.overlay[path]; ok {
			return io.NopCloser(bytes.NewReader(src)), nil
		}
		return os.Open(path)
	}
	files, err := a.standardFiles("internal/abi")
	if err != nil {
		t.Fatal(err)
	}
	var report RuntimeOwnership
	if err := a.inventory(&report, "internal/abi", "metadata-adapter", files); err != nil {
		t.Fatal(err)
	}
	owners := map[string]string{}
	for _, symbol := range report.Symbols {
		owners[symbol.Name] = symbol.Owner
	}
	want := map[string]string{"internal/abi.Kept": "goroot", "internal/abi.Replaced": "metadata-adapter", "internal/abi.Updated": "goroot", "internal/abi.Added": "goroot"}
	if !reflect.DeepEqual(owners, want) {
		t.Fatalf("owners = %v, want %v", owners, want)
	}
	// An explicit skip may remove a declaration without replacing its body.
	write(filepath.Join(alternate, "replace.go"), "package abi\n//llgo:skip Kept\nfunc Replaced() {}\n")
	files, err = a.standardFiles("internal/abi")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		for _, decl := range file.file.Decls {
			if slices.Contains(declPatchKeys(decl), "Kept") {
				t.Fatal("explicitly skipped declaration remains in the source view")
			}
		}
	}
	write(filepath.Join(alternate, "replace.go"), "package abi\n//llgo:skipall\nfunc Only() {}\n")
	files, err = a.standardFiles("internal/abi")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || len(files[0].file.Decls) != 1 {
		t.Fatalf("skipall retained upstream source: %+v", files)
	}
}

func TestRuntimeAuditAssemblyBoundary(t *testing.T) {
	a := runtimeAuditFixture(t)
	dir := filepath.Join(a.goroot, "src/internal/bytealg")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "compare_amd64.s")
	if err := os.WriteFile(path, []byte("TEXT runtime·cmpstring<ABIInternal>(SB),NOSPLIT,$0-0\nRET\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var report RuntimeOwnership
	if err := a.assembly(&report, "internal/bytealg", dir); err != nil {
		t.Fatal(err)
	}
	if len(report.Symbols) != 1 || report.Symbols[0].Name != "runtime.cmpstring" || report.Symbols[0].Owner != "goroot-assembly" {
		t.Fatalf("assembly entries: %+v", report.Symbols)
	}
	if err := a.assembly(&report, "internal/bytealg", dir); err == nil || !strings.Contains(err.Error(), "duplicate runtime assembly entry") {
		t.Fatalf("duplicate assembly provider: %v", err)
	}
	a.overlay[path] = []byte("// suppressed by source patch\n")
	report.Symbols = nil
	if err := a.assembly(&report, "internal/bytealg", dir); err != nil {
		t.Fatal(err)
	}
	if len(report.Symbols) != 0 {
		t.Fatal("suppressed assembly still provides runtime symbols")
	}
}

func TestRuntimeAuditUpgradeDiagnostics(t *testing.T) {
	for _, test := range []struct{ name, path, source, want string }{
		{"complete", "", "", ""},
		{"legacy-coverage", "", "", ""},
		{"dot-import", "go/src/consumer/use.go", "package consumer\nimport . \"runtime\"\nfunc use() { NewHook() }", "cannot resolve dot import"},
		{"new-stdlib-hook", "go/src/consumer/use.go", "package consumer\nimport \"runtime\"\nfunc use() { runtime.NewHook() }", "missing"},
		{"new-compiler-hook", "ssa/use.go", "package ssa\nfunc use() { p.rtFunc(\"NewHook\") }", "missing"},
		{"new-compiler-type", "ssa/use.go", "package ssa\nfunc use() { p.rtType(\"NewType\") }", "missing"},
		{"new-compiler-declaration", "cl/use.go", "package cl\nfunc use() { p.runtimeFunc(\"NewHook\", signature) }", "missing"},
		{"changed-upstream-helper", "go/src/runtime/utf8.go", "package runtime\nfunc encoderune(p []int, r rune) int { return 0 }", "unreviewed helper"},
		{"malformed-patch", "runtime/_patch/sync/bad.go", "package sync\nfunc (", "parse source patch directives"},
		{"duplicate-runtime", "runtime/internal/runtime/duplicate.go", "package runtime\nfunc countrunes(s string) int { return 0 }", "duplicate runtime declaration"},
		{"malformed-abi", "go/src/internal/abi/bad.go", "package abi\nfunc (", "expected"},
		{"malformed-stdlib", "go/src/consumer/bad.go", "package consumer\nfunc (", "expected"},
		{"invalid-source-selection", "go/src/consumer/bad.go", "//go:build &&\n\npackage consumer", "parsing //go:build line"},
		{"conflicting-stdlib-linknames", "go/src/consumer/use.go", "package consumer\nimport _ \"unsafe\"\n//go:linkname hook runtime.One\n//go:linkname hook runtime.Two\nfunc hook()", "conflicting linkname"},
		{"invalid-assembly-selection", "go/src/consumer/bad.s", "//go:build &&\n\nTEXT runtime·hook(SB),NOSPLIT,$0-0\nRET", "parsing //go:build line"},
		{"malformed-alternative", "runtime/internal/lib/internal/abi/bad.go", "package abi\nfunc (", "expected"},
		{"malformed-inactive-runtime", "runtime/internal/runtime/bad_windows.go", "package runtime\nfunc (", "expected"},
		{"malformed-compiler", "cl/use.go", "package cl\nfunc (", "expected"},
		{"missing-compiler", "", "", "requires compiler sources"},
		{"no-compiler-go-files", "", "", "requires compiler sources"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			write := func(path, source string) {
				t.Helper()
				path = filepath.Join(root, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(source), 0644); err != nil {
					t.Fatal(err)
				}
			}
			write("go/src/runtime/utf8.go", `package runtime
func countrunes(s string) int { return len(s) }
func decoderune(s string, k uint) (r rune, pos uint) { return 0, k }
func encoderune(p []byte, r rune) int { return 0 }
`)
			write("go/src/runtime/runtime.go", "package runtime\nfunc setCrashFD(fd uintptr) uintptr { return fd }\ntype MemStats struct{}\n")
			for _, name := range []string{"complex.go", "float.go"} {
				data, err := os.ReadFile(filepath.Join(runtime.GOROOT(), "src/runtime", name))
				if err != nil {
					t.Fatal(err)
				}
				write("go/src/runtime/"+name, string(data))
				write("runtime/internal/runtime/"+strings.TrimSuffix(name, ".go")+"_goroot.go", "package runtime\n")
			}
			write("runtime/internal/runtime/utf8.go", "package runtime\n")
			write("runtime/internal/runtime/utf8_index_go127.go", "package runtime\nfunc decoderune(s string, k uint) (r rune, pos uint)\n")
			write("runtime/internal/lib/runtime/crash_fd_go123.go", "package runtime\nfunc setCrashFD(fd uintptr) uintptr\n")
			write("go/src/sync/base.go", "package sync\n")
			write("ssa/use.go", "package ssa\nfunc use() { p.rtFunc(\"countrunes\"); p.rtFunc(name) }\n")
			write("cl/use.go", "package cl\n")
			goVersion := "go1.27.0"
			if test.name == "legacy-coverage" {
				goVersion = "go1.22.0"
				path := filepath.Join(root, "go/src/runtime/utf8.go")
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				write("go/src/runtime/utf8.go", strings.ReplaceAll(string(data), "uint", "int"))
				write("runtime/internal/runtime/utf8_index.go", "package runtime\nfunc decoderune(s string, k int) (r rune, pos int)\n")
				write("runtime/internal/runtime/utf8_index_go127.go", "//go:build go1.27\n\npackage runtime\nfunc decoderune(s string, k uint) (r rune, pos uint)\n")
			}
			if test.path != "" {
				write(test.path, test.source)
			}
			if test.name == "missing-compiler" || test.name == "no-compiler-go-files" {
				if err := os.RemoveAll(filepath.Join(root, "cl")); err != nil {
					t.Fatal(err)
				}
				if test.name == "no-compiler-go-files" {
					write("cl/README.md", "A directory without compiler sources must not produce a complete report.")
				}
			}
			report, err := auditRuntimeSources(filepath.Join(root, "runtime"), filepath.Join(root, "go"), sourcePatchBuildContext{goos: "linux", goarch: "amd64", goversion: goVersion, buildFlags: []string{"-tags=llgo"}})
			if test.want == "" || test.want == "missing" {
				if err != nil {
					t.Fatal(err)
				}
				if (len(report.Missing) != 0) != (test.want == "missing") {
					t.Fatalf("missing references: %+v", report.Missing)
				}
				if test.want == "" && len(report.DynamicCompilerUses) != 1 {
					t.Fatal("dynamic compiler lookup disappeared")
				}
				if test.name == "legacy-coverage" {
					found := false
					for _, symbol := range report.Symbols {
						if symbol.Linkname == "runtime.addExitHook" {
							found = symbol.Owner == "compiler-generated" && symbol.Condition != ""
						}
					}
					if !found {
						t.Fatal("legacy generated coverage hook is missing")
					}
				}
			} else if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %s", err, test.want)
			}
		})
	}
}

func TestRuntimeAuditInputErrors(t *testing.T) {
	if _, err := AuditRuntimeSources(RuntimeAuditConfig{}); err == nil || !strings.Contains(err.Error(), "requires RuntimeDir") {
		t.Fatalf("missing configuration: %v", err)
	}
	if _, err := AuditRuntimeSources(RuntimeAuditConfig{RuntimeDir: env.LLGoRuntimeDir(), GOOS: "linux", GOARCH: "amd64", Dir: filepath.Join(t.TempDir(), "missing")}); err == nil || !strings.Contains(err.Error(), "resolve Go source configuration") {
		t.Fatalf("unresolvable application toolchain: %v", err)
	}
	a := runtimeAuditFixture(t)
	if files, err := a.files(filepath.Join(t.TempDir(), "absent")); err != nil || len(files) != 0 {
		t.Fatalf("absent optional package: %v, %v", files, err)
	}
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.files(path); err == nil {
		t.Fatal("accepted a regular file as a source directory")
	}
	if err := a.assembly(&RuntimeOwnership{}, "consumer", path); err == nil {
		t.Fatal("accepted a regular file as an assembly directory")
	}
}
