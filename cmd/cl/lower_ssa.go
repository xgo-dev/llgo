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
	"go/constant"
	"go/token"
	gotypes "go/types"
	"sort"

	"github.com/xgo-dev/llgo/analysis/callgraph"
	"github.com/xgo-dev/llgo/analysis/escape"
	"github.com/xgo-dev/llgo/analysis/inline"
	"github.com/xgo-dev/llgo/exportdata"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/target"
	lltypes "github.com/xgo-dev/llgo/types"
	"golang.org/x/tools/go/ssa"
)

// LowerGoPackageTypes converts a type-checked stdlib *go/types.Package
// (including stdlib packages with Module == nil and runtime-patched effective
// scopes) into a self-contained pure-Go *llgo/types.Package suitable for
// deterministic binary exportdata.MarshalPackage persistence.
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

// LowerPackage translates a built golang.org/x/tools/go/ssa.Package into
// LLGo's pure-Go, backend-independent mid-level IR (*llgo/ssa.Package).
// The resulting package passes llgo/ssa.VerifyPackage and feeds llgo/analysis/*
// and exportdata.PackageMeta sidecar generation without LLVM or CGo.
func LowerPackage(prog *ssa.Program, goPkg *ssa.Package, spec target.Spec, tctx *lltypes.Context) (*llssa.Package, error) {
	if goPkg == nil || goPkg.Pkg == nil {
		return nil, fmt.Errorf("cl.LowerPackage: nil ssa.Package")
	}
	if spec.WordSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	if tctx == nil {
		tctx = lltypes.NewContextWithConfig(lltypes.ContextConfig{Spec: spec})
	}
	out := llssa.NewPackage(goPkg.Pkg.Path(), goPkg.Pkg.Name(), spec)
	for _, imp := range goPkg.Pkg.Imports() {
		if imp != nil {
			out.Imports = append(out.Imports, imp.Path())
		}
	}
	sort.Strings(out.Imports)

	l := &ssaLowerer{
		prog:       prog,
		goPkg:      goPkg,
		out:        out,
		tctx:       tctx,
		globals:    make(map[*ssa.Global]*llssa.Global),
		funcs:      make(map[*ssa.Function]*llssa.Function),
		globalByID: make(map[string]*llssa.Global),
	}
	l.collectMembers()
	for _, goFn := range l.orderedFuncs {
		l.lowerFunction(goFn)
	}
	if err := llssa.VerifyPackage(out); err != nil {
		return nil, err
	}
	return out, nil
}

// BuildPackageMeta runs LLGo's pure-Go analysis suite (Summarize, callgraph,
// escape, inline) on a lowered llgo/ssa.Package to produce a compact binary
// exportdata.PackageMeta sidecar.
func BuildPackageMeta(llPkg *llssa.Package, importedEsc map[string]*exportdata.EscapeSummary, inlineBudget int) *exportdata.PackageMeta {
	if llPkg == nil {
		return nil
	}
	sum := llssa.Summarize(llPkg)
	esc := escape.AnalyzePackage(llPkg, importedEsc)
	inl := inline.AnalyzePackage(llPkg, inlineBudget)
	edges := callgraph.ExtractEdges(llPkg)

	globals := make([]string, 0, len(llPkg.Globals))
	for _, g := range llPkg.Globals {
		globals = append(globals, g.Name)
	}
	sort.Strings(globals)

	var exportFuncs []string
	for _, fn := range llPkg.Funcs {
		if fn.ExportName != "" {
			exportFuncs = append(exportFuncs, fn.Name)
		}
	}
	sort.Strings(exportFuncs)

	meta := &exportdata.PackageMeta{
		Summary:        sum,
		DefinedGlobals: globals,
		CallEdges:      edges,
		ExportFuncs:    exportFuncs,
	}
	if esc != nil {
		meta.Escape = *esc
	}
	if inl != nil {
		meta.Inline = *inl
	}
	return meta
}

type ssaLowerer struct {
	prog         *ssa.Program
	goPkg        *ssa.Package
	out          *llssa.Package
	tctx         *lltypes.Context
	globals      map[*ssa.Global]*llssa.Global
	globalByID   map[string]*llssa.Global
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

func (l *ssaLowerer) ensureGlobal(g *ssa.Global) *llssa.Global {
	if lg, ok := l.globals[g]; ok {
		return lg
	}
	name := g.String()
	if lg, ok := l.globalByID[name]; ok {
		l.globals[g] = lg
		return lg
	}
	var elemType lltypes.Type
	if ptr, ok := g.Type().Underlying().(*gotypes.Pointer); ok {
		elemType = l.lowerType(ptr.Elem())
	} else {
		elemType = l.lowerType(g.Type())
	}
	lg := l.out.NewGlobal(name, elemType)
	lg.LinkName = name
	if g.Pkg != l.goPkg {
		lg.External = true
	}
	l.globals[g] = lg
	l.globalByID[name] = lg
	return lg
}

func (l *ssaLowerer) lowerFuncSig(goFn *ssa.Function) *lltypes.Signature {
	params := make([]*lltypes.Var, 0, len(goFn.Params)+len(goFn.FreeVars))
	for _, p := range goFn.Params {
		params = append(params, lltypes.NewParam(p.Pos(), nil, p.Name(), l.lowerType(p.Type())))
	}
	for _, fv := range goFn.FreeVars {
		params = append(params, lltypes.NewParam(fv.Pos(), nil, fv.Name(), l.lowerType(fv.Type())))
	}
	var results []*lltypes.Var
	if goFn.Signature != nil && goFn.Signature.Results() != nil {
		res := goFn.Signature.Results()
		results = make([]*lltypes.Var, res.Len())
		for i := 0; i < res.Len(); i++ {
			r := res.At(i)
			results[i] = lltypes.NewParam(r.Pos(), nil, r.Name(), l.lowerType(r.Type()))
		}
	}
	variadic := goFn.Signature != nil && goFn.Signature.Variadic()
	return lltypes.NewSignatureType(nil, nil, nil, lltypes.NewTuple(params...), lltypes.NewTuple(results...), variadic)
}

func (l *ssaLowerer) collectMembers() {
	memberNames := make([]string, 0, len(l.goPkg.Members))
	for name := range l.goPkg.Members {
		memberNames = append(memberNames, name)
	}
	sort.Strings(memberNames)

	seenFn := make(map[*ssa.Function]bool)
	seenName := make(map[string]bool)
	var addFn func(fn *ssa.Function)
	addFn = func(fn *ssa.Function) {
		if fn == nil || seenFn[fn] {
			return
		}
		if fn.Pkg != l.goPkg {
			return
		}
		if fn.TypeParams().Len() > 0 && len(fn.TypeArgs()) == 0 {
			// Skip uninstantiated generic templates; instantiated concrete bodies
			// are lowered when present.
			return
		}
		name := fn.String()
		if seenName[name] {
			return
		}
		seenFn[fn] = true
		seenName[name] = true
		l.orderedFuncs = append(l.orderedFuncs, fn)
		for _, anon := range fn.AnonFuncs {
			addFn(anon)
		}
	}

	for _, name := range memberNames {
		switch m := l.goPkg.Members[name].(type) {
		case *ssa.Global:
			l.ensureGlobal(m)
		case *ssa.Function:
			addFn(m)
		case *ssa.Type:
			if l.prog == nil {
				continue
			}
			for _, typ := range []gotypes.Type{m.Type(), gotypes.NewPointer(m.Type())} {
				mset := l.prog.MethodSets.MethodSet(typ)
				for i := 0; i < mset.Len(); i++ {
					if mfn := l.prog.MethodValue(mset.At(i)); mfn != nil {
						addFn(mfn)
					}
				}
			}
		}
	}

	sort.Slice(l.orderedFuncs, func(i, j int) bool {
		return l.orderedFuncs[i].String() < l.orderedFuncs[j].String()
	})
	for _, goFn := range l.orderedFuncs {
		sig := l.lowerFuncSig(goFn)
		llFn := l.out.NewFunction(goFn.String(), sig)
		llFn.LinkName = goFn.String()
		llFn.HasEnv = len(goFn.FreeVars) > 0
		l.funcs[goFn] = llFn
	}
}

func (l *ssaLowerer) lowerFunction(goFn *ssa.Function) {
	llFn := l.funcs[goFn]
	if llFn == nil || len(goFn.Blocks) == 0 {
		return
	}

	// Collect blocks reachable from entry (goFn.Blocks[0]) so dead unreachable
	// blocks cannot break Phi predecessor invariants in llgo/ssa.VerifyFunction.
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
		label := fmt.Sprintf("b%d", gb.Index)
		if gb.Comment != "" {
			label = fmt.Sprintf("b%d.%s", gb.Index, gb.Comment)
		}
		blkMap[gb] = llFn.NewBasicBlock(label)
	}

	valMap := make(map[ssa.Value]llssa.Value, len(goFn.Params)+len(goFn.FreeVars)+16)
	for i, p := range goFn.Params {
		if i < len(llFn.Params) {
			valMap[p] = llFn.Params[i]
		}
	}
	for i, fv := range goFn.FreeVars {
		idx := len(goFn.Params) + i
		if idx < len(llFn.Params) {
			valMap[fv] = llFn.Params[idx]
		}
	}

	type pendingPhi struct {
		goPhi *ssa.Phi
		llPhi *llssa.Phi
		goBlk *ssa.BasicBlock
	}
	var phis []pendingPhi

	// Pass 1: emit Phi placeholders at the start of each block (strictly before
	// non-Phi instructions as required by SSA form).
	b := llssa.NewBuilder(blkMap[reachable[0]])
	for _, gb := range reachable {
		lb := blkMap[gb]
		b.SetBlock(lb)
		for _, ins := range gb.Instrs {
			gp, ok := ins.(*ssa.Phi)
			if !ok {
				continue
			}
			b.SetPos(gp.Pos())
			lp := b.Phi(l.lowerType(gp.Type()))
			valMap[gp] = lp
			phis = append(phis, pendingPhi{goPhi: gp, llPhi: lp, goBlk: gb})
		}
	}

	// Pass 2: emit all non-Phi instructions and terminators.
	resolveVal := func(v ssa.Value) llssa.Value {
		if v == nil {
			return b.Const(lltypes.Typ[lltypes.UntypedNil], nil)
		}
		if lv, ok := valMap[v]; ok {
			return lv
		}
		switch v := v.(type) {
		case *ssa.Const:
			return b.Const(l.lowerType(v.Type()), v.Value)
		case *ssa.Global:
			return b.GlobalRef(l.ensureGlobal(v))
		case *ssa.Function:
			sig := l.lowerType(v.Type())
			return b.Const(sig, constant.MakeString(v.String()))
		case *ssa.Builtin:
			return b.Const(l.lowerType(v.Type()), constant.MakeString("builtin."+v.Name()))
		default:
			return b.Const(l.lowerType(v.Type()), nil)
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

	for _, gb := range reachable {
		lb := blkMap[gb]
		b.SetBlock(lb)
		for _, ins := range gb.Instrs {
			if _, isPhi := ins.(*ssa.Phi); isPhi {
				continue
			}
			b.SetPos(ins.Pos())
			switch ins := ins.(type) {
			case *ssa.Alloc:
				var elem lltypes.Type
				if ptr, ok := ins.Type().Underlying().(*gotypes.Pointer); ok {
					elem = l.lowerType(ptr.Elem())
				} else {
					elem = l.lowerType(ins.Type())
				}
				valMap[ins] = b.Alloc(elem, ins.Heap, !elem.HasPtrData())

			case *ssa.BinOp:
				valMap[ins] = b.BinOp(ins.Op, l.lowerType(ins.Type()), resolveVal(ins.X), resolveVal(ins.Y))

			case *ssa.UnOp:
				if ins.Op == token.MUL {
					valMap[ins] = b.Load(resolveVal(ins.X), l.lowerType(ins.Type()))
				} else {
					valMap[ins] = b.UnOp(ins.Op, l.lowerType(ins.Type()), resolveVal(ins.X))
				}

			case *ssa.Store:
				b.Store(resolveVal(ins.Addr), resolveVal(ins.Val))

			case *ssa.Convert:
				valMap[ins] = b.Convert(l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.ChangeType:
				valMap[ins] = b.Convert(l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.ChangeInterface:
				valMap[ins] = b.Convert(l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.MakeInterface:
				valMap[ins] = b.Convert(l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.SliceToArrayPointer:
				valMap[ins] = b.Convert(l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.MultiConvert:
				valMap[ins] = b.Convert(l.lowerType(ins.Type()), resolveVal(ins.X))

			case *ssa.FieldAddr:
				valMap[ins] = b.FieldAddr(resolveVal(ins.X), ins.Field, l.lowerType(ins.Type()))
			case *ssa.IndexAddr:
				valMap[ins] = b.IndexAddr(resolveVal(ins.X), resolveVal(ins.Index), l.lowerType(ins.Type()))

			case *ssa.Extract:
				tupleVal := resolveVal(ins.Tuple)
				if _, ok := tupleVal.Type().(*lltypes.Tuple); ok {
					valMap[ins] = b.Extract(tupleVal, ins.Index)
				} else {
					valMap[ins] = b.CallDirect("runtime.extract", lltypes.Go, l.lowerType(ins.Type()), tupleVal)
				}

			case *ssa.MakeClosure:
				var fnName string
				var sig *lltypes.Signature
				if fn, ok := ins.Fn.(*ssa.Function); ok {
					fnName = fn.String()
					if s, ok := l.lowerType(ins.Type()).Underlying().(*lltypes.Signature); ok {
						sig = s
					}
				}
				var env llssa.Value
				if len(ins.Bindings) == 1 {
					env = resolveVal(ins.Bindings[0])
				} else if len(ins.Bindings) > 1 {
					env = b.CallDirect("runtime.makeenv", lltypes.Go, lltypes.NewPointerWithKind(lltypes.Typ[lltypes.Uint8], lltypes.GoPtr), resolveArgs(ins.Bindings)...)
				}
				valMap[ins] = b.MakeClosure(sig, fnName, env)

			case *ssa.Call:
				valMap[ins] = l.lowerCallCommon(b, &ins.Call, l.lowerType(ins.Type()), resolveVal, resolveArgs)

			case *ssa.Defer:
				callee, args := l.lowerCallOperands(b, &ins.Call, resolveVal, resolveArgs)
				b.Defer(callee, false, args...)

			case *ssa.Go:
				callee, args := l.lowerCallOperands(b, &ins.Call, resolveVal, resolveArgs)
				b.Go(callee, args...)

			case *ssa.RunDefers:
				b.RunDefers()

			case *ssa.DebugRef:
				b.DebugRef(resolveVal(ins.X), nil, ins.IsAddr)

			case *ssa.MakeSlice:
				valMap[ins] = b.CallDirect("runtime.makeslice", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.Len), resolveVal(ins.Cap))
			case *ssa.MakeMap:
				var args []llssa.Value
				if ins.Reserve != nil {
					args = append(args, resolveVal(ins.Reserve))
				}
				valMap[ins] = b.CallDirect("runtime.makemap", lltypes.Go, l.lowerType(ins.Type()), args...)
			case *ssa.MakeChan:
				valMap[ins] = b.CallDirect("runtime.makechan", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.Size))
			case *ssa.Slice:
				args := []llssa.Value{resolveVal(ins.X)}
				if ins.Low != nil {
					args = append(args, resolveVal(ins.Low))
				}
				if ins.High != nil {
					args = append(args, resolveVal(ins.High))
				}
				if ins.Max != nil {
					args = append(args, resolveVal(ins.Max))
				}
				valMap[ins] = b.CallDirect("runtime.slice", lltypes.Go, l.lowerType(ins.Type()), args...)
			case *ssa.Lookup:
				valMap[ins] = b.CallDirect("runtime.lookup", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.X), resolveVal(ins.Index))
			case *ssa.MapUpdate:
				b.CallDirect("runtime.mapassign", lltypes.Go, nil, resolveVal(ins.Map), resolveVal(ins.Key), resolveVal(ins.Value))
			case *ssa.Field:
				valMap[ins] = b.CallDirect("runtime.field", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.Index:
				valMap[ins] = b.CallDirect("runtime.index", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.X), resolveVal(ins.Index))
			case *ssa.TypeAssert:
				valMap[ins] = b.CallDirect("runtime.typeassert", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.Range:
				valMap[ins] = b.CallDirect("runtime.range", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.X))
			case *ssa.Next:
				valMap[ins] = b.CallDirect("runtime.next", lltypes.Go, l.lowerType(ins.Type()), resolveVal(ins.Iter))
			case *ssa.Send:
				b.CallDirect("runtime.chansend", lltypes.Go, nil, resolveVal(ins.Chan), resolveVal(ins.X))
			case *ssa.Select:
				args := make([]llssa.Value, 0, len(ins.States)*2)
				for _, st := range ins.States {
					args = append(args, resolveVal(st.Chan))
					if st.Send != nil {
						args = append(args, resolveVal(st.Send))
					}
				}
				valMap[ins] = b.CallDirect("runtime.select", lltypes.Go, l.lowerType(ins.Type()), args...)

			case *ssa.Jump:
				b.Jump(blkMap[gb.Succs[0]])
			case *ssa.If:
				b.Branch(resolveVal(ins.Cond), blkMap[gb.Succs[0]], blkMap[gb.Succs[1]])
			case *ssa.Return:
				b.Return(resolveArgs(ins.Results)...)
			case *ssa.Panic:
				b.Panic(resolveVal(ins.X))
			}
		}
		if lb.Terminator() == nil {
			b.Unreachable()
		}
	}

	// Pass 3: wire Phi incoming edges matching the exact unique CFG predecessors
	// of each reachable block.
	cfgPreds := make(map[*llssa.BasicBlock]map[*llssa.BasicBlock]bool, len(llFn.Blocks))
	for _, lb := range llFn.Blocks {
		if term := lb.Terminator(); term != nil {
			for _, succ := range term.Successors() {
				if cfgPreds[succ] == nil {
					cfgPreds[succ] = make(map[*llssa.BasicBlock]bool)
				}
				cfgPreds[succ][lb] = true
			}
		}
	}
	for _, pp := range phis {
		lb := blkMap[pp.goBlk]
		preds := cfgPreds[lb]
		seenPred := make(map[*llssa.BasicBlock]bool, len(preds))
		for i, gPred := range pp.goBlk.Preds {
			lPred, ok := blkMap[gPred]
			if !ok || !preds[lPred] || seenPred[lPred] {
				continue
			}
			seenPred[lPred] = true
			var incoming llssa.Value
			if i < len(pp.goPhi.Edges) {
				incoming = resolveVal(pp.goPhi.Edges[i])
			} else {
				incoming = b.Const(pp.llPhi.Type(), nil)
			}
			pp.llPhi.Edges = append(pp.llPhi.Edges, llssa.PhiEdge{Pred: lPred, Val: incoming})
		}
		for lPred := range preds {
			if !seenPred[lPred] {
				pp.llPhi.Edges = append(pp.llPhi.Edges, llssa.PhiEdge{Pred: lPred, Val: b.Const(pp.llPhi.Type(), nil)})
			}
		}
	}
}

func (l *ssaLowerer) lowerCallCommon(
	b *llssa.Builder,
	call *ssa.CallCommon,
	retType lltypes.Type,
	resolveVal func(ssa.Value) llssa.Value,
	resolveArgs func([]ssa.Value) []llssa.Value,
) llssa.Value {
	args := resolveArgs(call.Args)
	if call.IsInvoke() {
		methodName := call.Method.Name()
		methodIdx := 0
		if iface, ok := call.Value.Type().Underlying().(*gotypes.Interface); ok {
			for i := 0; i < iface.NumMethods(); i++ {
				if iface.Method(i).Name() == methodName {
					methodIdx = i
					break
				}
			}
		}
		return b.CallInterface(resolveVal(call.Value), methodName, methodIdx, retType, args...)
	}
	if fn, ok := call.Value.(*ssa.Function); ok {
		return b.CallDirect(fn.String(), lltypes.Go, retType, args...)
	}
	if bi, ok := call.Value.(*ssa.Builtin); ok {
		if bi.Name() == "recover" {
			return b.Recover(retType)
		}
		return b.CallDirect("builtin."+bi.Name(), lltypes.Go, retType, args...)
	}
	callee := resolveVal(call.Value)
	if sig, ok := callee.Type().Underlying().(*lltypes.Signature); ok && sig.IsClosure() {
		return b.CallClosure(callee, retType, args...)
	}
	return b.CallDirect("runtime.call", lltypes.Go, retType, append([]llssa.Value{callee}, args...)...)
}

func (l *ssaLowerer) lowerCallOperands(
	b *llssa.Builder,
	call *ssa.CallCommon,
	resolveVal func(ssa.Value) llssa.Value,
	resolveArgs func([]ssa.Value) []llssa.Value,
) (llssa.Value, []llssa.Value) {
	args := resolveArgs(call.Args)
	if call.IsInvoke() {
		return resolveVal(call.Value), args
	}
	return resolveVal(call.Value), args
}
