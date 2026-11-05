package cl

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
	"sync"

	"github.com/xgo-dev/llgo/internal/directive"
	"github.com/xgo-dev/llgo/internal/locality"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

// These are frontend import costs, not LLVM instruction counts. LLVM makes
// the final call-site inlining decision after normal lowering. Like Go's
// inliner, charge calls whose bodies cannot be imported more than simple ops.
const (
	inlineBodyBudget   = 80
	inlineCallCost     = 57
	inlineModuleBudget = 1024
	inlineImportDepth  = 5
)

type inlineCost struct {
	cost      int
	recursive bool
	globals   []*ssa.Global
}

// Shared for one compilation, including dependencies whose backend archive is
// cached. Both accepted and rejected bodies are analyzed only once.
type inlineFacts struct {
	mu    sync.RWMutex
	costs map[*ssa.Function]inlineCost
}

func newInlineFacts() *inlineFacts {
	return &inlineFacts{costs: make(map[*ssa.Function]inlineCost)}
}

func (f *inlineFacts) cost(fn *ssa.Function) int {
	return f.info(fn).cost
}

func (f *inlineFacts) info(fn *ssa.Function) inlineCost {
	f.mu.RLock()
	c, ok := f.costs[fn]
	f.mu.RUnlock()
	if ok {
		return c
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.analyze(fn, make(map[*ssa.Function]bool))
}

func (f *inlineFacts) analyze(fn *ssa.Function, active map[*ssa.Function]bool) inlineCost {
	reject := inlineCost{cost: inlineBodyBudget + 1}
	if fn == nil {
		return reject
	}
	if c, ok := f.costs[fn]; ok {
		return c
	}
	if active[fn] {
		return inlineCost{cost: reject.cost, recursive: true}
	}
	if len(active) >= 32 {
		// Reject the unresolved chain as a whole, including a possible cycle
		// beyond the analysis limit, rather than treating it as a cheap call.
		return inlineCost{cost: reject.cost, recursive: true}
	}
	result := reject
	defer func() { f.costs[fn] = result }()
	if fn.Pkg == nil || fn.Parent() != nil || fn.Synthetic != "" ||
		len(fn.Blocks) == 0 || len(fn.FreeVars) != 0 || len(fn.AnonFuncs) != 0 || fn.Recover != nil ||
		fn.Signature.Recv() != nil || fn.Signature.Variadic() ||
		fn.Signature.TypeParams().Len() != 0 || fn.Origin() != nil ||
		fn.Name() == "init" || strings.HasPrefix(fn.Name(), "init#") ||
		inlineProhibitedDirective(fn) {
		return reject
	}
	active[fn] = true
	defer delete(active, fn)
	cost := 0
	var globals []*ssa.Global
	seenGlobals := make(map[*ssa.Global]bool)
	for _, block := range fn.Blocks {
		for _, succ := range block.Succs {
			if succ.Dominates(block) {
				return reject // Keep loops out of the initial import policy.
			}
		}
		for _, instr := range block.Instrs {
			switch v := instr.(type) {
			case *ssa.DebugRef:
				continue
			case *ssa.Go, *ssa.Defer, *ssa.RunDefers, *ssa.MakeClosure, *ssa.Panic:
				return reject
			case *ssa.Call:
				if builtin, ok := v.Call.Value.(*ssa.Builtin); ok {
					if builtin.Name() == "recover" {
						return reject
					}
				} else {
					callee := f.analyze(v.Call.StaticCallee(), active)
					if callee.recursive {
						result = callee
						return result
					}
					if callee.cost <= inlineBodyBudget {
						cost += callee.cost
					} else {
						cost += inlineCallCost
					}
				}
			}
			for _, operand := range instr.Operands(nil) {
				if operand != nil {
					if global, ok := (*operand).(*ssa.Global); ok && !seenGlobals[global] {
						seenGlobals[global] = true
						globals = append(globals, global)
					}
				}
			}
			cost++
			if cost > inlineBodyBudget {
				return reject
			}
		}
	}
	result = inlineCost{cost: cost, globals: globals}
	return result
}

func inlineProhibitedDirective(fn *ssa.Function) bool {
	decl, _ := fn.Syntax().(*ast.FuncDecl)
	if decl != nil {
		for _, d := range directive.ParseGroup(decl.Doc) {
			switch d.Name {
			case "go:noinline", "go:cgo_unsafe_args", "go:uintptrkeepalive", "go:uintptrescapes",
				"go:yeswritebarrierrec", "go:norace", "go:nocheckptr",
				"llgo:env", "llgo:link", "go:linkname":
				return true
			}
		}
	}
	return false
}

// importInlineBody supplies an optimization-only definition for a direct
// cross-package call. The defining package remains the owner of the real
// function entry, including when its address escapes or LLVM declines to inline.
func (p *context) importInlineBody(original *ssa.Function) {
	if disableInline || p.inlineFacts == nil || p.options.DebuggerFrames || p.options.ShadowStack ||
		p.inlineCost >= inlineModuleBudget || original.Pkg == nil || original.Pkg == p.goPkg ||
		llssa.PathOf(original.Pkg.Pkg) == p.pkg.Path() {
		return
	}
	if p.inlineChecked[original] {
		return
	}
	if p.inlineChecked == nil {
		p.inlineChecked = make(map[*ssa.Function]bool)
	}
	p.inlineChecked[original] = true
	depth := p.inlineImports[p.goFn] + 1
	if depth > inlineImportDepth {
		return
	}
	fn := original
	if patch, ok := p.patches[llssa.PathOf(fn.Pkg.Pkg)]; ok && fn.Signature.Recv() == nil {
		if replacement := patch.Alt.Func(fn.Name()); replacement != nil {
			fn = replacement
		}
	}
	if fn != original && inlineProhibitedDirective(original) {
		return
	}
	facts := p.inlineFacts.info(fn)
	cost := facts.cost
	if cost > inlineBodyBudget || p.inlineCost+cost > inlineModuleBudget ||
		p.callerTracking != nil && runtimeCallerFuncSet(p.callerTracking, fn.Pkg)[fn] ||
		needsRuntimeStackNoInline(fn.Pkg.Pkg, fn) || p.functionUsesRecover(fn) {
		return
	}
	// Locality storage is owned by its defining package and deliberately hidden
	// behind functions. Do not import bodies that expose that internal ABI.
	for _, global := range facts.globals {
		if info, ok := p.resolveLocality(global.Pkg.Pkg, llssa.FullName(global.Pkg.Pkg, global.Name())); ok && info.Locality != locality.None {
			return
		}
	}
	_, name, kind := p.funcName(original)
	_, replacementName, replacementKind := p.funcName(fn)
	if kind != goFunc || replacementKind != goFunc || name != replacementName ||
		!types.Identical(p.patchType(original.Signature), p.patchType(fn.Signature)) {
		return
	}
	if existing := p.pkg.Module().NamedFunction(name); !existing.IsNil() && !existing.IsDeclaration() {
		return // Never turn an existing real definition into an imported copy.
	}
	if p.inlineImports == nil {
		p.inlineImports = make(map[*ssa.Function]int)
	}
	if p.inlineImports[fn] != 0 {
		return
	}
	p.inlineImports[fn] = depth
	p.inlineCost += cost
	// Queue declaration/body construction between function lowerings: lowering
	// an imported body recursively would overwrite the current builder's state.
	p.inits = append(p.inits, func() {
		if existing := p.pkg.Module().NamedFunction(name); !existing.IsNil() && !existing.IsDeclaration() {
			return
		}
		if decl, ok := fn.Syntax().(*ast.FuncDecl); ok {
			fields := collectAddrOfFieldSelectors([]*ast.File{{Decls: []ast.Decl{decl}}})
			if len(fields) != 0 && p.addrOfFieldAddrs == nil {
				p.addrOfFieldAddrs = make(map[token.Pos]none)
			}
			for pos := range fields {
				p.addrOfFieldAddrs[pos] = none{}
			}
		}
		state := p.state
		p.state = pkgNormal
		entry, _, _ := p.compileFuncDecl(p.pkg, fn)
		p.state = state
		p.pkg.Module().NamedFunction(entry.Name()).SetLinkage(llvm.AvailableExternallyLinkage)
	})
}
