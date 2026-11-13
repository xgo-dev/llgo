package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/version"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Runtime source reuse is deliberately a small allowlist. The destinations are
// ordinary files seen by go list; the existing parser overlay and package
// fingerprinting then consume the selected GOROOT bytes. Nothing is written to
// either GOROOT or the runtime tree.
func runtimeSourceOverlay(base map[string][]byte, runtimeDir, goroot, goVersion string) (map[string][]byte, error) {
	out := cloneOverlay(base)
	if out == nil {
		out = make(map[string][]byte)
	}
	read := func(path string) ([]byte, error) {
		if data, ok := base[path]; ok {
			return data, nil
		}
		return os.ReadFile(path)
	}
	load := func(name string) ([]byte, *token.FileSet, *ast.File, error) {
		path := filepath.Join(goroot, "src", "runtime", name)
		src, err := read(path)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("reuse GOROOT runtime/%s: %w", name, err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("reuse GOROOT runtime/%s: %w", name, err)
		}
		return src, fset, file, nil
	}
	src, fset, file, err := load("utf8.go")
	if err != nil {
		return nil, err
	}
	indexType := "int"
	if version.Compare(goVersion, "go1.27") >= 0 {
		indexType = "uint"
	}
	want := map[string]string{
		"countrunes": "func(s string) int",
		"decoderune": fmt.Sprintf("func(s string, k %s) (r rune, pos %s)", indexType, indexType),
		"encoderune": "func(p []byte, r rune) int",
	}
	// Reject new declarations as well as missing/signature-changed helpers.
	// A Go upgrade must review new dependencies before adopting the file.
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build ") {
				return nil, fmt.Errorf("reuse GOROOT runtime/utf8.go: unreviewed directive %s", comment.Text)
			}
		}
	}
	constants := strings.Fields("runeError runeSelf maxRune surrogateMin surrogateMax t1 tx t2 t3 t4 t5 maskx mask2 mask3 mask4 rune1Max rune2Max rune3Max locb hicb")
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			if err := checkRuntimeSourceFunction(fset, decl, want[decl.Name.Name]); err != nil {
				return nil, err
			}
			delete(want, decl.Name.Name)
		case *ast.GenDecl:
			if decl.Tok != token.CONST {
				return nil, fmt.Errorf("reuse GOROOT runtime/utf8.go: unreviewed %s declaration", decl.Tok)
			}
			for _, spec := range decl.Specs {
				for _, name := range spec.(*ast.ValueSpec).Names {
					if !slices.Contains(constants, name.Name) {
						return nil, fmt.Errorf("reuse GOROOT runtime/utf8.go: unreviewed constant %s", name.Name)
					}
				}
			}
		default:
			return nil, fmt.Errorf("reuse GOROOT runtime/utf8.go: unreviewed declaration")
		}
	}
	if len(want) != 0 {
		return nil, fmt.Errorf("reuse GOROOT runtime/utf8.go: missing helpers %v", want)
	}
	out[filepath.Join(runtimeDir, "internal", "runtime", "utf8.go")] =
		buildInjectedSourcePatchFile(filepath.Join(goroot, "src", "runtime", "utf8.go"), src)
	indexFile := "utf8_index.go"
	if indexType == "uint" {
		indexFile = "utf8_index_go127.go"
	}
	indexPath := filepath.Join(runtimeDir, "internal", "runtime", indexFile)
	indexSource, err := read(indexPath)
	if err != nil {
		return nil, fmt.Errorf("read runtime UTF-8 ABI adapter: %w", err)
	}
	indexSource, changed, err := filterSourcePatchFile(indexSource, map[string]struct{}{"decoderune": {}})
	if err != nil {
		return nil, err
	}
	const adapterOverlayMarker = "\n// LLGo runtime source overlay: decoderune declaration removed.\n"
	if !changed && !bytes.HasSuffix(indexSource, []byte(adapterOverlayMarker)) {
		return nil, fmt.Errorf("read runtime UTF-8 ABI adapter %s: missing decoderune declaration", indexFile)
	}
	if changed {
		// Multi-package install reuses the configuration's overlay. Keep the
		// selection idempotent while still rejecting a missing declaration
		// in an adapter that has not passed through this transformation.
		indexSource = append(indexSource, adapterOverlayMarker...)
	}
	out[indexPath] = indexSource
	floating, err := runtimeFloatingPointSources(read, goroot)
	if err != nil {
		return nil, err
	}
	for name, data := range floating {
		out[filepath.Join(runtimeDir, "internal", "runtime", name+"_goroot.go")] = data
	}

	if version.Compare(goVersion, "go1.23") < 0 {
		return out, nil // SetCrashOutput was introduced in Go 1.23.
	}
	src, fset, file, err = load("runtime.go")
	if err != nil {
		return nil, err
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "setCrashFD" || fn.Recv != nil {
			continue
		}
		if err := checkRuntimeSourceFunction(fset, fn, "func(fd uintptr) uintptr"); err != nil {
			return nil, err
		}
		start := fn.Pos()
		if fn.Doc != nil {
			start = fn.Doc.Pos()
		}
		pos := fset.Position(start)
		data := fmt.Appendf(nil, "// Selected from the active GOROOT; see doc/runtime-source-reuse.md.\npackage runtime\nimport _ \"unsafe\"\n//line %s:%d\n",
			filepath.ToSlash(pos.Filename), pos.Line)
		data = append(data, src[pos.Offset:fset.Position(fn.End()).Offset]...)
		data = append(data, '\n')
		out[filepath.Join(runtimeDir, "internal", "lib", "runtime", "crash_fd_go123.go")] = data
		return out, nil
	}
	return nil, fmt.Errorf("reuse GOROOT runtime/runtime.go: missing setCrashFD")
}

func checkRuntimeSourceFunction(fset *token.FileSet, fn *ast.FuncDecl, signature string) error {
	var actual bytes.Buffer
	if err := format.Node(&actual, fset, fn.Type); err != nil {
		return err
	}
	// Parameter/result names are intentional: even cosmetic signature changes
	// request review of the upstream helper before adopting the new source.
	if signature == "" || fn.Recv != nil || fn.Body == nil || actual.String() != signature {
		return fmt.Errorf("reuse GOROOT runtime/%s: unreviewed helper %s %s (expected %q)",
			filepath.Base(fset.Position(fn.Pos()).Filename), fn.Name.Name, actual.String(), signature)
	}
	return nil
}
