package cl

import (
	"go/token"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

// OptionalMapInitializer is a map literal stored in a package variable.
type OptionalMapInitializer struct {
	Make   *ssa.MakeMap
	Store  *ssa.Store
	Global *ssa.Global
}

// OptionalMapInitializers finds map literals that can be guarded independently.
func OptionalMapInitializers(fn *ssa.Function) (groups []OptionalMapInitializer) {
	if fn != nil && fn.Synthetic == "package initializer" {
		for _, block := range fn.Blocks {
			groups = append(groups, optionalMapInitializersInBlock(block)...)
		}
	}
	return groups
}

// Accept a contiguous literal with no effects other than constructing the
// map and storing its address. Every value computed in the group is local.
func optionalMapInitializersInBlock(block *ssa.BasicBlock) []OptionalMapInitializer {
	var groups []OptionalMapInitializer
	var current *ssa.MakeMap
	start := 0
	for i, inst := range block.Instrs {
		switch v := inst.(type) {
		case *ssa.MakeMap:
			current = nil
			if _, basic := v.Type().Underlying().(*types.Map).Key().Underlying().(*types.Basic); basic {
				current, start = v, i
			}
		case *ssa.DebugRef:
		case *ssa.UnOp:
			global, ok := v.X.(*ssa.Global)
			if !ok || v.Op != token.MUL || global.Pkg != block.Parent().Pkg {
				current = nil
			}
		case *ssa.MapUpdate:
			if v.Map != current {
				current = nil
			}
		case *ssa.Store:
			global, ok := v.Addr.(*ssa.Global)
			if current != nil && v.Val == current && ok && global.Pkg == block.Parent().Pkg &&
				mapInitializerHasLocalUses(block.Instrs[start:i+1]) {
				groups = append(groups, OptionalMapInitializer{Make: current, Store: v, Global: global})
			}
			current = nil
		default:
			current = nil
		}
	}
	return groups
}

func mapInitializerHasLocalUses(instrs []ssa.Instruction) bool {
	owned := make(map[ssa.Instruction]bool, len(instrs))
	for _, inst := range instrs {
		owned[inst] = true
	}
	for _, inst := range instrs {
		val, ok := inst.(ssa.Value)
		if !ok {
			continue
		}
		uses := val.Referrers()
		if uses == nil {
			return false
		}
		for _, use := range *uses {
			if !owned[use] {
				return false
			}
		}
	}
	return true
}

// MapInitializerGuardName identifies a package variable's initializer guard.
func MapInitializerGuardName(g *ssa.Global) string {
	return "_llgo_init_map." + llssa.FullName(g.Pkg.Pkg, g.Name())
}

// DefineMapInitializerGuard emits a weak true default in the package archive
// or a strong false override for a closed final link.
func DefineMapInitializerGuard(pkg llssa.Package, name string, enabled bool) llssa.Global {
	prog := pkg.Prog
	guard := pkg.NewVarEx(name, prog.Pointer(prog.Bool()))
	guard.Init(prog.BoolVal(enabled))
	value := pkg.Module().NamedGlobal(name)
	value.SetGlobalConstant(true)
	value.SetVisibility(llvm.HiddenVisibility)
	if enabled {
		value.SetLinkage(llvm.WeakAnyLinkage)
	}
	return guard
}
