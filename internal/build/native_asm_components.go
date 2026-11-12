package build

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	llplan9asm "github.com/xgo-dev/llgo/internal/plan9asm"
	"github.com/xgo-dev/llgo/ssa/abi"
	extplan9asm "github.com/xgo-dev/plan9asm"
)

type plan9AsmSource struct {
	path   string
	source []byte
}

type nativeAsmPlan struct {
	files      map[string]extplan9asm.NativeOptions
	aliases    map[string]string // actual Go variable -> selected native object
	unresolved map[string][]string
}

type nativeAsmNode struct {
	file       string
	symbol     string
	object     string
	references []string
}

func captureSelectedPlan9Asm(ctx *context, pkg *packages.Package, files []string) ([]plan9AsmSource, error) {
	sources := make([]plan9AsmSource, 0, len(files))
	opt := plan9asmTranslateOptions(ctx)
	for _, path := range files {
		source, err := llplan9asm.PreprocessFileForPkg(pkg, path, ctx.buildConf.Goos, ctx.buildConf.Goarch, opt)
		if err != nil {
			return nil, fmt.Errorf("%s: preprocess %s: %w", pkg.PkgPath, path, err)
		}
		if darwinStdlibImportTrampolines(ctx, pkg, path, source) {
			continue
		}
		sources = append(sources, plan9AsmSource{path, source})
	}
	return sources, nil
}

// This is request-local source selection, not a cached permission or an
// inferred C signature. File contents are captured anew for each compiler use.
func planNativeAsmComponents(ctx *context, pkg *packages.Package, sources []plan9AsmSource) (*nativeAsmPlan, error) {
	plan := &nativeAsmPlan{files: make(map[string]extplan9asm.NativeOptions), aliases: make(map[string]string), unresolved: make(map[string][]string)}
	if !extplan9asm.SupportsNativeTarget(ctx.buildConf.Goos, ctx.buildConf.Goarch) || pkg.Types == nil {
		return plan, nil
	}
	variables, links, declarations := nativeAsmSourceDeclarations(pkg)
	parsed := make(map[string]*extplan9asm.File)
	nodes := make(map[string]*nativeAsmNode)
	roots := make(map[string]bool)
	dataRoots := make(map[string][]string)
	globalOwners := make(map[string]string)
	options := func() extplan9asm.NativeOptions {
		return extplan9asm.NativeOptions{GOOS: ctx.buildConf.Goos, GOARCH: ctx.buildConf.Goarch, PackagePath: abi.PathOf(pkg.Types), Imports: make(map[string]string), CodeEntries: make(map[string]string)}
	}
	for _, source := range sources {
		imports, native, err := foreignNativeImports(ctx, pkg, source.source)
		if err != nil {
			return nil, err
		}
		if native {
			opt := options()
			opt.Imports = imports
			plan.files[source.path] = opt
			continue // Preserve the existing explicitly imported file-local role.
		}
		file, err := extplan9asm.Parse(extplan9asm.Arch(ctx.buildConf.Goarch), string(source.source))
		if err != nil {
			continue
		} // The ordinary translator retains its diagnostic.
		typed := false
		for _, fn := range file.Funcs {
			typed = typed || !token.IsIdentifier(fn.Sym) || fn.Sym == "_" || pkg.Types.Scope().Lookup(fn.Sym) != nil || declarations[fn.Sym]
		}
		if typed || len(file.Funcs) == 0 && len(file.Globl) == 0 {
			continue
		}
		parsed[source.path] = file
		for _, fn := range file.Funcs {
			if nodes[fn.Sym] != nil {
				return nil, fmt.Errorf("duplicate selected native TEXT %s", fn.Sym)
			}
			identity := sha256.Sum256([]byte(pkg.Types.Path() + "\x00" + fn.Sym))
			node := &nativeAsmNode{file: source.path, symbol: fn.Sym, object: fmt.Sprintf("__llgo_native_address_%x", identity)}
			for _, ins := range fn.Instrs[1:] {
				for _, arg := range ins.Args {
					symbol := strings.TrimPrefix(arg.String(), "$")
					if strings.HasSuffix(symbol, "(SB)") {
						node.references = append(node.references, strings.TrimSuffix(symbol, "(SB)"))
					}
				}
			}
			nodes[fn.Sym] = node
		}
		for _, global := range file.Globl {
			if prior, duplicate := globalOwners[global.Sym]; duplicate {
				return nil, fmt.Errorf("native DATA owner %s is duplicated by %s and %s", global.Sym, prior, source.path)
			}
			globalOwners[global.Sym] = source.path
			name := strings.TrimPrefix(global.Sym, "·")
			if variable, ok := pkg.Types.Scope().Lookup(name).(*types.Var); ok && strings.HasPrefix(global.Sym, "·") {
				if variables[name] != variable || links[name] != "" {
					return nil, fmt.Errorf("native DATA does not own uninitialized Go variable %s", name)
				}
			}
		}
		globals := make(map[string]int64, len(file.Globl))
		counts := make(map[string]int, len(file.Data))
		for _, global := range file.Globl {
			if _, duplicate := globals[global.Sym]; duplicate {
				return nil, fmt.Errorf("duplicate native GLOBL %s", global.Sym)
			}
			globals[global.Sym] = global.Size
		}
		for _, data := range file.Data {
			counts[data.Sym]++
		}
		for _, data := range file.Data {
			name := strings.TrimPrefix(data.Sym, "·")
			variable := variables[name]
			if variable == nil || links[name] != "" || !strings.HasPrefix(data.Sym, "·") || data.Off != 0 || data.Width != 8 {
				continue
			}
			basic, ok := variable.Type().Underlying().(*types.Basic)
			if !ok || basic.Kind() != types.Uintptr {
				continue
			}
			target := strings.TrimSuffix(data.Addr, "(SB)")
			if globals[data.Sym] == 8 && counts[data.Sym] == 1 && data.Addr == target+"(SB)" {
				roots[target] = true
				dataRoots[source.path] = append(dataRoots[source.path], target)
			}
		}
	}
	for name, target := range links {
		variable := variables[name]
		if variable == nil || nodes[target] == nil || !nativeAsmAddressUses(pkg, variable) {
			continue
		}
		nodes[target].object = target // Preserve the explicit original code-address linkname.
		roots[target] = true
		plan.aliases[name] = target
	}
	if len(roots) == 0 {
		return plan, nil
	}
	objects := make(map[string]bool)
	for _, node := range nodes {
		objects[node.object] = true
	}
	for _, file := range pkg.Syntax {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name != "go:linkname" && item.Name != "llgo:link" && item.Name != "go:cgo_import_dynamic" {
					continue
				}
				fields := strings.Fields(item.Args)
				if len(fields) > 1 && objects[fields[1]] && plan.aliases[fields[0]] != fields[1] {
					return nil, fmt.Errorf("native code object collides with source alias %s", fields[1])
				}
			}
		}
	}
	reached := make(map[string]bool)
	var visit func(string)
	visit = func(symbol string) {
		node := nodes[symbol]
		if node == nil || reached[symbol] {
			return
		}
		reached[symbol] = true
		for _, target := range node.references {
			visit(target)
		}
	}
	for root := range roots {
		visit(root)
	}
	selectedData := make(map[string]bool)
	for symbol := range reached {
		for _, reference := range nodes[symbol].references {
			if path, exists := globalOwners[reference]; exists {
				selectedData[path] = true
			}
		}
	}
	for path, targets := range dataRoots {
		for _, target := range targets {
			selectedData[path] = selectedData[path] || reached[target]
		}
	}
	for path, file := range parsed {
		selected := selectedData[path]
		for _, fn := range file.Funcs {
			selected = selected || reached[fn.Sym]
		}
		if !selected {
			continue
		} // Unowned code keeps the original Go declaration checks.
		opt := options()
		for _, target := range dataRoots[path] {
			if node := nodes[target]; node != nil && node.file != path {
				opt.Imports[target] = node.object
			}
		}
		for _, fn := range file.Funcs {
			node := nodes[fn.Sym]
			if !reached[fn.Sym] {
				return nil, fmt.Errorf("native component has unreachable TEXT %s", fn.Sym)
			}
			opt.CodeEntries[fn.Sym] = node.object
			for _, target := range node.references {
				if dependency := nodes[target]; dependency != nil {
					if dependency.file != path {
						opt.Imports[target] = dependency.object
					}
					continue
				}
				if strings.HasPrefix(target, "·") {
					name := strings.TrimPrefix(target, "·")
					if _, ok := pkg.Types.Scope().Lookup(name).(*types.Var); ok {
						object := links[name]
						if object == "" || object == "<export>" {
							object = abi.PathOf(pkg.Types) + "." + name
						}
						opt.Imports[target] = object
						continue
					}
				}
				plan.unresolved[path] = append(plan.unresolved[path], target)
			}
		}
		plan.files[path] = opt
	}
	return plan, nil
}

func nativeAsmSourceDeclarations(pkg *packages.Package) (map[string]*types.Var, map[string]string, map[string]bool) {
	variables := make(map[string]*types.Var)
	links := make(map[string]string)
	functions := make(map[string]bool)
	for _, file := range pkg.Syntax {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR {
				continue
			}
			for _, spec := range group.Specs {
				value := spec.(*ast.ValueSpec)
				if len(value.Values) != 0 {
					continue
				}
				for _, name := range value.Names {
					if variable, ok := pkg.Types.Scope().Lookup(name.Name).(*types.Var); ok && variable.Pos() == name.Pos() {
						variables[name.Name] = variable
					}
				}
			}
		}
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name != "go:linkname" && item.Name != "llgo:link" {
					continue
				}
				fields := strings.Fields(item.Args)
				if len(fields) < 1 {
					continue
				}
				target := "<export>"
				if len(fields) > 1 {
					target = fields[1]
				}
				links[fields[0]] = target
				if _, ok := pkg.Types.Scope().Lookup(fields[0]).(*types.Func); ok {
					functions[target] = true
				}
			}
		}
	}
	return variables, links, functions
}

// All uses of the source byte alias must be actual pointer-to-uintptr
// conversions. A read, write or pointer escape cannot grant a code-address role.
func nativeAsmAddressUses(pkg *packages.Package, variable *types.Var) bool {
	basic, ok := variable.Type().Underlying().(*types.Basic)
	if !ok || basic.Kind() != types.Uint8 || pkg.TypesInfo == nil {
		return false
	}
	covered := make(map[token.Pos]bool)
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(node ast.Node) bool {
			outer, ok := node.(*ast.CallExpr)
			if !ok || len(outer.Args) != 1 {
				return true
			}
			conversion := pkg.TypesInfo.Types[outer.Fun]
			if !conversion.IsType() || conversion.Type == nil {
				return true
			}
			kind, ok := conversion.Type.Underlying().(*types.Basic)
			if !ok || kind.Kind() != types.Uintptr {
				return true
			}
			inner, ok := ast.Unparen(outer.Args[0]).(*ast.CallExpr)
			if !ok || len(inner.Args) != 1 {
				return true
			}
			pointer := pkg.TypesInfo.Types[inner.Fun]
			if !pointer.IsType() || pointer.Type == nil {
				return true
			}
			kind, ok = pointer.Type.Underlying().(*types.Basic)
			if !ok || kind.Kind() != types.UnsafePointer {
				return true
			}
			address, ok := ast.Unparen(inner.Args[0]).(*ast.UnaryExpr)
			if !ok || address.Op != token.AND {
				return true
			}
			name, ok := ast.Unparen(address.X).(*ast.Ident)
			if ok && pkg.TypesInfo.Uses[name] == variable {
				covered[name.Pos()] = true
			}
			return true
		})
	}
	uses := 0
	for name, object := range pkg.TypesInfo.Uses {
		if object == variable {
			uses++
			if !covered[name.Pos()] {
				return false
			}
		}
	}
	return uses > 0
}

// External native providers are bound only during real compilation, when the
// selected dependency graph has compiler-owned C export snapshots available.
func bindNativeAsmExports(ctx *context, current *aPackage, pkg *packages.Package, plan *nativeAsmPlan) error {
	if len(plan.files) == 0 {
		return nil
	}
	// A component cannot grant an ABI to unused exports. This also preserves
	// older native-only files whose explicit imports are already bound.
	interested := make(map[string]bool)
	for _, references := range plan.unresolved {
		for _, target := range references {
			interested[target] = true
		}
	}
	for _, options := range plan.files {
		for _, object := range options.CodeEntries {
			interested[object] = true
		}
	}
	if len(interested) == 0 {
		return nil
	}
	type provider struct{ object, owner string }
	providers := make(map[string]provider)
	insert := func(name, object, owner string) error {
		previous, exists := providers[name]
		if exists && (previous.object != object || previous.owner != "" && owner != "" && previous.owner != owner) {
			return fmt.Errorf("conflicting native providers for %s: %s (%s) and %s (%s)", name, previous.object, previous.owner, object, owner)
		}
		if owner == "" {
			owner = previous.owner
		}
		providers[name] = provider{object, owner}
		return nil
	}
	_, declarations := collectGoCgoPragmas(pkg.Syntax)
	for _, declaration := range declarations {
		if interested[declaration.local] {
			if err := insert(declaration.local, declaration.alias, ""); err != nil {
				return err
			}
		}
	}
	visited := make(map[*packages.Package]bool)
	var visit func(*packages.Package) error
	visit = func(dependency *packages.Package) error {
		if dependency == nil || visited[dependency] {
			return nil
		}
		visited[dependency] = true
		if compiled := ctx.pkgs[dependency]; compiled != nil {
			for _, export := range packageCExports(compiled) {
				if !interested[export.cName] {
					continue
				}
				if export.sig == nil {
					return fmt.Errorf("native export %s has no compiled signature", export.cName)
				}
				if err := insert(export.cName, export.cName, export.goName); err != nil {
					return err
				}
			}
		}
		for _, child := range dependency.Imports {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(pkg); err != nil {
		return err
	}
	if current.NeedRt {
		for dependency := range ctx.pkgs {
			if dependency.PkgPath == "runtime" {
				if err := visit(dependency); err != nil {
					return err
				}
			}
		}
	}
	for path, references := range plan.unresolved {
		options := plan.files[path]
		for _, target := range references {
			object := providers[target].object
			if object == "" {
				return fmt.Errorf("native assembly %s has no selected provider for %s", path, target)
			}
			options.Imports[target] = object
		}
		plan.files[path] = options
	}
	for _, options := range plan.files {
		for _, object := range options.CodeEntries {
			if providers[object].object != "" {
				return fmt.Errorf("native object %s conflicts with a selected C export", object)
			}
		}
	}
	return nil
}
