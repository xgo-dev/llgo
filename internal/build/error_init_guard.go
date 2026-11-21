package build

import (
	"go/constant"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/cl"
	"golang.org/x/tools/go/ssa"
)

// literalErrorConstructor resolves the standard constructor only when its
// selected body returns a fresh object initialized from a string. Removing an
// unobserved call must not remove anything that publishes the object.
func literalErrorConstructor(ctx *context, goroot string) *ssa.Function {
	pkg := ctx.pkgByID["errors"]
	if pkg == nil || pkg.AltPkg != nil || !isStandardLibraryPackage(pkg.Package, goroot) {
		return nil
	}
	if source := ctx.progSSA.Package(pkg.Types); source != nil {
		if ctor := source.Func("New"); isLiteralErrorConstructor(ctor) {
			return ctor
		}
	}
	return nil
}

func isLiteralErrorConstructor(fn *ssa.Function) bool {
	if fn == nil || len(fn.Params) != 1 || len(fn.FreeVars) != 0 || len(fn.Blocks) != 1 ||
		!types.Identical(fn.Params[0].Type(), types.Typ[types.String]) {
		return false
	}
	var body []ssa.Instruction
	for _, instr := range fn.Blocks[0].Instrs {
		if _, debug := instr.(*ssa.DebugRef); !debug {
			body = append(body, instr)
		}
	}
	if len(body) != 5 {
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

// optionalLiteralInitializers recognizes fresh literal errors, constant
// interface boxes and struct literals with a static callback. Each value is
// published by exactly one store.
func optionalLiteralInitializers(init, ctor *ssa.Function) (groups []cl.OptionalInitializer) {
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
			if first, _ := store.Val.(ssa.Instruction); first != block.Instrs[start] || !discardableLiteralValue(store.Val, ctor) {
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

func discardableLiteralValue(value ssa.Value, ctor *ssa.Function) bool {
	switch value := value.(type) {
	case *ssa.Call:
		if ctor != nil && value.Common().StaticCallee() == ctor && len(value.Common().Args) == 1 {
			text, _ := value.Common().Args[0].(*ssa.Const)
			return text != nil && text.Value != nil && text.Value.Kind() == constant.String
		}
	case *ssa.MakeInterface:
		_, ok := value.X.(*ssa.Const)
		return ok
	}
	return false
}
