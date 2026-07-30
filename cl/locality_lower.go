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
	"go/token"
	"go/types"
	"strings"

	"github.com/goplus/llgo/internal/locality"
	localitylayout "github.com/goplus/llgo/internal/locality/layout"
	llssa "github.com/goplus/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// localInitReady is part of the compiler/runtime ABI. Keep it in sync with
// runtime/internal/runtime.localInitReady.
const localInitReady = 2

type localVariable struct {
	planned localitylayout.Variable
	owner   *localPackage
}

type localPackage struct {
	plan      localitylayout.Package
	typ       llssa.Type
	blockFunc llssa.Function
	direct    map[string]llssa.Global
	init      map[locality.Kind]*localInitializer
}

type localInitializer struct {
	guard        llssa.Global
	failureCache llssa.Global
	dispatch     llssa.Function
	ensure       llssa.Function
}

type localBaseCacheKey struct {
	block *ssa.BasicBlock
	owner *localPackage
}

type localEnsureCacheKey struct {
	block *ssa.BasicBlock
	owner *localPackage
	kind  locality.Kind
}

// localityLowering owns all compiler state for TLS/GLS lowering. Only this
// value is embedded in the general compiler context.
type localityLowering struct {
	packages  map[string]*localPackage
	variables map[*ssa.Global]*localVariable
	function  localityFunction
}

type localityFunction struct {
	block          *ssa.BasicBlock
	packageBases   map[localBaseCacheKey]llssa.Expr
	packageEnsures map[localEnsureCacheKey]bool
	entry          *localEntryContext
}

type localEntryContext struct {
	context  llssa.Expr
	previous llssa.Expr
	entered  bool
}

func (p *context) prepareExportedLocalContext(f *ssa.Function) {
	if !p.prog.NeedsLocalContext() || f == nil || f.Pkg == nil {
		return
	}
	fullName := funcNameWithProgram(p.prog, f.Pkg.Pkg, f, false)
	if _, exported := p.pkg.ExportFuncs()[fullName]; !exported {
		return
	}
	p.locality.function.entry = &localEntryContext{}
}

func (p *context) enterExportedLocalContext(b llssa.Builder) {
	entry := p.locality.function.entry
	if entry == nil || entry.entered {
		return
	}
	context, previous := b.EnterLocalContext()
	entry.context = context
	entry.previous = previous
	entry.entered = true
}

func (p *context) leaveExportedLocalContext(b llssa.Builder) {
	entry := p.locality.function.entry
	if entry != nil && entry.entered {
		b.LeaveLocalContext(entry.context, entry.previous)
	}
}

func (p *context) prepareLocalVariables(pkg llssa.Package, globals []*ssa.Global) {
	_, err := p.localPackageFor(p.goTyps, pkg, true)
	if err != nil {
		panic(err)
	}
	if p.locality.variables == nil {
		p.locality.variables = make(map[*ssa.Global]*localVariable)
	}
	for _, global := range globals {
		variable, ok, err := p.localVariableFor(pkg, global, true)
		if err != nil {
			panic(err)
		}
		if !ok {
			continue
		}
		p.locality.variables[global] = variable
	}
}

// localVariableFor validates one Go SSA global and finds its declaring package
// plan. Both definitions and references use this path so forbidden linkname
// aliases and layout validation cannot diverge between lowering cases.
func (p *context) localVariableFor(pkg llssa.Package, global *ssa.Global, defineCurrent bool) (*localVariable, bool, error) {
	fullName := p.prog.FullName(global.Pkg.Pkg, global.Name())
	canonical, info, ok, err := p.prog.ResolveLocality(fullName)
	if err != nil {
		return nil, false, err
	}
	if !ok || info.Locality == locality.None {
		return nil, false, nil
	}
	typesPkg := p.localTypesPackage(canonical)
	if typesPkg == nil {
		return nil, false, fmt.Errorf("missing types package for local variable %s", canonical)
	}
	owner, err := p.localPackageFor(typesPkg, pkg, defineCurrent && typesPkg == p.goTyps)
	if err != nil {
		return nil, false, err
	}
	planned, ok := owner.plan.Lookup(canonical)
	if !ok {
		return nil, false, fmt.Errorf("missing locality layout for %s", canonical)
	}
	return &localVariable{planned: planned, owner: owner}, true, nil
}

func (p *context) localityGlobalStorage(pkg llssa.Package, global *ssa.Global, name string, typ types.Type, bg llssa.Background) (llssa.Global, bool) {
	info, ok := p.resolveLocality(p.prog.FullName(global.Pkg.Pkg, global.Name()))
	if !ok || info.Locality == locality.None {
		return pkg.NewVar(name, typ, bg), false
	}
	variable := p.locality.variables[global]
	if variable == nil {
		panic(fmt.Sprintf("missing locality layout for %s", name))
	}
	if variable.planned.Storage == localitylayout.StoragePackage {
		return nil, true
	}
	return variable.owner.direct[variable.planned.Name], false
}

func (p *context) localityAllowsGlobalDebug(global *ssa.Global) bool {
	variable := p.locality.variables[global]
	return variable == nil || variable.planned.Storage == localitylayout.StorageNativeTLS
}

func (p *context) localTypesPackage(fullName string) *types.Package {
	matches := func(pkg *types.Package) bool {
		if pkg == nil {
			return false
		}
		prefix := p.prog.PathOf(pkg) + "."
		if !strings.HasPrefix(fullName, prefix) {
			return false
		}
		name := strings.TrimPrefix(fullName, prefix)
		_, ok := pkg.Scope().Lookup(name).(*types.Var)
		return ok
	}
	if matches(p.goTyps) {
		return p.goTyps
	}
	if p.goProg != nil {
		for _, pkg := range p.goProg.AllPackages() {
			if pkg != nil && matches(pkg.Pkg) {
				return pkg.Pkg
			}
		}
	}
	for pkg := range p.loaded {
		if matches(pkg) {
			return pkg
		}
	}
	return nil
}

func (p *context) localPackageFor(typesPkg *types.Package, pkg llssa.Package, define bool) (*localPackage, error) {
	if typesPkg == nil {
		return nil, nil
	}
	path := p.prog.PathOf(typesPkg)
	if owner := p.locality.packages[path]; owner != nil {
		return owner, nil
	}
	plan, err := planLocalPackage(p.prog, typesPkg)
	if err != nil {
		return nil, err
	}
	if len(plan.Variables) == 0 {
		return nil, nil
	}
	if p.locality.packages == nil {
		p.locality.packages = make(map[string]*localPackage)
	}
	owner := &localPackage{
		plan:   plan,
		direct: make(map[string]llssa.Global),
		init:   make(map[locality.Kind]*localInitializer),
	}
	p.locality.packages[path] = owner
	p.buildLocalPackage(pkg, owner, define)
	return owner, nil
}

func (p *context) buildLocalPackage(pkg llssa.Package, owner *localPackage, define bool) {
	for _, variable := range owner.plan.Variables {
		if variable.Storage != localitylayout.StorageNativeTLS {
			continue
		}
		typ := types.NewPointer(p.patchType(variable.Type))
		global := pkg.NewThreadLocalVar(variable.Name, typ, llssa.InGo)
		owner.direct[variable.Name] = global
	}
	if len(owner.plan.Block) != 0 {
		fields := make([]*types.Var, len(owner.plan.Block))
		for index, variable := range owner.plan.Block {
			fields[index] = types.NewField(token.NoPos, nil, fmt.Sprintf("v%d", index), p.patchType(variable.Type), false)
		}
		structType := types.NewStruct(fields, nil)
		owner.typ = p.prog.Type(structType, llssa.InGo)
		cache := pkg.NewThreadLocalVar(localitylayout.BlockCacheName(owner.plan.Path), types.NewPointer(types.Typ[types.Uintptr]), llssa.InGo)
		if define {
			cache.InitNil()
		}
		result := types.NewPointer(structType)
		owner.blockFunc = pkg.NewFunc(localitylayout.BlockName(owner.plan.Path), noArgResultSignature(result), llssa.InGo)
		owner.blockFunc.Inline(llssa.AlwaysInline)
		if define && !owner.blockFunc.HasBody() {
			owner.blockFunc.BuildLocalPackageAccessor(
				cache.Expr,
				p.prog.IntVal(p.prog.SizeOf(owner.typ), p.prog.Uintptr()),
				p.prog.IntVal(p.prog.AlignOf(owner.typ), p.prog.Uintptr()),
			)
		}
	}
	for _, kind := range []locality.Kind{locality.Thread, locality.Goroutine} {
		initializers := owner.plan.Initializers(kind)
		if len(initializers) == 0 {
			continue
		}
		owner.init[kind] = p.buildLocalInitializer(pkg, owner, kind, initializers, define)
	}
}

func (p *context) buildLocalInitializer(pkg llssa.Package, owner *localPackage, kind locality.Kind, initializers []localitylayout.Initializer, define bool) *localInitializer {
	ret := &localInitializer{}
	ret.guard = pkg.NewThreadLocalVar(localitylayout.GuardName(owner.plan.Path, kind), types.NewPointer(types.Typ[types.Uint8]), llssa.InGo)
	ret.failureCache = pkg.NewThreadLocalVar(localitylayout.FailureCacheName(owner.plan.Path, kind), types.NewPointer(types.Typ[types.Uintptr]), llssa.InGo)
	ret.dispatch = pkg.NewFunc(localitylayout.InitName(owner.plan.Path, kind), llssa.NoArgsNoRet, llssa.InGo)
	ret.ensure = pkg.NewFunc(localitylayout.EnsureName(owner.plan.Path, kind), llssa.NoArgsNoRet, llssa.InGo)
	ret.ensure.Inline(llssa.AlwaysInline)
	if !define {
		return ret
	}
	ret.guard.InitNil()
	ret.failureCache.InitNil()
	if !ret.dispatch.HasBody() {
		b := ret.dispatch.MakeBody(1)
		for _, initializer := range initializers {
			helper := pkg.NewFunc(initializer.Name, llssa.NoArgsNoRet, llssa.InGo)
			b.Call(helper.Expr)
		}
		b.Return()
		b.EndBuild()
	}
	if !ret.ensure.HasBody() {
		b := ret.ensure.MakeBody(3)
		ready := b.BinOp(token.EQL, b.Load(ret.guard.Expr), p.prog.IntVal(localInitReady, p.prog.Byte()))
		b.If(ready, ret.ensure.Block(2), ret.ensure.Block(1))
		b.SetBlock(ret.ensure.Block(1))
		closure := b.MakeClosure(ret.dispatch.Expr, nil)
		b.Call(
			pkg.RuntimeFunc("EnsureLocalInitializer"),
			ret.guard.Expr,
			ret.failureCache.Expr,
			closure,
		)
		b.Jump(ret.ensure.Block(2))
		b.SetBlock(ret.ensure.Block(2))
		b.Return()
		b.EndBuild()
	}
	return ret
}

func noArgResultSignature(result types.Type) *types.Signature {
	results := types.NewTuple(types.NewVar(token.NoPos, nil, "", result))
	return types.NewSignatureType(nil, nil, nil, nil, results, false)
}

func (p *context) localVariableAddr(b llssa.Builder, v *ssa.Global, info llssa.VariableLocality, name string) llssa.Expr {
	variable := p.locality.variables[v]
	if variable == nil {
		var ok bool
		var err error
		variable, ok, err = p.localVariableFor(p.pkg, v, false)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic(fmt.Sprintf("missing locality metadata for %s", name))
		}
		p.locality.variables[v] = variable
	}
	p.ensureLocalInitializer(b, variable.owner, info.Locality)
	if variable.planned.Storage == localitylayout.StorageNativeTLS {
		direct := variable.owner.direct[variable.planned.Name]
		if direct == nil {
			panic(fmt.Sprintf("missing native TLS storage for %s", name))
		}
		return direct.Expr
	}
	base := p.localPackageBase(b, variable.owner)
	return b.FieldAddr(base, variable.planned.Field)
}

func (p *context) localVariableAddress(b llssa.Builder, variable *ssa.Global, name string) (llssa.Expr, bool) {
	info, ok := p.resolveLocality(p.prog.FullName(variable.Pkg.Pkg, variable.Name()))
	if !ok || info.Locality == locality.None {
		return llssa.Expr{}, false
	}
	return p.localVariableAddr(b, variable, info, name), true
}

func (p *context) resolveLocality(name string) (llssa.VariableLocality, bool) {
	_, info, ok, err := p.prog.ResolveLocality(name)
	if err != nil {
		panic(err)
	}
	return info, ok
}

func (p *context) localPackageBase(b llssa.Builder, owner *localPackage) llssa.Expr {
	state := &p.locality.function
	for block := state.block; block != nil; block = block.Idom() {
		if base, ok := state.packageBases[localBaseCacheKey{block: block, owner: owner}]; ok {
			return base
		}
	}
	base := b.Call(owner.blockFunc.Expr)
	if state.block != nil {
		if state.packageBases == nil {
			state.packageBases = make(map[localBaseCacheKey]llssa.Expr)
		}
		state.packageBases[localBaseCacheKey{block: state.block, owner: owner}] = base
	}
	return base
}

func (p *context) ensureLocalInitializer(b llssa.Builder, owner *localPackage, kind locality.Kind) {
	initializer := owner.init[kind]
	if initializer == nil {
		return
	}
	state := &p.locality.function
	for block := state.block; block != nil; block = block.Idom() {
		if state.packageEnsures[localEnsureCacheKey{block: block, owner: owner, kind: kind}] {
			return
		}
	}
	b.Call(initializer.ensure.Expr)
	if state.block != nil {
		if state.packageEnsures == nil {
			state.packageEnsures = make(map[localEnsureCacheKey]bool)
		}
		state.packageEnsures[localEnsureCacheKey{block: state.block, owner: owner, kind: kind}] = true
	}
}

func (p *context) initializeLocalGuards(b llssa.Builder) {
	owner := p.locality.packages[p.prog.PathOf(p.goTyps)]
	if owner == nil {
		return
	}
	for _, kind := range []locality.Kind{locality.Thread, locality.Goroutine} {
		if initializer := owner.init[kind]; initializer != nil {
			b.Store(initializer.guard.Expr, p.prog.IntVal(localInitReady, p.prog.Byte()))
		}
	}
}
