package jsonspecialize

import (
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/ssa"
)

func IsImplementation(path string) bool {
	return path == "encoding/json" || strings.HasPrefix(path, "encoding/json/")
}

func ImplementationSymbol(name string) bool {
	return strings.HasPrefix(name, "encoding/json.") || strings.HasPrefix(name, "encoding/json/")
}

// DiscardableInit reports whether init initializes only private JSON state.
// The linker must separately exclude any user of that state and keep the
// initializers of dependencies outside the codec.
func DiscardableInit(fn *ssa.Function, trusted func(*types.Package) bool) bool {
	p := initProof{trusted: trusted, checked: make(map[*ssa.Function]bool)}
	return fn != nil && fn.Synthetic == "package initializer" && p.check(fn)
}

type initProof struct {
	trusted func(*types.Package) bool
	checked map[*ssa.Function]bool
}

func (p *initProof) check(fn *ssa.Function) bool {
	pkg := fn.Package()
	if pkg == nil || !IsImplementation(pkg.Pkg.Path()) || !p.trusted(pkg.Pkg) || len(fn.Blocks) == 0 || p.checked[fn] {
		return false
	}
	p.checked[fn] = true
	defer delete(p.checked, fn)
	for _, block := range fn.Blocks {
		for _, succ := range block.Succs {
			if succ.Index <= block.Index {
				return false
			}
		}
		slices := make(map[*ssa.Global]bool)
		fullLoads := make(map[*ssa.UnOp]bool)
		for _, instr := range block.Instrs {
			if !p.instruction(fn, instr, slices, fullLoads) {
				return false
			}
		}
	}
	return true
}

func (p *initProof) instruction(fn *ssa.Function, instr ssa.Instruction, fullSlices map[*ssa.Global]bool, fullLoads map[*ssa.UnOp]bool) bool {
	switch v := instr.(type) {
	case *ssa.Alloc, *ssa.ChangeType, *ssa.MakeClosure, *ssa.MakeInterface, *ssa.Field,
		*ssa.Phi, *ssa.DebugRef, *ssa.Jump, *ssa.If, *ssa.Return:
		return true
	case *ssa.FieldAddr, *ssa.IndexAddr:
		return address(v.(ssa.Value)) != nil
	case *ssa.UnOp:
		if global, ok := v.X.(*ssa.Global); ok && v.Op == token.MUL && fullSlices[global] {
			fullLoads[v] = true
		}
		return v.Op == token.MUL && address(v.X) != nil
	case *ssa.BinOp:
		_, scalar := v.X.Type().Underlying().(*types.Basic)
		return scalar && (v.Op == token.EQL || v.Op == token.NEQ)
	case *ssa.Convert:
		from, ok := v.X.Type().Underlying().(*types.Basic)
		to, slice := v.Type().Underlying().(*types.Slice)
		return ok && slice && from.Kind() == types.String && types.Identical(to.Elem(), types.Typ[types.Byte])
	case *ssa.Slice:
		_, ok := fullSlice(v)
		return ok
	case *ssa.Store:
		switch to := address(v.Addr).(type) {
		case *ssa.Alloc:
			return true
		case *ssa.Global:
			delete(fullSlices, to)
			if slice, ok := v.Val.(*ssa.Slice); ok && to == v.Addr {
				_, fullSlices[to] = fullSlice(slice)
			}
			return IsImplementation(to.Pkg.Pkg.Path()) && p.trusted(to.Pkg.Pkg)
		}
	case *ssa.Call:
		return p.call(fn, v.Common(), fullSlices, fullLoads)
	}
	return false
}

func (p *initProof) call(fn *ssa.Function, call *ssa.CallCommon, fullSlices map[*ssa.Global]bool, fullLoads map[*ssa.UnOp]bool) bool {
	if builtin, ok := call.Value.(*ssa.Builtin); ok {
		switch builtin.Name() {
		case "len":
			return true
		case "append":
			// A full slice forces fresh backing storage on a nonempty
			// append. The zero-element case performs no writes.
			load, _ := call.Args[0].(*ssa.UnOp)
			return fullLoads[load]
		}
		return false
	}
	to := call.StaticCallee()
	if to == nil {
		return false
	}
	// LLGo calls dependency initializers independently from its entry module.
	if fn.Synthetic == "package initializer" && to.Synthetic == "package initializer" &&
		to.Package() != nil && to.Package() != fn.Package() && len(call.Args) == 0 {
		for _, dep := range fn.Package().Pkg.Imports() {
			if dep == to.Package().Pkg {
				return true
			}
		}
	}
	// These standard-library constructors initialize fresh objects or
	// return type descriptors. OnceValue stores its callback for later.
	obj := to.Object()
	if obj == nil && to.Origin() != nil {
		obj = to.Origin().Object()
	}
	if obj != nil && obj.Pkg() != nil && p.trusted(obj.Pkg()) {
		switch obj.Pkg().Path() + "." + obj.Name() {
		case "errors.New", "reflect.TypeFor", "sync.OnceValue",
			"github.com/xgo-dev/llgo/runtime/internal/lib/reflect.TypeFor":
			return true
		case "strings.NewReplacer":
			if len(call.Args) == 1 {
				if arg, ok := call.Args[0].(*ssa.Slice); ok {
					n, known := fullSlice(arg)
					return known && n%2 == 0
				}
			}
			return false
		case "encoding/json/jsontext.Export":
			return jsonExportGuard(to, call.Args)
		}
	}
	clear(fullSlices)
	return p.check(to)
}

func jsonExportGuard(fn *ssa.Function, args []ssa.Value) bool {
	// The private Export method only checks that its argument is the
	// shared allow token. Verify the actual equality guard and return.
	if len(args) != 2 || len(fn.Params) != 2 || len(fn.Blocks) != 3 {
		return false
	}
	entry := fn.Blocks[0]
	if len(entry.Instrs) != 2 || len(entry.Succs) != 2 {
		return false
	}
	cmp, ok := entry.Instrs[0].(*ssa.BinOp)
	branch, isBranch := entry.Instrs[1].(*ssa.If)
	if !ok || !isBranch || branch.Cond != cmp || cmp.Op != token.NEQ || cmp.X != fn.Params[1] || cmp.Y != args[1] {
		return false
	}
	equal := entry.Succs[1].Instrs
	if len(equal) != 1 {
		return false
	}
	ret, ok := equal[0].(*ssa.Return)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	_, ok = ret.Results[0].(*ssa.Const)
	return ok
}

func fullSlice(v *ssa.Slice) (int64, bool) {
	arr := arrayPointer(v.X)
	if arr == nil || v.Low != nil || v.High != nil || v.Max != nil || address(v.X) == nil {
		return 0, false
	}
	return arr.Len(), true
}

func arrayPointer(v ssa.Value) *types.Array {
	ptr, ok := v.Type().Underlying().(*types.Pointer)
	if !ok {
		return nil
	}
	array, _ := ptr.Elem().Underlying().(*types.Array)
	return array
}

// address accepts only non-nil addresses with statically in-range indexing.
func address(v ssa.Value) ssa.Value {
	switch v := v.(type) {
	case *ssa.Alloc, *ssa.Global:
		return v
	case *ssa.FieldAddr:
		return address(v.X)
	case *ssa.IndexAddr:
		index, ok := v.Index.(*ssa.Const)
		array := arrayPointer(v.X)
		if ok && array != nil && index.Value != nil && index.Value.Kind() == constant.Int {
			if n, ok := constant.Int64Val(index.Value); ok && n >= 0 && n < array.Len() {
				return address(v.X)
			}
		}
	}
	return nil
}
