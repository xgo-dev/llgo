package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/packages"
)

func nativeAssemblyFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".s" || ext == ".S"
}

// Respect the last explicit Clang language override, including flags from the
// configured driver. This is only an IR-emission decision, not source routing.
func llgoFileIsAssembly(ctx *context, args []string, path string) bool {
	assembly := nativeAssemblyFile(path)
	args = ctx.compilerForSource(path).CompileArguments(args...)
	for i := 0; i < len(args); i++ {
		language := ""
		if args[i] == "-x" && i+1 < len(args) {
			i++
			language = args[i]
		} else if strings.HasPrefix(args[i], "-x") && !strings.HasPrefix(args[i], "-xray-") {
			language = strings.TrimPrefix(args[i], "-x")
		}
		if language != "" {
			assembly = language == "assembler" || language == "assembler-with-cpp" || language == "none" && nativeAssemblyFile(path)
		}
	}
	return assembly
}

// cgoAssemblyInputs gives every selected GCC assembly file a real compiler
// owner. Go's runtime/cgo is mixed: gcc_* uses GCC/Clang, the other files use
// the Go assembler. Extension case alone cannot distinguish those languages.
func cgoAssemblyInputs(ctx *context, pkg *packages.Package, explicit []llgoFileInput) ([]llgoFileInput, error) {
	selected := selectedSFiles(pkg.OtherFiles)
	if len(selected) == 0 {
		return nil, nil
	}
	fset, files, err := selectedOriginalCgoFiles(ctx, pkg)
	if err != nil || len(files) == 0 {
		return nil, err
	}
	var native []string
	for _, file := range selected {
		if pkg.PkgPath != "runtime/cgo" || strings.HasPrefix(filepath.Base(file), "gcc_") {
			native = append(native, file)
		}
	}
	if len(native) == 0 {
		return nil, nil
	}
	for _, file := range native {
		for _, input := range explicit {
			if filepath.Clean(input.path) == filepath.Clean(file) {
				return nil, fmt.Errorf("assembly %s has both cgo and explicit LLGoFiles owners", file)
			}
		}
	}
	buildCtx := cgoBuildContext(ctx)
	preambles, decls, err := parseCgoFilePreambles(ctx.commands, &buildCtx, fset, files, "C")
	if err != nil {
		return nil, err
	}
	cflags, _, _ := cgoCompilationFlags(preambles, decls)
	// GCC's package assembly also needs the package include directory when
	// there is no preamble (notably GAS .include in an unpreprocessed .s).
	if pkg.Dir != "" && !slices.Contains(cflags, "-I"+pkg.Dir) {
		cflags = append(cflags, "-I"+pkg.Dir)
	}
	inputs := make([]llgoFileInput, 0, len(native))
	for _, file := range native {
		inputs = append(inputs, llgoFileInput{path: file, compilerArgs: slices.Clone(cflags)})
	}
	return inputs, nil
}

// Syntax/CompiledGoFiles may already contain cgo's rewritten import "unsafe".
// Only selected original GoFiles, with the invocation overlay, establish cgo
// ownership. Do not glob the directory or use a generated file's imports.
func selectedOriginalCgoFiles(ctx *context, pkg *packages.Package) (*token.FileSet, []*ast.File, error) {
	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range pkg.GoFiles {
		var source []byte
		var overlaid bool
		if ctx.buildConf != nil {
			source, overlaid = ctx.buildConf.Overlay[name]
		}
		if overlaid && source == nil {
			return nil, nil, fmt.Errorf("selected Go source %s is deleted by overlay", name)
		}
		if !overlaid {
			var err error
			source, err = os.ReadFile(name)
			if err != nil {
				return nil, nil, fmt.Errorf("read selected Go source %s: %w", name, err)
			}
		}
		file, err := parser.ParseFile(fset, name, source, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return nil, nil, fmt.Errorf("inspect selected Go imports %s: %w", name, err)
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, nil, fmt.Errorf("inspect selected Go import %s: %w", name, err)
			}
			if path == "C" {
				files = append(files, file)
				break
			}
		}
	}
	return fset, files, nil
}

func validateExplicitAssemblyOwners(inputs []llgoFileInput) error {
	seen := make(map[string]bool)
	for _, input := range inputs {
		if !nativeAssemblyFile(input.path) {
			continue
		}
		path := filepath.Clean(input.path)
		if seen[path] {
			return fmt.Errorf("assembly %s has duplicate explicit LLGoFiles owners", input.path)
		}
		seen[path] = true
	}
	return nil
}
