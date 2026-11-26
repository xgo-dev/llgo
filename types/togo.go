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

package types

import (
	"go/token"
	gotypes "go/types"
)

// toGoPackage converts or binds an LLGo *Package into a go/types.Package
// registered in c so that go/types.Checker sees its full declaration graph and
// every resulting go/types object/type lowers back to the exact same LLGo
// identity in c.
func (c *Context) toGoPackage(pkg *Package) *gotypes.Package {
	if pkg == nil {
		return nil
	}
	if pkg == Unsafe {
		return gotypes.Unsafe
	}
	goCtx := c.ensureGoContext()
	c.mu.Lock()
	defer c.mu.Unlock()
	if pkg.goPkg != nil {
		if pkg.ctx == c {
			return pkg.goPkg
		}
		if pkg.ctx != nil {
			c.absorbContextLocked(pkg.ctx)
		}
		c.goPkgs[pkg.goPkg] = pkg
		if gs := pkg.goPkg.Scope(); gs != nil && pkg.scope != nil {
			c.goScopes[gs] = pkg.scope
		}
		return pkg.goPkg
	}
	if gp, ok := c.llToGoPkgs[pkg]; ok {
		return gp
	}
	conv := &toGoConverter{
		ctx:     c,
		goCtx:   goCtx,
		pkgs:    make(map[*Package]*gotypes.Package),
		types:   make(map[Type]gotypes.Type),
		objs:    make(map[Object]gotypes.Object),
		tparams: make(map[*TypeParam]*gotypes.TypeParam),
	}
	return conv.pkg(pkg)
}

// syncPackageScopeToGo bridges any objects inserted into pkg.Scope() after
// pkg.goPkg was created so Eval and CheckExpr see post-Check scope mutations.
func (c *Context) syncPackageScopeToGo(pkg *Package) {
	if pkg == nil || pkg.goPkg == nil || pkg.scope == nil || pkg.goPkg.Scope() == nil {
		return
	}
	goScope := pkg.goPkg.Scope()
	var missing []Object
	for _, name := range pkg.scope.Names() {
		if goScope.Lookup(name) == nil {
			if obj := pkg.scope.Lookup(name); obj != nil {
				missing = append(missing, obj)
			}
		}
	}
	if len(missing) == 0 {
		return
	}
	goCtx := c.ensureGoContext()
	c.mu.Lock()
	defer c.mu.Unlock()
	conv := &toGoConverter{
		ctx:     c,
		goCtx:   goCtx,
		pkgs:    map[*Package]*gotypes.Package{pkg: pkg.goPkg},
		types:   make(map[Type]gotypes.Type),
		objs:    make(map[Object]gotypes.Object),
		tparams: make(map[*TypeParam]*gotypes.TypeParam),
	}
	for _, obj := range missing {
		if _, ok := obj.(*TypeName); ok {
			if goObj := conv.obj(obj); goObj != nil {
				goScope.Insert(goObj)
			}
		}
	}
	for _, obj := range missing {
		if _, ok := obj.(*TypeName); ok {
			continue
		}
		if goObj := conv.obj(obj); goObj != nil {
			goScope.Insert(goObj)
		}
	}
}

func (c *Context) absorbContextLocked(src *Context) {
	if src == nil || src == c {
		return
	}
	src.mu.RLock()
	defer src.mu.RUnlock()
	for k, v := range src.goPkgs {
		if _, ok := c.goPkgs[k]; !ok {
			c.goPkgs[k] = v
		}
	}
	for k, v := range src.goScopes {
		if _, ok := c.goScopes[k]; !ok {
			c.goScopes[k] = v
		}
	}
	for k, v := range src.goObjs {
		if _, ok := c.goObjs[k]; !ok {
			c.goObjs[k] = v
		}
	}
	for k, v := range src.goTypes {
		if _, ok := c.goTypes[k]; !ok {
			c.goTypes[k] = v
		}
	}
	for k, v := range src.goNamed {
		if _, ok := c.goNamed[k]; !ok {
			c.goNamed[k] = v
		}
	}
	for k, v := range src.goAliases {
		if _, ok := c.goAliases[k]; !ok {
			c.goAliases[k] = v
		}
	}
	for k, v := range src.goTParams {
		if _, ok := c.goTParams[k]; !ok {
			c.goTParams[k] = v
		}
	}
}

type toGoConverter struct {
	ctx     *Context
	goCtx   *gotypes.Context
	pkgs    map[*Package]*gotypes.Package
	types   map[Type]gotypes.Type
	objs    map[Object]gotypes.Object
	tparams map[*TypeParam]*gotypes.TypeParam
}

func (g *toGoConverter) pkg(p *Package) *gotypes.Package {
	if p == nil {
		return nil
	}
	if p == Unsafe {
		return gotypes.Unsafe
	}
	if gp, ok := g.pkgs[p]; ok {
		return gp
	}
	if gp, ok := g.ctx.llToGoPkgs[p]; ok {
		g.pkgs[p] = gp
		return gp
	}
	if p.goPkg != nil {
		if p.ctx != nil && p.ctx != g.ctx {
			g.ctx.absorbContextLocked(p.ctx)
		}
		g.pkgs[p] = p.goPkg
		g.ctx.goPkgs[p.goPkg] = p
		if gs := p.goPkg.Scope(); gs != nil && p.scope != nil {
			g.ctx.goScopes[gs] = p.scope
		}
		return p.goPkg
	}
	gp := gotypes.NewPackage(p.path, p.name)
	g.pkgs[p] = gp
	if g.ctx.llToGoPkgs == nil {
		g.ctx.llToGoPkgs = make(map[*Package]*gotypes.Package)
	}
	g.ctx.llToGoPkgs[p] = gp
	g.ctx.goPkgs[gp] = p
	if p.scope != nil && gp.Scope() != nil {
		g.ctx.goScopes[gp.Scope()] = p.scope
	}

	if len(p.imports) > 0 {
		imps := make([]*gotypes.Package, len(p.imports))
		for i, imp := range p.imports {
			imps[i] = g.pkg(imp)
		}
		gp.SetImports(imps)
	}

	if p.scope != nil {
		// Materialize TypeNames first so method/type references see complete shells.
		names := p.scope.Names()
		for _, name := range names {
			obj := p.scope.Lookup(name)
			if _, ok := obj.(*TypeName); ok {
				if goObj := g.obj(obj); goObj != nil {
					gp.Scope().Insert(goObj)
				}
			}
		}
		for _, name := range names {
			obj := p.scope.Lookup(name)
			if _, ok := obj.(*TypeName); ok {
				continue
			}
			if goObj := g.obj(obj); goObj != nil {
				gp.Scope().Insert(goObj)
			}
		}
	}

	if p.complete {
		gp.MarkComplete()
	}
	return gp
}

func (g *toGoConverter) bindType(ll Type, gt gotypes.Type) {
	if ll == nil || gt == nil {
		return
	}
	g.types[ll] = gt
	g.ctx.goTypes[goTypeCacheKey{typ: gt, cc: Go, hasOverride: false}] = ll
	if n, ok := ll.(*Named); ok {
		if gn, ok := gt.(*gotypes.Named); ok {
			g.ctx.goNamed[gn] = n
		}
	}
	if a, ok := ll.(*Alias); ok {
		g.ctx.goAliases[gt] = a
	}
	if tp, ok := ll.(*TypeParam); ok {
		if gtp, ok := gt.(*gotypes.TypeParam); ok {
			g.ctx.goTParams[gtp] = tp
		}
	}
}

func (g *toGoConverter) bindObj(ll Object, goObj gotypes.Object) {
	if ll == nil || goObj == nil {
		return
	}
	g.objs[ll] = goObj
	g.ctx.goObjs[goObj] = ll
}

func (g *toGoConverter) obj(o Object) gotypes.Object {
	if o == nil {
		return nil
	}
	if goObj, ok := g.objs[o]; ok {
		return goObj
	}
	if o.Parent() == Universe {
		if uo := gotypes.Universe.Lookup(o.Name()); uo != nil {
			g.bindObj(o, uo)
			return uo
		}
	}
	if o.Pkg() == Unsafe {
		if uo := gotypes.Unsafe.Scope().Lookup(o.Name()); uo != nil {
			g.bindObj(o, uo)
			return uo
		}
	}

	switch v := o.(type) {
	case *TypeName:
		if v.goObj != nil {
			g.bindObj(v, v.goObj)
			return v.goObj
		}
		if v.isAlias {
			if a, ok := v.typ.(*Alias); ok {
				ga := g.typ(a)
				if goObj, ok := g.objs[v]; ok {
					return goObj
				}
				tn := gotypes.NewTypeName(v.pos, g.pkg(v.pkg), v.name, ga)
				g.bindObj(v, tn)
				return tn
			}
			tn := gotypes.NewTypeName(v.pos, g.pkg(v.pkg), v.name, g.typ(v.typ))
			g.bindObj(v, tn)
			return tn
		}
		if n, ok := v.typ.(*Named); ok {
			_ = g.typ(n)
			if goObj, ok := g.objs[v]; ok {
				return goObj
			}
		}
		tn := gotypes.NewTypeName(v.pos, g.pkg(v.pkg), v.name, nil)
		g.bindObj(v, tn)
		if v.typ != nil {
			_ = g.typ(v.typ)
		}
		return tn

	case *Func:
		if v.goObj != nil {
			g.bindObj(v, v.goObj)
			return v.goObj
		}
		sig, _ := g.typ(v.typ).(*gotypes.Signature)
		fn := gotypes.NewFunc(v.pos, g.pkg(v.pkg), v.name, sig)
		g.bindObj(v, fn)
		return fn

	case *Var:
		if v.goObj != nil {
			g.bindObj(v, v.goObj)
			return v.goObj
		}
		var gv *gotypes.Var
		if v.isField {
			gv = gotypes.NewField(v.pos, g.pkg(v.pkg), v.name, g.typ(v.typ), v.embedded)
		} else if v.kind == ParamVar || v.kind == RecvVar || v.kind == ResultVar {
			gv = gotypes.NewParam(v.pos, g.pkg(v.pkg), v.name, g.typ(v.typ))
		} else {
			gv = gotypes.NewVar(v.pos, g.pkg(v.pkg), v.name, g.typ(v.typ))
		}
		g.bindObj(v, gv)
		return gv

	case *Const:
		if v.goObj != nil {
			g.bindObj(v, v.goObj)
			return v.goObj
		}
		gc := gotypes.NewConst(v.pos, g.pkg(v.pkg), v.name, g.typ(v.typ), v.val)
		g.bindObj(v, gc)
		return gc

	case *PkgName:
		gp := gotypes.NewPkgName(v.pos, g.pkg(v.pkg), v.name, g.pkg(v.imported))
		g.bindObj(v, gp)
		return gp

	case *Label:
		gl := gotypes.NewLabel(v.pos, g.pkg(v.pkg), v.name)
		g.bindObj(v, gl)
		return gl
	}
	return nil
}

func (g *toGoConverter) tparam(tp *TypeParam) *gotypes.TypeParam {
	if tp == nil {
		return nil
	}
	if gtp, ok := g.tparams[tp]; ok {
		return gtp
	}
	if gt, ok := tp.goType.(*gotypes.TypeParam); ok && gt != nil {
		g.tparams[tp] = gt
		g.bindType(tp, gt)
		return gt
	}
	var tn *gotypes.TypeName
	if tp.obj != nil {
		tn = gotypes.NewTypeName(tp.obj.pos, g.pkg(tp.obj.pkg), tp.obj.name, nil)
		g.bindObj(tp.obj, tn)
	} else {
		tn = gotypes.NewTypeName(token.NoPos, nil, "T", nil)
	}
	gtp := gotypes.NewTypeParam(tn, nil)
	g.tparams[tp] = gtp
	g.bindType(tp, gtp)
	if tp.constraint != nil {
		gtp.SetConstraint(g.typ(tp.constraint))
	}
	return gtp
}

func (g *toGoConverter) tuple(t *Tuple) *gotypes.Tuple {
	if t == nil || len(t.vars) == 0 {
		return nil
	}
	vars := make([]*gotypes.Var, len(t.vars))
	for i, v := range t.vars {
		vars[i] = g.obj(v).(*gotypes.Var)
	}
	gt := gotypes.NewTuple(vars...)
	g.bindType(t, gt)
	return gt
}

func (g *toGoConverter) typ(t Type) gotypes.Type {
	if t == nil {
		return nil
	}
	if gt, ok := g.types[t]; ok {
		return gt
	}
	if gt := t.goTypeRef(); gt != nil {
		g.bindType(t, gt)
		return gt
	}
	switch u := t.(type) {
	case *Basic:
		if u.kind >= 0 && int(u.kind) < len(gotypes.Typ) {
			return gotypes.Typ[u.kind]
		}
		return gotypes.Typ[gotypes.Invalid]

	case *Pointer:
		gt := gotypes.NewPointer(g.typ(u.elem))
		g.bindType(u, gt)
		return gt

	case *Slice:
		gt := gotypes.NewSlice(g.typ(u.elem))
		g.bindType(u, gt)
		return gt

	case *Array:
		gt := gotypes.NewArray(g.typ(u.elem), u.length)
		g.bindType(u, gt)
		return gt

	case *SIMDVec:
		gt := gotypes.NewArray(g.typ(u.elem), int64(u.length))
		g.bindType(u, gt)
		return gt

	case *Map:
		gt := gotypes.NewMap(g.typ(u.key), g.typ(u.elem))
		g.bindType(u, gt)
		return gt

	case *Chan:
		gt := gotypes.NewChan(u.dir, g.typ(u.elem))
		g.bindType(u, gt)
		return gt

	case *Tuple:
		return g.tuple(u)

	case *Signature:
		var recv *gotypes.Var
		if u.recv != nil {
			recv = g.obj(u.recv).(*gotypes.Var)
		}
		var rtps, tps []*gotypes.TypeParam
		if u.recvTypeParams != nil {
			rtps = make([]*gotypes.TypeParam, u.recvTypeParams.Len())
			for i := 0; i < u.recvTypeParams.Len(); i++ {
				rtps[i] = g.tparam(u.recvTypeParams.At(i))
			}
		}
		if u.typeParams != nil {
			tps = make([]*gotypes.TypeParam, u.typeParams.Len())
			for i := 0; i < u.typeParams.Len(); i++ {
				tps[i] = g.tparam(u.typeParams.At(i))
			}
		}
		gt := gotypes.NewSignatureType(recv, rtps, tps, g.tuple(u.params), g.tuple(u.results), u.variadic)
		g.bindType(u, gt)
		return gt

	case *Struct:
		fields := make([]*gotypes.Var, len(u.fields))
		for i, f := range u.fields {
			fields[i] = g.obj(f).(*gotypes.Var)
		}
		gt := gotypes.NewStruct(fields, u.tags)
		g.bindType(u, gt)
		return gt

	case *Interface:
		if u.ctx != nil && u.ctx != g.ctx {
			u.resolve()
		}
		methods := make([]*gotypes.Func, len(u.explicitMethods))
		for i, m := range u.explicitMethods {
			if m == nil {
				continue
			}
			if goObj, ok := g.objs[m]; ok {
				methods[i] = goObj.(*gotypes.Func)
				continue
			}
			var gsig *gotypes.Signature
			if sig, ok := m.typ.(*Signature); ok && sig != nil {
				gsig = gotypes.NewSignatureType(nil, nil, nil, g.tuple(sig.params), g.tuple(sig.results), sig.variadic)
			}
			fn := gotypes.NewFunc(m.pos, g.pkg(m.pkg), m.name, gsig)
			g.bindObj(m, fn)
			methods[i] = fn
		}
		embs := make([]gotypes.Type, len(u.embeddeds))
		for i, e := range u.embeddeds {
			embs[i] = g.typ(e)
		}
		gt := gotypes.NewInterfaceType(methods, embs)
		gt.Complete()
		g.bindType(u, gt)
		return gt

	case *Union:
		terms := make([]*gotypes.Term, len(u.terms))
		for i, term := range u.terms {
			terms[i] = gotypes.NewTerm(term.tilde, g.typ(term.typ))
		}
		gt := gotypes.NewUnion(terms)
		g.bindType(u, gt)
		return gt

	case *TypeParam:
		return g.tparam(u)

	case *Named:
		if u.obj != nil && u.obj.parent == Universe {
			if uo := gotypes.Universe.Lookup(u.obj.name); uo != nil && uo.Type() != nil {
				g.bindType(u, uo.Type())
				g.bindObj(u.obj, uo)
				return uo.Type()
			}
		}
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			gOrig := g.typ(orig)
			gArgs := make([]gotypes.Type, u.TypeArgs().Len())
			for i := 0; i < u.TypeArgs().Len(); i++ {
				gArgs[i] = g.typ(u.TypeArgs().At(i))
			}
			inst, err := gotypes.Instantiate(g.goCtx, gOrig, gArgs, false)
			if err == nil {
				g.bindType(u, inst)
				return inst
			}
		}
		if u.ctx != nil && u.ctx != g.ctx {
			u.resolve()
			u.resolveMethods()
		}
		var tn *gotypes.TypeName
		if u.obj != nil {
			tn = gotypes.NewTypeName(u.obj.pos, g.pkg(u.obj.pkg), u.obj.name, nil)
			g.bindObj(u.obj, tn)
		} else {
			tn = gotypes.NewTypeName(token.NoPos, nil, "", nil)
		}
		gn := gotypes.NewNamed(tn, nil, nil)
		g.bindType(u, gn)
		if tps := u.TypeParams(); tps != nil && tps.Len() > 0 {
			list := make([]*gotypes.TypeParam, tps.Len())
			for i := 0; i < tps.Len(); i++ {
				list[i] = g.tparam(tps.At(i))
			}
			gn.SetTypeParams(list)
		}
		if u.underlying != nil {
			gn.SetUnderlying(g.typ(u.underlying))
		}
		for _, m := range u.methods {
			gn.AddMethod(g.obj(m).(*gotypes.Func))
		}
		return gn

	case *Alias:
		if u.obj != nil && u.obj.parent == Universe {
			if uo := gotypes.Universe.Lookup(u.obj.name); uo != nil && uo.Type() != nil {
				g.bindType(u, uo.Type())
				g.bindObj(u.obj, uo)
				return uo.Type()
			}
		}
		if orig := u.Origin(); orig != nil && orig != u && u.TypeArgs().Len() > 0 {
			gOrig := g.typ(orig)
			gArgs := make([]gotypes.Type, u.TypeArgs().Len())
			for i := 0; i < u.TypeArgs().Len(); i++ {
				gArgs[i] = g.typ(u.TypeArgs().At(i))
			}
			inst, err := gotypes.Instantiate(g.goCtx, gOrig, gArgs, false)
			if err == nil {
				g.bindType(u, inst)
				return inst
			}
		}
		var tn *gotypes.TypeName
		if u.obj != nil {
			tn = gotypes.NewTypeName(u.obj.pos, g.pkg(u.obj.pkg), u.obj.name, nil)
			g.bindObj(u.obj, tn)
		} else {
			tn = gotypes.NewTypeName(token.NoPos, nil, "", nil)
		}
		rhs := g.typ(u.rhs)
		var tps []*gotypes.TypeParam
		if u.typeParams != nil && u.typeParams.Len() > 0 {
			tps = make([]*gotypes.TypeParam, u.typeParams.Len())
			for i := 0; i < u.typeParams.Len(); i++ {
				tps[i] = g.tparam(u.typeParams.At(i))
			}
		}
		gt := newGoAlias(tn, rhs, tps)
		g.bindType(u, gt)
		return gt
	}
	return gotypes.Typ[gotypes.Invalid]
}
