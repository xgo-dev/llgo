package plan9asm

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	archcfg "github.com/xgo-dev/llgo/internal/goarch"
	"github.com/xgo-dev/llgo/internal/packages"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

const (
	maxAssemblySourceBytes = 4 << 20
	maxAssemblyInputBytes  = 16 << 20
	// The shared observation engine charges at most 16 MiB of token bytes
	// and 1,000,000 tokens, each with at most one emitted separator.
	maxAssemblyOutputBytes = (16 << 20) + 1_000_000
)

var errDeletedAssemblyOverlay = errors.New("assembly source deleted by overlay")

// PreprocessSourceForPkg returns the actual expanded bytes used by translation,
// routing, signature inspection and cache fingerprints. Includes always search
// the package directory, then the selected SDK; an include's parent never
// changes this order. Module.Dir permits module-local relative paths but is not
// an additional include search directory.
func PreprocessSourceForPkg(pkg *packages.Package, sfile string, src []byte, goos, goarch string, opt TranslateOptions) ([]byte, error) {
	if pkg == nil {
		return nil, fmt.Errorf("nil package")
	}
	env, err := assemblyTargetEnvironment(goos, goarch, opt)
	if err != nil {
		return nil, err
	}
	if opt.SourcePreprocessed {
		if len(src) > maxAssemblyOutputBytes {
			return nil, fmt.Errorf("expanded assembly exceeds its output bound")
		}
		return src, nil
	}
	if len(src) > maxAssemblySourceBytes {
		return nil, fmt.Errorf("assembly source exceeds its %d-byte bound", maxAssemblySourceBytes)
	}
	file, directory, err := assemblyPackageFile(pkg, sfile)
	if err != nil {
		return nil, err
	}
	root := opt.GOROOT
	if root == "" {
		// Compatibility for direct callers. The build driver supplies its
		// resolved source SDK explicitly, not the launcher's environment.
		root = runtime.GOROOT()
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	includeDir := filepath.Join(root, "pkg", "include")
	allowed := []string{directory, includeDir}
	if pkg.Module != nil && pkg.Module.Dir != "" {
		moduleDir, err := filepath.Abs(pkg.Module.Dir)
		if err != nil {
			return nil, err
		}
		allowed = append(allowed, moduleDir)
	}
	for _, dir := range append([]string(nil), allowed...) {
		if actual, err := filepath.EvalSymlinks(dir); err == nil {
			allowed = append(allowed, actual)
		}
	}
	remaining := maxAssemblyInputBytes - len(src)
	charge := func(data []byte) error {
		if len(data) > maxAssemblySourceBytes || len(data) > remaining {
			return fmt.Errorf("assembly includes exceed their input byte bound")
		}
		remaining -= len(data)
		return nil
	}
	readInclude := func(_ string, name string) (string, []byte, error) {
		if name == "" || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
			return "", nil, fmt.Errorf("include must be package-relative: %q", name)
		}
		searchDirs := []string{directory, includeDir}
		if name == "go_asm.h" {
			// cmd/go's generated objdir header precedes pkg/include. A real
			// package header still wins, but an SDK namesake cannot replace
			// definitions generated from this selected package's declarations.
			searchDirs = searchDirs[:1]
		}
		for _, search := range searchDirs {
			candidate := filepath.Join(search, filepath.FromSlash(name))
			if !assemblyPathAllowed(candidate, allowed) {
				return "", nil, fmt.Errorf("include escapes package/module/SDK roots: %q", name)
			}
			// Explicit overlays replace the source, including symlinks. Disk
			// sources must also remain inside the allowed roots after resolving.
			_, overlaid, err := assemblyOverlay(opt.Overlay, candidate)
			if err != nil {
				return "", nil, err
			}
			if !overlaid {
				actual, err := filepath.EvalSymlinks(candidate)
				if err == nil && !assemblyPathAllowed(actual, allowed) {
					return "", nil, fmt.Errorf("include symlink escapes allowed roots: %q", name)
				}
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return "", nil, err
				}
			}
			data, err := ReadFileWithOverlay(opt.Overlay, candidate)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return "", nil, err
			}
			if err := charge(data); err != nil {
				return "", nil, err
			}
			return candidate, data, nil
		}
		if name == "go_asm.h" {
			data, err := extplan9asm.GoAssemblyHeader(extplan9asm.GoPackage{
				Path: pkg.PkgPath, Types: pkg.Types, Syntax: pkg.Syntax,
			}, goarch)
			if err != nil {
				return "", nil, err
			}
			if err := charge(data); err != nil {
				return "", nil, err
			}
			return filepath.Join(directory, "go_asm.h"), data, nil
		}
		return "", nil, fmt.Errorf("active include %q not found in package or selected GOROOT", name)
	}
	version, err := assemblySDKVersion(root)
	if err != nil {
		return nil, fmt.Errorf("read selected SDK version: %w", err)
	}
	env["GOOS"], env["GOARCH"] = goos, goarch
	env["GOVERSION"] = version
	minor, _ := strconv.Atoi(strings.Split(strings.TrimPrefix(env["GOVERSION"], "go1."), ".")[0])
	if goarch == "arm64" && minor < 23 {
		// The build driver normalizes its default to v8.0, but older SDKs
		// never defined GOARM64 assembler macros, even for an explicit CPU.
		env["GOARM64"] = ""
	}
	if goarch == "wasm" {
		env["GOWASM"] = "" // cmd/asm registers no GOWASM_* macro.
	}
	defines, err := extplan9asm.GoAssemblerDefinesForEnvironment(goos, goarch, env)
	if err != nil {
		return nil, err
	}
	experiments, err := assemblyExperimentDefines(root, pkg.PkgPath, opt.ToolTags)
	if err != nil {
		return nil, err
	}
	defines = append(defines, experiments...)
	// PreprocessAssemblySource has no live output budget. This query executes
	// that same strict engine once, with its token/event/output accounting
	// enabled; discard the observation rather than expanding a second time.
	observed, _, err := extplan9asm.ObserveAssemblyCPPOutput(string(src), extplan9asm.AssemblyPreprocessOptions{
		FileName: file, Defines: defines, ReadInclude: readInclude,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: preprocess assembly: %w", sfile, err)
	}
	if len(observed.Output) > maxAssemblyOutputBytes {
		return nil, fmt.Errorf("expanded assembly exceeds its output bound")
	}
	return []byte(observed.Output), nil
}

// PreprocessFileForPkg reads the selected overlay/disk source and expands it
// once. Pass its result with SourcePreprocessed to the translation interface.
func PreprocessFileForPkg(pkg *packages.Package, sfile, goos, goarch string, opt TranslateOptions) ([]byte, error) {
	file, _, err := assemblyPackageFile(pkg, sfile)
	if err != nil {
		return nil, err
	}
	src, err := ReadFileWithOverlay(opt.Overlay, file)
	if err != nil {
		return nil, err
	}
	return PreprocessSourceForPkg(pkg, file, src, goos, goarch, opt)
}

// HasAnyTextAsmForPkg inspects the same selected, expanded source as translation.
// Callers already retaining expanded bytes can use HasTextAsmSource directly.
func HasAnyTextAsmForPkg(pkg *packages.Package, files []string, goos, goarch string, opt TranslateOptions) (bool, error) {
	for _, file := range files {
		src, err := PreprocessFileForPkg(pkg, file, goos, goarch, opt)
		if err != nil {
			return false, err
		}
		if HasTextAsmSource(src) {
			return true, nil
		}
	}
	return false, nil
}

func HasTextAsmSource(src []byte) bool { return reTextDirective.Match(src) }

func assemblyTargetEnvironment(goos, goarch string, opt TranslateOptions) (map[string]string, error) {
	if goos == "" || types.SizesFor("gc", goarch) == nil {
		return nil, fmt.Errorf("assembly requires an explicit valid target: %s/%s", goos, goarch)
	}
	env := map[string]string{}
	var err error
	switch goarch {
	case "386":
		env["GO386"], err = archcfg.Resolve386(opt.GO386)
	case "amd64":
		env["GOAMD64"], err = archcfg.ResolveAMD64(opt.GOAMD64)
	case "arm":
		var config archcfg.ARM
		config, err = archcfg.ParseARM(opt.GOARM)
		env["GOARM"] = opt.GOARM
		if opt.GOARM == "" {
			env["GOARM"] = config.Version
		}
	case "arm64":
		var config archcfg.ARM64
		config, err = archcfg.ParseARM64(opt.GOARM64)
		env["GOARM64"] = config.String()
	}
	return env, err
}

func assemblyPackageFile(pkg *packages.Package, file string) (string, string, error) {
	if pkg == nil || file == "" {
		return "", "", fmt.Errorf("assembly requires a package and filename")
	}
	directory := pkg.Dir
	if directory != "" && !filepath.IsAbs(file) {
		file = filepath.Join(directory, file)
	}
	file, err := filepath.Abs(file)
	if err != nil {
		return "", "", err
	}
	if directory == "" {
		directory = filepath.Dir(file)
	}
	directory, err = filepath.Abs(directory)
	return file, directory, err
}

func assemblyPathAllowed(file string, roots []string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, file)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

// cmd/asm registers experiment macros only for its SDK's AllowAsmABI package
// paths. Read the selected sources, not a baked-in package list or host tags.
// SDKs preceding that registration deliberately get no experiment macros.
func assemblyExperimentDefines(root, pkgPath string, tags []string) ([]string, error) {
	var experiments []string
	for _, tag := range tags {
		if name, ok := strings.CutPrefix(tag, "goexperiment."); ok {
			if name == "" || name[0] < 'a' || name[0] > 'z' || strings.IndexFunc(name, func(r rune) bool {
				return (r < 'a' || r > 'z') && (r < '0' || r > '9')
			}) >= 0 {
				return nil, fmt.Errorf("invalid selected Go experiment tag %q", tag)
			}
			experiments = append(experiments, "GOEXPERIMENT_"+name)
		}
	}
	if len(experiments) == 0 {
		return nil, nil
	}
	readAST := func(name string) (*ast.File, error) {
		file := filepath.Join(root, "src", filepath.FromSlash(name))
		data, err := readAssemblyFile(nil, file)
		if err != nil {
			return nil, fmt.Errorf("read selected SDK assembler registration: %w", err)
		}
		return parser.ParseFile(token.NewFileSet(), file, data, 0)
	}
	main, err := readAST("cmd/asm/main.go")
	if err != nil {
		return nil, err
	}
	registrations := 0
	ast.Inspect(main, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if ok && literal.Kind == token.STRING {
			value, _ := strconv.Unquote(literal.Value)
			if strings.HasPrefix(value, "GOEXPERIMENT_") {
				registrations++
			}
		}
		return true
	})
	if registrations == 0 {
		return nil, nil
	}
	if registrations != 1 {
		return nil, fmt.Errorf("unsupported selected SDK assembler experiment registration")
	}
	roleConditions := 0
	ast.Inspect(main, func(node ast.Node) bool {
		statement, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		var condition, body bytes.Buffer
		if format.Node(&condition, token.NewFileSet(), statement.Cond) == nil &&
			format.Node(&body, token.NewFileSet(), statement.Body) == nil &&
			condition.String() == "objabi.LookupPkgSpecial(ctxt.Pkgpath).AllowAsmABI" &&
			strings.Contains(body.String(), `"GOEXPERIMENT_"`) {
			roleConditions++
		}
		return true
	})
	if roleConditions != 1 {
		return nil, fmt.Errorf("unsupported selected SDK assembler experiment package-role condition")
	}
	special, err := readAST("cmd/internal/objabi/pkgspecial.go")
	if err != nil {
		return nil, err
	}
	var allowed *ast.CompositeLit
	for _, decl := range special.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				if name.Name == "allowAsmABIPkgs" {
					if allowed != nil || len(value.Names) != 1 || len(value.Values) != 1 {
						return nil, fmt.Errorf("ambiguous selected SDK assembler package roles")
					}
					allowed, ok = value.Values[0].(*ast.CompositeLit)
					if !ok {
						return nil, fmt.Errorf("nonliteral selected SDK assembler package roles")
					}
				}
			}
		}
	}
	if allowed == nil {
		return nil, fmt.Errorf("missing selected SDK assembler package roles")
	}
	array, ok := allowed.Type.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return nil, fmt.Errorf("nonliteral selected SDK assembler package role list")
	}
	if element, ok := array.Elt.(*ast.Ident); !ok || element.Name != "string" {
		return nil, fmt.Errorf("invalid selected SDK assembler package role list")
	}
	paths := map[string]bool{}
	for _, element := range allowed.Elts {
		literal, ok := element.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return nil, fmt.Errorf("nonliteral selected SDK assembler package role")
		}
		name, err := strconv.Unquote(literal.Value)
		if err != nil || name == "" || paths[name] {
			return nil, fmt.Errorf("invalid selected SDK assembler package role")
		}
		paths[name] = true
	}
	if paths[pkgPath] {
		return experiments, nil
	}
	return nil, nil
}

func assemblyOverlay(overlay map[string][]byte, file string) ([]byte, bool, error) {
	file, err := filepath.Abs(file)
	if err != nil {
		return nil, false, err
	}
	var selected []byte
	found := false
	for name, data := range overlay {
		absolute, err := filepath.Abs(name)
		if err != nil {
			return nil, false, err
		}
		if absolute != file {
			continue
		}
		if found && ((selected == nil) != (data == nil) || !bytes.Equal(selected, data)) {
			return nil, false, fmt.Errorf("conflicting assembly overlay aliases for %s", file)
		}
		selected, found = data, true
	}
	if found && selected == nil {
		return nil, true, fmt.Errorf("%w: %s", errDeletedAssemblyOverlay, file)
	}
	return selected, found, nil
}

func readAssemblyFile(overlay map[string][]byte, file string) ([]byte, error) {
	data, found, err := assemblyOverlay(overlay, file)
	if err != nil {
		return nil, err
	}
	if found {
		if len(data) > maxAssemblySourceBytes {
			return nil, fmt.Errorf("assembly overlay exceeds its input byte bound: %s", file)
		}
		return append([]byte{}, data...), nil
	}
	info, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("assembly source is not a regular file: %s", file)
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("assembly source is not a regular file: %s", file)
	}
	data, err = io.ReadAll(io.LimitReader(f, maxAssemblySourceBytes+1))
	if err == nil && len(data) > maxAssemblySourceBytes {
		err = fmt.Errorf("assembly source exceeds its input byte bound: %s", file)
	}
	return data, err
}
