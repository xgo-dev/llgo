/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package cl

import (
	"fmt"
	gotypes "go/types"
	"sort"

	"github.com/xgo-dev/llgo/cmd/compiler/pkgmeta"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
	"golang.org/x/tools/go/ssa"
)

// LowerGoPackageTypes converts a type-checked stdlib *go/types.Package
// (including stdlib packages with Module == nil and runtime-patched effective
// scopes) into a self-contained pure-Go *llgo/types.Package suitable for
// deterministic binary exportdata.Write persistence.
func LowerGoPackageTypes(goPkg *gotypes.Package, tctx *lltypes.Context) *lltypes.Package {
	if goPkg == nil {
		return nil
	}
	if tctx == nil {
		tctx = lltypes.NewContext()
	}
	visited := make(map[*gotypes.Package]*lltypes.Package)
	var convert func(gp *gotypes.Package) *lltypes.Package
	convert = func(gp *gotypes.Package) *lltypes.Package {
		if gp == nil {
			return nil
		}
		if gp == gotypes.Unsafe {
			return lltypes.Unsafe
		}
		if cached, ok := visited[gp]; ok {
			return cached
		}
		lp := lltypes.NewPackage(gp.Path(), gp.Name())
		visited[gp] = lp
		if imps := gp.Imports(); len(imps) > 0 {
			llImps := make([]*lltypes.Package, 0, len(imps))
			for _, imp := range imps {
				if conv := convert(imp); conv != nil {
					llImps = append(llImps, conv)
				}
			}
			lp.SetImports(llImps)
		}
		if gs := gp.Scope(); gs != nil {
			for _, name := range gs.Names() {
				if obj := gs.Lookup(name); obj != nil {
					if lobj := tctx.FromGoObject(obj); lobj != nil {
						lp.Scope().Insert(lobj)
					}
				}
			}
		}
		if gp.Complete() {
			lp.MarkComplete()
		}
		return lp
	}
	return convert(goPkg)
}

// LowerProgram translates a set of built golang.org/x/tools/go/ssa.Packages
// into a unified pure-Go *llgo/ssa.Program (1:1 instruction and CFG parity),
// returning the program and map from package path to *llgo/ssa.Package.
func LowerProgram(prog *ssa.Program, goPkgs []*ssa.Package, spec target.Spec, tctx *lltypes.Context) (*llssa.Program, map[string]*llssa.Package, error) {
	if spec.WordSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	if tctx == nil {
		tctx = lltypes.NewContextWithConfig(lltypes.ContextConfig{Spec: spec})
	}
	var fset = prog.Fset
	llProg := llssa.NewProgram(fset, llssa.SanityCheckFunctions)
	llProg.Target = spec

	l := &ssaLowerer{
		prog:    prog,
		llProg:  llProg,
		tctx:    tctx,
		pkgs:    make(map[*ssa.Package]*llssa.Package),
		byPath:  make(map[string]*llssa.Package),
		globals: make(map[*ssa.Global]*llssa.Global),
		funcs:   make(map[*ssa.Function]*llssa.Function),
	}

	for _, gp := range goPkgs {
		if gp == nil || gp.Pkg == nil {
			continue
		}
		l.ensurePackage(gp)
	}
	for _, gp := range goPkgs {
		if gp == nil || gp.Pkg == nil {
			continue
		}
		l.collectPackageMembers(gp)
	}
	for _, goFn := range l.orderedFuncs {
		l.lowerFunction(goFn)
	}
	llProg.Build()
	for _, lp := range l.byPath {
		if err := llssa.VerifyPackage(lp); err != nil {
			return nil, nil, err
		}
	}
	return llProg, l.byPath, nil
}

// LowerPackage translates a single golang.org/x/tools/go/ssa.Package into a
// verified pure-Go *llgo/ssa.Package.
func LowerPackage(prog *ssa.Program, goPkg *ssa.Package, spec target.Spec, tctx *lltypes.Context) (*llssa.Package, error) {
	if goPkg == nil || goPkg.Pkg == nil {
		return nil, fmt.Errorf("cl.LowerPackage: nil ssa.Package")
	}
	_, pkgs, err := LowerProgram(prog, []*ssa.Package{goPkg}, spec, tctx)
	if err != nil {
		return nil, err
	}
	return pkgs[goPkg.Pkg.Path()], nil
}

// BuildPackageMeta produces a compact binary compiler summary over llPkg.
func BuildPackageMeta(llPkg *llssa.Package, importedEsc map[string]*pkgmeta.EscapeSummary, inlineBudget int) *pkgmeta.PackageMeta {
	return pkgmeta.BuildPackageMeta(llPkg, nil, importedEsc, inlineBudget)
}

type ssaLowerer struct {
	prog         *ssa.Program
	llProg       *llssa.Program
	tctx         *lltypes.Context
	pkgs         map[*ssa.Package]*llssa.Package
	byPath       map[string]*llssa.Package
	globals      map[*ssa.Global]*llssa.Global
	funcs        map[*ssa.Function]*llssa.Function
	orderedFuncs []*ssa.Function
}

func (l *ssaLowerer) lowerType(t gotypes.Type) lltypes.Type {
	if t == nil {
		return lltypes.Typ[lltypes.Invalid]
	}
	res := l.tctx.FromGo(t)
	if res == nil {
		return lltypes.Typ[lltypes.Invalid]
	}
	return res
}

func (l *ssaLowerer) lowerSig(sig *gotypes.Signature) *lltypes.Signature {
	if sig == nil {
		return lltypes.NewSignatureType(nil, nil, nil, nil, nil, false)
	}
	if s, ok := l.lowerType(sig).(*lltypes.Signature); ok && s != nil {
		return s
	}
	return lltypes.NewSignatureType(nil, nil, nil, nil, nil, false)
}

func (l *ssaLowerer) ensurePackage(gp *ssa.Package) *llssa.Package {
	if gp == nil || gp.Pkg == nil {
		return nil
	}
	if lp, ok := l.pkgs[gp]; ok {
		return lp
	}
	llGoPkg := LowerGoPackageTypes(gp.Pkg, l.tctx)
	lp := l.llProg.CreatePackage(llGoPkg, nil, nil, true)
	l.pkgs[gp] = lp
	l.byPath[gp.Pkg.Path()] = lp
	return lp
}

func (l *ssaLowerer) ensureGlobal(g *ssa.Global) *llssa.Global {
	if g == nil {
		return nil
	}
	if lg, ok := l.globals[g]; ok {
		return lg
	}
	lp := l.ensurePackage(g.Pkg)
	var obj *lltypes.Var
	if g.Object() != nil {
		if v, ok := l.tctx.FromGoObject(g.Object()).(*lltypes.Var); ok {
			obj = v
		}
	}
	lg := lp.AddGlobal(g.Name(), obj, l.lowerType(g.Type()))
	lg.LinkName = g.String()
	l.globals[g] = lg
	return lg
}

func (l *ssaLowerer) ensureFunc(goFn *ssa.Function, declareBody bool) *llssa.Function {
	if goFn == nil {
		return nil
	}
	if lf, ok := l.funcs[goFn]; ok {
		return lf
	}
	sig := l.lowerSig(goFn.Signature)
	lf := l.llProg.NewFunction(goFn.Name(), sig, goFn.Synthetic)
	lf.LinkName = goFn.String()
	var fnObj *lltypes.Func
	if goFn.Object() != nil {
		if f, ok := l.tctx.FromGoObject(goFn.Object()).(*lltypes.Func); ok {
			fnObj = f
		}
	}
	if goFn.Pkg != nil {
		lp := l.ensurePackage(goFn.Pkg)
		if goFn.Parent() != nil {
			parentFn := l.ensureFunc(goFn.Parent(), false)
			parentFn.AddAnonFunc(lf)
		} else {
			lp.AddFunction(lf, fnObj)
		}
		if goFn.Signature != nil && goFn.Signature.Recv() != nil {
			recvType := l.lowerType(goFn.Signature.Recv().Type())
			var methodPkg *lltypes.Package
			if lp.Pkg != nil {
				methodPkg = lp.Pkg
			}
			l.llProg.RegisterMethod(recvType, methodPkg, goFn.Name(), lf)
		}
	}
	for _, p := range goFn.Params {
		var pObj lltypes.Object
		if p.Object() != nil {
			pObj = l.tctx.FromGoObject(p.Object())
		}
		lf.AddParam(p.Name(), pObj, l.lowerType(p.Type()))
	}
	for _, fv := range goFn.FreeVars {
		lf.AddFreeVar(fv.Name(), l.lowerType(fv.Type()), fv.Pos())
	}
	l.funcs[goFn] = lf
	if declareBody {
		l.orderedFuncs = append(l.orderedFuncs, goFn)
	}
	return lf
}

func (l *ssaLowerer) collectPackageMembers(gp *ssa.Package) {
	lp := l.ensurePackage(gp)
	memberNames := make([]string, 0, len(gp.Members))
	for name := range gp.Members {
		memberNames = append(memberNames, name)
	}
	sort.Strings(memberNames)

	seenFn := make(map[*ssa.Function]bool)
	var addFn func(fn *ssa.Function)
	addFn = func(fn *ssa.Function) {
		if fn == nil || seenFn[fn] || fn.Pkg != gp {
			return
		}
		if fn.TypeParams().Len() > 0 && len(fn.TypeArgs()) == 0 {
			return
		}
		seenFn[fn] = true
		l.ensureFunc(fn, true)
		for _, anon := range fn.AnonFuncs {
			addFn(anon)
		}
	}

	for _, name := range memberNames {
		switch m := gp.Members[name].(type) {
		case *ssa.Global:
			l.ensureGlobal(m)
		case *ssa.Function:
			addFn(m)
		case *ssa.NamedConst:
			if cObj, ok := l.tctx.FromGoObject(m.Object()).(*lltypes.Const); ok && m.Value != nil {
				lp.AddConst(cObj, llssa.NewConst(m.Value.Value, l.lowerType(m.Value.Type())))
			}
		case *ssa.Type:
			if tObj, ok := l.tctx.FromGoObject(m.Object()).(*lltypes.TypeName); ok {
				lp.AddType(tObj)
			}
			if l.prog == nil {
				continue
			}
			for _, typ := range []gotypes.Type{m.Type(), gotypes.NewPointer(m.Type())} {
				mset := l.prog.MethodSets.MethodSet(typ)
				for i := 0; i < mset.Len(); i++ {
					if mfn := l.prog.MethodValue(mset.At(i)); mfn != nil {
						addFn(mfn)
						// Ensure pointer/value receiver method is registered under typ.
						if lf := l.funcs[mfn]; lf != nil {
							l.llProg.RegisterMethod(l.lowerType(typ), lp.Pkg, mfn.Name(), lf)
						}
					}
				}
			}
		}
	}
}

func (l *ssaLowerer) lowerFunction(goFn *ssa.Function) {
	llFn := l.funcs[goFn]
	if llFn == nil || len(goFn.Blocks) == 0 {
		return
	}

	// Keep CFG-reachable blocks from entry (goFn.Blocks[0]).
	reachable := make([]*ssa.BasicBlock, 0, len(goFn.Blocks))
	visited := make(map[*ssa.BasicBlock]bool, len(goFn.Blocks))
	var dfs func(b *ssa.BasicBlock)
	dfs = func(b *ssa.BasicBlock) {
		if b == nil || visited[b] {
			return
		}
		visited[b] = true
		reachable = append(reachable, b)
		for _, s := range b.Succs {
			dfs(s)
		}
	}
	dfs(goFn.Blocks[0])

	blkMap := make(map[*ssa.BasicBlock]*llssa.BasicBlock, len(reachable))
	for _, gb := range reachable {
		blkMap[gb] = llFn.NewBasicBlock(gb.Comment)
	}
	if goFn.Recover != nil {
		llFn.Recover = blkMap[goFn.Recover]
	}

	// Wire CFG edges symmetrically before Phi resolution.
	for _, gb := range reachable {
		lb := blkMap[gb]
		for _, gs := range gb.Succs {
			if ls := blkMap[gs]; ls != nil {
				lb.AddEdge(ls)
			}
		}
	}

	valMap := make(map[ssa.Value]llssa.Value, len(goFn.Params)+len(goFn.FreeVars)+16)
	for i, p := range goFn.Params {
		if i < len(llFn.Params) {
			valMap[p] = llFn.Params[i]
		}
	}
	for i, fv := range goFn.FreeVars {
		if i < len(llFn.FreeVars) {
			valMap[fv] = llFn.FreeVars[i]
		}
	}

	type pendingPhi struct {
		goPhi *ssa.Phi
		llPhi *llssa.Phi
		goBlk *ssa.BasicBlock
	}
	var phis []pendingPhi

	// Pass 1: emit Phi placeholders at top of each block.
	for _, gb := range reachable {
		lb := blkMap[gb]
		for _, ins := range gb.Instrs {
			gp, ok := ins.(*ssa.Phi)
			if !ok {
				continue
			}
			lp := &llssa.Phi{Comment: gp.Comment}
			lb.Emit(lp, l.lowerType(gp.Type()), gp.Pos())
			valMap[gp] = lp
			phis = append(phis, pendingPhi{goPhi: gp, llPhi: lp, goBlk: gb})
		}
	}

	resolveVal := func(v ssa.Value) llssa.Value {
		if v == nil {
			return nil
		}
		if lv, ok := valMap[v]; ok {
			return lv
		}
		switch v := v.(type) {
		case *ssa.Const:
			return llssa.NewConst(v.Value, l.lowerType(v.Type()))
		case *ssa.Global:
			return l.ensureGlobal(v)
		case *ssa.Function:
			return l.ensureFunc(v, false)
		case *ssa.Builtin:
			sig, _ := l.lowerType(v.Type()).(*lltypes.Signature)
			return llssa.NewBuiltin(v.Name(), sig)
		default:
			return llssa.NewConst(nil, l.lowerType(v.Type()))
		}
	}

	resolveArgs := func(args []ssa.Value) []llssa.Value {
		if len(args) == 0 {
			return nil
		}
		out := make([]llssa.Value, len(args))
		for i, a := range args {
			out[i] = resolveVal(a)
		}
		return out
	}

	lowerCallCommon := func(c *ssa.CallCommon) llssa.CallCommon {
		cc := llssa.CallCommon{
			Value: resolveVal(c.Value),
			Args:  resolveArgs(c.Args),
		}
		if c.Method != nil {
			if mObj, ok := l.tctx.FromGoObject(c.Method).(*lltypes.Func); ok {
				cc.Method = mObj
			}
		}
		return cc
	}

	// Pass 2: lower all 37 standard x/tools/go/ssa instructions 1:1.
	for _, gb := range reachable {
		lb := blkMap[gb]
		for _, ins := range gb.Instrs {
			if _, isPhi := ins.(*ssa.Phi); isPhi {
				continue
			}
			var llIns llssa.Instruction
			var insType lltypes.Type
			if v, ok := ins.(ssa.Value); ok {
				insType = l.lowerType(v.Type())
			}
			switch ins := ins.(type) {
			case *ssa.Alloc:
				noScan := false
				if ptr, ok := insType.(*lltypes.Pointer); ok && ptr.Elem() != nil {
					noScan = !ptr.Elem().HasPtrData()
				}
				llIns = &llssa.Alloc{Comment: ins.Comment, Heap: ins.Heap, NoScan: noScan}
			case *ssa.BinOp:
				llIns = &llssa.BinOp{Op: ins.Op, X: resolveVal(ins.X), Y: resolveVal(ins.Y)}
			case *ssa.UnOp:
				llIns = &llssa.UnOp{Op: ins.Op, X: resolveVal(ins.X), CommaOk: ins.CommaOk}
			case *ssa.Call:
				llIns = &llssa.Call{Call: lowerCallCommon(&ins.Call)}
			case *ssa.ChangeInterface:
				llIns = &llssa.ChangeInterface{X: resolveVal(ins.X)}
			case *ssa.ChangeType:
				llIns = &llssa.ChangeType{X: resolveVal(ins.X)}
			case *ssa.Convert:
				llIns = &llssa.Convert{X: resolveVal(ins.X)}
			case *ssa.MultiConvert:
				llIns = &llssa.MultiConvert{X: resolveVal(ins.X)}
			case *ssa.SliceToArrayPointer:
				llIns = &llssa.SliceToArrayPointer{X: resolveVal(ins.X)}
			case *ssa.DebugRef:
				llIns = &llssa.DebugRef{Expr: ins.Expr, IsAddr: ins.IsAddr, X: resolveVal(ins.X)}
			case *ssa.Defer:
				llIns = &llssa.Defer{Call: lowerCallCommon(&ins.Call), DeferStack: resolveVal(ins.DeferStack)}
			case *ssa.Extract:
				llIns = &llssa.Extract{Tuple: resolveVal(ins.Tuple), Index: ins.Index}
			case *ssa.Field:
				llIns = &llssa.Field{X: resolveVal(ins.X), Field: ins.Field}
			case *ssa.FieldAddr:
				llIns = &llssa.FieldAddr{X: resolveVal(ins.X), Field: ins.Field}
			case *ssa.Go:
				llIns = &llssa.Go{Call: lowerCallCommon(&ins.Call)}
			case *ssa.If:
				llIns = &llssa.If{Cond: resolveVal(ins.Cond)}
			case *ssa.Index:
				llIns = &llssa.Index{X: resolveVal(ins.X), Index: resolveVal(ins.Index)}
			case *ssa.IndexAddr:
				llIns = &llssa.IndexAddr{X: resolveVal(ins.X), Index: resolveVal(ins.Index)}
			case *ssa.Jump:
				llIns = &llssa.Jump{}
			case *ssa.Lookup:
				llIns = &llssa.Lookup{X: resolveVal(ins.X), Index: resolveVal(ins.Index), CommaOk: ins.CommaOk}
			case *ssa.MakeChan:
				llIns = &llssa.MakeChan{Size: resolveVal(ins.Size)}
			case *ssa.MakeClosure:
				llIns = &llssa.MakeClosure{Fn: resolveVal(ins.Fn), Bindings: resolveArgs(ins.Bindings)}
			case *ssa.MakeInterface:
				x := resolveVal(ins.X)
				if x != nil {
					l.llProg.AddRuntimeType(x.Type())
				}
				llIns = &llssa.MakeInterface{X: x}
			case *ssa.MakeMap:
				llIns = &llssa.MakeMap{Reserve: resolveVal(ins.Reserve)}
			case *ssa.MakeSlice:
				llIns = &llssa.MakeSlice{Len: resolveVal(ins.Len), Cap: resolveVal(ins.Cap)}
			case *ssa.MapUpdate:
				llIns = &llssa.MapUpdate{Map: resolveVal(ins.Map), Key: resolveVal(ins.Key), Value: resolveVal(ins.Value)}
			case *ssa.Next:
				llIns = &llssa.Next{Iter: resolveVal(ins.Iter), IsString: ins.IsString}
			case *ssa.Panic:
				llIns = &llssa.Panic{X: resolveVal(ins.X)}
			case *ssa.Range:
				llIns = &llssa.Range{X: resolveVal(ins.X)}
			case *ssa.Return:
				llIns = &llssa.Return{Results: resolveArgs(ins.Results)}
			case *ssa.RunDefers:
				llIns = &llssa.RunDefers{}
			case *ssa.Select:
				states := make([]*llssa.SelectState, len(ins.States))
				for i, st := range ins.States {
					states[i] = &llssa.SelectState{
						Dir:       lltypes.ChanDir(st.Dir),
						Chan:      resolveVal(st.Chan),
						Send:      resolveVal(st.Send),
						Pos:       st.Pos,
						DebugNode: st.DebugNode,
					}
				}
				llIns = &llssa.Select{States: states, Blocking: ins.Blocking}
			case *ssa.Send:
				llIns = &llssa.Send{Chan: resolveVal(ins.Chan), X: resolveVal(ins.X)}
			case *ssa.Slice:
				llIns = &llssa.Slice{X: resolveVal(ins.X), Low: resolveVal(ins.Low), High: resolveVal(ins.High), Max: resolveVal(ins.Max)}
			case *ssa.Store:
				llIns = &llssa.Store{Addr: resolveVal(ins.Addr), Val: resolveVal(ins.Val)}
			case *ssa.TypeAssert:
				llIns = &llssa.TypeAssert{X: resolveVal(ins.X), AssertedType: l.lowerType(ins.AssertedType), CommaOk: ins.CommaOk}
			}
			if llIns != nil {
				lb.Emit(llIns, insType, ins.Pos())
				if lv, ok := llIns.(llssa.Value); ok {
					if gv, ok := ins.(ssa.Value); ok {
						valMap[gv] = lv
					}
				}
			}
		}
	}

	// Pass 3: populate Phi edges aligned strictly with lb.Preds order.
	for _, pp := range phis {
		lb := blkMap[pp.goBlk]
		// Map each reachable Go predecessor -> its incoming value in goPhi.Edges.
		predVal := make(map[*llssa.BasicBlock]llssa.Value, len(pp.goBlk.Preds))
		for i, gPred := range pp.goBlk.Preds {
			lPred := blkMap[gPred]
			if lPred == nil {
				continue
			}
			if i < len(pp.goPhi.Edges) {
				predVal[lPred] = resolveVal(pp.goPhi.Edges[i])
			} else {
				predVal[lPred] = llssa.NewConst(nil, pp.llPhi.Type())
			}
		}
		pp.llPhi.Edges = make([]llssa.Value, len(lb.Preds))
		for i, lPred := range lb.Preds {
			if v, ok := predVal[lPred]; ok && v != nil {
				pp.llPhi.Edges[i] = v
			} else {
				pp.llPhi.Edges[i] = llssa.NewConst(nil, pp.llPhi.Type())
			}
		}
	}
}
