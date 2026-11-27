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

package ssaopt

import (
	"go/ast"
	"go/constant"
	"go/token"
	"sort"

	"github.com/xgo-dev/llgo/cmd/compiler/abipolicy"
	"github.com/xgo-dev/llgo/cmd/compiler/genmethod"
	"github.com/xgo-dev/llgo/cmd/compiler/goescape"
	llgenabi "github.com/xgo-dev/llgo/cmd/compiler/llvmgen/abi"
	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssautil"
	"github.com/xgo-dev/llgo/types"
)

// EscapeConfig configures pure-Go escape analysis without creating or touching
// any LLVM backend Program.
type EscapeConfig struct {
	TypeCtx         *types.Context
	GCRoots         bool
	PreloadedSyntax bool
	TargetPkg       *ssa.Package // non-nil only when PreloadedSyntax is false
	RuntimePkg      *types.Package
	Patches         map[string]*ssa.Package
	PatchTypes      map[string]*types.Package
	Cache           *goescape.Cache
	Imports         map[*ssa.Function]goescape.Summary
	ImportKeys      map[*ssa.Function]string
	ExtraRoots      []*ssa.Function
	// Linkname optionally resolves preloaded Program-level linkname directives
	// (pure-Go string lookup) when a package was loaded without populating
	// Function.LinkName directly.
	Linkname func(fullName string) (string, bool)
	// ResolveIntrinsic optionally overrides intrinsic classification for a
	// function when a caller needs custom symbol-table resolution.
	ResolveIntrinsic func(fn *ssa.Function) (isGoBody bool, intrinsic ssa.Intrinsic)
}

// PrepareEscapeAnalysis runs pure-Go escape analysis across prog using
// *types.Context and abipolicy with zero LLVM dependencies.
func PrepareEscapeAnalysis(prog *ssa.Program, cfg EscapeConfig) *goescape.Result {
	if prog == nil {
		return nil
	}
	typeCtx := cfg.TypeCtx
	if typeCtx == nil {
		typeCtx = types.NewContext()
	}
	spec := typeCtx.Spec()
	wordSize := spec.WordSize
	if wordSize <= 0 {
		wordSize = 8
	}
	ptrSize := spec.PtrSize
	if ptrSize <= 0 {
		ptrSize = wordSize
	}
	is32Bits := wordSize == 4
	mapHeaderType := resolveRuntimeMapType(cfg.RuntimePkg)

	patchType := func(t types.Type) types.Type {
		return patchEscapeType(t, cfg.PatchTypes)
	}

	var roots []*ssa.Function
	for fn := range ssautil.AllFunctions(prog) {
		roots = append(roots, fn)
	}
	for _, pkg := range prog.AllPackages() {
		roots = append(roots, collectPackageMethods(pkg)...)
	}
	roots = append(roots, cfg.ExtraRoots...)
	for fn := range cfg.Imports {
		roots = append(roots, fn)
	}

	inspectBody := func(fn *ssa.Function) bool {
		return shouldInspectEscapeBody(fn, cfg)
	}
	external := func(fn *ssa.Function) (goescape.Summary, bool) {
		if summary, ok := cfg.Imports[fn]; ok && goescape.ValidSummary(fn, summary) {
			return summary, true
		}
		return escapeCallContract(fn, cfg)
	}
	stackLayout := func(site ssa.Instruction, typ types.Type) (uint64, uint64) {
		return escapeStackLayout(site, typ, typeCtx, patchType, mapHeaderType, wordSize, cfg.GCRoots)
	}
	forceHeap := func(site ssa.Instruction, typ types.Type) string {
		return forceEscapeHeap(site, typ, typeCtx, patchType, spec.GOARCH, ptrSize)
	}

	return goescape.Analyze(roots, goescape.Config{
		Cache:       cfg.Cache,
		Inspect:     inspectBody,
		External:    external,
		ExternalKey: func(fn *ssa.Function) string { return cfg.ImportKeys[fn] },
		IndirectInterface: func(typ types.Type) bool {
			return InterfaceIndirect(patchType(typ), typeCtx, is32Bits)
		},
		ForceHeap:   forceHeap,
		StackLayout: stackLayout,
		LocalStackLayout: func(_ ssa.Instruction, typ types.Type) (uint64, uint64) {
			return StackRootBytes(patchType(typ), typeCtx, wordSize, cfg.GCRoots), uint64(wordSize)
		},
		MaxStackBytes: abipolicy.MaxEscapeStackBytes,
	})
}

// GCRootCount reports how many heap pointer roots t contributes to a Wasm GC
// stack root frame, matching llvmgen.Program.GCRootCount exactly in pure Go.
func GCRootCount(t types.Type, ctx *types.Context) int {
	if t == nil {
		return 0
	}
	if ctx != nil {
		if _, isNative := ctx.NativeTypeCallConv(t); isNative {
			if _, ok := types.Unalias(t).Underlying().(*types.Signature); ok {
				return 0
			}
		}
	}
	switch u := types.Unalias(t).(type) {
	case *types.Basic:
		if u.Kind() == types.String || u.Kind() == types.UnsafePointer {
			return 1
		}
		return 0
	case *types.Pointer:
		if u.Kind() == types.CPtr {
			return 0
		}
		return 1
	case *types.Slice, *types.Map, *types.Chan, *types.Interface:
		return 1
	case *types.Signature:
		if u.CallConv().IsNative() {
			return 0
		}
		return 1
	case *types.Struct:
		count := 0
		for i := 0; i < u.NumFields(); i++ {
			count += GCRootCount(u.Field(i).Type(), ctx)
		}
		return count
	case *types.Array:
		if u.Len() == 0 {
			return 0
		}
		return int(u.Len()) * GCRootCount(u.Elem(), ctx)
	case *types.Tuple:
		count := 0
		for i := 0; i < u.Len(); i++ {
			count += GCRootCount(u.At(i).Type(), ctx)
		}
		return count
	case *types.Named:
		if u.SIMDVec() != nil {
			return 0
		}
		return GCRootCount(u.Underlying(), ctx)
	case *types.SIMDVec:
		return 0
	default:
		return 0
	}
}

// StackRootBytes reserves field-address slots plus a possible frame header for
// a stack-allocated object of type t when cooperative GC roots are enabled.
func StackRootBytes(t types.Type, ctx *types.Context, wordSize int, gcRoots bool) uint64 {
	if !gcRoots {
		return 0
	}
	count := GCRootCount(t, ctx)
	if count == 0 {
		return 0
	}
	return uint64(count+2) * uint64(wordSize)
}

// InterfaceIndirect reports whether an interface data word points at a boxed
// copy of t rather than storing t directly, matching llvmgen.Program.InterfaceIndirect.
func InterfaceIndirect(t types.Type, ctx *types.Context, is32Bits bool) bool {
	if t == nil {
		return true
	}
	if !directIfaceType(t, ctx, false) {
		return true
	}
	kind, inner, _ := llgenabi.DataKindOf(t, 0, is32Bits)
	if kind == llgenabi.Indirect && isNativeFuncType(inner, ctx) {
		kind = llgenabi.Pointer
	}
	return kind == llgenabi.Indirect
}

func directIfaceType(t types.Type, ctx *types.Context, publicFunc bool) bool {
	if isClosureType(t, ctx, publicFunc) {
		return false
	}
	switch u := types.Unalias(t).(type) {
	case *types.Named:
		return directIfaceType(u.Underlying(), ctx, true)
	case *types.Pointer, *types.Chan, *types.Map, *types.Signature:
		return true
	case *types.Basic:
		return u.Kind() == types.UnsafePointer
	case *types.Array:
		return u.Len() == 1 && directIfaceType(u.Elem(), ctx, false)
	case *types.Struct:
		return u.NumFields() == 1 && directIfaceType(u.Field(0).Type(), ctx, false)
	}
	return false
}

func isClosureType(t types.Type, ctx *types.Context, publicFunc bool) bool {
	if t == nil || isNativeFuncType(t, ctx) {
		return false
	}
	switch u := types.Unalias(t).(type) {
	case *types.Signature:
		return u.CallConv().IsClosure()
	case *types.Named:
		if publicFunc {
			return isClosureType(u.Underlying(), ctx, true)
		}
		if sig, ok := u.Underlying().(*types.Signature); ok {
			return sig.CallConv().IsClosure()
		}
	}
	return false
}

func isNativeFuncType(t types.Type, ctx *types.Context) bool {
	if t == nil {
		return false
	}
	if ctx != nil {
		if _, ok := ctx.NativeTypeCallConv(t); ok {
			if _, isSig := types.Unalias(t).Underlying().(*types.Signature); isSig {
				return true
			}
		}
	}
	switch u := t.(type) {
	case *types.Alias:
		if _, ok := types.Unalias(u).Underlying().(*types.Signature); !ok {
			return false
		}
		if u.CallConv().IsNative() {
			return true
		}
		return isNativeFuncType(u.Rhs(), ctx)
	case *types.Named:
		if sig, ok := u.Underlying().(*types.Signature); ok {
			return u.CallConv().IsNative() || sig.CallConv().IsNative()
		}
		return false
	case *types.Signature:
		return u.CallConv().IsNative()
	}
	return false
}

func resolveRuntimeMapType(runtimePkg *types.Package) types.Type {
	if runtimePkg != nil && runtimePkg.Scope() != nil {
		if obj := runtimePkg.Scope().Lookup("Map"); obj != nil {
			return obj.Type()
		}
	}
	// Canonical fallback matching runtime/internal/runtime.Map (hmap).
	uptr := types.Typ[types.UnsafePointer]
	return types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "count", types.Typ[types.Int], false),
		types.NewField(token.NoPos, nil, "flags", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, nil, "B", types.Typ[types.Uint8], false),
		types.NewField(token.NoPos, nil, "noverflow", types.Typ[types.Uint16], false),
		types.NewField(token.NoPos, nil, "hash0", types.Typ[types.Uint32], false),
		types.NewField(token.NoPos, nil, "buckets", uptr, false),
		types.NewField(token.NoPos, nil, "oldbuckets", uptr, false),
		types.NewField(token.NoPos, nil, "nevacuate", types.Typ[types.Uintptr], false),
		types.NewField(token.NoPos, nil, "extra", uptr, false),
	}, nil)
}

func escapeStackLayout(
	site ssa.Instruction,
	typ types.Type,
	typeCtx *types.Context,
	patchType func(types.Type) types.Type,
	mapHeaderType types.Type,
	wordSize int,
	gcRoots bool,
) (size, alignment uint64) {
	switch v := site.(type) {
	case *ssa.MakeSlice:
		elem := patchType(typ.Underlying().(*types.Slice).Elem())
		n := uint64(typeCtx.Sizeof(elem))
		if n == 0 || n > abipolicy.SmallStackBufferSize {
			return 0, 1
		}
		typ = types.NewArray(elem, int64(abipolicy.SmallStackBufferSize/n))
	case *ssa.Convert, *ssa.BinOp:
		elem := types.Typ[types.Byte]
		if _, conversion := site.(*ssa.Convert); conversion {
			if dst, ok := typ.Underlying().(*types.Slice); ok {
				if basic, ok := dst.Elem().Underlying().(*types.Basic); ok && basic.Kind() == types.Int32 {
					elem = types.Typ[types.Rune]
				}
			}
		}
		typ = types.NewArray(elem, abipolicy.SmallStackBufferSize)
	case *ssa.MakeMap:
		mt := patchType(v.Type()).Underlying().(*types.Map)
		size = uint64(typeCtx.Sizeof(mapHeaderType))
		alignment = uint64(typeCtx.Alignof(mapHeaderType))
		var bucket types.Type
		if uint64(typeCtx.Alignof(mt.Key())) <= uint64(wordSize) && uint64(typeCtx.Alignof(mt.Elem())) <= uint64(wordSize) {
			bt := llgenabi.MapBucketType(mt, typeCtx)
			if uint64(typeCtx.Sizeof(bt)) <= abipolicy.MaxImplicitStackVarSize {
				bucket = bt
			}
		}
		if bucket != nil {
			a := uint64(typeCtx.Alignof(bucket))
			size += -size & (a - 1)
			size += uint64(typeCtx.Sizeof(bucket))
			if a > alignment {
				alignment = a
			}
		}
		if gcRoots {
			size += StackRootBytes(mapHeaderType, typeCtx, wordSize, true)
			if bucket != nil {
				size += StackRootBytes(bucket, typeCtx, wordSize, true)
			}
		}
		return size, alignment
	}
	patched := patchType(typ)
	size = uint64(typeCtx.Sizeof(patched))
	if gcRoots {
		size += StackRootBytes(patched, typeCtx, wordSize, true)
	}
	return size, uint64(typeCtx.Alignof(patched))
}

func forceEscapeHeap(
	site ssa.Instruction,
	typ types.Type,
	typeCtx *types.Context,
	patchType func(types.Type) types.Type,
	goarch string,
	ptrSize int,
) string {
	if value, ok := site.(ssa.Value); ok && FreshNilAppendAllocation(value) {
		return "fresh append result storage"
	}
	switch site.(type) {
	case *ssa.MakeSlice:
		elem := patchType(typ.Underlying().(*types.Slice).Elem())
		size := uint64(typeCtx.Sizeof(elem))
		if size == 0 || size > abipolicy.SmallStackBufferSize {
			return "no small slice stack buffer"
		}
		typ = types.NewArray(elem, int64(abipolicy.SmallStackBufferSize/size))
	case *ssa.Convert, *ssa.BinOp:
		typ = types.NewArray(types.Typ[types.Byte], abipolicy.SmallStackBufferSize)
	}
	size := uint64(typeCtx.Sizeof(patchType(typ)))
	limit := abipolicy.MaxImplicitStackVarSize
	if alloc, ok := site.(*ssa.Alloc); ok {
		limit = abipolicy.MaxStackVarSize
		if alloc.Heap {
			limit = abipolicy.MaxImplicitStackVarSize
		}
		if size == 0 && alloc.Heap {
			return "zero-size heap identity"
		}
		recoverLimit := abipolicy.MaxRecoverStackValueSize(goarch, ptrSize)
		if alloc.Heap && alloc.Parent().Recover != nil && size > recoverLimit {
			return "recover-visible storage"
		}
	} else if size == 0 {
		return "zero-size heap identity"
	}
	if size > limit {
		return "stack size limit"
	}
	return ""
}

func shouldInspectEscapeBody(fn *ssa.Function, cfg EscapeConfig) bool {
	if fn == nil {
		return false
	}
	if fn.TypeParams().Len() != 0 && len(fn.TypeArgs()) == 0 {
		return false
	}
	pkg := fn.Pkg
	for parent := fn.Parent(); pkg == nil && parent != nil; parent = parent.Parent() {
		pkg = parent.Pkg
	}
	if pkg != nil {
		if !cfg.PreloadedSyntax && cfg.TargetPkg != nil && pkg != cfg.TargetPkg {
			return false
		}
		if altPkg, ok := cfg.Patches[pkg.Pkg.Path()]; ok && altPkg != nil && pkg != altPkg {
			return false
		}
	}
	if cfg.ResolveIntrinsic != nil {
		isGoBody, _ := cfg.ResolveIntrinsic(fn)
		return isGoBody
	}
	if fn.CallKind() != ssa.CallGo {
		return false
	}
	// If the function carries a Go-to-Go linkname pointing at a different symbol,
	// its body is not the symbol that lowering calls.
	link := effectiveFuncLinkName(fn)
	if link != "" {
		ns, target := directive.ParseLinkTarget(link)
		if ns != directive.LinkGo {
			return false
		}
		exp := effectiveFuncExportName(fn)
		if target != "" && target != exp && target != canonicalFuncSymbol(fn) {
			return false
		}
	}
	return true
}

func escapeCallContract(fn *ssa.Function, cfg EscapeConfig) (goescape.Summary, bool) {
	if fn == nil {
		return goescape.Summary{}, false
	}
	var intr ssa.Intrinsic
	if cfg.ResolveIntrinsic != nil {
		_, intr = cfg.ResolveIntrinsic(fn)
	} else {
		intr = fn.Intrinsic()
	}
	if intr != ssa.IntrinsicNone {
		return IntrinsicEscapeContract(intr, fn.Signature)
	}
	if len(fn.Blocks) == 0 && (fn.NoEscape || hasFuncDirective(fn, "go:noescape")) {
		summary := SignatureEscapeSummary(fn.Signature)
		for i := range summary.Parameters {
			summary.Parameters[i].Mutator = 1
			summary.Parameters[i].Callee = 1
		}
		return summary, true
	}
	return goescape.Summary{}, false
}

// SignatureEscapeSummary allocates a zero-effect escape summary matching sig.
func SignatureEscapeSummary(sig *types.Signature) goescape.Summary {
	n := sig.Params().Len()
	if sig.Recv() != nil {
		n++
	}
	s := goescape.Summary{Parameters: make([]goescape.Parameter, n)}
	for i := range s.Parameters {
		s.Parameters[i].Results = make([]goescape.Effect, sig.Results().Len())
	}
	return s
}

// IntrinsicEscapeContract returns the escape contract for an LLGo compiler
// intrinsic (pure Go, keyed by typed ssa.Intrinsic).
func IntrinsicEscapeContract(kind ssa.Intrinsic, sig *types.Signature) (goescape.Summary, bool) {
	if sig == nil {
		return goescape.Summary{}, false
	}
	s := SignatureEscapeSummary(sig)
	n, results := len(s.Parameters), sig.Results().Len()
	switch kind {
	case ssa.IntrinsicKeepAlive:
		if n == 1 && results == 0 && sig.Recv() == nil && isAnyType(sig.Params().At(0).Type()) {
			return s, true
		}
	case ssa.IntrinsicSkip:
		return s, true
	case ssa.IntrinsicNoEscape:
		if noEscapeSignature(sig) {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Callee = 1
			return s, true
		}
	case ssa.IntrinsicAtomicLoad:
		if n == 1 && results == 1 {
			s.Parameters[0].Results[0] = 2
			return s, true
		}
	case ssa.IntrinsicAtomicStore, ssa.IntrinsicAtomicStoreUnordered:
		if n == 2 && results == 0 {
			s.Parameters[0].Mutator = 1
			s.Parameters[1].Heap = 1
			return s, true
		}
	case ssa.IntrinsicAtomicCmpXchg, ssa.IntrinsicAtomicCmpXchgOK:
		if n == 3 && results >= 1 && results <= 2 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[2].Heap = 1
			return s, true
		}
	case ssa.IntrinsicAtomicAddReturnNew:
		if n == 2 && results == 1 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[1].Heap = 1
			s.Parameters[1].Results[0] = 1
			return s, true
		}
	case ssa.IntrinsicAtomicAddRelaxed:
		if n == 2 && results == 1 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[1].Heap = 1
			return s, true
		}
	case ssa.IntrinsicAtomicXchg, ssa.IntrinsicAtomicAdd, ssa.IntrinsicAtomicSub,
		ssa.IntrinsicAtomicAnd, ssa.IntrinsicAtomicNand, ssa.IntrinsicAtomicOr,
		ssa.IntrinsicAtomicXor, ssa.IntrinsicAtomicMax, ssa.IntrinsicAtomicMin,
		ssa.IntrinsicAtomicUMax, ssa.IntrinsicAtomicUMin:
		if n == 2 && results == 1 {
			s.Parameters[0].Mutator = 1
			s.Parameters[0].Results[0] = 2
			s.Parameters[1].Heap = 1
			return s, true
		}
	}
	return goescape.Summary{}, false
}

func isAnyType(t types.Type) bool {
	iface, ok := types.Unalias(t).Underlying().(*types.Interface)
	return ok && iface.Empty()
}

func noEscapeSignature(sig *types.Signature) bool {
	return sig.Recv() == nil && sig.Params().Len() == 1 && sig.Results().Len() == 1 &&
		types.Identical(sig.Params().At(0).Type(), types.Typ[types.UnsafePointer]) &&
		types.Identical(sig.Results().At(0).Type(), types.Typ[types.UnsafePointer])
}

func effectiveFuncLinkName(fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	if fn.LinkName != "" {
		return fn.LinkName
	}
	if orig := fn.Origin(); orig != nil && orig != fn {
		if ln := effectiveFuncLinkName(orig); ln != "" {
			return ln
		}
	}
	if obj, ok := fn.Object().(*types.Func); ok && obj != nil {
		return obj.LinkName()
	}
	return ""
}

func effectiveFuncExportName(fn *ssa.Function) string {
	if fn == nil {
		return ""
	}
	if fn.ExportName != "" {
		return fn.ExportName
	}
	if obj, ok := fn.Object().(*types.Func); ok && obj != nil {
		return obj.ExportName()
	}
	return ""
}

func canonicalFuncSymbol(fn *ssa.Function) string {
	if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return ""
	}
	pkgPath := fn.Pkg.Pkg.Path()
	if recv := fn.Signature.Recv(); recv != nil {
		return pkgPath + "." + recvSymbolName(recv.Type()) + "." + fn.Name()
	}
	return pkgPath + "." + fn.Name()
}

func recvSymbolName(t types.Type) string {
	switch u := types.Unalias(t).(type) {
	case *types.Pointer:
		return "(*" + recvSymbolName(u.Elem()) + ")"
	case *types.Named:
		if u.Obj() != nil {
			return u.Obj().Name()
		}
	}
	return ""
}

func patchEscapeType(typ types.Type, patchTypes map[string]*types.Package) types.Type {
	if typ == nil || len(patchTypes) == 0 {
		return typ
	}
	switch u := types.Unalias(typ).(type) {
	case *types.Named:
		if obj := u.Obj(); obj != nil && obj.Pkg() != nil {
			if altPkg, ok := patchTypes[obj.Pkg().Path()]; ok && altPkg != nil {
				if altObj := altPkg.Scope().Lookup(obj.Name()); altObj != nil {
					if altNamed, ok := altObj.Type().(*types.Named); ok {
						if targs := u.TypeArgs(); targs != nil && targs.Len() > 0 {
							args := make([]types.Type, targs.Len())
							for i := 0; i < targs.Len(); i++ {
								args[i] = patchEscapeType(targs.At(i), patchTypes)
							}
							if inst, err := types.Instantiate(nil, altNamed, args, false); err == nil {
								return inst
							}
						}
						return altNamed
					}
				}
			}
		}
	}
	return typ
}

func collectPackageMethods(pkg *ssa.Package) []*ssa.Function {
	if pkg == nil || pkg.Prog == nil {
		return nil
	}
	type methodType struct {
		key string
		typ types.Type
	}
	var typesToVisit []methodType
	seenTypes := make(map[string]bool)
	addType := func(typ types.Type) {
		key := types.TypeString(typ, func(owner *types.Package) string {
			if owner == nil {
				return ""
			}
			return owner.Path()
		})
		if !seenTypes[key] {
			seenTypes[key] = true
			typesToVisit = append(typesToVisit, methodType{key: key, typ: typ})
		}
	}
	for _, member := range pkg.Members {
		if typ, ok := member.(*ssa.Type); ok {
			addType(typ.Type())
			addType(types.NewPointer(typ.Type()))
		}
	}
	sort.Slice(typesToVisit, func(i, j int) bool {
		return typesToVisit[i].key < typesToVisit[j].key
	})
	var methods []*ssa.Function
	seen := make(map[*ssa.Function]bool)
	for _, entry := range typesToVisit {
		mset := pkg.Prog.MethodSets.MethodSet(entry.typ)
		for i := 0; i < mset.Len(); i++ {
			meth := mset.At(i)
			if genmethod.SupportsGenericMethods && genmethod.IsGenericMethod(meth.Type()) {
				continue
			}
			if fn := pkg.Prog.MethodValue(meth); fn != nil && !seen[fn] {
				seen[fn] = true
				methods = append(methods, fn)
			}
		}
	}
	return methods
}

func hasFuncDirective(f *ssa.Function, name string) bool {
	if f == nil {
		return false
	}
	decl, _ := f.Syntax().(*ast.FuncDecl)
	if decl == nil || decl.Doc == nil {
		return false
	}
	for _, item := range directive.ParseGroup(decl.Doc) {
		if item.Name == name {
			return true
		}
	}
	return false
}

// FreshNilAppendSource identifies backing storage that can become an
// append(nil, make([]T, n)...) result itself without copying.
func FreshNilAppendSource(call *ssa.Call) ssa.Value {
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	if !ok || builtin.Name() != "append" || len(call.Call.Args) != 2 {
		return nil
	}
	destination := call.Call.Args[0]
	for {
		change, ok := destination.(*ssa.ChangeType)
		if !ok {
			break
		}
		destination = change.X
	}
	dst, ok := destination.(*ssa.Const)
	if !ok || !dst.IsNil() {
		return nil
	}
	source := call.Call.Args[1]
	current, parent := source, ssa.Instruction(call)
	for {
		if singleNonDebugReferrer(current) != parent {
			return nil
		}
		change, ok := current.(*ssa.ChangeType)
		if !ok {
			break
		}
		current, parent = change.X, change
	}
	switch src := current.(type) {
	case *ssa.MakeSlice:
		if src.Block() != call.Block() {
			return nil
		}
		if src.Len == src.Cap {
			return source
		}
	case *ssa.Slice:
		alloc, ok := src.X.(*ssa.Alloc)
		if !ok || alloc.Comment != "makeslice" || src.Low != nil || src.Max != nil {
			return nil
		}
		if alloc.Block() != call.Block() {
			return nil
		}
		pointer, ok := alloc.Type().(*types.Pointer)
		if !ok {
			return nil
		}
		array, ok := pointer.Elem().Underlying().(*types.Array)
		if !ok {
			return nil
		}
		if src.High != nil {
			high, ok := src.High.(*ssa.Const)
			if !ok || high.Value == nil {
				return nil
			}
			if n, exact := constant.Int64Val(high.Value); !exact || n != array.Len() {
				return nil
			}
		}
		if singleNonDebugReferrer(alloc) == src {
			return source
		}
	}
	return nil
}

// FreshNilAppendAllocation reports whether value is the backing allocation for
// a forwarded append(nil, make(...)) expression and therefore must remain on
// the heap.
func FreshNilAppendAllocation(value ssa.Value) bool {
	switch allocation := value.(type) {
	case *ssa.MakeSlice:
		if allocation.Len != allocation.Cap {
			return false
		}
	case *ssa.Alloc:
		if allocation.Comment != "makeslice" {
			return false
		}
	default:
		return false
	}
	for {
		switch use := singleNonDebugReferrer(value).(type) {
		case *ssa.Slice:
			value = use
		case *ssa.ChangeType:
			value = use
		case *ssa.Call:
			return FreshNilAppendSource(use) == value
		default:
			return false
		}
	}
}

func singleNonDebugReferrer(v ssa.Value) ssa.Instruction {
	refs := v.Referrers()
	if refs == nil {
		return nil
	}
	var single ssa.Instruction
	for _, ref := range *refs {
		if _, isDbg := ref.(*ssa.DebugRef); isDbg {
			continue
		}
		if single != nil {
			return nil
		}
		single = ref
	}
	return single
}
