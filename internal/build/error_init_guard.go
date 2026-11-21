package build

import (
	"go/constant"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/ssa"
)

type literalConstructors struct {
	newError *ssa.Function
	sync     *ssa.Package
}

func standardInitializerPackage(ctx *context, goroot, path string) *ssa.Package {
	pkg := ctx.pkgByID[path]
	if pkg == nil || pkg.AltPkg != nil || !isStandardLibraryPackage(pkg.Package, goroot) {
		return nil
	}
	return ctx.progSSA.Package(pkg.Types)
}

func findLiteralConstructors(ctx *context, goroot string) (constructors literalConstructors) {
	if pkg := standardInitializerPackage(ctx, goroot, "errors"); pkg != nil {
		if fn := pkg.Func("New"); isLiteralErrorConstructor(fn) {
			constructors.newError = fn
		}
	}
	constructors.sync = standardInitializerPackage(ctx, goroot, "sync")
	return
}

func constructorBody(fn *ssa.Function) []ssa.Instruction {
	if fn == nil || len(fn.Params) != 1 || len(fn.FreeVars) != 0 || len(fn.Blocks) != 1 {
		return nil
	}
	var body []ssa.Instruction
	for _, instr := range fn.Blocks[0].Instrs {
		if _, debug := instr.(*ssa.DebugRef); !debug {
			body = append(body, instr)
		}
	}
	return body
}

func isLiteralErrorConstructor(fn *ssa.Function) bool {
	body := constructorBody(fn)
	if len(body) != 5 || !types.Identical(fn.Params[0].Type(), types.Typ[types.String]) {
		return false
	}
	object, _ := body[0].(*ssa.Alloc)
	field, _ := body[1].(*ssa.FieldAddr)
	store, _ := body[2].(*ssa.Store)
	box, _ := body[3].(*ssa.MakeInterface)
	ret, _ := body[4].(*ssa.Return)
	return object != nil && field != nil && store != nil && box != nil && ret != nil &&
		field.X == object && store.Addr == field && store.Val == fn.Params[0] && box.X == object &&
		types.Identical(box.Type(), types.Universe.Lookup("error").Type()) &&
		len(ret.Results) == 1 && ret.Results[0] == box
}

// The selected sync factory fills fresh state and returns a wrapper that
// captures it. The callback and the wrapper run only when invoked later.
func isStaticCallbackConstructor(fn *ssa.Function, sync *ssa.Package) bool {
	if fn == nil || sync == nil || initializerFunctionOrigin(fn).Pkg != sync {
		return false
	}
	body := constructorBody(fn)
	if len(body) != 8 {
		return false
	}
	state, _ := body[0].(*ssa.Alloc)
	local, _ := body[1].(*ssa.Alloc)
	field, _ := body[2].(*ssa.FieldAddr)
	write, _ := body[3].(*ssa.Store)
	load, _ := body[4].(*ssa.UnOp)
	copy, _ := body[5].(*ssa.Store)
	closure, _ := body[6].(*ssa.MakeClosure)
	ret, _ := body[7].(*ssa.Return)
	return state != nil && local != nil && field != nil && field.X == local &&
		write != nil && write.Addr == field && write.Val == fn.Params[0] &&
		load != nil && load.Op == token.MUL && load.X == local &&
		copy != nil && copy.Addr == state && copy.Val == load &&
		closure != nil && len(closure.Bindings) == 1 && closure.Bindings[0] == state &&
		ret != nil && len(ret.Results) == 1 && ret.Results[0] == closure
}

func initializerFunctionOrigin(fn *ssa.Function) *ssa.Function {
	if origin := fn.Origin(); origin != nil {
		return origin
	}
	return fn
}

// optionalLiteralInitializers recognizes fresh literal errors, constant
// interface boxes and stored callbacks or wrappers. Each value is
// published by exactly one store.
func optionalLiteralInitializers(init *ssa.Function, constructors literalConstructors) (groups []cl.OptionalInitializer) {
	if init == nil || init.Synthetic != "package initializer" {
		return nil
	}
	for _, block := range init.Blocks {
		var positions []int
		for i, instr := range block.Instrs {
			if _, debug := instr.(*ssa.DebugRef); debug {
				continue
			}
			positions = append(positions, i)
			store, ok := instr.(*ssa.Store)
			if !ok || len(positions) < 2 {
				continue
			}
			global, _ := store.Addr.(*ssa.Global)
			if global == nil || global.Pkg != init.Pkg {
				continue
			}
			start := positions[len(positions)-2]
			if first, _ := store.Val.(ssa.Instruction); first != block.Instrs[start] || !constructors.discardable(store.Val) {
				start = staticCallbackInitializerStart(store, block.Instrs, positions)
			}
			if start >= 0 && cl.InitializerHasLocalUses(block.Instrs[start:i+1]) {
				groups = append(groups, cl.OptionalInitializer{First: block.Instrs[start], Store: store, Global: global})
			}
		}
	}
	return groups
}

// A zeroed local struct is filled with a noncapturing callback and copied
// into the global. The callback is stored without running it.
func staticCallbackInitializerStart(store *ssa.Store, instructions []ssa.Instruction, positions []int) int {
	load, ok := store.Val.(*ssa.UnOp)
	if !ok || load.Op != token.MUL || len(positions) < 5 {
		return -1
	}
	start := positions[len(positions)-5]
	local, _ := instructions[start].(*ssa.Alloc)
	field, _ := instructions[positions[len(positions)-4]].(*ssa.FieldAddr)
	write, _ := instructions[positions[len(positions)-3]].(*ssa.Store)
	if local == nil || local.Heap || field == nil || field.X != local || write == nil || write.Addr != field ||
		load.X != local || load != instructions[positions[len(positions)-2]] {
		return -1
	}
	if fn, ok := write.Val.(*ssa.Function); ok && len(fn.FreeVars) == 0 {
		return start
	}
	return -1
}

func (constructors literalConstructors) discardable(value ssa.Value) bool {
	switch value := value.(type) {
	case *ssa.Call:
		call := value.Common()
		if len(call.Args) != 1 {
			return false
		}
		callee := call.StaticCallee()
		if callee != nil && callee == constructors.newError {
			text, _ := call.Args[0].(*ssa.Const)
			return text != nil && text.Value != nil && text.Value.Kind() == constant.String
		}
		callback, ok := call.Args[0].(*ssa.Function)
		return ok && len(callback.FreeVars) == 0 && isStaticCallbackConstructor(callee, constructors.sync)
	case *ssa.MakeInterface:
		_, ok := value.X.(*ssa.Const)
		return ok
	}
	return false
}
