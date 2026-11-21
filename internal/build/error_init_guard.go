package build

import (
	"go/constant"
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

// optionalLiteralInitializers recognizes fresh literal errors and interface
// boxes of constant values. Each value is published by exactly one store.
func optionalLiteralInitializers(init, ctor *ssa.Function) (groups []cl.OptionalInitializer) {
	if init == nil || init.Synthetic != "package initializer" {
		return nil
	}
	for _, block := range init.Blocks {
		previous := -1
		for i, instr := range block.Instrs {
			if _, debug := instr.(*ssa.DebugRef); debug {
				continue
			}
			start := previous
			previous = i
			store, ok := instr.(*ssa.Store)
			if !ok || start < 0 {
				continue
			}
			first, _ := store.Val.(ssa.Instruction)
			if first != block.Instrs[start] {
				continue
			}
			global, _ := store.Addr.(*ssa.Global)
			if global != nil && global.Pkg == init.Pkg && discardableLiteralValue(store.Val, ctor) &&
				cl.InitializerHasLocalUses(block.Instrs[start:i+1]) {
				groups = append(groups, cl.OptionalInitializer{First: first, Store: store, Global: global})
			}
		}
	}
	return groups
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
