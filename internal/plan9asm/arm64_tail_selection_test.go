package plan9asm

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/packages"
	"github.com/xgo-dev/llvm"
)

func arm64TailSelectionFixture(t *testing.T, overlay, tuple bool) (*packages.Package, TranslateOptions, string) {
	t.Helper()
	return arm64TailSelectionPackageFixture(t, overlay, tuple, "example.com/abizero", "abizero")
}

func arm64TailSelectionPackageFixture(t *testing.T, overlay, tuple bool, packagePath, packageName string) (*packages.Package, TranslateOptions, string) {
	t.Helper()
	declaration := "func Slots(value uint64) (out uint64)\n"
	assembly := `TEXT ·Slots(SB),516,$0-16
MOVD value+0(FP),R0
MOVD $out+8(FP),R8
B resultbody<>(SB)
TEXT resultbody<>(SB),516,$0
ADD $7,R0,R11
MOVD R11,(R8)
RET
`
	if tuple {
		declaration = "func Slots(value uint64) (low uint64, high uint64)\n"
		assembly = `TEXT ·Slots(SB),516,$0-24
MOVD value+0(FP),R0
MOVD $low+8(FP),R8
MOVD $high+16(FP),R9
B resultbody<>(SB)
TEXT resultbody<>(SB),516,$0
ADD $7,R0,R11
MOVD R11,(R8)
SUB $3,R0,R12
MOVD R12,(R9)
RET
`
	}

	dir := t.TempDir()
	fset := token.NewFileSet()
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	sizes := types.SizesFor("gc", "arm64")
	pkg := &packages.Package{
		PkgPath: packagePath, Name: packageName,
		Fset: fset, TypesInfo: info, TypesSizes: sizes,
	}
	opt := TranslateOptions{Overlay: make(map[string][]byte)}
	// A complete selection contains more than the one consumed declaration.
	// Keep its order identical to the parser/type checker's selected input.
	otherSource := "const Selected = 1\n"
	if packageName == "main" {
		otherSource += "func main() {}\n"
	}
	for i, body := range []string{declaration, otherSource} {
		filename := filepath.Join(dir, fmt.Sprintf("selected%d.go", i))
		source := []byte("package " + packageName + "\n" + body)
		onDisk := source
		if overlay {
			// Stale disk content must not supersede the actual parser overlay.
			onDisk = []byte("package stale\n")
			opt.Overlay[filename] = source
		}
		if err := os.WriteFile(filename, onDisk, 0o600); err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, filename, source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		pkg.CompiledGoFiles = append(pkg.CompiledGoFiles, filename)
		pkg.GoFiles = append(pkg.GoFiles, filename)
		pkg.Syntax = append(pkg.Syntax, file)
	}
	var err error
	pkg.Types, err = (&types.Config{Sizes: sizes}).Check(pkg.PkgPath, fset, pkg.Syntax, info)
	if err != nil {
		t.Fatal(err)
	}
	return pkg, opt, assembly
}

func TestTranslateARM64ABI0TailSelectedDeclarations(t *testing.T) {
	for _, overlay := range []bool{false, true} {
		for _, tuple := range []bool{false, true} {
			for _, preprocessed := range []bool{false, true} {
				name := fmt.Sprintf("overlay=%t/tuple=%t/preprocessed=%t", overlay, tuple, preprocessed)
				t.Run(name, func(t *testing.T) {
					pkg, opt, source := arm64TailSelectionFixture(t, overlay, tuple)
					opt.SourcePreprocessed = preprocessed
					file := filepath.Join(filepath.Dir(pkg.CompiledGoFiles[0]), "slots_arm64.s")
					tr, err := TranslateSourceModuleForPkgWithOptions(pkg, file, []byte(source), "linux", "arm64", opt)
					if err != nil {
						t.Fatal("actual selected ABI0 declaration was not supplied to the translator:", err)
					}
					defer tr.Module.Dispose()
					if err := llvm.VerifyModule(tr.Module, llvm.ReturnStatusAction); err != nil {
						t.Fatal(err)
					}
					function := tr.Module.NamedFunction(pkg.PkgPath + ".Slots")
					if function.IsNil() || function.BasicBlocksCount() == 0 {
						t.Fatal("missing translated ABI0 entry body")
					}
				})
			}
		}
	}
}

func TestTranslateARM64ABI0TailRejectsIncompleteSelection(t *testing.T) {
	for _, mutation := range []string{
		"missing-fset", "missing-types-info", "missing-sizes", "missing-files",
		"missing-syntax", "reordered-files", "changed-bytes", "deleted-source",
	} {
		t.Run(mutation, func(t *testing.T) {
			pkg, opt, source := arm64TailSelectionFixture(t, true, false)
			file := filepath.Join(filepath.Dir(pkg.CompiledGoFiles[0]), "slots_arm64.s")
			opt.SourcePreprocessed = true
			switch mutation {
			case "missing-fset":
				pkg.Fset = nil
			case "missing-types-info":
				pkg.TypesInfo = nil
			case "missing-sizes":
				pkg.TypesSizes = nil
			case "missing-files":
				pkg.CompiledGoFiles = nil
			case "missing-syntax":
				pkg.Syntax = pkg.Syntax[:1]
			case "reordered-files":
				pkg.CompiledGoFiles[0], pkg.CompiledGoFiles[1] = pkg.CompiledGoFiles[1], pkg.CompiledGoFiles[0]
			case "changed-bytes":
				opt.Overlay[pkg.CompiledGoFiles[1]] = []byte("package abizero\nconst Selected = 2\n")
			case "deleted-source":
				opt.Overlay[pkg.CompiledGoFiles[1]] = nil
			}
			tr, err := TranslateSourceModuleForPkgWithOptions(pkg, file, []byte(source), "linux", "arm64", opt)
			if tr != nil {
				tr.Module.Dispose()
			}
			if tr != nil || err == nil {
				t.Fatal("incomplete or mismatched selection acquired an ABI0 tail binding:", err)
			}
		})
	}
}
