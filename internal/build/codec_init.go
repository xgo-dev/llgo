package build

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"slices"
	"strconv"

	"github.com/xgo-dev/llgo/internal/build/jsonspecialize"
	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func configureCodecInits(ctx *context, inputs *linkPruningInputs) {
	if len(ctx.initial) != 1 || inputs == nil || inputs.opaqueAsmPkg != "" ||
		(ctx.buildConf.Goarch == "wasm" && !ctx.wasmClosedProgram) {
		return
	}
	entry := ctx.initial[0]
	if entry.TypesInfo == nil {
		return
	}
	ssaEntry := ctx.progSSA.Package(entry.Types)
	if ssaEntry == nil || ssaEntry.Func("init") == nil {
		return
	}

	if !ctx.jsonCodecPackages[entry.Types] {
		return
	}
	skip := findUnusedJSONInits(ctx, inputs, entry)
	if len(skip) == 0 {
		return
	}
	// LLGo has both ordered entry-module initialization and guarded
	// dependency calls in main.init. Update only those two caller roots.
	for _, block := range ssaEntry.Func("init").Blocks {
		block.Instrs = slices.DeleteFunc(block.Instrs, func(instr ssa.Instruction) bool {
			if call, ok := instr.(*ssa.Call); ok {
				fn := call.Common().StaticCallee()
				if fn != nil && fn.Package() != nil && fn.Synthetic == "package initializer" {
					return skip[llssa.FullName(fn.Package().Pkg, "init")]
				}
			}
			return false
		})
	}
	ctx.codecInitDrops = skip
	if ctx.buildConf.Verbose || ctx.buildConf.PrintCommands {
		fmt.Fprintln(os.Stderr, "llgo: omit unused JSON package initialization")
	}
}

// Remove a codec's private initialization only when the specialized entry
// has no remaining use of that state. Preserve all other packages' order.
func findUnusedJSONInits(ctx *context, inputs *linkPruningInputs, entry *packages.Package) map[string]bool {
	typedDecode, hasOtherUse := entryJSONUses(entry)
	if slices.ContainsFunc(inputs.roots, jsonspecialize.ImplementationSymbol) || hasOtherUse ||
		(typedDecode && !ctx.embeddedClosedProgram) {
		return nil
	}
	trusted := make(map[*types.Package]bool)
	var members []*aPackage
	for _, pkg := range ctx.pkgByID {
		if pkg.Types == nil {
			return nil
		}
		trusted[pkg.Types] = isStandardLibraryPackage(pkg.Package, inputs.goroot)
		if jsonspecialize.IsImplementation(pkg.PkgPath) {
			if pkg.AltPkg != nil {
				return nil
			}
			if !typedDecode || genericJSONInit(pkg.PkgPath) {
				members = append(members, pkg)
			} else {
				for _, dep := range effectiveDependencies(pkg) {
					if genericJSONInit(dep.PkgPath) {
						return nil
					}
				}
			}
			continue
		}
		for _, dep := range effectiveDependencies(pkg) {
			if jsonspecialize.IsImplementation(dep.PkgPath) && (pkg.Package != entry ||
				dep.PkgPath != "encoding/json" && !(typedDecode && (dep.PkgPath == "encoding/json/v2" || dep.PkgPath == "encoding/json/jsontext"))) {
				return nil
			}
		}
		if hasJSONLinkname(pkg.Syntax) || pkg.AltPkg != nil && hasJSONLinkname(pkg.AltPkg.Syntax) {
			return nil
		}
	}

	skip := make(map[string]bool, len(members))
	for _, pkg := range members {
		ssaPkg := ctx.progSSA.Package(pkg.Types)
		if typedDecode && pkg.PkgPath == "encoding/json/v2" && !jsonspecialize.StatelessTokenizerHandle(pkg.Types) {
			return nil
		}
		if ssaPkg == nil || !jsonspecialize.DiscardableInit(ssaPkg.Func("init"),
			func(p *types.Package) bool {
				return trusted[p] || p.Path() == "github.com/xgo-dev/llgo/runtime/internal/lib/reflect"
			}) {
			return nil
		}
		skip[llssa.FullName(pkg.Types, "init")] = true
	}
	return skip
}

func genericJSONInit(path string) bool {
	return path == "encoding/json" || path == "encoding/json/v2"
}

func entryJSONUses(entry *packages.Package) (typedDecode, hasOtherUse bool) {
	var typedSupport bool
	for _, file := range entry.Syntax {
		for _, imp := range file.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if jsonspecialize.IsImplementation(path) && imp.Name != nil && imp.Name.Name == "_" {
				return typedDecode, true
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			id, ok := node.(*ast.Ident)
			if !ok {
				return true
			}
			obj := entry.TypesInfo.Uses[id]
			if obj == nil || obj.Pkg() == nil || !jsonspecialize.IsImplementation(obj.Pkg().Path()) {
				return true
			}
			_, isFunction := obj.(*types.Func)
			switch obj.Pkg().Path() {
			case "encoding/json":
				if isFunction && obj.Name() == "LlgoDecodeTyped" {
					typedDecode = true
					return true
				}
				if jsonspecialize.TypedDecodeErrorObject(obj) {
					typedSupport = true
					return true
				}
			case "encoding/json/v2":
				if isFunction {
					switch obj.Name() {
					case "LlgoReadBool", "LlgoReadString", "LlgoReadInt", "LlgoReadUint", "LlgoBadToken", "LlgoRecoverable":
						typedSupport = true
						return true
					}
				}
			case "encoding/json/jsontext":
				// The tokenizer and its dependency initializers are kept.
				typedSupport = true
				return true
			}
			hasOtherUse = true
			return false
		})
	}
	return typedDecode, hasOtherUse || typedSupport && !typedDecode
}

func hasJSONLinkname(files []*ast.File) bool {
	for _, file := range files {
		for _, group := range file.Comments {
			for _, item := range directive.ParseGroup(group) {
				if item.Name != "go:linkname" && item.Name != "llgo:link" {
					continue
				}
				_, target := directive.ParseLinknameArgs(item.Args)
				if jsonspecialize.ImplementationSymbol(target) {
					return true
				}
			}
		}
	}
	return false
}
