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
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types"
)

// Error describes a type-checking error (mirroring go/types.Error).
type Error struct {
	Fset *token.FileSet
	Pos  token.Pos
	Msg  string
	Soft bool
}

func (err Error) Error() string {
	if err.Fset != nil && err.Pos.IsValid() {
		return fmt.Sprintf("%s: %s", err.Fset.Position(err.Pos), err.Msg)
	}
	return err.Msg
}

// ArgumentError holds an error associated with a type argument index
// (mirroring go/types.ArgumentError).
type ArgumentError struct {
	Index int
	Err   error
}

func (e *ArgumentError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ArgumentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func wrapGoError(err error) error {
	if err == nil {
		return nil
	}
	var ge gotypes.Error
	if errors.As(err, &ge) {
		return Error{
			Fset: ge.Fset,
			Pos:  ge.Pos,
			Msg:  ge.Msg,
			Soft: ge.Soft,
		}
	}
	var ae *gotypes.ArgumentError
	if errors.As(err, &ae) {
		return &ArgumentError{
			Index: ae.Index,
			Err:   wrapGoError(ae.Err),
		}
	}
	return err
}

// Importer resolves import paths to Packages (mirroring go/types.Importer).
type Importer interface {
	Import(path string) (*Package, error)
}

// ImporterFrom resolves import paths relative to a directory
// (mirroring go/types.ImporterFrom).
type ImporterFrom interface {
	Importer
	ImportFrom(path, dir string, mode ImportMode) (*Package, error)
}

// Config specifies the configuration for type checking (mirroring go/types.Config).
type Config struct {
	Context                  *Context
	GoVersion                string
	IgnoreFuncBodies         bool
	FakeImportC              bool
	Error                    func(err error)
	Importer                 Importer
	Sizes                    Sizes
	DisableUnusedImportCheck bool
}

// Check type-checks a package and returns the resulting package object and the
// first error if any (mirroring go/types.Config.Check).
func (conf *Config) Check(path string, fset *token.FileSet, files []*ast.File, info *Info) (*Package, error) {
	pkg := NewPackage(path, "")
	err := NewChecker(conf, fset, pkg, info).Files(files)
	return pkg, err
}

// Checker maintains the state of the type checker (mirroring go/types.Checker).
type Checker struct {
	*Info

	ctx       *Context
	pkg       *Package
	goPkg     *gotypes.Package
	goInfo    *gotypes.Info
	goChecker *gotypes.Checker
	goVersion string
}

type llgoContextProvider interface {
	LLGoContext() *Context
}

// NewChecker returns a new Checker instance for a given package
// (mirroring go/types.NewChecker).
func NewChecker(conf *Config, fset *token.FileSet, pkg *Package, info *Info) *Checker {
	var cfg Config
	if conf != nil {
		cfg = *conf
	}
	ctx := cfg.Context
	if ctx == nil && cfg.Sizes != nil {
		switch s := cfg.Sizes.(type) {
		case *Context:
			ctx = s
		case *archSizes:
			ctx = NewContextWithConfig(ContextConfig{
				Spec:             s.spec,
				HostBasicLayouts: s.hostBasicLayouts,
				SIMDAlignments:   s.simdAlignments,
			})
		case llgoContextProvider:
			ctx = s.LLGoContext()
		}
	}
	if ctx == nil {
		ctx = NewContext()
	}
	if pkg == nil {
		pkg = NewPackage("", "")
	}
	goPkg := pkg.goPkg
	if goPkg == nil {
		goPkg = gotypes.NewPackage(pkg.path, pkg.name)
	}

	goConf := &gotypes.Config{
		Context:                  ctx.ensureGoContext(),
		GoVersion:                cfg.GoVersion,
		IgnoreFuncBodies:         cfg.IgnoreFuncBodies,
		FakeImportC:              cfg.FakeImportC,
		DisableUnusedImportCheck: cfg.DisableUnusedImportCheck,
	}
	if cfg.Error != nil {
		errFn := cfg.Error
		goConf.Error = func(e error) {
			errFn(wrapGoError(e))
		}
	}
	if cfg.Importer != nil {
		goConf.Importer = &goImporterAdapter{ctx: ctx, imp: cfg.Importer}
	}
	goConf.Sizes = bridgeToGoSizes(ctx, cfg.Sizes)

	goInfo := makeMirrorGoInfo(info)
	return &Checker{
		Info:      info,
		ctx:       ctx,
		pkg:       pkg,
		goPkg:     goPkg,
		goInfo:    goInfo,
		goChecker: gotypes.NewChecker(goConf, fset, goPkg, goInfo),
		goVersion: cfg.GoVersion,
	}
}

// Files checks the provided files as part of the checker's package
// (mirroring go/types.Checker.Files).
func (check *Checker) Files(files []*ast.File) error {
	if check == nil || check.goChecker == nil {
		return nil
	}
	check.ctx.registerCheckedGoPackage(check.pkg.path, check.goPkg)
	check.ctx.prescanTypeDirectives(check.pkg.path, files)
	err := check.goChecker.Files(files)
	check.ctx.populateFromGoPackage(check.pkg, check.goPkg, check.Info, check.goInfo, check.goVersion, files)
	return wrapGoError(err)
}

func makeMirrorGoInfo(info *Info) *gotypes.Info {
	if info == nil {
		return nil
	}
	goInfo := &gotypes.Info{}
	if info.Types != nil {
		goInfo.Types = make(map[ast.Expr]gotypes.TypeAndValue)
	}
	if info.Instances != nil {
		goInfo.Instances = make(map[*ast.Ident]gotypes.Instance)
	}
	if info.Defs != nil {
		goInfo.Defs = make(map[*ast.Ident]gotypes.Object)
	}
	if info.Uses != nil {
		goInfo.Uses = make(map[*ast.Ident]gotypes.Object)
	}
	if info.Implicits != nil {
		goInfo.Implicits = make(map[ast.Node]gotypes.Object)
	}
	if info.Selections != nil {
		goInfo.Selections = make(map[*ast.SelectorExpr]*gotypes.Selection)
	}
	if info.Scopes != nil {
		goInfo.Scopes = make(map[ast.Node]*gotypes.Scope)
	}
	if info.FileVersions != nil {
		initGoInfoFileVersions(goInfo)
	}
	return goInfo
}

func bridgeToGoSizes(ctx *Context, s Sizes) gotypes.Sizes {
	if s == nil {
		return ctx.GoSizes()
	}
	switch v := s.(type) {
	case *Context:
		return v.GoSizes()
	case *archSizes:
		if ctx != nil && ctx.spec.GOARCH == v.spec.GOARCH &&
			ctx.spec.WordSize == v.spec.WordSize && ctx.spec.MaxAlign == v.spec.MaxAlign {
			return ctx.GoSizes()
		}
		return NewContextWithConfig(ContextConfig{
			Spec:             v.spec,
			HostBasicLayouts: v.hostBasicLayouts,
			SIMDAlignments:   v.simdAlignments,
		}).GoSizes()
	case llgoContextProvider:
		if c := v.LLGoContext(); c != nil {
			return c.GoSizes()
		}
		return &goSizesBridge{ctx: ctx, sizes: s}
	case *StdSizes:
		return &gotypes.StdSizes{WordSize: v.WordSize, MaxAlign: v.MaxAlign}
	default:
		return &goSizesBridge{ctx: ctx, sizes: s}
	}
}

type goSizesBridge struct {
	ctx   *Context
	sizes Sizes
}

func (b *goSizesBridge) Alignof(T gotypes.Type) int64 {
	return b.sizes.Alignof(b.ctx.FromGo(T))
}

func (b *goSizesBridge) Sizeof(T gotypes.Type) int64 {
	return b.sizes.Sizeof(b.ctx.FromGo(T))
}

func (b *goSizesBridge) Offsetsof(fields []*gotypes.Var) []int64 {
	llFields := make([]*Var, len(fields))
	for i, f := range fields {
		if v, ok := b.ctx.FromGoObject(f).(*Var); ok {
			llFields[i] = v
		}
	}
	return b.sizes.Offsetsof(llFields)
}

type goImporterAdapter struct {
	ctx *Context
	imp Importer
}

func (a *goImporterAdapter) Import(path string) (*gotypes.Package, error) {
	return a.ImportFrom(path, "", 0)
}

func (a *goImporterAdapter) ImportFrom(path, dir string, mode gotypes.ImportMode) (*gotypes.Package, error) {
	var (
		pkg *Package
		err error
	)
	if impFrom, ok := a.imp.(ImporterFrom); ok {
		pkg, err = impFrom.ImportFrom(path, dir, ImportMode(mode))
	} else {
		pkg, err = a.imp.Import(path)
	}
	if pkg == nil {
		return nil, err
	}
	if pkg == Unsafe {
		return gotypes.Unsafe, err
	}
	if a.ctx != nil {
		return a.ctx.toGoPackage(pkg), err
	}
	if pkg.goPkg != nil {
		return pkg.goPkg, err
	}
	gp := gotypes.NewPackage(pkg.path, pkg.name)
	if pkg.complete {
		gp.MarkComplete()
	}
	return gp, err
}

func evalGoPackage(pkg *Package) (*Context, *gotypes.Package) {
	if pkg == nil {
		return NewContext(), nil
	}
	if pkg.goPkg != nil {
		ctx := pkg.ctx
		if ctx == nil {
			ctx = NewContext()
		}
		ctx.syncPackageScopeToGo(pkg)
		return ctx, pkg.goPkg
	}
	// Use a fresh Context when bridging a publicly constructed Package so
	// subsequent pkg.Scope().Insert mutations are always observed.
	ctx := NewContext()
	return ctx, ctx.toGoPackage(pkg)
}

// Eval returns the type and, if constant, the value for the expression expr,
// evaluated at position pos of package pkg (mirroring go/types.Eval).
func Eval(fset *token.FileSet, pkg *Package, pos token.Pos, expr string) (_ TypeAndValue, err error) {
	ctx, goPkg := evalGoPackage(pkg)
	gtv, err := gotypes.Eval(fset, goPkg, pos, expr)
	if err != nil {
		return TypeAndValue{}, wrapGoError(err)
	}
	ctx.mu.Lock()
	tv := ctx.fromGoTypeAndValueLocked(gtv)
	ctx.mu.Unlock()
	return tv, nil
}

// CheckExpr type-checks the expression expr as if it had appeared at position
// pos of package pkg (mirroring go/types.CheckExpr).
func CheckExpr(fset *token.FileSet, pkg *Package, pos token.Pos, expr ast.Expr, info *Info) error {
	ctx, goPkg := evalGoPackage(pkg)
	goInfo := makeMirrorGoInfo(info)
	err := gotypes.CheckExpr(fset, goPkg, pos, expr, goInfo)
	if info != nil && goInfo != nil {
		ctx.mu.Lock()
		ctx.populateInfoFromGoLocked(info, goInfo, "", nil)
		ctx.mu.Unlock()
	}
	return wrapGoError(err)
}

// Instantiate instantiates the generic type orig with type arguments targs
// (mirroring go/types.Instantiate).
func Instantiate(ctxt *Context, orig Type, targs []Type, validate bool) (Type, error) {
	if orig == nil {
		if validate {
			return nil, fmt.Errorf("cannot instantiate nil type")
		}
		return nil, nil
	}

	// Fast path: delegate directly to go/types.Instantiate when backed by go/types.
	var ownerCtx *Context
	switch o := orig.(type) {
	case *Named:
		ownerCtx = o.ctx
	case *Alias:
		if o.obj != nil && o.obj.pkg != nil {
			ownerCtx = o.obj.pkg.ctx
		}
	case *Signature:
		ownerCtx = o.ctx
	}
	if ownerCtx == nil {
		for _, a := range targs {
			switch ta := Unalias(a).(type) {
			case *Named:
				if ta.ctx != nil {
					ownerCtx = ta.ctx
				}
			case *Signature:
				if ta.ctx != nil {
					ownerCtx = ta.ctx
				}
			}
			if ownerCtx != nil {
				break
			}
		}
	}
	switch orig.(type) {
	case *Named, *Alias, *Signature:
		if goOrig := orig.goTypeRef(); goOrig != nil {
			goArgs := make([]gotypes.Type, len(targs))
			allGo := true
			for i, a := range targs {
				if a == nil || a.goTypeRef() == nil {
					allGo = false
					break
				}
				goArgs[i] = a.goTypeRef()
			}
			if allGo {
				instCtx := ctxt
				if instCtx == nil {
					instCtx = ownerCtx
				}
				if instCtx == nil {
					instCtx = NewContext()
				}
				res, err := gotypes.Instantiate(instCtx.ensureGoContext(), goOrig, goArgs, validate)
				if err != nil {
					return nil, wrapGoError(err)
				}
				lowerCtx := ownerCtx
				if lowerCtx == nil {
					lowerCtx = instCtx
				}
				return lowerCtx.FromGo(res), nil
			}
		}
	}

	// Synthetic pure-Go instantiation for programmatically constructed generic types.
	return instantiateSynthetic(orig, targs, validate)
}

func instantiateSynthetic(orig Type, targs []Type, validate bool) (Type, error) {
	var tparams *TypeParamList
	switch t := orig.(type) {
	case *Named:
		tparams = t.TypeParams()
	case *Alias:
		tparams = t.TypeParams()
	case *Signature:
		tparams = t.TypeParams()
	default:
		if validate {
			return nil, fmt.Errorf("cannot instantiate non-generic type %s", orig)
		}
		return orig, nil
	}

	n := tparams.Len()
	if validate {
		if n == 0 {
			return nil, fmt.Errorf("cannot instantiate non-generic type %s", orig)
		}
		if len(targs) != n {
			return nil, fmt.Errorf("got %d type arguments but want %d", len(targs), n)
		}
		for i := 0; i < n; i++ {
			tp := tparams.At(i)
			arg := targs[i]
			if tp != nil && tp.constraint != nil {
				if iface, ok := tp.constraint.Underlying().(*Interface); ok {
					if !Satisfies(arg, iface) {
						return nil, &ArgumentError{
							Index: i,
							Err:   fmt.Errorf("%s does not satisfy %s", arg, tp.constraint),
						}
					}
				}
			}
		}
	}

	subst := make(map[*TypeParam]Type, n)
	for i := 0; i < n && i < len(targs); i++ {
		subst[tparams.At(i)] = targs[i]
	}

	switch t := orig.(type) {
	case *Named:
		inst := &Named{
			synthInstOrigin: t,
			synthInstSubst:  subst,
			obj:             t.obj,
			origin:          t.Origin(),
			typeArgs:        newTypeList(targs),
			callConv:        t.callConv,
			hasCallConv:     t.hasCallConv,
			hostLayout:      t.hostLayout,
			nativeLayout:    t.nativeLayout,
			simdTagIndex:    t.simdTagIndex,
			simdVec:         t.simdVec,
		}
		return inst, nil

	case *Alias:
		seen := make(map[Type]Type)
		return &Alias{
			obj:         t.obj,
			origin:      t.Origin(),
			typeArgs:    newTypeList(targs),
			rhs:         substType(t.rhs, subst, seen),
			callConv:    t.callConv,
			hasCallConv: t.hasCallConv,
		}, nil

	case *Signature:
		seen := make(map[Type]Type)
		res := substSignature(t, subst, seen)
		res.typeParams = nil
		return res, nil
	}
	return orig, nil
}

func substType(t Type, subst map[*TypeParam]Type, seen map[Type]Type) Type {
	if t == nil {
		return nil
	}
	if cached, ok := seen[t]; ok {
		return cached
	}
	switch u := t.(type) {
	case *TypeParam:
		if rep, ok := subst[u]; ok {
			return rep
		}
		return u
	case *Basic, *SIMDVec:
		return u
	case *Pointer:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return &Pointer{elem: e, kind: u.kind}
	case *Slice:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return NewSlice(e)
	case *Array:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return &Array{elem: e, length: u.length, nativeStorage: u.nativeStorage}
	case *Map:
		k := substType(u.key, subst, seen)
		e := substType(u.elem, subst, seen)
		if k == u.key && e == u.elem {
			return u
		}
		return NewMap(k, e)
	case *Chan:
		e := substType(u.elem, subst, seen)
		if e == u.elem {
			return u
		}
		return NewChan(u.dir, e)
	case *Tuple:
		return substTuple(u, subst, seen)
	case *Signature:
		return substSignature(u, subst, seen)
	case *Struct:
		fields := make([]*Var, len(u.fields))
		changed := false
		for i, f := range u.fields {
			ft := substType(f.typ, subst, seen)
			if ft != f.typ {
				changed = true
				nf := NewField(f.pos, f.pkg, f.name, ft, f.embedded)
				nf.origin = f.Origin()
				fields[i] = nf
			} else {
				fields[i] = f
			}
		}
		if !changed {
			return u
		}
		return &Struct{
			fields:          fields,
			tags:            u.tags,
			hostLayout:      u.hostLayout,
			nativeLayout:    u.nativeLayout,
			isClosureStruct: u.isClosureStruct,
			simdTagIndex:    u.simdTagIndex,
		}
	case *Interface:
		u.resolve()
		out := &Interface{resolved: true, methodSet: true}
		seen[u] = out
		methods := make([]*Func, len(u.explicitMethods))
		changed := false
		for i, m := range u.explicitMethods {
			if sig, ok := m.typ.(*Signature); ok && sig != nil {
				params := substTuple(sig.params, subst, seen)
				results := substTuple(sig.results, subst, seen)
				if params != sig.params || results != sig.results {
					changed = true
					cp := *sig
					cp.params = params
					cp.results = results
					cp.recv = NewVar(m.pos, m.pkg, "", out)
					nm := NewFunc(m.pos, m.pkg, m.name, &cp)
					nm.origin = m.Origin()
					methods[i] = nm
					continue
				}
			}
			methods[i] = m
		}
		embs := make([]Type, len(u.embeddeds))
		for i, e := range u.embeddeds {
			ne := substType(e, subst, seen)
			if ne != e {
				changed = true
			}
			embs[i] = ne
		}
		if !changed {
			seen[u] = u
			return u
		}
		out.explicitMethods = methods
		out.embeddeds = embs
		return out.Complete()
	case *Named:
		if u.TypeArgs().Len() > 0 {
			args := make([]Type, u.TypeArgs().Len())
			changed := false
			for i := 0; i < u.TypeArgs().Len(); i++ {
				a := u.TypeArgs().At(i)
				na := substType(a, subst, seen)
				if na != a {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if existing, ok := seen[u.Origin()]; ok {
					if en, ok := existing.(*Named); ok && typeListMatches(en.TypeArgs(), args) {
						return en
					}
				}
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		} else if u.TypeParams().Len() > 0 {
			args := make([]Type, u.TypeParams().Len())
			changed := false
			for i := 0; i < u.TypeParams().Len(); i++ {
				tp := u.TypeParams().At(i)
				na := substType(tp, subst, seen)
				if na != tp {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if existing, ok := seen[u.Origin()]; ok {
					if en, ok := existing.(*Named); ok && typeListMatches(en.TypeArgs(), args) {
						return en
					}
				}
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		}
		return u
	case *Alias:
		if u.TypeArgs().Len() > 0 {
			args := make([]Type, u.TypeArgs().Len())
			changed := false
			for i := 0; i < u.TypeArgs().Len(); i++ {
				a := u.TypeArgs().At(i)
				na := substType(a, subst, seen)
				if na != a {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		} else if u.TypeParams().Len() > 0 {
			args := make([]Type, u.TypeParams().Len())
			changed := false
			for i := 0; i < u.TypeParams().Len(); i++ {
				tp := u.TypeParams().At(i)
				na := substType(tp, subst, seen)
				if na != tp {
					changed = true
				}
				args[i] = na
			}
			if changed {
				if inst, err := Instantiate(nil, u.Origin(), args, false); err == nil {
					return inst
				}
			}
		}
		return u
	case *Union:
		terms := make([]*Term, len(u.terms))
		changed := false
		for i, term := range u.terms {
			nt := substType(term.typ, subst, seen)
			if nt != term.typ {
				changed = true
				terms[i] = NewTerm(term.tilde, nt)
			} else {
				terms[i] = term
			}
		}
		if !changed {
			return u
		}
		return NewUnion(terms)
	}
	return t
}

func typeListMatches(tl *TypeList, args []Type) bool {
	if tl.Len() != len(args) {
		return false
	}
	for i, a := range args {
		if !Identical(tl.At(i), a) {
			return false
		}
	}
	return true
}

func substTuple(t *Tuple, subst map[*TypeParam]Type, seen map[Type]Type) *Tuple {
	if t == nil {
		return nil
	}
	vars := make([]*Var, len(t.vars))
	changed := false
	for i, v := range t.vars {
		vt := substType(v.typ, subst, seen)
		if vt != v.typ {
			changed = true
			nv := &Var{
				object:   object{pos: v.pos, pkg: v.pkg, name: v.name, typ: vt},
				origin:   v.Origin(),
				embedded: v.embedded,
				isField:  v.isField,
				kind:     v.kind,
			}
			vars[i] = nv
		} else {
			vars[i] = v
		}
	}
	if !changed {
		return t
	}
	return &Tuple{vars: vars}
}

func substSignature(s *Signature, subst map[*TypeParam]Type, seen map[Type]Type) *Signature {
	if s == nil {
		return nil
	}
	cp := *s
	if s.recv != nil {
		rt := substType(s.recv.typ, subst, seen)
		if rt != s.recv.typ {
			cp.recv = &Var{
				object: object{pos: s.recv.pos, pkg: s.recv.pkg, name: s.recv.name, typ: rt},
				origin: s.recv.Origin(),
				kind:   RecvVar,
			}
		}
	}
	cp.params = substTuple(s.params, subst, seen)
	cp.results = substTuple(s.results, subst, seen)
	return &cp
}
