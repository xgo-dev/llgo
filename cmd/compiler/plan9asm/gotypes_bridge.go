package plan9asm

import (
	"go/token"
	gotypes "go/types"

	"github.com/xgo-dev/llgo/types"
)

// llToGoBridge converts an llgo/types.Package graph into stdlib go/types for
// the external github.com/xgo-dev/plan9asm boundary.
type llToGoBridge struct {
	pkgs    map[*types.Package]*gotypes.Package
	types   map[types.Type]gotypes.Type
	objs    map[types.Object]gotypes.Object
	tparams map[*types.TypeParam]*gotypes.TypeParam
}

func newLLToGoBridge() *llToGoBridge {
	return &llToGoBridge{
		pkgs:    make(map[*types.Package]*gotypes.Package),
		types:   make(map[types.Type]gotypes.Type),
		objs:    make(map[types.Object]gotypes.Object),
		tparams: make(map[*types.TypeParam]*gotypes.TypeParam),
	}
}

func (b *llToGoBridge) pkg(p *types.Package) *gotypes.Package {
	if p == nil {
		return nil
	}
	if p == types.Unsafe {
		return gotypes.Unsafe
	}
	if gp, ok := b.pkgs[p]; ok {
		return gp
	}
	gp := gotypes.NewPackage(p.Path(), p.Name())
	b.pkgs[p] = gp
	if imps := p.Imports(); len(imps) > 0 {
		gimps := make([]*gotypes.Package, len(imps))
		for i, imp := range imps {
			gimps[i] = b.pkg(imp)
		}
		gp.SetImports(gimps)
	}
	if sc := p.Scope(); sc != nil {
		names := sc.Names()
		for _, name := range names {
			if obj := sc.Lookup(name); obj != nil {
				if _, ok := obj.(*types.TypeName); ok {
					if goObj := b.obj(obj); goObj != nil {
						gp.Scope().Insert(goObj)
					}
				}
			}
		}
		for _, name := range names {
			if obj := sc.Lookup(name); obj != nil {
				if _, ok := obj.(*types.TypeName); ok {
					continue
				}
				if goObj := b.obj(obj); goObj != nil {
					gp.Scope().Insert(goObj)
				}
			}
		}
	}
	if p.Complete() {
		gp.MarkComplete()
	}
	return gp
}

func (b *llToGoBridge) obj(o types.Object) gotypes.Object {
	if o == nil {
		return nil
	}
	if goObj, ok := b.objs[o]; ok {
		return goObj
	}
	if o.Parent() == types.Universe {
		if uo := gotypes.Universe.Lookup(o.Name()); uo != nil {
			b.objs[o] = uo
			return uo
		}
	}
	if o.Pkg() == types.Unsafe {
		if uo := gotypes.Unsafe.Scope().Lookup(o.Name()); uo != nil {
			b.objs[o] = uo
			return uo
		}
	}
	switch v := o.(type) {
	case *types.TypeName:
		if v.IsAlias() {
			if a, ok := v.Type().(*types.Alias); ok {
				ga := b.typ(a)
				if goObj, ok := b.objs[v]; ok {
					return goObj
				}
				tn := gotypes.NewTypeName(v.Pos(), b.pkg(v.Pkg()), v.Name(), ga)
				b.objs[v] = tn
				return tn
			}
			tn := gotypes.NewTypeName(v.Pos(), b.pkg(v.Pkg()), v.Name(), b.typ(v.Type()))
			b.objs[v] = tn
			return tn
		}
		if n, ok := v.Type().(*types.Named); ok {
			_ = b.typ(n)
			if goObj, ok := b.objs[v]; ok {
				return goObj
			}
		}
		tn := gotypes.NewTypeName(v.Pos(), b.pkg(v.Pkg()), v.Name(), nil)
		b.objs[v] = tn
		return tn
	case *types.Func:
		sig, _ := b.typ(v.Type()).(*gotypes.Signature)
		fn := gotypes.NewFunc(v.Pos(), b.pkg(v.Pkg()), v.Name(), sig)
		b.objs[v] = fn
		return fn
	case *types.Var:
		var gv *gotypes.Var
		if v.IsField() {
			gv = gotypes.NewField(v.Pos(), b.pkg(v.Pkg()), v.Name(), b.typ(v.Type()), v.Embedded())
		} else {
			gv = gotypes.NewVar(v.Pos(), b.pkg(v.Pkg()), v.Name(), b.typ(v.Type()))
		}
		b.objs[v] = gv
		return gv
	case *types.Const:
		gc := gotypes.NewConst(v.Pos(), b.pkg(v.Pkg()), v.Name(), b.typ(v.Type()), v.Val())
		b.objs[v] = gc
		return gc
	}
	return nil
}

func (b *llToGoBridge) tparam(tp *types.TypeParam) *gotypes.TypeParam {
	if tp == nil {
		return nil
	}
	if gtp, ok := b.tparams[tp]; ok {
		return gtp
	}
	tn := gotypes.NewTypeName(token.NoPos, nil, "T", nil)
	if tp.Obj() != nil {
		tn = gotypes.NewTypeName(tp.Obj().Pos(), b.pkg(tp.Obj().Pkg()), tp.Obj().Name(), nil)
		b.objs[tp.Obj()] = tn
	}
	gtp := gotypes.NewTypeParam(tn, nil)
	b.tparams[tp] = gtp
	b.types[tp] = gtp
	if tp.Constraint() != nil {
		gtp.SetConstraint(b.typ(tp.Constraint()))
	}
	return gtp
}

func (b *llToGoBridge) tuple(t *types.Tuple) *gotypes.Tuple {
	if t == nil || t.Len() == 0 {
		return nil
	}
	vars := make([]*gotypes.Var, t.Len())
	for i := 0; i < t.Len(); i++ {
		vars[i] = b.obj(t.At(i)).(*gotypes.Var)
	}
	return gotypes.NewTuple(vars...)
}

func (b *llToGoBridge) typ(t types.Type) gotypes.Type {
	if t == nil {
		return nil
	}
	if gt, ok := b.types[t]; ok {
		return gt
	}
	switch u := t.(type) {
	case *types.Basic:
		if u.Kind() >= 0 && int(u.Kind()) < len(gotypes.Typ) {
			return gotypes.Typ[u.Kind()]
		}
		return gotypes.Typ[gotypes.Invalid]
	case *types.Pointer:
		gt := gotypes.NewPointer(b.typ(u.Elem()))
		b.types[u] = gt
		return gt
	case *types.Slice:
		gt := gotypes.NewSlice(b.typ(u.Elem()))
		b.types[u] = gt
		return gt
	case *types.Array:
		gt := gotypes.NewArray(b.typ(u.Elem()), u.Len())
		b.types[u] = gt
		return gt
	case *types.SIMDVec:
		gt := gotypes.NewArray(b.typ(u.Elem()), int64(u.Len()))
		b.types[u] = gt
		return gt
	case *types.Map:
		gt := gotypes.NewMap(b.typ(u.Key()), b.typ(u.Elem()))
		b.types[u] = gt
		return gt
	case *types.Chan:
		gt := gotypes.NewChan(u.Dir(), b.typ(u.Elem()))
		b.types[u] = gt
		return gt
	case *types.Tuple:
		return b.tuple(u)
	case *types.Signature:
		var recv *gotypes.Var
		if u.Recv() != nil {
			recv = b.obj(u.Recv()).(*gotypes.Var)
		}
		var rtps, tps []*gotypes.TypeParam
		for i := 0; i < u.RecvTypeParams().Len(); i++ {
			rtps = append(rtps, b.tparam(u.RecvTypeParams().At(i)))
		}
		for i := 0; i < u.TypeParams().Len(); i++ {
			tps = append(tps, b.tparam(u.TypeParams().At(i)))
		}
		gt := gotypes.NewSignatureType(recv, rtps, tps, b.tuple(u.Params()), b.tuple(u.Results()), u.Variadic())
		b.types[u] = gt
		return gt
	case *types.Struct:
		fields := make([]*gotypes.Var, u.NumFields())
		tags := make([]string, u.NumFields())
		for i := 0; i < u.NumFields(); i++ {
			fields[i] = b.obj(u.Field(i)).(*gotypes.Var)
			tags[i] = u.Tag(i)
		}
		gt := gotypes.NewStruct(fields, tags)
		b.types[u] = gt
		return gt
	case *types.Interface:
		methods := make([]*gotypes.Func, u.NumExplicitMethods())
		for i := 0; i < u.NumExplicitMethods(); i++ {
			m := u.ExplicitMethod(i)
			if m == nil {
				continue
			}
			if goObj, ok := b.objs[m]; ok {
				methods[i] = goObj.(*gotypes.Func)
				continue
			}
			var gsig *gotypes.Signature
			if sig, ok := m.Type().(*types.Signature); ok && sig != nil {
				gsig = gotypes.NewSignatureType(nil, nil, nil, b.tuple(sig.Params()), b.tuple(sig.Results()), sig.Variadic())
			}
			fn := gotypes.NewFunc(m.Pos(), b.pkg(m.Pkg()), m.Name(), gsig)
			b.objs[m] = fn
			methods[i] = fn
		}
		embs := make([]gotypes.Type, u.NumEmbeddeds())
		for i := 0; i < u.NumEmbeddeds(); i++ {
			embs[i] = b.typ(u.EmbeddedType(i))
		}
		gt := gotypes.NewInterfaceType(methods, embs)
		gt.Complete()
		b.types[u] = gt
		return gt
	case *types.Union:
		terms := make([]*gotypes.Term, u.Len())
		for i := 0; i < u.Len(); i++ {
			terms[i] = gotypes.NewTerm(u.Term(i).Tilde(), b.typ(u.Term(i).Type()))
		}
		gt := gotypes.NewUnion(terms)
		b.types[u] = gt
		return gt
	case *types.TypeParam:
		return b.tparam(u)
	case *types.Named:
		if u.Obj() != nil && u.Obj().Parent() == types.Universe {
			if uo := gotypes.Universe.Lookup(u.Obj().Name()); uo != nil && uo.Type() != nil {
				b.types[u] = uo.Type()
				return uo.Type()
			}
		}
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			gOrig := b.typ(orig)
			gArgs := make([]gotypes.Type, u.TypeArgs().Len())
			for i := 0; i < u.TypeArgs().Len(); i++ {
				gArgs[i] = b.typ(u.TypeArgs().At(i))
			}
			if inst, err := gotypes.Instantiate(nil, gOrig, gArgs, false); err == nil {
				b.types[u] = inst
				return inst
			}
		}
		tn := gotypes.NewTypeName(token.NoPos, nil, "", nil)
		if u.Obj() != nil {
			tn = gotypes.NewTypeName(u.Obj().Pos(), b.pkg(u.Obj().Pkg()), u.Obj().Name(), nil)
			b.objs[u.Obj()] = tn
		}
		gn := gotypes.NewNamed(tn, nil, nil)
		b.types[u] = gn
		if u.TypeParams().Len() > 0 {
			list := make([]*gotypes.TypeParam, u.TypeParams().Len())
			for i := 0; i < u.TypeParams().Len(); i++ {
				list[i] = b.tparam(u.TypeParams().At(i))
			}
			gn.SetTypeParams(list)
		}
		if u.Underlying() != nil {
			gn.SetUnderlying(b.typ(u.Underlying()))
		}
		for i := 0; i < u.NumMethods(); i++ {
			gn.AddMethod(b.obj(u.Method(i)).(*gotypes.Func))
		}
		return gn
	case *types.Alias:
		if u.Obj() != nil && u.Obj().Parent() == types.Universe {
			if uo := gotypes.Universe.Lookup(u.Obj().Name()); uo != nil && uo.Type() != nil {
				b.types[u] = uo.Type()
				return uo.Type()
			}
		}
		tn := gotypes.NewTypeName(token.NoPos, nil, "", nil)
		if u.Obj() != nil {
			tn = gotypes.NewTypeName(u.Obj().Pos(), b.pkg(u.Obj().Pkg()), u.Obj().Name(), nil)
			b.objs[u.Obj()] = tn
		}
		ga := gotypes.NewAlias(tn, b.typ(u.Rhs()))
		if u.TypeParams().Len() > 0 {
			list := make([]*gotypes.TypeParam, u.TypeParams().Len())
			for i := 0; i < u.TypeParams().Len(); i++ {
				list[i] = b.tparam(u.TypeParams().At(i))
			}
			ga.SetTypeParams(list)
		}
		b.types[u] = ga
		return ga
	}
	return gotypes.Typ[gotypes.Invalid]
}

func toGoSizes(s types.Sizes, goarch string) gotypes.Sizes {
	if s == nil {
		return gotypes.SizesFor("gc", goarch)
	}
	if ctx, ok := s.(*types.Context); ok {
		return ctx.GoSizes()
	}
	if std, ok := s.(*types.StdSizes); ok {
		return &gotypes.StdSizes{WordSize: std.WordSize, MaxAlign: std.MaxAlign}
	}
	return &gotypes.StdSizes{
		WordSize: s.Sizeof(types.Typ[types.Int]),
		MaxAlign: s.Alignof(types.Typ[types.Complex128]),
	}
}
