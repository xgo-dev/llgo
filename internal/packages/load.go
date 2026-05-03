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
	"go/scanner"
	"go/types"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"unsafe"

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

	typecheckCgo = NeedModule - 1 // TODO(xsw): how to check
)

const (
	DebugPackagesLoad = false
)

var runtimeGoMinor = parseRuntimeGoMinor(runtime.Version())

func parseRuntimeGoMinor(version string) int {
	const prefix = "go1."
	if !strings.HasPrefix(version, prefix) {
		return 0
	}
	minor := 0
	for i := len(prefix); i < len(version); i++ {
		c := version[i]
		if c < '0' || c > '9' {
			break
		}
		minor = minor*10 + int(c-'0')
	}
	return minor
}

// A Config specifies details about how packages should be loaded.
// The zero value is a valid configuration.
// Calls to Load do not modify this struct.
type Config = packages.Config

// A Package describes a loaded Go package.
type Package = packages.Package

// loaderPackage augments Package with state used during the loading phase
type loaderPackage struct {
	*Package
	importErrors map[string]error // maps each bad import to its error
	loadOnce     sync.Once
	color        uint8 // for cycle detection
	needsrc      bool  // load from source (Mode >= LoadTypes)
	needtypes    bool  // type information is either requested or depended on
	initial      bool  // package was matched by a pattern
	goVersion    int   // minor version number of go command on PATH
}

// loader holds the working state of a single call to load.
type loader struct {
	pkgs map[string]*loaderPackage
	Config
	sizes        types.Sizes // TODO(xsw): ensure offset of sizes
	parseCache   map[string]unsafe.Pointer
	parseCacheMu sync.Mutex
	exportMu     sync.Mutex // enforces mutual exclusion of exportdata operations

	// Config.Mode contains the implied mode (see impliedLoadMode).
	// Implied mode contains all the fields we need the data for.
	// In requestedMode there are the actually requested fields.
	// We'll zero them out before returning packages to the user.
	// This makes it easier for us to get the conditions where
	// we need certain modes right.
	requestedMode LoadMode
}

type Cached struct {
	*packages.Package
	Types     *types.Package
	TypesInfo *types.Info
	Syntax    []*ast.File
}

type aDeduper struct {
	cache   sync.Map
	setpath func(path string, name string) string
	preload func(pkg *types.Package, syntax []*ast.File)
}

type Deduper = *aDeduper

func NewDeduper() Deduper {
	return &aDeduper{}
}

func (p Deduper) SetPreload(fn func(pkg *types.Package, syntax []*ast.File)) {
	p.preload = fn
}

func (p Deduper) SetPkgPath(fn func(path, name string) string) {
	p.setpath = fn
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

//go:linkname defaultDriver golang.org/x/tools/go/packages.defaultDriver
func defaultDriver(cfg *Config, patterns ...string) (*packages.DriverResponse, bool, error)

//go:linkname newLoader golang.org/x/tools/go/packages.newLoader
func newLoader(cfg *Config) *loader

//go:linkname loadFromExportData golang.org/x/tools/go/packages.(*loader).loadFromExportData
func loadFromExportData(ld *loader, lpkg *loaderPackage) error

//go:linkname parseFiles golang.org/x/tools/go/packages.(*loader).parseFiles
func parseFiles(ld *loader, filenames []string) ([]*ast.File, []error)

//go:linkname typesinternalSetUsesCgo golang.org/x/tools/internal/typesinternal.SetUsesCgo
func typesinternalSetUsesCgo(conf *types.Config) bool

// An importFunc is an implementation of the single-method
// types.Importer interface based on a function value.
type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

func loadPackageEx(dedup Deduper, ld *loader, lpkg *loaderPackage) {
	if lpkg.PkgPath == "unsafe" {
		// Fill in the blanks to avoid surprises.
		lpkg.Types = types.Unsafe
		lpkg.Fset = ld.Fset
		lpkg.Syntax = []*ast.File{}
		lpkg.TypesInfo = new(types.Info)
		lpkg.TypesSizes = ld.sizes
		return
	}

	if dedup != nil {
		if cp := dedup.Check(lpkg.ID); cp != nil {
			lpkg.Types = cp.Types
			lpkg.Fset = ld.Fset
			lpkg.TypesInfo = cp.TypesInfo
			lpkg.Syntax = cp.Syntax
			lpkg.TypesSizes = ld.sizes
			return
		}
		defer func() {
			if !lpkg.IllTyped && lpkg.needtypes && lpkg.needsrc {
				dedup.set(lpkg.PkgPath, &Cached{
					Package:   lpkg.Package,
					Types:     lpkg.Types,
					TypesInfo: lpkg.TypesInfo,
					Syntax:    lpkg.Syntax,
				})
			}
		}()
		if dedup.setpath != nil {
			lpkg.PkgPath = dedup.setpath(lpkg.PkgPath, lpkg.Name)
		}
	}

	// Call NewPackage directly with explicit name.
	// This avoids skew between golist and go/types when the files'
	// package declarations are inconsistent.
	lpkg.Types = types.NewPackage(lpkg.PkgPath, lpkg.Name)
	lpkg.Fset = ld.Fset

	// Start shutting down if the context is done and do not load
	// source or export data files.
	// Packages that import this one will have ld.Context.Err() != nil.
	// ld.Context.Err() will be returned later by refine.
	if ld.Context.Err() != nil {
		return
	}

	// Subtle: we populate all Types fields with an empty Package
	// before loading export data so that export data processing
	// never has to create a types.Package for an indirect dependency,
	// which would then require that such created packages be explicitly
	// inserted back into the Import graph as a final step after export data loading.
	// (Hence this return is after the Types assignment.)
	// The Diamond test exercises this case.
	if !lpkg.needtypes && !lpkg.needsrc {
		return
	}
	if !lpkg.needsrc {
		if err := loadFromExportData(ld, lpkg); err != nil {
			lpkg.Errors = append(lpkg.Errors, packages.Error{
				Pos:  "-",
				Msg:  err.Error(),
				Kind: packages.UnknownError, // e.g. can't find/open/parse export data
			})
		}
		return // not a source package, don't get syntax trees
	}

	appendError := func(err error) {
		// Convert various error types into the one true Error.
		var errs []packages.Error
		switch err := err.(type) {
		case packages.Error:
			// from driver
			errs = append(errs, err)

		case *os.PathError:
			// from parser
			errs = append(errs, packages.Error{
				Pos:  err.Path + ":1",
				Msg:  err.Err.Error(),
				Kind: packages.ParseError,
			})

		case scanner.ErrorList:
			// from parser
			for _, err := range err {
				errs = append(errs, packages.Error{
					Pos:  err.Pos.String(),
					Msg:  err.Msg,
					Kind: packages.ParseError,
				})
			}

		case types.Error:
			// from type checker
			lpkg.TypeErrors = append(lpkg.TypeErrors, err)
			errs = append(errs, packages.Error{
				Pos:  err.Fset.Position(err.Pos).String(),
				Msg:  err.Msg,
				Kind: packages.TypeError,
			})

		default:
			// unexpected impoverished error from parser?
			errs = append(errs, packages.Error{
				Pos:  "-",
				Msg:  err.Error(),
				Kind: packages.UnknownError,
			})

			// If you see this error message, please file a bug.
			log.Printf("internal error: error %q (%T) without position", err, err)
		}

		lpkg.Errors = append(lpkg.Errors, errs...)
	}

	// If the go command on the PATH is newer than the runtime,
	// then the go/{scanner,ast,parser,types} packages from the
	// standard library may be unable to process the files
	// selected by go list.
	//
	// There is currently no way to downgrade the effective
	// version of the go command (see issue 52078), so we proceed
	// with the newer go command but, in case of parse or type
	// errors, we emit an additional diagnostic.
	//
	// See:
	// - golang.org/issue/52078 (flag to set release tags)
	// - golang.org/issue/50825 (gopls legacy version support)
	// - golang.org/issue/55883 (go/packages confusing error)
	//
	// Should we assert a hard minimum of (currently) go1.16 here?
	if runtimeVersion := runtimeGoMinor; runtimeVersion != 0 && runtimeVersion < lpkg.goVersion {
		defer func() {
			if len(lpkg.Errors) > 0 {
				appendError(packages.Error{
					Pos:  "-",
					Msg:  fmt.Sprintf("This application uses version go1.%d of the source-processing packages but runs version go1.%d of 'go list'. It may fail to process source files that rely on newer language features. If so, rebuild the application using a newer version of Go.", runtimeVersion, lpkg.goVersion),
					Kind: packages.UnknownError,
				})
			}
		}()
	}

	if ld.Config.Mode&NeedTypes != 0 && len(lpkg.CompiledGoFiles) == 0 && lpkg.ExportFile != "" {
		// The config requested loading sources and types, but sources are missing.
		// Add an error to the package and fall back to loading from export data.
		appendError(packages.Error{
			Pos:  "-",
			Msg:  fmt.Sprintf("sources missing for package %s", lpkg.ID),
			Kind: packages.ParseError,
		})
		_ = loadFromExportData(ld, lpkg) // ignore any secondary errors

		return // can't get syntax trees for this package
	}

	files, errs := parseFiles(ld, lpkg.CompiledGoFiles)
	for _, err := range errs {
		appendError(err)
	}

	lpkg.Syntax = files
	if ld.Config.Mode&NeedTypes == 0 {
		return
	}

	// Start shutting down if the context is done and do not type check.
	// Packages that import this one will have ld.Context.Err() != nil.
	// ld.Context.Err() will be returned later by refine.
	if ld.Context.Err() != nil {
		return
	}

	typeInfoCap := len(lpkg.Syntax) * 1024
	if typeInfoCap < 16 {
		typeInfoCap = 16
	}
	identCap := typeInfoCap / 2
	if identCap < 16 {
		identCap = 16
	}
	smallCap := len(lpkg.Syntax) * 8
	if smallCap < 4 {
		smallCap = 4
	}
	lpkg.TypesInfo = &types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue, typeInfoCap),
		Defs:      make(map[*ast.Ident]types.Object, identCap),
		Uses:      make(map[*ast.Ident]types.Object, identCap),
		Implicits: make(map[ast.Node]types.Object, smallCap),
		Instances: make(map[*ast.Ident]types.Instance, smallCap),
		// Scopes are not consumed by LLGo or x/tools/go/ssa during compilation.
		// Leaving it nil avoids recording every lexical scope during type checking.
		Selections: make(map[*ast.SelectorExpr]*types.Selection, smallCap),
	}
	lpkg.TypesSizes = ld.sizes

	importer := importerFunc(func(path string) (*types.Package, error) {
		if path == "unsafe" {
			return types.Unsafe, nil
		}

		// The imports map is keyed by import path.
		ipkg := lpkg.Imports[path]
		if ipkg == nil {
			if err := lpkg.importErrors[path]; err != nil {
				return nil, err
			}
			// There was skew between the metadata and the
			// import declarations, likely due to an edit
			// race, or because the ParseFile feature was
			// used to supply alternative file contents.
			return nil, fmt.Errorf("no metadata for %s", path)
		}

		if ipkg.Types != nil && ipkg.Types.Complete() {
			return ipkg.Types, nil
		}
		log.Fatalf("internal error: package %q without types was imported from %q", path, lpkg)
		panic("unreachable")
	})

	if dedup != nil && dedup.preload != nil {
		dedup.preload(lpkg.Types, lpkg.Syntax)
	}

	// type-check
	tc := &types.Config{
		Importer: importer,

		// Type-check bodies of functions only in initial packages.
		// Example: for import graph A->B->C and initial packages {A,C},
		// we can ignore function bodies in B.
		IgnoreFuncBodies: ld.Mode&NeedDeps == 0 && !lpkg.initial,

		Error: appendError,
		Sizes: ld.sizes, // may be nil
	}
	if lpkg.Module != nil && lpkg.Module.GoVersion != "" {
		tc.GoVersion = "go" + lpkg.Module.GoVersion
	}
	if (ld.Mode & typecheckCgo) != 0 {
		if !typesinternalSetUsesCgo(tc) {
			appendError(packages.Error{
				Msg:  "typecheckCgo requires Go 1.15+",
				Kind: packages.ListError,
			})
			return
		}
	}

	typErr := types.NewChecker(tc, ld.Fset, lpkg.Types, lpkg.TypesInfo).Files(lpkg.Syntax)
	lpkg.importErrors = nil // no longer needed

	// In go/types go1.21 and go1.22, Checker.Files failed fast with a
	// a "too new" error, without calling tc.Error and without
	// proceeding to type-check the package (#66525).
	// We rely on the runtimeVersion error to give the suggested remedy.
	if typErr != nil && len(lpkg.Errors) == 0 && len(lpkg.Syntax) > 0 {
		if msg := typErr.Error(); strings.HasPrefix(msg, "package requires newer Go version") {
			appendError(types.Error{
				Fset: ld.Fset,
				Pos:  lpkg.Syntax[0].Package,
				Msg:  msg,
			})
		}
	}

	// If !Cgo, the type-checker uses FakeImportC mode, so
	// it doesn't invoke the importer for import "C",
	// nor report an error for the import,
	// or for any undefined C.f reference.
	// We must detect this explicitly and correctly
	// mark the package as IllTyped (by reporting an error).
	// TODO(adonovan): if these errors are annoying,
	// we could just set IllTyped quietly.
	if tc.FakeImportC {
	outer:
		for _, f := range lpkg.Syntax {
			for _, imp := range f.Imports {
				if imp.Path.Value == `"C"` {
					err := types.Error{Fset: ld.Fset, Pos: imp.Pos(), Msg: `import "C" ignored`}
					appendError(err)
					break outer
				}
			}
		}
	}

	// If types.Checker.Files had an error that was unreported,
	// make sure to report the unknown error so the package is illTyped.
	if typErr != nil && len(lpkg.Errors) == 0 {
		appendError(typErr)
	}

	// Record accumulated errors.
	illTyped := len(lpkg.Errors) > 0
	if !illTyped {
		for _, imp := range lpkg.Imports {
			if imp.IllTyped {
				illTyped = true
				break
			}
		}
	}
	lpkg.IllTyped = illTyped
}

const packageLoadParallelFanout = 4

func loadRecursiveEx(dedup Deduper, ld *loader, lpkg *loaderPackage) {
	lpkg.loadOnce.Do(func() {
		// Most packages have a narrow import fanout; loading those branches
		// sequentially avoids a large number of short-lived goroutines while
		// still using parallelism for wider parts of the graph.
		if len(lpkg.Imports) <= packageLoadParallelFanout {
			for _, ipkg := range lpkg.Imports {
				loadRecursiveEx(dedup, ld, ld.pkgs[ipkg.ID])
			}
		} else {
			var wg sync.WaitGroup
			for _, ipkg := range lpkg.Imports {
				imp := ld.pkgs[ipkg.ID]
				wg.Add(1)
				go func(imp *loaderPackage) {
					loadRecursiveEx(dedup, ld, imp)
					wg.Done()
				}(imp)
			}
			wg.Wait()
		}
		loadPackageEx(dedup, ld, lpkg)
	})
}

func refineEx(dedup Deduper, ld *loader, response *packages.DriverResponse) ([]*Package, error) {
	roots := response.Roots
	rootMap := make(map[string]int, len(roots))
	for i, root := range roots {
		rootMap[root] = i
	}
	ld.pkgs = make(map[string]*loaderPackage)
	// first pass, fixup and build the map and roots
	var initial = make([]*loaderPackage, len(roots))
	for _, pkg := range response.Packages {
		rootIndex := -1
		if i, found := rootMap[pkg.ID]; found {
			rootIndex = i
		}

		// Overlays can invalidate export data.
		// TODO(matloob): make this check fine-grained based on dependencies on overlaid files
		exportDataInvalid := len(ld.Overlay) > 0 || pkg.ExportFile == "" && pkg.PkgPath != "unsafe"
		// This package needs type information if the caller requested types and the package is
		// either a root, or it's a non-root and the user requested dependencies ...
		needtypes := (ld.Mode&NeedTypes|NeedTypesInfo != 0 && (rootIndex >= 0 || ld.Mode&NeedDeps != 0))
		// This package needs source if the call requested source (or types info, which implies source)
		// and the package is either a root, or itas a non- root and the user requested dependencies...
		needsrc := ((ld.Mode&(NeedSyntax|NeedTypesInfo) != 0 && (rootIndex >= 0 || ld.Mode&NeedDeps != 0)) ||
			// ... or if we need types and the exportData is invalid. We fall back to (incompletely)
			// typechecking packages from source if they fail to compile.
			(ld.Mode&(NeedTypes|NeedTypesInfo) != 0 && exportDataInvalid)) && pkg.PkgPath != "unsafe"
		lpkg := &loaderPackage{
			Package:   pkg,
			needtypes: needtypes,
			needsrc:   needsrc,
			goVersion: response.GoVersion,
		}
		ld.pkgs[lpkg.ID] = lpkg
		if rootIndex >= 0 {
			initial[rootIndex] = lpkg
			lpkg.initial = true
		}
	}
	for i, root := range roots {
		if initial[i] == nil {
			return nil, fmt.Errorf("root package %v is missing", root)
		}
	}

	if ld.Mode&NeedImports != 0 {
		// Materialize the import graph.

		const (
			white = 0 // new
			grey  = 1 // in progress
			black = 2 // complete
		)

		// visit traverses the import graph, depth-first,
		// and materializes the graph as Packages.Imports.
		//
		// Valid imports are saved in the Packages.Import map.
		// Invalid imports (cycles and missing nodes) are saved in the importErrors map.
		// Thus, even in the presence of both kinds of errors,
		// the Import graph remains a DAG.
		//
		// visit returns whether the package needs src or has a transitive
		// dependency on a package that does. These are the only packages
		// for which we load source code.
		var stack []*loaderPackage
		var visit func(lpkg *loaderPackage) bool
		visit = func(lpkg *loaderPackage) bool {
			switch lpkg.color {
			case black:
				return lpkg.needsrc
			case grey:
				panic("internal error: grey node")
			}
			lpkg.color = grey
			stack = append(stack, lpkg) // push
			stubs := lpkg.Imports       // the structure form has only stubs with the ID in the Imports
			lpkg.Imports = make(map[string]*Package, len(stubs))
			for importPath, ipkg := range stubs {
				var importErr error
				imp := ld.pkgs[ipkg.ID]
				if imp == nil {
					// (includes package "C" when DisableCgo)
					importErr = fmt.Errorf("missing package: %q", ipkg.ID)
				} else if imp.color == grey {
					importErr = fmt.Errorf("import cycle: %s", stack)
				}
				if importErr != nil {
					if lpkg.importErrors == nil {
						lpkg.importErrors = make(map[string]error)
					}
					lpkg.importErrors[importPath] = importErr
					continue
				}

				if visit(imp) {
					lpkg.needsrc = true
				}
				lpkg.Imports[importPath] = imp.Package
			}

			// Complete type information is required for the
			// immediate dependencies of each source package.
			if lpkg.needsrc && ld.Mode&NeedTypes != 0 {
				for _, ipkg := range lpkg.Imports {
					ld.pkgs[ipkg.ID].needtypes = true
				}
			}

			// NeedTypeSizes causes TypeSizes to be set even
			// on packages for which types aren't needed.
			if ld.Mode&NeedTypesSizes != 0 {
				lpkg.TypesSizes = ld.sizes
			}
			stack = stack[:len(stack)-1] // pop
			lpkg.color = black

			return lpkg.needsrc
		}

		// For each initial package, create its import DAG.
		for _, lpkg := range initial {
			visit(lpkg)
		}

	} else {
		// !NeedImports: drop the stub (ID-only) import packages
		// that we are not even going to try to resolve.
		for _, lpkg := range initial {
			lpkg.Imports = nil
		}
	}

	// Load type data and syntax if needed, starting at
	// the initial packages (roots of the import DAG).
	if ld.Mode&NeedTypes != 0 || ld.Mode&NeedSyntax != 0 {
		if len(initial) == 1 {
			loadRecursiveEx(dedup, ld, initial[0])
		} else {
			var wg sync.WaitGroup
			for _, lpkg := range initial {
				wg.Add(1)
				go func(lpkg *loaderPackage) {
					loadRecursiveEx(dedup, ld, lpkg)
					wg.Done()
				}(lpkg)
			}
			wg.Wait()
		}
	}

	// If the context is done, return its error and
	// throw out [likely] incomplete packages.
	if err := ld.Context.Err(); err != nil {
		return nil, err
	}

	result := make([]*Package, len(initial))
	for i, lpkg := range initial {
		result[i] = lpkg.Package
	}
	for i := range ld.pkgs {
		// Clear all unrequested fields,
		// to catch programs that use more than they request.
		if ld.requestedMode&NeedName == 0 {
			ld.pkgs[i].Name = ""
			ld.pkgs[i].PkgPath = ""
		}
		if ld.requestedMode&NeedFiles == 0 {
			ld.pkgs[i].GoFiles = nil
			ld.pkgs[i].OtherFiles = nil
			ld.pkgs[i].IgnoredFiles = nil
		}
		if ld.requestedMode&NeedEmbedFiles == 0 {
			ld.pkgs[i].EmbedFiles = nil
		}
		if ld.requestedMode&NeedEmbedPatterns == 0 {
			ld.pkgs[i].EmbedPatterns = nil
		}
		if ld.requestedMode&NeedCompiledGoFiles == 0 {
			ld.pkgs[i].CompiledGoFiles = nil
		}
		if ld.requestedMode&NeedImports == 0 {
			ld.pkgs[i].Imports = nil
		}
		if ld.requestedMode&NeedExportFile == 0 {
			ld.pkgs[i].ExportFile = ""
		}
		if ld.requestedMode&NeedTypes == 0 {
			ld.pkgs[i].Types = nil
			ld.pkgs[i].Fset = nil
			ld.pkgs[i].IllTyped = false
		}
		if ld.requestedMode&NeedSyntax == 0 {
			ld.pkgs[i].Syntax = nil
		}
		if ld.requestedMode&NeedTypesInfo == 0 {
			ld.pkgs[i].TypesInfo = nil
		}
		if ld.requestedMode&NeedTypesSizes == 0 {
			ld.pkgs[i].TypesSizes = nil
		}
		if ld.requestedMode&NeedModule == 0 {
			ld.pkgs[i].Module = nil
		}
	}

	return result, nil
}

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
	ld := newLoader(cfg)
	response, external, err := defaultDriver(&ld.Config, patterns...)
	if err != nil {
		return nil, err
	}

	ld.sizes = types.SizesFor(response.Compiler, response.Arch)
	if ld.sizes == nil && ld.Config.Mode&(NeedTypes|NeedTypesSizes|NeedTypesInfo) != 0 {
		// Type size information is needed but unavailable.
		if external {
			// An external driver may fail to populate the Compiler/GOARCH fields,
			// especially since they are relatively new (see #63700).
			// Provide a sensible fallback in this case.
			ld.sizes = types.SizesFor("gc", runtime.GOARCH)
			if ld.sizes == nil { // gccgo-only arch
				ld.sizes = types.SizesFor("gc", "amd64")
			}
		} else {
			// Go list should never fail to deliver accurate size information.
			// Reject the whole Load since the error is the same for every package.
			return nil, fmt.Errorf("can't determine type sizes for compiler %q on GOARCH %q",
				response.Compiler, response.Arch)
		}
	}

	if sizes != nil {
		ld.sizes = sizes(ld.sizes, response.Compiler, response.Arch)
	}
	return refineEx(dedup, ld, response)
}

// Visit visits all the packages in the import graph whose roots are
// pkgs, calling the optional pre function the first time each package
// is encountered (preorder), and the optional post function after a
// package's dependencies have been visited (postorder).
// The boolean result of pre(pkg) determines whether
// the imports of package pkg are visited.
//
//go:linkname Visit golang.org/x/tools/go/packages.Visit
func Visit(pkgs []*Package, pre func(*Package) bool, post func(*Package))
