package build

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"slices"
	"strings"
)

const wasmFSSnapshotTag = "llgo.wasm.fs_snapshot"
const wasmFSSnapshotPatch = "fs_js_snapshot.go"

// Add the compiler's tag to the effective Go tag flag without changing the
// precedence of caller-supplied and GOFLAGS tag flags.
func appendWasmFSSnapshotTag(flags []string) []string {
	for i := len(flags) - 1; i >= 0; i-- {
		switch {
		case strings.HasPrefix(flags[i], "-tags="):
			flags[i] = "-tags=" + strings.Join(append(splitSourcePatchBuildTags(strings.TrimPrefix(flags[i], "-tags=")), wasmFSSnapshotTag), ",")
			return flags
		case flags[i] == "-tags" && i+1 < len(flags):
			flags[i+1] = strings.Join(append(splitSourcePatchBuildTags(flags[i+1]), wasmFSSnapshotTag), ",")
			return flags
		}
	}
	return append(flags, "-tags="+wasmFSSnapshotTag)
}

const wasmFSStart = `var jsProcess = js.Global().Get("process")`
const wasmFSPath = `var jsPath = js.Global().Get("path")`
const wasmFSValues = `
var jsFS = js.Global().Get("fs")
var constants = jsFS.Get("constants")
var uint8Array = js.Global().Get("Uint8Array")
`
const wasmFSFlags = `
	nodeWRONLY = constants.Get("O_WRONLY").Int()
	nodeRDWR = constants.Get("O_RDWR").Int()
	nodeCREATE = constants.Get("O_CREAT").Int()
	nodeTRUNC = constants.Get("O_TRUNC").Int()
	nodeAPPEND = constants.Get("O_APPEND").Int()
	nodeEXCL = constants.Get("O_EXCL").Int()
`
const wasmFSDirectory = `
func init() {
	oDir := constants.Get("O_DIRECTORY")
	if !oDir.IsUndefined() {
		nodeDIRECTORY = oDir.Int()
	}
}
`

// Recognize only the known contiguous filesystem initializers. Go 1.20–1.23
// have the shorter version, and newer sources can include path and O_DIRECTORY.
// Keep the original dependency and declaration order, including the directory
// lookup in init after the package's variable initializers.
func rewriteWasmFSInitializers(src []byte) []byte {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fs_js.go", src, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != "syscall" {
		return nil
	}
	first := slices.IndexFunc(file.Decls, func(decl ast.Decl) bool {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR || len(gen.Specs) != 1 {
			return false
		}
		names := gen.Specs[0].(*ast.ValueSpec).Names
		return len(names) == 1 && names[0].Name == "jsProcess"
	})
	if first < 0 {
		return nil
	}
	count := wasmFSInitializerCount(fset, file.Decls[first:])
	if count == 0 || !wasmFSUsesKnownOperations(file.Decls[first+count:]) {
		return nil
	}

	replacements := map[string]string{
		"jsProcess":  "llgoCaptureFSValue(0)",
		"jsPath":     "llgoCaptureFSValue(1)",
		"jsFS":       "llgoCaptureFSValue(2)",
		"uint8Array": "llgoCaptureFSValue(3)",
		"constants":  "llgoCaptureFSConstants(jsFS)",
		"nodeWRONLY": "constants.intFlag(0)",
		"nodeRDWR":   "constants.intFlag(1)",
		"nodeCREATE": "constants.intFlag(2)",
		"nodeTRUNC":  "constants.intFlag(3)",
		"nodeAPPEND": "constants.intFlag(4)",
		"nodeEXCL":   "constants.intFlag(5)",
	}
	pos := 0
	var out bytes.Buffer
	replace := func(node ast.Node, with string) {
		start, end := fset.Position(node.Pos()).Offset, fset.Position(node.End()).Offset
		out.Write(src[pos:start])
		out.WriteString(with)
		for n := bytes.Count(src[start:end], []byte("\n")); n > 0; n-- {
			out.WriteByte('\n')
		}
		pos = end
	}
	for _, decl := range file.Decls[first : first+count] {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				value := spec.(*ast.ValueSpec)
				if with := replacements[value.Names[0].Name]; with != "" {
					replace(value.Values[0], with)
				}
			}
		case *ast.FuncDecl:
			replace(decl.Body, "{ nodeDIRECTORY = constants.intFlag(6) }")
		}
	}
	out.Write(src[pos:])
	return out.Bytes()
}

func wasmFSInitializerCount(fset *token.FileSet, decls []ast.Decl) int {
	canonical := func(set *token.FileSet, decl ast.Decl) string {
		var text bytes.Buffer
		_ = printer.Fprint(&text, set, decl)
		return strings.Join(strings.Fields(text.String()), " ")
	}
	for _, path := range []string{"", wasmFSPath} {
		for _, directory := range []bool{false, true} {
			source := "package syscall\n" + wasmFSStart + "\n" + path + wasmFSValues + "\nvar (" + wasmFSFlags
			if directory {
				source += "\tnodeDIRECTORY = -1\n"
			}
			source += ")\n"
			if directory {
				source += wasmFSDirectory
			}
			set := token.NewFileSet()
			expected, _ := parser.ParseFile(set, "", source, parser.SkipObjectResolution)
			count := len(expected.Decls)
			if count >= len(decls) {
				continue
			}
			// A new initializer inserted in the middle or after the recognized
			// sequence needs to be reviewed before the rewrite applies to it.
			next, ok := decls[count].(*ast.GenDecl)
			if !ok || next.Tok != token.TYPE || next.Specs[0].(*ast.TypeSpec).Name.Name != "jsFile" {
				continue
			}
			match := true
			for i, want := range expected.Decls {
				if canonical(fset, decls[i]) != canonical(set, want) {
					match = false
					break
				}
			}
			if match {
				return count
			}
		}
	}
	return 0
}

func wasmFSUsesKnownOperations(decls []ast.Decl) bool {
	changed := func(name string) bool {
		switch name {
		case "jsProcess", "jsPath", "jsFS", "uint8Array", "constants":
			return true
		}
		return false
	}
	known := true
	for _, decl := range decls {
		ast.Inspect(decl, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.SelectorExpr:
				if base, ok := x.X.(*ast.Ident); ok && changed(base.Name) && base.Name != "constants" {
					known = known && (x.Sel.Name == "Get" || x.Sel.Name == "Call" || x.Sel.Name == "New")
					return false
				}
			case *ast.Ident:
				known = known && !changed(x.Name)
			}
			return known
		})
	}
	return known
}
