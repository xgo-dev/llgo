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

package ssa

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/directive"
	"github.com/xgo-dev/llgo/types"
)

// CallKind classifies the calling convention / lowering mechanism of an
// LLGo SSA function or call site.
type CallKind uint8

const (
	CallGo CallKind = iota
	CallC
	CallStdcall
	CallPy
	CallWasm
	CallLLGoIntrinsic
	CallLLVMIntrinsic
)

func (k CallKind) String() string {
	switch k {
	case CallC:
		return "c"
	case CallStdcall:
		return "stdcall"
	case CallPy:
		return "py"
	case CallWasm:
		return "wasm"
	case CallLLGoIntrinsic:
		return "llgo"
	case CallLLVMIntrinsic:
		return "llvm"
	default:
		return "go"
	}
}

// Intrinsic identifies a first-class LLGo compiler intrinsic bound via
// //go:linkname ... llgo.<op>, CGo runtime helper, SIMD operation, or
// compiler-recognized stdlib bitcast/escape intrinsic.
type Intrinsic uint16

const (
	IntrinsicNone Intrinsic = iota
	IntrinsicUnreachable
	IntrinsicCStr
	IntrinsicAlloca
	IntrinsicAllocaCStr
	IntrinsicAllocaCStrs
	IntrinsicAllocCStr
	IntrinsicAdvance
	IntrinsicIndex
	IntrinsicStringData
	IntrinsicString
	IntrinsicDeferData
	IntrinsicSigjmpbuf
	IntrinsicSigsetjmp
	IntrinsicSiglongjmp
	IntrinsicSetjmp
	IntrinsicLongjmp
	IntrinsicFuncAddr
	IntrinsicFuncPCABI0
	IntrinsicPyList
	IntrinsicPyStr
	IntrinsicPyTuple
	IntrinsicAtomicLoad
	IntrinsicAtomicStore
	IntrinsicAtomicStoreUnordered
	IntrinsicAtomicAddRelaxed
	IntrinsicAtomicCmpXchg
	IntrinsicAtomicCmpXchgOK
	IntrinsicAtomicAddReturnNew
	IntrinsicAtomicXchg
	IntrinsicAtomicAdd
	IntrinsicAtomicSub
	IntrinsicAtomicAnd
	IntrinsicAtomicNand
	IntrinsicAtomicOr
	IntrinsicAtomicXor
	IntrinsicAtomicMax
	IntrinsicAtomicMin
	IntrinsicAtomicUMax
	IntrinsicAtomicUMin
	IntrinsicCgoCString
	IntrinsicCgoCBytes
	IntrinsicCgoGoString
	IntrinsicCgoGoStringN
	IntrinsicCgoGoBytes
	IntrinsicCgoCMalloc
	IntrinsicCgoCheckPointer
	IntrinsicCgoCgocall
	IntrinsicAsm
	IntrinsicStackSave
	IntrinsicSkip
	IntrinsicSyscall
	IntrinsicBoolToUint8
	IntrinsicClosureEnv
	IntrinsicFloat32FromBits
	IntrinsicFloat32Bits
	IntrinsicFloat64FromBits
	IntrinsicFloat64Bits
	IntrinsicUMulOverflow
	IntrinsicSIMD
	IntrinsicInterfaceKindUsed
	IntrinsicAESRound
	IntrinsicAESSubWord
	IntrinsicAESAlloca
	IntrinsicKeepAlive
	IntrinsicNoEscape
)

var llgoIntrinsicByName = map[string]Intrinsic{
	"aesRound":             IntrinsicAESRound,
	"aesSubWord":           IntrinsicAESSubWord,
	"aesAlloca":            IntrinsicAESAlloca,
	"simd":                 IntrinsicSIMD,
	"cstr":                 IntrinsicCStr,
	"advance":              IntrinsicAdvance,
	"index":                IntrinsicIndex,
	"alloca":               IntrinsicAlloca,
	"allocCStr":            IntrinsicAllocCStr,
	"allocaCStr":           IntrinsicAllocaCStr,
	"allocaCStrs":          IntrinsicAllocaCStrs,
	"string":               IntrinsicString,
	"stringData":           IntrinsicStringData,
	"funcAddr":             IntrinsicFuncAddr,
	"funcPCABI0":           IntrinsicFuncPCABI0,
	"keepAlive":            IntrinsicKeepAlive,
	"skip":                 IntrinsicSkip,
	"noescape":             IntrinsicNoEscape,
	"syscall":              IntrinsicSyscall,
	"boolToUint8":          IntrinsicBoolToUint8,
	"interfaceKindUsed":    IntrinsicInterfaceKindUsed,
	"closureEnv":           IntrinsicClosureEnv,
	"pystr":                IntrinsicPyStr,
	"pyList":               IntrinsicPyList,
	"pyTuple":              IntrinsicPyTuple,
	"sigjmpbuf":            IntrinsicSigjmpbuf,
	"sigsetjmp":            IntrinsicSigsetjmp,
	"siglongjmp":           IntrinsicSiglongjmp,
	"setjmp":               IntrinsicSetjmp,
	"longjmp":              IntrinsicLongjmp,
	"deferData":            IntrinsicDeferData,
	"unreachable":          IntrinsicUnreachable,
	"atomicLoad":           IntrinsicAtomicLoad,
	"atomicStore":          IntrinsicAtomicStore,
	"atomicStoreUnordered": IntrinsicAtomicStoreUnordered,
	"atomicAddRelaxed":     IntrinsicAtomicAddRelaxed,
	"atomicCmpXchg":        IntrinsicAtomicCmpXchg,
	"atomicCmpXchgOK":      IntrinsicAtomicCmpXchgOK,
	"atomicAddReturnNew":   IntrinsicAtomicAddReturnNew,
	"atomicXchg":           IntrinsicAtomicXchg,
	"atomicAdd":            IntrinsicAtomicAdd,
	"atomicSub":            IntrinsicAtomicSub,
	"atomicAnd":            IntrinsicAtomicAnd,
	"atomicNand":           IntrinsicAtomicNand,
	"atomicOr":             IntrinsicAtomicOr,
	"atomicXor":            IntrinsicAtomicXor,
	"atomicMax":            IntrinsicAtomicMax,
	"atomicMin":            IntrinsicAtomicMin,
	"atomicUMax":           IntrinsicAtomicUMax,
	"atomicUMin":           IntrinsicAtomicUMin,
	"_Cfunc_CString":       IntrinsicCgoCString,
	"_Cfunc_CBytes":        IntrinsicCgoCBytes,
	"_Cfunc_GoString":      IntrinsicCgoGoString,
	"_Cfunc_GoStringN":     IntrinsicCgoGoStringN,
	"_Cfunc_GoBytes":       IntrinsicCgoGoBytes,
	"_Cfunc__CMalloc":      IntrinsicCgoCMalloc,
	"_cgoCheckPointer":     IntrinsicCgoCheckPointer,
	"_cgo_runtime_cgocall": IntrinsicCgoCgocall,
	"asm":                  IntrinsicAsm,
	"stackSave":            IntrinsicStackSave,
	"float32FromBits":      IntrinsicFloat32FromBits,
	"float32Bits":          IntrinsicFloat32Bits,
	"float64FromBits":      IntrinsicFloat64FromBits,
	"float64Bits":          IntrinsicFloat64Bits,
	"umulOverflow":         IntrinsicUMulOverflow,
}

var llgoIntrinsicNames map[Intrinsic]string

func init() {
	llgoIntrinsicNames = make(map[Intrinsic]string, len(llgoIntrinsicByName))
	for name, kind := range llgoIntrinsicByName {
		llgoIntrinsicNames[kind] = name
	}
}

// LookupIntrinsic resolves an llgo.<name> or cgo helper name into its typed
// Intrinsic constant, or IntrinsicNone if unknown.
func LookupIntrinsic(name string) Intrinsic {
	return llgoIntrinsicByName[name]
}

// String returns the canonical LLGo intrinsic name (e.g. "alloca", "cstr").
func (k Intrinsic) String() string {
	return llgoIntrinsicNames[k]
}

// ReceiverNilCheck preserves pointer-method receiver nil-dereference semantics
// that SSA field-address folding otherwise erases (e.g. (*p).M or
// p.EmbeddedValue.PtrMethod).
type ReceiverNilCheck struct {
	Pos     token.Pos // source position of the selector expression
	Address bool      // true when taking the address of an embedded value field
}

// PyModule returns the Python module name when p is a PkgPyModule package.
func (p *Package) PyModule() string {
	if p != nil && p.Kind == directive.PkgPyModule {
		return p.KindParam
	}
	return ""
}

// SkipsInit reports whether p is excluded from Go package initialization.
func (p *Package) SkipsInit() bool {
	return p != nil && p.Kind >= directive.PkgNoInit
}

// IsCPackage reports whether p selects LLGo's package C export convention.
func (p *Package) IsCPackage() bool {
	return p != nil && p.Pkg != nil && p.Pkg.IsCPackage()
}

// Skipped reports whether symbol name is skipped via //llgo:skip on p.
func (p *Package) Skipped(name string) bool {
	return p != nil && p.Skips != nil && p.Skips[name]
}

// LinkTarget splits g.LinkName into its link namespace and target symbol.
func (g *Global) LinkTarget() (directive.LinkNamespace, string) {
	if g == nil {
		return directive.LinkGo, ""
	}
	return directive.ParseLinkTarget(g.LinkName)
}

func (f *Function) effectiveLinkName() string {
	if f == nil {
		return ""
	}
	if f.LinkName != "" {
		return f.LinkName
	}
	if f.topLevelOrigin != nil && f.topLevelOrigin != f {
		if link := f.topLevelOrigin.effectiveLinkName(); link != "" {
			return link
		}
	}
	if f.object != nil {
		return f.object.LinkName()
	}
	return ""
}

func (f *Function) effectiveWasmImport() *directive.WasmImport {
	if f == nil {
		return nil
	}
	if f.WasmImport != nil {
		return f.WasmImport
	}
	if f.topLevelOrigin != nil && f.topLevelOrigin != f {
		if wi := f.topLevelOrigin.effectiveWasmImport(); wi != nil {
			return wi
		}
	}
	if f.object != nil {
		return f.object.WasmImport()
	}
	return nil
}

// CallKind classifies how f should be called or lowered in LLGo.
func (f *Function) CallKind() CallKind {
	if f == nil {
		return CallGo
	}
	if link := f.effectiveLinkName(); link != "" {
		ns, _ := directive.ParseLinkTarget(link)
		switch ns {
		case directive.LinkC:
			return CallC
		case directive.LinkStdcall:
			return CallStdcall
		case directive.LinkPy:
			return CallPy
		case directive.LinkLLGo:
			return CallLLGoIntrinsic
		case directive.LinkLLVM:
			return CallLLVMIntrinsic
		}
	}
	if f.effectiveWasmImport() != nil {
		return CallWasm
	}
	if f.Intrinsic() != IntrinsicNone {
		return CallLLGoIntrinsic
	}
	if f.Signature != nil {
		switch f.Signature.CallConv() {
		case types.CDecl:
			return CallC
		case types.Stdcall:
			return CallStdcall
		case types.Py:
			return CallPy
		case types.Wasm:
			return CallWasm
		}
	}
	return CallGo
}

// CallConv returns the effective LLGo calling convention of f.
func (f *Function) CallConv() types.CallConv {
	if f == nil {
		return types.Go
	}
	switch f.CallKind() {
	case CallC:
		return types.CDecl
	case CallStdcall:
		return types.Stdcall
	case CallPy:
		return types.Py
	case CallWasm:
		return types.Wasm
	}
	if f.Signature != nil {
		return f.Signature.CallConv()
	}
	return types.Go
}

// LinkSymbol returns the external symbol name for f after stripping any
// namespace prefix (C., stdcall., py., llgo.). For llvm.* intrinsics, the
// full intrinsic name is returned.
func (f *Function) LinkSymbol() string {
	link := f.effectiveLinkName()
	if link == "" {
		return ""
	}
	_, sym := directive.ParseLinkTarget(link)
	return sym
}

// LLVMIntrinsic returns the full "llvm.*" intrinsic name when f binds an LLVM
// intrinsic, or "" otherwise.
func (f *Function) LLVMIntrinsic() string {
	link := f.effectiveLinkName()
	if link == "" {
		return ""
	}
	if ns, sym := directive.ParseLinkTarget(link); ns == directive.LinkLLVM {
		return sym
	}
	return ""
}

// Intrinsic returns the typed LLGo intrinsic for f, or IntrinsicNone if f is
// not an LLGo intrinsic.
func (f *Function) Intrinsic() Intrinsic {
	if f == nil {
		return IntrinsicNone
	}
	if link := f.effectiveLinkName(); link != "" {
		if ns, sym := directive.ParseLinkTarget(link); ns == directive.LinkLLGo {
			return LookupIntrinsic(sym)
		}
	}
	if f.topLevelOrigin == nil {
		if k := LookupIntrinsic(f.name); k != IntrinsicNone && isCgoHelperName(f.name) {
			return k
		}
	}
	pkgPath := ""
	if f.Pkg != nil && f.Pkg.Pkg != nil {
		pkgPath = f.Pkg.Pkg.Path()
	} else if f.object != nil && f.object.Pkg() != nil {
		pkgPath = f.object.Pkg().Path()
	}
	if pkgPath == "simd/archsimd" && f.Signature != nil && isArchSIMDIntrinsic(f) {
		return IntrinsicSIMD
	}
	switch pkgPath + "." + f.name {
	case "hash/maphash.escapeForHash":
		return IntrinsicSkip
	case "math.Float32frombits":
		return IntrinsicFloat32FromBits
	case "math.Float32bits":
		return IntrinsicFloat32Bits
	case "math.Float64frombits":
		return IntrinsicFloat64FromBits
	case "math.Float64bits":
		return IntrinsicFloat64Bits
	}
	return IntrinsicNone
}

func isCgoHelperName(name string) bool {
	return strings.HasPrefix(name, "_Cfunc_") ||
		name == "_cgoCheckPointer" ||
		name == "_cgo_runtime_cgocall"
}

func isArchSIMDIntrinsic(f *Function) bool {
	if decl, ok := f.Syntax().(*ast.FuncDecl); ok && decl.Body != nil {
		return false
	}
	sig := f.Signature
	if recv := sig.Recv(); recv != nil {
		rt := types.Unalias(recv.Type())
		if rt.SIMDVec() != nil {
			return true
		}
		if named, ok := rt.(*types.Named); ok && named.Obj() != nil && strings.HasPrefix(named.Obj().Name(), "Mask") {
			return true
		}
		return false
	}
	if f.name == "ClearAVXUpperBits" {
		return sig.Params().Len() == 0 && sig.Results().Len() == 0 && !sig.Variadic()
	}
	if strings.HasPrefix(f.name, "Load") && strings.HasSuffix(f.name, "Array") {
		return true
	}
	if strings.HasPrefix(f.name, "Broadcast") {
		return true
	}
	if strings.HasPrefix(f.name, "Mask") && strings.HasSuffix(f.name, "FromBits") {
		return true
	}
	return false
}

// CallKind classifies the call convention / lowering mode of c, inspecting the
// static callee when available and otherwise the callee Signature's CallConv.
func (c *CallCommon) CallKind() CallKind {
	if c == nil {
		return CallGo
	}
	if c.IsInvoke() {
		return CallGo
	}
	if fn, ok := c.Value.(*Function); ok {
		return fn.CallKind()
	}
	if sig := c.Signature(); sig != nil {
		switch sig.CallConv() {
		case types.CDecl:
			return CallC
		case types.Stdcall:
			return CallStdcall
		case types.Py:
			return CallPy
		case types.Wasm:
			return CallWasm
		}
	}
	return CallGo
}

// Intrinsic returns the typed LLGo intrinsic invoked by c, or IntrinsicNone.
func (c *CallCommon) Intrinsic() Intrinsic {
	if c == nil || c.IsInvoke() {
		return IntrinsicNone
	}
	if fn, ok := c.Value.(*Function); ok {
		return fn.Intrinsic()
	}
	return IntrinsicNone
}

// IsCVariadic reports whether c passes C-style variadic arguments (either via
// a native variadic Signature or LLGo's __llgo_va_list marker parameter).
func (c *CallCommon) IsCVariadic() bool {
	if c == nil || c.IsInvoke() {
		return false
	}
	sig := c.Signature()
	if sig == nil {
		return false
	}
	if sig.Variadic() && sig.CallConv().IsNative() {
		return true
	}
	if params := sig.Params(); params != nil && params.Len() > 0 {
		last := params.At(params.Len() - 1)
		if last != nil && last.Name() == "__llgo_va_list" {
			return true
		}
	}
	return false
}

func selectionNeedsAddressCheck(sel *selection) bool {
	if sel == nil || sel.kind != types.MethodVal {
		return false
	}
	typ := sel.recv
	for _, index := range sel.index[:len(sel.index)-1] {
		if ptr, ok := types.Unalias(typ).Underlying().(*types.Pointer); ok {
			typ = ptr.Elem()
		}
		st, ok := types.Unalias(typ).Underlying().(*types.Struct)
		if !ok || index >= st.NumFields() {
			return false
		}
		typ = st.Field(index).Type()
	}
	_, isPtr := types.Unalias(typ).Underlying().(*types.Pointer)
	return !isPtr
}

// minStaticByteArrayLen is the element threshold at which a package-level
// constant [N]byte / [...]byte initializer is stored directly on Global.StaticBytes
// instead of emitting element-by-element IndexAddr+Store instructions in init.
const minStaticByteArrayLen = 4096

func staticByteArrayInit(g *Global, rhs ast.Expr, retainInit map[string]bool) ([]byte, bool) {
	if g == nil || rhs == nil {
		return nil, false
	}
	if g.Locality != directive.LocalityNone || g.LinkName != "" {
		return nil, false
	}
	if retainInit != nil && retainInit[g.name] {
		return nil, false
	}
	cl, ok := rhs.(*ast.CompositeLit)
	if !ok || len(cl.Elts) < minStaticByteArrayLen {
		return nil, false
	}
	ptr, ok := g.typ.(*types.Pointer)
	if !ok {
		return nil, false
	}
	arr, ok := ptr.Elem().Underlying().(*types.Array)
	if !ok || arr.Len() < minStaticByteArrayLen {
		return nil, false
	}
	basic, ok := arr.Elem().Underlying().(*types.Basic)
	if !ok || (basic.Kind() != types.Byte && basic.Kind() != types.Uint8) {
		return nil, false
	}
	at, ok := cl.Type.(*ast.ArrayType)
	if !ok || at.Len == nil || !isByteIdentExpr(at.Elt) {
		return nil, false
	}
	n := int(arr.Len())
	if int64(n) != arr.Len() {
		return nil, false
	}
	buf := make([]byte, n)
	next := 0
	for _, elt := range cl.Elts {
		idx := next
		val := elt
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			idx, ok = parseStaticIntLit(kv.Key)
			if !ok {
				return nil, false
			}
			val = kv.Value
		} else {
			next++
		}
		b, ok := parseStaticByteLit(val)
		if !ok || idx < 0 || idx >= n {
			return nil, false
		}
		buf[idx] = b
		if _, isKV := elt.(*ast.KeyValueExpr); isKV {
			next = idx + 1
		}
	}
	return buf, true
}

func isByteIdentExpr(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && (id.Name == "byte" || id.Name == "uint8")
}

func parseStaticIntLit(e ast.Expr) (int, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	n, err := strconv.ParseInt(lit.Value, 0, 32)
	if err != nil || n < 0 {
		return 0, false
	}
	return int(n), true
}

func parseStaticByteLit(e ast.Expr) (byte, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok {
		return 0, false
	}
	switch lit.Kind {
	case token.INT:
		n, err := strconv.ParseUint(lit.Value, 0, 8)
		if err != nil {
			return 0, false
		}
		return byte(n), true
	case token.CHAR:
		s, err := strconv.Unquote(lit.Value)
		if err != nil || len(s) != 1 {
			return 0, false
		}
		return s[0], true
	default:
		return 0, false
	}
}
