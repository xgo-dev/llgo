package cl

import (
	"go/token"
	"go/types"
	"slices"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
	"golang.org/x/tools/go/ssa"
)

// OptionalInitializer is an instruction range ending with a package variable store.
type OptionalInitializer struct {
	First  ssa.Instruction
	Store  *ssa.Store
	Global *ssa.Global
}

// OptionalMapInitializers finds map literals that can be guarded independently.
func OptionalMapInitializers(fn *ssa.Function) (groups []OptionalInitializer) {
	if fn != nil && fn.Synthetic == "package initializer" {
		for _, block := range fn.Blocks {
			groups = append(groups, optionalMapInitializersInBlock(block)...)
		}
	}
	return groups
}

// OptionalSliceInitializers recognizes literal slices whose backing arrays are
// populated with constants or copies of other package variables.
func OptionalSliceInitializers(fn *ssa.Function) (groups []OptionalInitializer) {
	if fn == nil || fn.Synthetic != "package initializer" {
		return nil
	}
	for _, block := range fn.Blocks {
		for i, instr := range block.Instrs {
			store, ok := instr.(*ssa.Store)
			if ok {
				if start := optionalSliceInitializerStart(block, i, store); start >= 0 {
					groups = append(groups, OptionalInitializer{First: block.Instrs[start], Store: store, Global: store.Addr.(*ssa.Global)})
				}
			}
		}
	}
	return groups
}

func optionalSliceInitializerStart(block *ssa.BasicBlock, i int, store *ssa.Store) int {
	global, _ := store.Addr.(*ssa.Global)
	value, _ := store.Val.(*ssa.Slice)
	if global == nil || global.Pkg != block.Parent().Pkg || value == nil ||
		value.Low != nil || value.High != nil || value.Max != nil {
		return -1
	}
	alloc, _ := value.X.(*ssa.Alloc)
	if alloc == nil || alloc.Block() != block {
		return -1
	}
	array, ok := alloc.Type().Underlying().(*types.Pointer).Elem().Underlying().(*types.Array)
	start := slices.Index(block.Instrs[:i], ssa.Instruction(alloc))
	if ok && start >= 0 && literalSliceStores(block.Instrs[start+1:i], alloc, array, value) &&
		InitializerHasLocalUses(block.Instrs[start:i+1]) {
		return start
	}
	return -1
}

func literalSliceStores(instrs []ssa.Instruction, alloc *ssa.Alloc, array *types.Array, result *ssa.Slice) bool {
	for _, instr := range instrs {
		switch v := instr.(type) {
		case *ssa.DebugRef:
		case *ssa.UnOp:
			g, ok := v.X.(*ssa.Global)
			if !ok || g.Pkg != alloc.Parent().Pkg || v.Op != token.MUL {
				return false
			}
		case *ssa.IndexAddr:
			n, ok := staticInitConstIndex(v.Index)
			if v.X != alloc || !ok || int64(n) >= array.Len() {
				return false
			}
		case *ssa.Store:
			addr, ok := v.Addr.(*ssa.IndexAddr)
			if !ok || addr.X != alloc {
				return false
			}
		case *ssa.Slice:
			if v != result {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// Accept a contiguous literal with no effects other than constructing the
// map and storing its address. Every value computed in the group is local.
func optionalMapInitializersInBlock(block *ssa.BasicBlock) []OptionalInitializer {
	var groups []OptionalInitializer
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
				InitializerHasLocalUses(block.Instrs[start:i+1]) {
				groups = append(groups, OptionalInitializer{First: current, Store: v, Global: global})
			}
			current = nil
		default:
			current = nil
		}
	}
	return groups
}

// InitializerHasLocalUses determines whether all computed values stay within the range.
func InitializerHasLocalUses(instrs []ssa.Instruction) bool {
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

// InitializerGuardName identifies a package variable's initializer guard.
func InitializerGuardName(g *ssa.Global) string {
	return "_llgo_init_var." + llssa.FullName(g.Pkg.Pkg, g.Name())
}

// DefineInitializerGuard emits a weak true default in the package archive
// or a strong false override for a closed final link.
func DefineInitializerGuard(pkg llssa.Package, name string, enabled bool) llssa.Global {
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
