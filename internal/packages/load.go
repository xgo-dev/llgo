/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package packages

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"go/version"
	"log"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/directive"
	"golang.org/x/tools/go/packages"
)

// A LoadMode controls the amount of detail to return when loading.
// The bits below can be combined to specify which fields should be
// filled in the result packages.
// The zero value is a special case, equivalent to combining
// the NeedName, NeedFiles, and NeedCompiledGoFiles bits.
// ID and Errors (if present) will always be filled.
// Load may return more information than requested.
type LoadMode = packages.LoadMode

const (
	NeedName  = packages.NeedName
	NeedFiles = packages.NeedFiles

	NeedSyntax     = packages.NeedSyntax
	NeedImports    = packages.NeedImports
	NeedDeps       = packages.NeedDeps
	NeedModule     = packages.NeedModule
	NeedExportFile = packages.NeedExportFile

	NeedEmbedFiles      = packages.NeedEmbedFiles
	NeedEmbedPatterns   = packages.NeedEmbedPatterns
	NeedCompiledGoFiles = packages.NeedCompiledGoFiles

	NeedTypes      = packages.NeedTypes
	NeedTypesSizes = packages.NeedTypesSizes
	NeedTypesInfo  = packages.NeedTypesInfo

	NeedForTest = packages.NeedForTest
)

const (
	DebugPackagesLoad = false
)

// A Config specifies details about how packages should be loaded.
// The zero value is a valid configuration.
// Calls to Load do not modify this struct.
type Config = packages.Config
type Error = packages.Error

// A Package describes a loaded Go package.
type Package = packages.Package

type Cached struct {
	*packages.Package
	Types     *types.Package
	TypesInfo *types.Info
	Syntax    []*ast.File
}

type aDeduper struct {
	directives *directive.Index
	cache      sync.Map
	checked    sync.Map
	setpath    func(path string, name string) string
	preload    func(pkg *packages.Package)
	prepare    func([]*Package, *Config) error
	llgoFiles  map[string][]string
}

type Deduper = *aDeduper

func NewDeduper() Deduper {
	return &aDeduper{}
}

func (p Deduper) SetDirectives(index *directive.Index) { p.directives = index }

func (p Deduper) SetPreload(fn func(pkg *packages.Package)) {
	p.preload = fn
}

// SetPrepare installs a graph preprocessing hook before parsing/type checking.
// The hook may add generated files and their imports (for example, coverage).
func (p Deduper) SetPrepare(fn func([]*Package, *Config) error) {
	p.prepare = fn
}

func (p Deduper) SetPkgPath(fn func(path, name string) string) {
	p.setpath = fn
}

func (p Deduper) SetLLGoFiles(files map[string][]string) {
	p.llgoFiles = files
}

func (p Deduper) Check(id string) *Cached {
	if v, ok := p.cache.Load(id); ok {
		return v.(*Cached)
	}
	return nil
}

func (p Deduper) set(id string, cp *Cached) {
	if DebugPackagesLoad {
		log.Println("==> Import", id)
	}
	p.cache.Store(id, cp)
}

// Visit visits all the packages in the import graph whose roots are
// pkgs, calling the optional pre function the first time each package
// is encountered (preorder), and the optional post function after a
// package's dependencies have been visited (postorder).
// The boolean result of pre(pkg) determines whether
// the imports of package pkg are visited.
func Visit(pkgs []*Package, pre func(*Package) bool, post func(*Package)) {
	packages.Visit(pkgs, pre, post)
}

// An importFunc is an implementation of the single-method
// types.Importer interface based on a function value.
type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// LoadEx loads and returns the Go packages named by the given patterns.
//
// Config specifies loading options;
// nil behaves the same as an empty Config.
//
// If any of the patterns was invalid as defined by the
// underlying build system, Load returns an error.
// It may return an empty list of packages without an error,
// for instance for an empty expansion of a valid wildcard.
// Errors associated with a particular package are recorded in the
// corresponding Package's Errors list, and do not cause Load to
// return an error. Clients may need to handle such errors before
// proceeding with further analysis. The PrintErrors function is
// provided for convenient display of all errors.
func LoadEx(dedup Deduper, sizes func(sizes types.Sizes, compiler, arch string) types.Sizes, cfg *Config, patterns ...string) ([]*Package, error) {
	return LoadExWithGoVersion(dedup, sizes, cfg, "", patterns...)
}

// LoadExWithGoVersion is LoadEx with an optional go/types language version
// override. The version uses go/types syntax, such as "go1.22".
func LoadExWithGoVersion(dedup Deduper, sizes func(sizes types.Sizes, compiler, arch string) types.Sizes, cfg *Config, goVersion string, patterns ...string) ([]*Package, error) {
	var driverCfg Config
	if cfg != nil {
		driverCfg = *cfg
	}
	origMode := driverCfg.Mode

	// When type information or custom syntax parsing is requested, we do not let
	// packages.Load typecheck or parse directly. We request files, imports, embed patterns,
	// and module metadata from packages.Load (go list driver), and perform custom parsing
	// and typechecking ourselves.
	driverCfg.Mode = (origMode &^ (NeedTypes | NeedTypesSizes | NeedTypesInfo | NeedSyntax)) | NeedCompiledGoFiles | NeedImports | NeedName | NeedFiles
	if origMode&(NeedEmbedPatterns|NeedEmbedFiles|NeedTypes|NeedTypesInfo|NeedSyntax) != 0 {
		driverCfg.Mode |= NeedEmbedPatterns | NeedEmbedFiles | NeedExportFile
	}
	if origMode&NeedTypesSizes != 0 {
		driverCfg.Mode |= NeedTypesSizes
	}
	if origMode&NeedModule != 0 || origMode&(NeedTypes|NeedTypesInfo) != 0 {
		driverCfg.Mode |= NeedModule
	}

	initial, err := packages.Load(&driverCfg, patterns...)
	if err != nil {
		return nil, err
	}
	if dedup != nil && dedup.prepare != nil {
		if err := dedup.prepare(initial, &driverCfg); err != nil {
			return nil, err
		}
	}

	fset := driverCfg.Fset
	if fset == nil {
		fset = token.NewFileSet()
	}

	if origMode&(NeedTypes|NeedTypesInfo|NeedTypesSizes|NeedSyntax) != 0 {
		tc := &typecheckContext{
			dedup:     dedup,
			sizesFn:   sizes,
			cfg:       &driverCfg,
			fset:      fset,
			goVersion: goVersion,
			origMode:  origMode,
		}

		// Perform bottom-up typechecking in dependency post-order
		packages.Visit(initial, nil, func(pkg *Package) {
			tc.typecheckPackage(pkg)
		})
	}

	return initial, nil
}

type typecheckContext struct {
	dedup     Deduper
	sizesFn   func(sizes types.Sizes, compiler, arch string) types.Sizes
	cfg       *Config
	fset      *token.FileSet
	goVersion string
	origMode  LoadMode
}

func (tc *typecheckContext) targetGoVersion(pkg *Package) string {
	if tc.goVersion != "" {
		return tc.goVersion
	}
	if pkg.Module != nil && pkg.Module.GoVersion != "" {
		return "go" + pkg.Module.GoVersion
	}
	return ""
}

func (tc *typecheckContext) parseFile(filename string, fset *token.FileSet) (*ast.File, error) {
	fullPath := filename
	if !filepath.IsAbs(fullPath) && tc.cfg.Dir != "" {
		fullPath = filepath.Join(tc.cfg.Dir, fullPath)
	}
	var src []byte
	hasSrc := false
	if tc.cfg.Overlay != nil {
		if data, ok := tc.cfg.Overlay[fullPath]; ok {
			src, hasSrc = data, true
		} else if data, ok := tc.cfg.Overlay[filename]; ok {
			src, hasSrc = data, true
		}
	}
	if !hasSrc {
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		src = data
	}
	if tc.cfg.ParseFile != nil {
		return tc.cfg.ParseFile(fset, fullPath, src)
	}
	return parser.ParseFile(fset, fullPath, src, parser.AllErrors|parser.ParseComments)
}

func (tc *typecheckContext) targetCompilerAndArch() (compiler, arch string) {
	compiler = "gc"
	if tc.cfg != nil {
		for _, env := range tc.cfg.Env {
			if strings.HasPrefix(env, "GOARCH=") {
				arch = env[len("GOARCH="):]
			}
		}
	}
	if arch == "" {
		arch = os.Getenv("GOARCH")
	}
	if arch == "" {
		arch = runtime.GOARCH
	}
	return compiler, arch
}

// computedSizes determines the appropriate types.Sizes for the target package.
// When cross-compiling for WebAssembly (wasm), types.SizesFor("gc", "wasm") may
// return nil on certain Go toolchain configurations; we explicitly fall back to
// 32-bit word size and 4-byte alignment (&types.StdSizes{WordSize: 4, MaxAlign: 4})
// matching the wasm32 ABI.
func (tc *typecheckContext) computedSizes(pkg *Package) types.Sizes {
	compiler, arch := tc.targetCompilerAndArch()
	s := pkg.TypesSizes
	if s == nil {
		s = types.SizesFor(compiler, arch)
		if s == nil {
			if arch == "wasm" {
				s = &types.StdSizes{WordSize: 4, MaxAlign: 4}
			} else {
				s = types.SizesFor("gc", "amd64")
			}
		}
	}
	if tc.sizesFn != nil {
		s = tc.sizesFn(s, compiler, arch)
	}
	return s
}

func (tc *typecheckContext) typecheckPackage(pkg *Package) {
	fset := pkg.Fset
	if fset == nil {
		fset = tc.fset
		pkg.Fset = fset
	}

	if pkg.PkgPath == "unsafe" {
		pkg.Types = types.Unsafe
		pkg.Fset = fset
		pkg.Syntax = []*ast.File{}
		pkg.TypesInfo = new(types.Info)
		pkg.TypesSizes = tc.computedSizes(pkg)
		return
	}

	if tc.dedup != nil {
		if cp := tc.dedup.Check(pkg.ID); cp != nil {
			pkg.Types = cp.Types
			pkg.Fset = fset
			pkg.TypesInfo = cp.TypesInfo
			pkg.Syntax = cp.Syntax
			pkg.TypesSizes = tc.computedSizes(pkg)
			return
		}
		defer func() {
			if !pkg.IllTyped && pkg.Types != nil && pkg.Types.Complete() {
				// The ordinary package and its test-augmented variant share
				// PkgPath but not types.Package identity. Match the lookup key
				// above so a later load cannot import the wrong variant.
				tc.dedup.set(pkg.ID, &Cached{
					Package:   pkg,
					Types:     pkg.Types,
					TypesInfo: pkg.TypesInfo,
					Syntax:    pkg.Syntax,
				})
			}
		}()
		if tc.dedup.setpath != nil {
			pkg.PkgPath = tc.dedup.setpath(pkg.PkgPath, pkg.Name)
		}
		// Source-patch files are selected per import path and intentionally
		// shared by the ordinary and test-augmented variants.
		if _, ok := tc.dedup.checked.Load(pkg.PkgPath); !ok {
			tc.dedup.checked.Store(pkg.PkgPath, struct{}{})
			if files, ok := tc.dedup.llgoFiles[pkg.PkgPath]; ok {
				pkg.CompiledGoFiles = append(pkg.CompiledGoFiles, files...)
			}
		}
	}

	pkg.Fset = fset
	pkg.TypesSizes = tc.computedSizes(pkg)

	hasCompilerSyntaxError := false
	for _, err := range pkg.Errors {
		if strings.Contains(err.Msg, "syntax error") {
			hasCompilerSyntaxError = true
			break
		}
		if err.Kind != packages.ListError || !strings.HasPrefix(err.Msg, "# ") {
			continue
		}
		if _, diagnostics, ok := strings.Cut(err.Msg, "\n"); ok && strings.Contains(diagnostics, ": syntax error: ") {
			hasCompilerSyntaxError = true
			break
		}
	}

	appendError := func(err error) {
		var errs []packages.Error
		switch err := err.(type) {
		case packages.Error:
			errs = append(errs, err)
		case *os.PathError:
			errs = append(errs, packages.Error{
				Pos:  err.Path + ":1",
				Msg:  err.Err.Error(),
				Kind: packages.ParseError,
			})
		case scanner.ErrorList:
			if hasCompilerSyntaxError {
				return
			}
			for _, e := range err {
				errs = append(errs, packages.Error{
					Pos:  e.Pos.String(),
					Msg:  e.Msg,
					Kind: packages.ParseError,
				})
			}
		case types.Error:
			if hasCompilerSyntaxError {
				return
			}
			pkg.TypeErrors = append(pkg.TypeErrors, err)
			errs = append(errs, packages.Error{
				Pos:  err.Fset.Position(err.Pos).String(),
				Msg:  err.Msg,
				Kind: packages.TypeError,
			})
		default:
			errs = append(errs, packages.Error{
				Pos:  "-",
				Msg:  err.Error(),
				Kind: packages.UnknownError,
			})
		}
		pkg.Errors = append(pkg.Errors, errs...)
	}

	if len(pkg.Syntax) == 0 && len(pkg.CompiledGoFiles) > 0 {
		for _, file := range pkg.CompiledGoFiles {
			f, err := tc.parseFile(file, fset)
			if err != nil {
				appendError(err)
			} else {
				pkg.Syntax = append(pkg.Syntax, f)
			}
		}
	}

	pkgGoVersion := tc.targetGoVersion(pkg)
	index := new(directive.Index)
	if tc.dedup != nil && tc.dedup.directives != nil {
		index = tc.dedup.directives
	}
	index.Files(pkg.Syntax)
	normalizeEmbedDriverDiagnostics(pkg.Errors, fset, pkg.Syntax, pkgGoVersion, index)

	if tc.origMode&NeedTypes == 0 && tc.origMode&NeedTypesInfo == 0 {
		return
	}

	pkg.Types = types.NewPackage(pkg.PkgPath, pkg.Name)

	pkg.TypesInfo = &types.Info{
		Types:        make(map[ast.Expr]types.TypeAndValue),
		Defs:         make(map[*ast.Ident]types.Object),
		Uses:         make(map[*ast.Ident]types.Object),
		Implicits:    make(map[ast.Node]types.Object),
		Instances:    make(map[*ast.Ident]types.Instance),
		Scopes:       make(map[ast.Node]*types.Scope),
		Selections:   make(map[*ast.SelectorExpr]*types.Selection),
		FileVersions: make(map[*ast.File]string),
	}

	importer := importerFunc(func(path string) (*types.Package, error) {
		if path == "unsafe" {
			return types.Unsafe, nil
		}
		if pathpkg.IsAbs(path) {
			return nil, fmt.Errorf("import path cannot be absolute path")
		}

		ipkg := pkg.Imports[path]
		if ipkg == nil {
			return nil, fmt.Errorf("no metadata for %s", path)
		}

		if ipkg.Types != nil && ipkg.Types.Complete() {
			return ipkg.Types, nil
		}
		return nil, fmt.Errorf("package %q without types was imported from %q", path, pkg.ID)
	})

	if tc.dedup != nil && tc.dedup.preload != nil {
		tc.dedup.preload(pkg)
	}

	typeConf := &types.Config{
		Importer:  importer,
		Sizes:     pkg.TypesSizes,
		Error:     appendError,
		GoVersion: pkgGoVersion,
	}

	typErr := types.NewChecker(typeConf, fset, pkg.Types, pkg.TypesInfo).Files(pkg.Syntax)
	if typErr != nil && len(pkg.Errors) == 0 && len(pkg.Syntax) > 0 {
		if msg := typErr.Error(); strings.HasPrefix(msg, "package requires newer Go version") {
			appendError(types.Error{
				Fset: fset,
				Pos:  pkg.Syntax[0].Package,
				Msg:  msg,
			})
		}
	}

	if typErr != nil && len(pkg.Errors) == 0 {
		appendError(typErr)
	}

	illTyped := len(pkg.Errors) > 0
	if !illTyped {
		for _, imp := range pkg.Imports {
			if imp.IllTyped {
				illTyped = true
				break
			}
		}
	}
	pkg.IllTyped = illTyped
}

const embedPatternDriverDiagnostic = "pattern //: invalid pattern syntax"

func normalizeEmbedDriverDiagnostics(errs []packages.Error, fset *token.FileSet, files []*ast.File, goVersion string, indexes ...*directive.Index) {
	index := new(directive.Index)
	if len(indexes) > 0 {
		index = indexes[0]
	}
	for i := range errs {
		if errs[i].Msg != embedPatternDriverDiagnostic {
			continue
		}
		for _, file := range files {
			context := embedDirectiveContextAt(fset, file, errs[i].Pos, index)
			switch {
			case context == embedDirectiveLocalVar:
				errs[i].Msg = "go:embed cannot apply to var inside func"
			case context == embedDirectivePackageVar && version.IsValid(goVersion) && version.Compare(goVersion, "go1.16") < 0:
				errs[i].Msg = fmt.Sprintf("go:embed requires go1.16 or later (-lang was set to %s; check go.mod)", goVersion)
			}
			if errs[i].Msg != embedPatternDriverDiagnostic {
				break
			}
		}
	}
}

type embedDirectiveContext uint8

const (
	embedDirectiveUnknown embedDirectiveContext = iota
	embedDirectivePackageVar
	embedDirectiveLocalVar
)

func embedDirectiveContextAt(fset *token.FileSet, file *ast.File, errorPos string, indexes ...*directive.Index) embedDirectiveContext {
	index := new(directive.Index)
	if len(indexes) > 0 {
		index = indexes[0]
	}
	if fset == nil || file == nil {
		return embedDirectiveUnknown
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if !index.File(file).EmbedComments[comment] || !sameDiagnosticLine(errorPos, fset.Position(comment.Pos())) {
				continue
			}
			if localVarHasDocComment(file, comment) {
				return embedDirectiveLocalVar
			}
			for _, decl := range file.Decls {
				gen, ok := decl.(*ast.GenDecl)
				if ok && gen.Tok == token.VAR && genDeclHasDocComment(gen, comment) {
					return embedDirectivePackageVar
				}
			}
		}
	}
	return embedDirectiveUnknown
}

func isEmbedDirectiveComment(comment *ast.Comment) bool { return directive.IsEmbedComment(comment) }

func sameDiagnosticLine(errorPos string, commentPos token.Position) bool {
	if errorPos == "" || commentPos.Filename == "" || commentPos.Line == 0 {
		return false
	}
	errorFile, errorLine, ok := diagnosticFileLine(errorPos)
	if !ok || errorLine != commentPos.Line {
		return false
	}
	errorFile = filepath.Clean(errorFile)
	commentFile := filepath.Clean(commentPos.Filename)
	if errorFile == commentFile {
		return true
	}
	return !filepath.IsAbs(errorFile) &&
		(commentFile == errorFile || strings.HasSuffix(commentFile, string(filepath.Separator)+errorFile))
}

func diagnosticFileLine(pos string) (file string, line int, ok bool) {
	lastColon := strings.LastIndexByte(pos, ':')
	if lastColon < 0 {
		return "", 0, false
	}
	last, err := strconv.Atoi(pos[lastColon+1:])
	if err != nil {
		return "", 0, false
	}
	prefix := pos[:lastColon]
	if lineColon := strings.LastIndexByte(prefix, ':'); lineColon >= 0 {
		if parsedLine, err := strconv.Atoi(prefix[lineColon+1:]); err == nil {
			return prefix[:lineColon], parsedLine, true
		}
	}
	return prefix, last, true
}

func localVarHasDocComment(file *ast.File, comment *ast.Comment) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if found {
			return false
		}
		stmt, ok := node.(*ast.DeclStmt)
		if !ok {
			return true
		}
		gen, ok := stmt.Decl.(*ast.GenDecl)
		if ok && gen.Tok == token.VAR && genDeclHasDocComment(gen, comment) {
			found = true
		}
		return false
	})
	return found
}

func genDeclHasDocComment(gen *ast.GenDecl, comment *ast.Comment) bool {
	if commentGroupContains(gen.Doc, comment) {
		return true
	}
	for _, spec := range gen.Specs {
		if value, ok := spec.(*ast.ValueSpec); ok && commentGroupContains(value.Doc, comment) {
			return true
		}
	}
	return false
}

func commentGroupContains(group *ast.CommentGroup, comment *ast.Comment) bool {
	if group == nil {
		return false
	}
	for _, candidate := range group.List {
		if candidate == comment {
			return true
		}
	}
	return false
}
