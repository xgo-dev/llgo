package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/goplus/llgo/internal/env"
	llruntime "github.com/goplus/llgo/runtime"
)

func TestBuildSourcePatchOverlayForIter(t *testing.T) {
	overlay, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), runtime.GOROOT(), sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}

	iterDir := filepath.Join(runtime.GOROOT(), "src", "iter")
	patchFile := filepath.Join(iterDir, "z_llgo_patch_iter.go")
	patchSrc, ok := overlay[patchFile]
	if !ok {
		t.Fatalf("missing source patch file %s", patchFile)
	}
	if !strings.Contains(string(patchSrc), "func Pull[V any]") {
		t.Fatalf("source patch file %s does not contain iter replacement", patchFile)
	}
	if !strings.HasPrefix(string(patchSrc), sourcePatchLineDirective(filepath.Join(env.LLGoRuntimeDir(), "_patch", "iter", "iter.go"))) {
		t.Fatalf("source patch file %s is missing line directive, got:\n%s", patchFile, patchSrc)
	}

	stdFile := filepath.Join(iterDir, "iter.go")
	stdSrc, ok := overlay[stdFile]
	if !ok {
		t.Fatalf("missing stub overlay for %s", stdFile)
	}
	got := string(stdSrc)
	if !strings.Contains(got, "package iter") {
		t.Fatalf("stub overlay for %s lost package clause", stdFile)
	}
	if strings.Contains(got, "func Pull") {
		t.Fatalf("stub overlay for %s still contains original declarations", stdFile)
	}
}

func TestIterUsesSourcePatchInsteadOfAltPkg(t *testing.T) {
	if !llruntime.HasSourcePatchPkg("iter") {
		t.Fatal("iter should be registered as a source patch package")
	}
	if llruntime.HasAltPkg("iter") {
		t.Fatal("iter should not remain an alt package")
	}
}

func TestBuildSourcePatchOverlayForGo126Payloads(t *testing.T) {
	goroot := t.TempDir()
	mustWriteFile(t, filepath.Join(goroot, "src", "internal", "sync", "hashtriemap.go"), `package sync

type HashTrieMap[K comparable, V any] struct{}
`)
	mustWriteFile(t, filepath.Join(goroot, "src", "internal", "sync", "mutex.go"), `package sync

type Mutex struct{}
`)
	mustWriteFile(t, filepath.Join(goroot, "src", "crypto", "internal", "constanttime", "constant_time.go"), `package constanttime

func boolToUint8(bool) uint8
`)

	overlay, err := buildSourcePatchOverlayForGOROOT(nil, env.LLGoRuntimeDir(), goroot, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: "go1.26.0",
	})
	if err != nil {
		t.Fatal(err)
	}

	syncDir := filepath.Join(goroot, "src", "internal", "sync")
	syncPatch := filepath.Join(syncDir, "z_llgo_patch_hashtriemap.go")
	if src, ok := overlay[syncPatch]; !ok {
		t.Fatalf("missing source patch file %s", syncPatch)
	} else if !strings.Contains(string(src), "type HashTrieMap") {
		t.Fatalf("source patch file %s does not contain HashTrieMap replacement", syncPatch)
	}
	if stdSrc := string(overlay[filepath.Join(syncDir, "hashtriemap.go")]); strings.Contains(stdSrc, "type HashTrieMap") {
		t.Fatalf("stub overlay for internal/sync still contains HashTrieMap: %s", stdSrc)
	}

	constanttimeDir := filepath.Join(goroot, "src", "crypto", "internal", "constanttime")
	constanttimePatch := filepath.Join(constanttimeDir, "z_llgo_patch_constant_time.go")
	if src, ok := overlay[constanttimePatch]; !ok {
		t.Fatalf("missing source patch file %s", constanttimePatch)
	} else if !strings.Contains(string(src), "//go:linkname boolToUint8 llgo.boolToUint8") {
		t.Fatalf("source patch file %s does not contain boolToUint8 linkname", constanttimePatch)
	}
}

func TestGo126PayloadsUseSourcePatchInsteadOfAltPkg(t *testing.T) {
	for _, pkgPath := range []string{"internal/sync", "crypto/internal/constanttime"} {
		if !llruntime.HasSourcePatchPkg(pkgPath) {
			t.Fatalf("%s should be registered as a source patch package", pkgPath)
		}
		if llruntime.HasAltPkg(pkgPath) {
			t.Fatalf("%s should not remain an alt package", pkgPath)
		}
	}
}

func TestSyncAtomicRemainsAltPkg(t *testing.T) {
	if llruntime.HasSourcePatchPkg("sync/atomic") {
		t.Fatal("sync/atomic should not be registered as a source patch package")
	}
	if !llruntime.HasAltPkg("sync/atomic") {
		t.Fatal("sync/atomic should remain an alt package")
	}
}

func TestInternalRuntimeMapsRemainsAltPkg(t *testing.T) {
	if llruntime.HasSourcePatchPkg("internal/runtime/maps") {
		t.Fatal("internal/runtime/maps should not be registered as a source patch package")
	}
	if !llruntime.HasAltPkg("internal/runtime/maps") {
		t.Fatal("internal/runtime/maps should remain an alt package")
	}
}

func TestInternalRuntimeSysRemainsAltPkg(t *testing.T) {
	if llruntime.HasSourcePatchPkg("internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should not be registered as a source patch package")
	}
	if !llruntime.HasAltPkg("internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should remain an alt package")
	}
	if !llruntime.HasAdditiveAltPkg("internal/runtime/sys") {
		t.Fatal("internal/runtime/sys should remain an additive alt package")
	}
}

func TestApplySourcePatchForPkg_Cases(t *testing.T) {
	for _, caseName := range []string{
		"default-override",
		"generic-constraints-and-interface",
		"generic-type-and-method",
		"multi-file-skipall",
		"multi-file-with-asm",
		"skip-and-override",
		"skipall",
		"type-alias-and-grouped-values",
	} {
		t.Run(caseName, func(t *testing.T) {
			runSourcePatchCase(t, caseName)
		})
	}
}

func TestApplySourcePatchForPkg_MissingStdlibPkg(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "iter"
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	mustWriteFile(t, filepath.Join(patchDir, "iter.go"), `package iter

//llgo:skipall

func Pull[V any](seq func(func(V) bool)) {}
`)

	changed, overlay, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected missing stdlib package to skip source patching")
	}
	if overlay != nil {
		t.Fatalf("expected no overlay for missing stdlib package, got %v entries", len(overlay))
	}
}

func TestApplySourcePatchForPkg_BuildTaggedPatch(t *testing.T) {
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "demo"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	mustWriteFile(t, filepath.Join(srcDir, "demo.go"), `package demo

func Old() string { return "old" }
`)
	mustWriteFile(t, filepath.Join(patchDir, "patch.go"), `//go:build go1.26
//llgo:skipall
package demo

const Only = "patched"
`)

	changed, overlay, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: "go1.24.11",
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatalf("expected go1.26-tagged patch to be ignored on go1.24, got overlay: %#v", overlay)
	}

	changed, overlay, err = applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{
		goos:      runtime.GOOS,
		goarch:    runtime.GOARCH,
		goversion: "go1.26.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected go1.26-tagged patch to apply on go1.26")
	}
}

func TestApplySourcePatchForPkg_UnreadableStdlibPkg(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission test is Unix-only")
	}
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	pkgPath := "iter"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(srcDir, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(srcDir, 0755)
	mustWriteFile(t, filepath.Join(patchDir, "iter.go"), `package iter

//llgo:skipall

func Pull[V any](seq func(func(V) bool)) {}
`)

	changed, overlay, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected unreadable stdlib package to skip source patching")
	}
	if overlay != nil {
		t.Fatalf("expected no overlay for unreadable stdlib package, got %v entries", len(overlay))
	}
}

func runSourcePatchCase(t *testing.T, caseName string) {
	t.Helper()

	assetRoot := filepath.Join(env.LLGoRuntimeDir(), "_patch", "_test", caseName)
	goroot := t.TempDir()
	runtimeDir := t.TempDir()
	const pkgPath = "demo"
	srcDir := filepath.Join(goroot, "src", pkgPath)
	patchDir := filepath.Join(runtimeDir, "_patch", pkgPath)

	copyTree(t, filepath.Join(assetRoot, "pkg"), srcDir)
	copyTree(t, filepath.Join(assetRoot, "patch"), patchDir)

	changed, overlay, err := applySourcePatchForPkg(nil, nil, runtimeDir, goroot, pkgPath, sourcePatchBuildContext{})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected source patch overlay to change package")
	}

	assertOverlayMatchesOutput(t, overlay, srcDir, filepath.Join(assetRoot, "output"), runtimeDir)
	assertGeneratedPatchPositions(t, overlay, srcDir, patchDir)
}

func copyTree(t *testing.T, srcRoot, dstRoot string) {
	t.Helper()
	err := filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func assertOverlayMatchesOutput(t *testing.T, overlay map[string][]byte, srcRoot, outputRoot, runtimeDir string) {
	t.Helper()

	got := overlayFilesUnderRoot(t, overlay, srcRoot)
	want := readTextFiles(t, outputRoot, runtimeDir)

	gotNames := sortedMapKeys(got)
	wantNames := sortedMapKeys(want)
	assertExactString(t, "overlay file list", strings.Join(gotNames, "\n"), strings.Join(wantNames, "\n"))

	for _, name := range wantNames {
		assertExactString(t, "overlay file "+name, got[name], want[name])
	}
}

func overlayFilesUnderRoot(t *testing.T, overlay map[string][]byte, root string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for filename, src := range overlay {
		rel, err := filepath.Rel(root, filename)
		if err != nil {
			t.Fatal(err)
		}
		if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
			continue
		}
		out[filepath.ToSlash(rel)] = string(src)
	}
	return out
}

func readTextFiles(t *testing.T, root, runtimeDir string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		if strings.HasSuffix(key, ".txt") {
			key = strings.TrimSuffix(key, ".txt")
		}
		out[key] = expandSourcePatchOutputTemplate(string(data), runtimeDir)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func expandSourcePatchOutputTemplate(src, runtimeDir string) string {
	patchRoot := filepath.ToSlash(filepath.Join(runtimeDir, "_patch"))
	return strings.ReplaceAll(src, "{{PATCH_ROOT}}", patchRoot)
}

func sortedMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertGeneratedPatchPositions(t *testing.T, overlay map[string][]byte, srcRoot, patchRoot string) {
	t.Helper()
	for rel, src := range overlayFilesUnderRoot(t, overlay, srcRoot) {
		base := filepath.Base(rel)
		if !strings.HasPrefix(base, "z_llgo_patch_") {
			continue
		}
		original := strings.TrimPrefix(base, "z_llgo_patch_")
		patchFile := filepath.Join(patchRoot, filepath.Dir(rel), original)
		for _, target := range patchedTargetsOfFile(t, patchFile) {
			assertPatchedPosition(t, src, filepath.Join(srcRoot, filepath.FromSlash(rel)), patchFile, target.key, target.line)
		}
	}
}

type patchedTarget struct {
	key  string
	line int
}

func patchedTargetsOfFile(t *testing.T, filename string) []patchedTarget {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	targets := []patchedTarget{{
		key:  "package",
		line: fset.Position(file.Package).Line,
	}}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			key := "func:" + decl.Name.Name
			if decl.Recv != nil && len(decl.Recv.List) != 0 {
				key = "method:" + recvPatchKey(decl.Recv.List[0].Type) + "." + decl.Name.Name
			}
			targets = append(targets, patchedTarget{
				key:  key,
				line: fset.Position(decl.Name.Pos()).Line,
			})
		case *ast.GenDecl:
			kind := strings.ToLower(decl.Tok.String())
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					targets = append(targets, patchedTarget{
						key:  "type:" + spec.Name.Name,
						line: fset.Position(spec.Name.Pos()).Line,
					})
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						targets = append(targets, patchedTarget{
							key:  kind + ":" + name.Name,
							line: fset.Position(name.Pos()).Line,
						})
					}
				}
			}
		}
	}
	return targets
}

func mustWriteFile(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func sourcePatchLineDirective(filename string) string {
	return "//line " + filepath.ToSlash(filename) + ":1\n"
}

func assertExactString(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch\nwant:\n%q\n\ngot:\n%q", label, want, got)
	}
}

func assertPatchedPosition(t *testing.T, src, generatedFilename, wantFilename, target string, wantLine int) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, generatedFilename, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pos, ok := findPatchedPosition(file, target)
	if !ok {
		t.Fatalf("target %q not found", target)
	}
	got := fset.Position(pos)
	if filepath.ToSlash(got.Filename) != filepath.ToSlash(wantFilename) || got.Line != wantLine {
		t.Fatalf("target %q position mismatch: want %s:%d, got %s:%d", target, filepath.ToSlash(wantFilename), wantLine, filepath.ToSlash(got.Filename), got.Line)
	}
}

func findPatchedPosition(file *ast.File, target string) (token.Pos, bool) {
	if target == "package" {
		return file.Package, true
	}
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			key := "func:" + decl.Name.Name
			if decl.Recv != nil && len(decl.Recv.List) != 0 {
				key = "method:" + recvPatchKey(decl.Recv.List[0].Type) + "." + decl.Name.Name
			}
			if key == target {
				return decl.Name.Pos(), true
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if "type:"+spec.Name.Name == target {
						return spec.Name.Pos(), true
					}
				case *ast.ValueSpec:
					kind := strings.ToLower(decl.Tok.String())
					for _, name := range spec.Names {
						if kind+":"+name.Name == target {
							return name.Pos(), true
						}
					}
				}
			}
		}
	}
	return token.NoPos, false
}
