package build

import (
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"slices"

	llssa "github.com/xgo-dev/llgo/ssa"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type startupEffects struct {
	trusted   map[*types.Package]bool
	active    map[*ssa.Function]bool
	passed    map[*ssa.Function]bool
	providers map[*ssa.Function]*ssa.Function
	disabled  map[*ssa.Global]bool
	sync      *ssa.Package
}

func (p *startupEffects) check(fn *ssa.Function) error {
	if fn == nil {
		return fmt.Errorf("indirect call")
	}
	if provider := p.providers[fn]; provider != nil {
		fn = provider
	}
	if p.passed[fn] || isStaticCallbackConstructor(fn, p.sync) {
		return nil
	}
	owner := initializerFunctionOrigin(fn).Pkg
	if owner == nil || !p.trusted[owner.Pkg] || len(fn.Blocks) == 0 || p.active[fn] {
		return fmt.Errorf("opaque call to %s", fn)
	}
	p.active[fn] = true
	defer delete(p.active, fn)
	seen := make(map[*ssa.BasicBlock]uint8)
	var visit func(*ssa.BasicBlock) error
	visit = func(block *ssa.BasicBlock) error {
		if seen[block] == 1 {
			return fmt.Errorf("%s: loop", fn)
		}
		if seen[block] == 2 {
			return nil
		}
		seen[block] = 1
		for _, inst := range block.Instrs {
			if err := p.instruction(fn, inst); err != nil {
				return fmt.Errorf("%s: %w", fn, err)
			}
		}
		if n := len(block.Instrs); n != 0 {
			if branch, ok := block.Instrs[n-1].(*ssa.If); ok {
				if value, known := p.boolean(branch.Cond); known {
					succ := block.Succs[1]
					if value {
						succ = block.Succs[0]
					}
					if err := visit(succ); err != nil {
						return err
					}
					seen[block] = 2
					return nil
				}
			}
		}
		for _, succ := range block.Succs {
			if err := visit(succ); err != nil {
				return err
			}
		}
		seen[block] = 2
		return nil
	}
	if err := visit(fn.Blocks[0]); err != nil {
		return err
	}
	p.passed[fn] = true
	return nil
}

func (p *startupEffects) instruction(fn *ssa.Function, inst ssa.Instruction) error {
	switch v := inst.(type) {
	case *ssa.Alloc, *ssa.ChangeType, *ssa.MakeClosure, *ssa.MakeInterface, *ssa.Field,
		*ssa.Phi, *ssa.DebugRef, *ssa.Jump, *ssa.If, *ssa.Return:
		return nil
	case *ssa.FieldAddr, *ssa.IndexAddr:
		if startupAddress(v.(ssa.Value)) != nil {
			return nil
		}
	case *ssa.UnOp:
		if _, known := p.boolean(v); known || v.Op == token.MUL && startupAddress(v.X) != nil {
			return nil
		}
	case *ssa.Slice:
		if _, ok := startupArray(v.X); ok && startupAddress(v.X) != nil && v.Low == nil && v.High == nil && v.Max == nil {
			return nil
		}
	case *ssa.Store:
		switch root := startupAddress(v.Addr).(type) {
		case *ssa.Alloc:
			return nil
		case *ssa.Global:
			if root.Pkg == fn.Pkg {
				return nil
			}
		}
	case *ssa.MakeMap:
		if v.Reserve == nil {
			return nil
		}
		if reserve, ok := v.Reserve.(*ssa.Const); ok && reserve.Value != nil &&
			reserve.Value.Kind() == constant.Int && constant.Sign(reserve.Value) >= 0 {
			return nil
		}
	case *ssa.MapUpdate:
		if fresh, ok := v.Map.(*ssa.MakeMap); ok && fresh.Parent() == fn {
			if key, ok := v.Key.(*ssa.Const); ok && key.Value != nil {
				return nil
			}
		}
	case *ssa.BinOp:
		if _, known := p.boolean(v); known {
			return nil
		}
		if basic, ok := v.X.Type().Underlying().(*types.Basic); ok && basic.Info()&types.IsInteger != 0 {
			switch v.Op {
			case token.ADD, token.SUB, token.MUL, token.EQL, token.NEQ, token.LEQ, token.LSS, token.GEQ, token.GTR:
				return nil
			}
		}
	case *ssa.Convert:
		from, fromOK := v.X.Type().Underlying().(*types.Basic)
		to, toOK := v.Type().Underlying().(*types.Basic)
		if fromOK && toOK && from.Info()&types.IsInteger != 0 && to.Info()&types.IsInteger != 0 {
			return nil
		}
	case *ssa.Call:
		if builtin, ok := v.Call.Value.(*ssa.Builtin); ok && (builtin.Name() == "len" || builtin.Name() == "cap") {
			return nil
		}
		callee := v.Call.StaticCallee()
		// The entry has a separate initializer list ordered by imports.
		if callee != nil && callee.Pkg != nil && callee.Synthetic == "package initializer" &&
			fn.Synthetic == "package initializer" && callee.Pkg != fn.Pkg && len(v.Call.Args) == 0 {
			for _, dep := range fn.Pkg.Pkg.Imports() {
				if dep == callee.Pkg.Pkg {
					return nil
				}
			}
		}
		return p.check(callee)
	}
	return fmt.Errorf("unproved %T %v", inst, inst)
}

func startupArray(value ssa.Value) (*types.Array, bool) {
	ptr, ok := value.Type().Underlying().(*types.Pointer)
	if !ok {
		return nil, false
	}
	array, ok := ptr.Elem().Underlying().(*types.Array)
	return array, ok
}

func startupAddress(value ssa.Value) ssa.Value {
	switch v := value.(type) {
	case *ssa.Alloc, *ssa.Global:
		return v
	case *ssa.FieldAddr:
		return startupAddress(v.X)
	case *ssa.IndexAddr:
		array, ok := startupArray(v.X)
		index, constantIndex := v.Index.(*ssa.Const)
		if ok && constantIndex && index.Value != nil && index.Value.Kind() == constant.Int {
			if i, valid := constant.Int64Val(index.Value); valid && i >= 0 && i < array.Len() {
				return startupAddress(v.X)
			}
		}
	}
	return nil
}

func newStartupEffects(ctx *context, goroot string) startupEffects {
	p := startupEffects{
		trusted:   make(map[*types.Package]bool),
		active:    make(map[*ssa.Function]bool),
		passed:    make(map[*ssa.Function]bool),
		providers: make(map[*ssa.Function]*ssa.Function),
		disabled:  make(map[*ssa.Global]bool),
		sync:      ctx.progSSA.ImportedPackage("sync"),
	}
	for _, pkg := range ctx.pkgByID {
		p.trusted[pkg.Types] = isStandardLibraryPackage(pkg.Package, goroot) || isRuntimePkg(pkg.PkgPath)
		if pkg.AltPkg != nil {
			p.trusted[pkg.AltPkg.Types] = true
		}
	}
	if rt := ctx.progSSA.ImportedPackage(wasiRuntimePath); rt != nil {
		p.trusted[rt.Pkg] = true
		for _, spec := range []struct{ pkg, name, impl string }{
			{"syscall", "runtime_envs", "syscall_runtime_envs"},
			{"os", "runtime_args", "os_runtime_args"},
			{"time", "runtimeNano", "time_runtimeNano"},
		} {
			if pkg := ctx.progSSA.ImportedPackage(spec.pkg); pkg != nil {
				from, to := pkg.Func(spec.name), rt.Func(spec.impl)
				link, _ := ctx.prog.Linkname("runtime." + spec.impl)
				if from != nil && to != nil && strings.TrimPrefix(link, "go:") == spec.pkg+"."+spec.name {
					p.providers[from] = to
				}
			}
		}
	}
	for _, pkg := range ctx.progSSA.AllPackages() {
		for _, member := range pkg.Members {
			if a, ok := member.(*ssa.Global); ok && slices.Contains(ctx.initializerGuardsOff, llssa.FullName(pkg.Pkg, a.Name())) {
				p.disabled[a] = true
			}
		}
	}
	return p
}

func (p *startupEffects) boolean(value ssa.Value) (bool, bool) {
	switch v := value.(type) {
	case *ssa.Const:
		if v.Value != nil && v.Value.Kind() == constant.Bool {
			return constant.BoolVal(v.Value), true
		}
	case *ssa.UnOp:
		if g, ok := v.X.(*ssa.Global); v.Op == token.MUL && ok && p.disabled[g] {
			return false, true
		}
		if v.Op == token.NOT {
			on, known := p.boolean(v.X)
			return !on, known
		}
	case *ssa.BinOp:
		left, knownLeft := p.integer(v.X)
		right, knownRight := p.integer(v.Y)
		if knownLeft && knownRight {
			switch v.Op {
			case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
				return constant.Compare(left, v.Op, right), true
			}
		}
		if v.Op != token.EQL && v.Op != token.NEQ {
			return false, false
		}
		nilCompare, ok := v.Y.(*ssa.Const)
		call, isCall := v.X.(*ssa.Call)
		if ok && nilCompare.IsNil() && isCall && startupReturnsBox(call.Common().StaticCallee()) {
			return v.Op == token.NEQ, true
		}
	}
	return false, false
}

func startupReturnsBox(callee *ssa.Function) bool {
	if callee == nil || len(callee.Blocks) != 1 {
		return false
	}
	list := callee.Blocks[0].Instrs
	if len(list) != 2 {
		return false
	}
	box, b := list[0].(*ssa.MakeInterface)
	ret, r := list[1].(*ssa.Return)
	return b && r && len(ret.Results) == 1 && ret.Results[0] == box
}

func (p *startupEffects) integer(value ssa.Value) (constant.Value, bool) {
	switch v := value.(type) {
	case *ssa.Const:
		return v.Value, v.Value != nil && v.Value.Kind() == constant.Int
	case *ssa.Call:
		if builtin, ok := v.Call.Value.(*ssa.Builtin); ok && len(v.Call.Args) == 1 && (builtin.Name() == "len" || builtin.Name() == "cap") {
			if c, ok := p.constantResult(v.Call.Args[0]); ok && c.IsNil() {
				return constant.MakeInt64(0), true
			}
		}
	}
	return nil, false
}

func (p *startupEffects) constantResult(value ssa.Value) (*ssa.Const, bool) {
	switch v := value.(type) {
	case *ssa.Const:
		return v, true
	case *ssa.Call:
		fn := v.Call.StaticCallee()
		if fn == nil {
			return nil, false
		}
		if provider := p.providers[fn]; provider != nil {
			fn = provider
		}
		if owner := initializerFunctionOrigin(fn).Pkg; owner == nil || !p.trusted[owner.Pkg] || len(fn.Blocks) != 1 {
			return nil, false
		}
		list := fn.Blocks[0].Instrs
		if len(list) == 1 {
			if ret, ok := list[0].(*ssa.Return); ok && len(ret.Results) == 1 {
				result, ok := ret.Results[0].(*ssa.Const)
				return result, ok
			}
		}
	}
	return nil, false
}
