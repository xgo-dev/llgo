// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"go/version"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/plan9asm"
	llruntime "github.com/xgo-dev/llgo/runtime"
	llssa "github.com/xgo-dev/llgo/ssa"
)

// RuntimeOwnership is a source inventory, before LLVM optimization or linking.
// A declaration with no Go body records a boundary; it does not establish that
// an assembly implementation or external library satisfies that boundary.
type RuntimeOwnership struct {
	GoVersion               string            `json:"go_version"`
	GOOS                    string            `json:"goos"`
	GOARCH                  string            `json:"goarch"`
	Tags                    []string          `json:"tags"`
	ToolTags                []string          `json:"tool_tags"`
	CgoEnabled              bool              `json:"cgo_enabled"`
	Symbols                 []RuntimeSymbol   `json:"symbols"`
	Required                []RuntimeUse      `json:"required"`
	Missing                 []RuntimeUse      `json:"missing,omitempty"`
	DynamicCompilerUses     []RuntimeUse      `json:"dynamic_compiler_uses,omitempty"`
	CompilerSelectors       []RuntimeSelector `json:"compiler_selectors,omitempty"`
	CompilerForwarders      []RuntimeUse      `json:"compiler_forwarders,omitempty"`
	ExternalBoundaries      []RuntimeBoundary `json:"external_boundaries,omitempty"`
	UnresolvedBoundaries    []RuntimeUse      `json:"unresolved_boundaries,omitempty"`
	ABI                     *RuntimeABI       `json:"abi,omitempty"`
	OtherTargetCompilerUses []RuntimeUse      `json:"other_target_compiler_uses,omitempty"`
}

type RuntimeSelector struct {
	Expression string   `json:"expression"`
	Source     string   `json:"source"`
	Candidates []string `json:"candidates"`
}

type RuntimeSymbol struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Signature  string   `json:"signature,omitempty"`
	Owner      string   `json:"owner"`
	Source     string   `json:"source"`
	Linkname   string   `json:"linkname,omitempty"`
	Body       bool     `json:"body,omitempty"`
	Imports    []string `json:"imports,omitempty"`
	Directives []string `json:"directives,omitempty"`
	Condition  string   `json:"condition,omitempty"`
}

type RuntimeUse struct {
	Symbol string `json:"symbol"`
	Source string `json:"source"`
	Kind   string `json:"kind"`
}

type runtimeAuditFile struct {
	path string
	file *ast.File
}

type RuntimeAuditConfig struct {
	Dir          string // Application directory, used to resolve its Go toolchain.
	RuntimeDir   string
	GOOS, GOARCH string
	Tags         []string
	Env          []string
}

// AuditRuntimeSources uses the build's source overlays and target selection.
// It inventories the two runtime layers, their metadata ABI and internal/abi,
// and checks selected standard-library references to the public/linknamed
// runtime surface. The application's Go command selects GOROOT and feature tags.
// This developer diagnostic neither compiles nor modifies either source tree.
func AuditRuntimeSources(conf RuntimeAuditConfig) (*RuntimeOwnership, error) {
	if conf.RuntimeDir == "" || conf.GOOS == "" || conf.GOARCH == "" {
		return nil, fmt.Errorf("runtime audit requires RuntimeDir, GOOS and GOARCH")
	}
	root, err := filepath.Abs(conf.RuntimeDir)
	if err != nil {
		return nil, err
	}
	conf.RuntimeDir = root
	tags := append(splitSourcePatchBuildTags(DefaultBuildTags()), conf.Tags...)
	slices.Sort(tags)
	tags = slices.Compact(tags)
	flags := []string{"-tags=" + strings.Join(tags, ",")}
	environ := conf.Env
	if environ == nil {
		environ = os.Environ()
	}
	environ = withEnv(environ, "GOOS="+conf.GOOS, "GOARCH="+conf.GOARCH)
	source, err := resolveSourceGoConfig(commandEnv{dir: conf.Dir, environ: environ}, "", flags...)
	if err != nil {
		return nil, err
	}
	target := &llssa.Target{GOOS: conf.GOOS, GOARCH: conf.GOARCH}
	if conf.GOOS == "js" && conf.GOARCH == "wasm" && slices.Contains(tags, "llgo.wasm.emscripten.memory64") {
		target.LLVMTarget = "wasm64-unknown-emscripten"
	}
	tags = append(tags, target.ClosureEnvBuildTag())
	slices.Sort(tags)
	flags = []string{"-tags=" + strings.Join(slices.Compact(tags), ",")}
	cgoEnabled := source.CGO_ENABLED == "1"
	report, err := auditRuntimeSources(conf.RuntimeDir, source.GOROOT, sourcePatchBuildContext{
		goos: conf.GOOS, goarch: conf.GOARCH, goversion: source.GOVERSION,
		buildFlags: flags, toolTags: source.toolTags, cgoEnabled: &cgoEnabled})
	if err != nil {
		return nil, err
	}
	report.ABI, err = auditRuntimeABI(conf.RuntimeDir, source.GOROOT, source.GOVERSION, target)
	return report, err
}

func auditRuntimeSources(runtimeDir, goroot string, ctx sourcePatchBuildContext) (*RuntimeOwnership, error) {
	overlay, _, err := buildSourcePatchOverlayForGOROOT(nil, runtimeDir, goroot, ctx)
	if err != nil {
		return nil, err
	}
	overlay, err = runtimeSourceOverlay(overlay, runtimeDir, goroot, ctx.goversion)
	if err != nil {
		return nil, err
	}
	match, err := newSourcePatchMatchContext(goroot, ctx)
	if err != nil {
		return nil, err
	}
	match.OpenFile = func(path string) (io.ReadCloser, error) {
		if data, ok := overlay[path]; ok {
			return io.NopCloser(strings.NewReader(string(data))), nil
		}
		return os.Open(path)
	}
	a := runtimeAuditor{runtimeDir: runtimeDir, goroot: goroot, match: match,
		overlay: overlay, fset: token.NewFileSet(), definitions: make(map[string]string), implementations: make(map[string]string)}
	report := &RuntimeOwnership{GoVersion: ctx.goversion, GOOS: ctx.goos, GOARCH: ctx.goarch,
		Tags: match.BuildTags, ToolTags: ctx.toolTags, CgoEnabled: match.CgoEnabled}
	for _, pkg := range []struct{ dir, path, owner string }{
		{"internal/runtime", "github.com/xgo-dev/llgo/runtime/internal/runtime", "compiler-runtime"},
		{"abi", "github.com/xgo-dev/llgo/runtime/abi", "metadata-abi"},
		{"internal/lib/runtime", "runtime", "standard-runtime"},
	} {
		files, err := a.files(filepath.Join(runtimeDir, filepath.FromSlash(pkg.dir)))
		if err != nil {
			return nil, err
		}
		if err := a.inventory(report, pkg.path, pkg.owner, files); err != nil {
			return nil, err
		}
	}
	// skipall replaces executable declarations, but public GOROOT types and
	// constants remain visible to clients during package loading.
	runtimeFiles, err := a.files(filepath.Join(goroot, "src/runtime"))
	if err != nil {
		return nil, err
	}
	for _, file := range runtimeFiles {
		var decls []ast.Decl
		for _, decl := range file.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || (gen.Tok != token.TYPE && gen.Tok != token.CONST) {
				continue
			}
			for _, spec := range gen.Specs {
				var name string
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					name = spec.Name.Name
				case *ast.ValueSpec:
					if len(spec.Names) == 1 {
						name = spec.Names[0].Name
					}
				}
				if ast.IsExported(name) && a.definitions["runtime."+name] == "" {
					decls = append(decls, &ast.GenDecl{TokPos: gen.TokPos, Tok: gen.Tok, Specs: []ast.Spec{spec}})
				}
			}
		}
		file.file.Decls = decls
		if err := a.inventory(report, "runtime", "goroot", []runtimeAuditFile{file}); err != nil {
			return nil, err
		}
	}
	files, err := a.standardFiles("internal/abi")
	if err != nil {
		return nil, err
	}
	if err := a.inventory(report, "internal/abi", "metadata-adapter", files); err != nil {
		return nil, err
	}
	if version.Compare(ctx.goversion, "go1.23") < 0 {
		// This entry is injected into coverage mains by the existing builder;
		// it is not an absent runtime implementation on older Go versions.
		file, err := parser.ParseFile(a.fset, "$GENERATED/coverage_legacy.go",
			"package main\nimport _ \"unsafe\"\n"+coverageLegacyExitSupport, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		start := len(report.Symbols)
		if err := a.inventory(report, "main", "compiler-generated", []runtimeAuditFile{{"$GENERATED/coverage_legacy.go", file}}); err != nil {
			return nil, err
		}
		for i := start; i < len(report.Symbols); i++ {
			report.Symbols[i].Condition = "coverage instrumentation; Go < 1.23"
		}
	}
	err = filepath.WalkDir(filepath.Join(goroot, "src"), func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if name == "cmd" || name == "testdata" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(filepath.Join(goroot, "src"), path)
		if err != nil {
			return err
		}
		pkg := filepath.ToSlash(rel)
		if pkg == "runtime/cgo" {
			return filepath.SkipDir
		} // cl.pkgKindByPath: declaration-only.
		if pkg == "runtime" { // Its implementation was inventoried above.
			return nil
		}
		files, err := a.standardFiles(pkg)
		if err != nil {
			return err
		}
		if err := a.assembly(report, pkg, path); err != nil {
			return err
		}
		links, err := runtimeAuditLinknames(files)
		if err != nil {
			return err
		}
		var providers []runtimeAuditFile
		for _, file := range files {
			// Some packages provide runtime symbols, rather than consume them
			// (for example internal/fuzz and internal/runtime/maps).
			copy := *file.file
			copy.Decls = nil
			for _, decl := range file.file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Body != nil && strings.HasPrefix(links[fn.Name.Name], "runtime.") {
					copy.Decls = append(copy.Decls, fn)
				}
			}
			providers = append(providers, runtimeAuditFile{file.path, &copy})
			if err := a.requirements(report, file, links); err != nil {
				return err
			}
		}
		return a.inventory(report, pkg, "standard-library", providers)
	})
	if err != nil {
		return nil, err
	}
	if err := a.compilerUses(report); err != nil {
		return nil, err
	}
	if err := a.externalBoundaries(report); err != nil {
		return nil, err
	}
	a.resolve(report)
	return report, nil
}

func (a *runtimeAuditor) resolve(report *RuntimeOwnership) {
	known := make(map[string]bool)
	for _, symbol := range report.Symbols {
		known[symbol.Name] = true
		if symbol.Linkname != "" && symbol.Body {
			known[symbol.Linkname] = true
		}
	}
	slices.SortFunc(report.Symbols, func(a, b RuntimeSymbol) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(report.Required, func(a, b RuntimeUse) int {
		if c := strings.Compare(a.Symbol, b.Symbol); c != 0 {
			return c
		}
		return strings.Compare(a.Source, b.Source)
	})
	report.Required = slices.Compact(report.Required)
	for _, use := range report.Required {
		found := known[use.Symbol]
		if use.Kind == "linkname-variable" && strings.HasPrefix(use.Symbol, "runtime.") {
			// cl.replaceGoName sends variable aliases to the low-level runtime.
			found = known["github.com/xgo-dev/llgo/runtime/internal/runtime."+strings.TrimPrefix(use.Symbol, "runtime.")]
		}
		if !found {
			report.Missing = append(report.Missing, use)
		}
	}
}

type runtimeAuditor struct {
	runtimeDir, goroot string
	match              build.Context
	overlay            map[string][]byte
	fset               *token.FileSet
	definitions        map[string]string
	implementations    map[string]string
}

func (a *runtimeAuditor) source(pos token.Pos) string {
	p := a.fset.Position(pos)
	for _, root := range []struct{ path, name string }{{a.goroot, "$GOROOT"}, {filepath.Dir(a.runtimeDir), "$LLGO"}} {
		if rel, err := filepath.Rel(root.path, p.Filename); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Sprintf("%s/%s:%d", root.name, filepath.ToSlash(rel), p.Line)
		}
	}
	return p.String()
}

func (a *runtimeAuditor) files(dir string) ([]runtimeAuditFile, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		// Windows also reports ERROR_PATH_NOT_FOUND when ReadDir receives a
		// regular file. Only an absent path may represent an optional package.
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			return nil, nil
		}
	}
	if err != nil {
		return nil, err
	}
	names := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			names[entry.Name()] = true
		}
	}
	for path := range a.overlay {
		if filepath.Dir(path) == dir {
			names[filepath.Base(path)] = true
		}
	}
	var paths []string
	for name := range names {
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		ok, err := a.match.MatchFile(dir, name)
		if err != nil {
			return nil, err
		}
		if ok {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	slices.Sort(paths)
	var files []runtimeAuditFile
	for _, path := range paths {
		var src any
		if data, ok := a.overlay[path]; ok {
			src = data
		}
		file, err := parser.ParseFile(a.fset, path, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, runtimeAuditFile{path, file})
	}
	return files, nil
}

// Merge alternative packages using the same declaration keys and skip rules
// as source patching. runtime and reflect can skip all; internal/abi preserves
// upstream declarations not replaced by its boundary types and helpers.
func (a *runtimeAuditor) standardFiles(pkg string) ([]runtimeAuditFile, error) {
	files, err := a.files(filepath.Join(a.goroot, "src", filepath.FromSlash(pkg)))
	if err != nil || !llruntime.HasAltPkgForGOARCH(pkg, a.match.GOARCH) {
		return files, err
	}
	alts, err := a.files(filepath.Join(a.runtimeDir, "internal/lib", filepath.FromSlash(pkg)))
	if err != nil {
		return nil, err
	}
	skips := make(map[string]struct{})
	for _, alt := range alts {
		for _, comments := range alt.file.Comments {
			for _, comment := range comments.List {
				all, names, _ := parseSourcePatchDirective(comment.Text)
				if all {
					return alts, nil
				}
				for _, name := range names {
					skips[name] = struct{}{}
				}
			}
		}
		for _, decl := range alt.file.Decls {
			for _, key := range declPatchKeys(decl) {
				skips[key] = struct{}{}
			}
		}
	}
	for i, file := range files {
		src, ok := a.overlay[file.path]
		if !ok {
			src, err = os.ReadFile(file.path)
			if err != nil {
				return nil, err
			}
		}
		filtered, _, err := filterSourcePatchFile(src, skips)
		if err != nil {
			return nil, err
		}
		parsed, err := parser.ParseFile(a.fset, file.path, filtered, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files[i].file = parsed
	}
	return append(files, alts...), nil
}

func runtimeAuditLinknames(files []runtimeAuditFile) (map[string]string, error) {
	links := make(map[string]string)
	for _, file := range files {
		importsUnsafe := false
		for _, imp := range file.file.Imports {
			if imp.Path.Value == `"unsafe"` {
				importsUnsafe = true
			}
		}
		for _, group := range file.file.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name != "llgo:link" && (item.Name != "go:linkname" || !importsUnsafe) {
					continue
				}
				parts := strings.Fields(item.Args)
				if len(parts) != 2 {
					continue
				}
				if previous := links[parts[0]]; previous != "" && previous != parts[1] {
					return nil, fmt.Errorf("conflicting linkname for %s: %s and %s", parts[0], previous, parts[1])
				}
				links[parts[0]] = parts[1]
			}
		}
	}
	return links, nil
}

func (a *runtimeAuditor) inventory(report *RuntimeOwnership, pkg, owner string, files []runtimeAuditFile) error {
	links, err := runtimeAuditLinknames(files)
	if err != nil {
		return err
	}
	for _, file := range files {
		var imports []string
		for _, spec := range file.file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			imports = append(imports, path)
		}
		slices.Sort(imports)
		for _, decl := range file.file.Decls {
			kind, body := "", false
			switch d := decl.(type) {
			case *ast.FuncDecl:
				kind, body = "func", d.Body != nil
			case *ast.GenDecl:
				kind = d.Tok.String()
			}
			for _, key := range declPatchKeys(decl) {
				if key == "_" || key == "init" {
					continue
				}
				symbol := RuntimeSymbol{Name: pkg + "." + key, Kind: kind, Owner: owner,
					Source: a.source(decl.Pos()), Linkname: links[key], Body: body, Imports: imports}
				if fn, ok := decl.(*ast.FuncDecl); ok {
					var sig bytes.Buffer
					if err := format.Node(&sig, a.fset, fn.Type); err != nil {
						return err
					}
					symbol.Signature = sig.String()
				}
				if strings.HasPrefix(symbol.Source, "$GOROOT/") {
					symbol.Owner = "goroot"
				}
				if kind == "func" && !body {
					symbol.Owner = "external-boundary"
				}
				if strings.HasPrefix(symbol.Linkname, "llgo.") {
					symbol.Owner = "compiler-intrinsic"
				}
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Doc != nil {
					for _, c := range fn.Doc.List {
						if strings.HasPrefix(c.Text, "//go:") || strings.HasPrefix(c.Text, "//llgo:") {
							symbol.Directives = append(symbol.Directives, c.Text)
						}
					}
				}
				if previous := a.definitions[symbol.Name]; previous != "" {
					return fmt.Errorf("duplicate runtime declaration %s: %s and %s", symbol.Name, previous, symbol.Source)
				}
				a.definitions[symbol.Name] = symbol.Source
				if body || (kind == "var" && symbol.Linkname == "") {
					name := symbol.Name
					if symbol.Linkname != "" {
						name = symbol.Linkname
					}
					if previous := a.implementations[name]; previous != "" {
						return fmt.Errorf("duplicate runtime implementation %s: %s and %s", name, previous, symbol.Source)
					}
					a.implementations[name] = symbol.Source
				}
				report.Symbols = append(report.Symbols, symbol)
			}
		}
	}
	return nil
}

func (a *runtimeAuditor) requirements(report *RuntimeOwnership, file runtimeAuditFile, links map[string]string) error {
	for _, decl := range file.file.Decls {
		kind := "linkname-variable"
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if fn.Body != nil {
				continue
			}
			kind = "linkname-function"
		} else if gen, ok := decl.(*ast.GenDecl); !ok || gen.Tok != token.VAR {
			continue
		}
		for _, key := range declPatchKeys(decl) {
			if target := links[key]; runtimeAuditOwnedSymbol(target) {
				report.Required = append(report.Required, RuntimeUse{target, a.source(decl.Pos()), kind})
			}
		}
	}
	runtimeImports := make(map[string]string)
	for _, spec := range file.file.Imports {
		path, _ := strconv.Unquote(spec.Path.Value)
		if !runtimeAuditOwnedSymbol(path + ".") {
			continue
		}
		name := filepath.Base(path)
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			return fmt.Errorf("runtime audit cannot resolve dot import %s at %s", path, a.source(spec.Pos()))
		}
		runtimeImports[name] = path
	}
	ast.Inspect(file.file, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Obj == nil && runtimeImports[ident.Name] != "" {
				report.Required = append(report.Required, RuntimeUse{runtimeImports[ident.Name] + "." + sel.Sel.Name, a.source(sel.Pos()), "selector"})
			}
		}
		return true
	})
	return nil
}

func runtimeAuditOwnedSymbol(name string) bool {
	for _, pkg := range []string{"runtime", "internal/abi", "github.com/xgo-dev/llgo/runtime/abi", "github.com/xgo-dev/llgo/runtime/internal/runtime"} {
		if strings.HasPrefix(name, pkg+".") {
			return true
		}
	}
	return false
}

// Check literal lookups and every candidate of bounded dynamic selectors.
// Unbounded expressions remain explicit failures of the ownership gate.
func (a *runtimeAuditor) compilerUses(report *RuntimeOwnership) error {
	// Compiler sources contain lowering for every target. A literal used only
	// by a different target must still have a declaration somewhere in the ABI,
	// but is not a requirement of this target's standard-library source view.
	available := make(map[string]bool)
	dir := filepath.Join(a.runtimeDir, "internal/runtime")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(a.fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			for _, key := range declPatchKeys(decl) {
				available["github.com/xgo-dev/llgo/runtime/internal/runtime."+key] = true
			}
		}
	}
	for _, pkg := range []string{"ssa", "cl"} {
		dir := filepath.Join(filepath.Dir(a.runtimeDir), pkg)
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("runtime audit requires compiler sources: %w", err)
		}
		parsedFiles := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(a.fset, path, nil, 0)
			if err != nil {
				return err
			}
			parsedFiles++
			forwarders := runtimeAuditForwarders(pkg, entry.Name(), file)
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || !slices.Contains([]string{"rtFunc", "rtEnvFunc", "rtNamed", "rtType", "RuntimeFunc", "runtimeFunc"}, sel.Sel.Name) {
					return true
				}
				if forwarding := forwarders[call]; forwarding != "" {
					report.CompilerForwarders = append(report.CompilerForwarders, RuntimeUse{forwarding, a.source(call.Pos()), "compiler-forwarder"})
					return true
				}
				names, bounded := runtimeAuditCandidates(file, call.Args[0])
				if bounded {
					var expr bytes.Buffer
					_ = format.Node(&expr, a.fset, call.Args[0].(*ast.CallExpr).Args[0])
					report.CompilerSelectors = append(report.CompilerSelectors, RuntimeSelector{expr.String(), a.source(call.Pos()), names})
				}
				for _, name := range names {
					name = "github.com/xgo-dev/llgo/runtime/internal/runtime." + name
					use := RuntimeUse{name, a.source(call.Pos()), "compiler"}
					if a.definitions[name] == "" && available[name] {
						report.OtherTargetCompilerUses = append(report.OtherTargetCompilerUses, use)
					} else {
						report.Required = append(report.Required, use)
					}
				}
				if len(names) == 0 {
					var expr bytes.Buffer
					_ = format.Node(&expr, a.fset, call.Args[0])
					report.DynamicCompilerUses = append(report.DynamicCompilerUses, RuntimeUse{expr.String(), a.source(call.Pos()), "compiler-selector"})
				}
				return true
			})
		}
		if parsedFiles == 0 {
			return fmt.Errorf("runtime audit requires compiler sources in %s", dir)
		}
	}
	return nil
}

var runtimeAsmText = regexp.MustCompile(`(?m)^\s*TEXT\s+(runtime[·.][^\s(]+)\(SB\)`)

// Record source-level assembly entry declarations. Translation, preprocessor
// conditionals, signatures and register ABI still require the existing Plan 9
// validator and execution tests; this inventory is not an ABI proof.
func (a *runtimeAuditor) assembly(report *RuntimeOwnership, pkg, dir string) error {
	if llruntime.HasAltPkgForGOARCH(pkg, a.match.GOARCH) || llruntime.SourcePatchReplacesAsmForGOARCH(pkg, a.match.GOARCH) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".s") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		ok, err := a.match.MatchFile(dir, entry.Name())
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		src, err := plan9asm.ReadFileWithOverlay(a.overlay, path)
		if err != nil {
			return err
		}
		for _, entry := range runtimeAsmText.FindAllSubmatchIndex(src, -1) {
			name := plan9asm.ResolveSymFunc(pkg)(string(src[entry[2]:entry[3]]))
			file := a.fset.AddFile(path, -1, len(src))
			file.SetLinesForContent(src)
			if previous := a.implementations[name]; previous != "" {
				return fmt.Errorf("duplicate runtime assembly entry %s: %s and %s", name, previous, a.source(file.Pos(entry[2])))
			}
			a.implementations[name] = a.source(file.Pos(entry[2]))
			report.Symbols = append(report.Symbols, RuntimeSymbol{Name: name, Kind: "assembly-entry", Owner: "goroot-assembly", Source: a.source(file.Pos(entry[2]))})
		}
	}
	return nil
}
