/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"go/types"
	"log"
	"os"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ssa"

	llssa "github.com/goplus/llgo/ssa"
)

var asmRegisterRegex = regexp.MustCompile(`\{[a-zA-Z]+\}`)

// -----------------------------------------------------------------------------

func constStr(v ssa.Value) (ret string, ok bool) {
	if c, ok := v.(*ssa.Const); ok {
		if v := c.Value; v.Kind() == constant.String {
			return constant.StringVal(v), true
		}
	}
	return
}

func constBool(v ssa.Value) (ret bool, ok bool) {
	if c, ok := v.(*ssa.Const); ok {
		if v := c.Value; v.Kind() == constant.Bool {
			return constant.BoolVal(v), true
		}
	}
	return
}

func isNamedType(t types.Type, pkgPath, name string) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Name() == name && obj.Pkg() != nil && obj.Pkg().Path() == pkgPath
}

func staticCallMethod(call *ssa.CallCommon) (recv types.Type, method string, ok bool) {
	fn := call.StaticCallee()
	if fn == nil || fn.Signature == nil || fn.Signature.Recv() == nil {
		return nil, "", false
	}
	return fn.Signature.Recv().Type(), fn.Name(), true
}

func reflectMethodNameArg(call *ssa.CallCommon) (nameArg ssa.Value, ok bool) {
	if method := call.Method; method != nil {
		// invoke-mode on reflect.Type: Value is the interface receiver and
		// Args[0] is the first real argument to MethodByName.
		if !isNamedType(call.Value.Type(), "reflect", "Type") {
			return nil, false
		}
		if method.Name() != "MethodByName" {
			return nil, method.Name() == "Method"
		}
		return call.Args[0], true
	}

	recv, methodName, ok := staticCallMethod(call)
	// static-call mode on reflect.Value: StaticCallee carries the concrete
	// method, and the MethodByName string is the last explicit argument.
	if !ok || !isNamedType(recv, "reflect", "Value") {
		return nil, false
	}
	if methodName != "MethodByName" {
		return nil, methodName == "Method"
	}
	return call.Args[len(call.Args)-1], true
}

func (p *context) markReflectMethodCall(call *ssa.CallCommon) {
	owner := p.fn.Name()
	nameArg, ok := reflectMethodNameArg(call)
	// Not a reflect.Type/reflect.Value Method* call.
	if !ok {
		return
	}
	// Method(index) and dynamic MethodByName(name) must stay conservative.
	if nameArg == nil {
		p.pkg.EmitReflectMethod(owner)
		return
	}
	// Constant MethodByName("X") can record an exact method-name demand.
	if name, ok := constStr(nameArg); ok {
		p.pkg.EmitUseNamedMethod(owner, name)
		return
	}
	// Dynamic MethodByName(name) falls back to conservative reflection.
	p.pkg.EmitReflectMethod(owner)
}

// func pystr(string) *py.Object
func pystr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		if sv, ok := constStr(args[0]); ok {
			return b.PyStr(sv)
		}
	}
	panic("pystr(<string-literal>): invalid arguments")
}

// func cstr(string) *int8
func cstr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		if sv, ok := constStr(args[0]); ok {
			return b.CStr(sv)
		}
	}
	panic("cstr(<string-literal>): invalid arguments")
}

// func asm(string)
// func asm(string, map[string]any)
func (p *context) asm(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 0 || len(args) > 2 {
		panic("asm: invalid arguments - expected asm(<string-literal>) or asm(<string-literal>, <map-literal>)")
	}

	asmString, ok := constStr(args[0])
	if !ok {
		panic("asm: inline assembly requires a constant string")
	}
	if len(args) == 1 {
		b.InlineAsm(asmString)
		return llssa.Expr{Type: b.Prog.Void()}
	}

	registers := make(map[string]llssa.Expr)
	if registerMap, ok := args[1].(*ssa.MakeMap); ok {
		referrers := registerMap.Referrers()
		for _, r := range *referrers {
			switch r := r.(type) {
			case *ssa.DebugRef, *ssa.Call:
				// ignore
			case *ssa.MapUpdate:
				if r.Block() != registerMap.Block() {
					panic("asm: register value map must be created in the same basic block")
				}
				key, ok := constStr(r.Key)
				if !ok {
					panic("asm: register key must be a string constant")
				}
				llvmValue := p.compileValue(b, r.Value.(*ssa.MakeInterface).X)
				registers[key] = llvmValue
			default:
				panic(fmt.Sprintf("asm: don't know how to handle argument to inline assembly: %s", r.String()))
			}
		}
	}

	finalAsm := asmString
	var hasOutput bool
	var inputValues []llssa.Expr
	var constraints []string
	registerNumbers := map[string]int{}

	if strings.Contains(finalAsm, "{}") {
		finalAsm = strings.ReplaceAll(finalAsm, "{}", "$0")
		constraints = append(constraints, "=&r")
		registerNumbers[""] = 0
		hasOutput = true
	}

	finalAsm = asmRegisterRegex.ReplaceAllStringFunc(finalAsm, func(s string) string {
		// TODO: skip strings like {r4} etc. that look like ARM push/pop
		// instructions.
		name := s[1 : len(s)-1]
		value, ok := registers[name]
		if !ok {
			panic(fmt.Sprintf("asm: register not found: %s", name))
		}
		if _, ok := registerNumbers[name]; !ok {
			// Type checking - only allow integer basic types
			rawType := value.Type.RawType()
			if basic, ok := rawType.Underlying().(*types.Basic); ok && basic.Info()&types.IsInteger != 0 {
				registerNumbers[name] = len(registerNumbers)
				inputValues = append(inputValues, value)
				constraints = append(constraints, "r")
			} else {
				// Pointer operands support was dropped, following TinyGo
				// NOTE(tinygo): Memory references require a type starting with LLVM 14, probably as a preparation for opaque pointers.
				panic(fmt.Sprintf("asm: unsupported type in inline assembly for operand: %s, only integer types are supported", name))
			}
		}
		return fmt.Sprintf("${%v}", registerNumbers[name])
	})

	constraintStr := strings.Join(constraints, ",")
	dbgInstrf("asm: %q -> %q, constraints: %q", asmString, finalAsm, constraintStr)

	if !hasOutput {
		// Make sure we return something valid
		b.InlineAsmFull(finalAsm, constraintStr, b.Prog.Void(), inputValues)
		return b.Prog.Val((uintptr(0)))
	}

	return b.InlineAsmFull(finalAsm, constraintStr, b.Prog.Uintptr(), inputValues)
}

// -----------------------------------------------------------------------------

// func _Cfunc_CString(s string) *int8
func (p *context) cgoCString(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		return b.CString(p.compileValue(b, args[0]))
	}
	panic("cgoCString(string): invalid arguments")
}

// func _Cfunc_CBytes(bytes []byte) *int8
func (p *context) cgoCBytes(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		return b.CBytes(p.compileValue(b, args[0]))
	}
	panic("cgoCBytes([]byte): invalid arguments")
}

// func _Cfunc_GoString(s *int8) string
func (p *context) cgoGoString(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		return b.GoString(p.compileValue(b, args[0]))
	}
	panic("cgoGoString(<cstr>): invalid arguments")
}

// func _Cfunc_GoStringN(s *int8, n int) string
func (p *context) cgoGoStringN(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		return b.GoStringN(p.compileValue(b, args[0]), p.compileValue(b, args[1]))
	}
	panic("cgoGoStringN(<cstr>, n int): invalid arguments")
}

// func _Cfunc_GoBytes(s *int8, n int) []byte
func (p *context) cgoGoBytes(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		return b.GoBytes(p.compileValue(b, args[0]), p.compileValue(b, args[1]))
	}
	panic("cgoGoBytes(<cstr>, n int): invalid arguments")
}

// func _Cfunc__CMalloc(n int) unsafe.Pointer
func (p *context) cgoCMalloc(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		return b.CMalloc(p.compileValue(b, args[0]))
	}
	panic("cgoCMalloc(n int): invalid arguments")
}

// func _cgoCheckPointer(ptr any, arg any)
func (p *context) cgoCheckPointer(b llssa.Builder, args []ssa.Value) {
	// don't need to do anything
}

// func _cgo_runtime_cgocall(fn unsafe.Pointer, arg unsafe.Pointer) int
func (p *context) cgoCgocall(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	fnName := p.fn.Name()
	if dot := strings.LastIndex(fnName, "."); dot >= 0 {
		fnName = fnName[dot+1:]
	}
	isC2 := isCgoC2func(fnName)
	var sig *types.Signature
	if isC2 {
		sig = p.fn.Type.RawType().(*types.Signature)
	}
	if len(p.cgoArgs) == 0 && isC2 {
		n := sig.Params().Len()
		p.cgoArgs = make([]llssa.Expr, n)
		for i := 0; i < n; i++ {
			p.cgoArgs[i] = b.Param(i)
		}
	}

	pfn := p.compileValue(b, args[0])
	fnTy := p.fn.Type
	if isC2 {
		if sig.Results().Len() == 0 {
			panic("cgo C2func should have at least one result")
		}
		directSig := types.NewSignatureType(nil, nil, nil, sig.Params(),
			types.NewTuple(types.NewVar(token.NoPos, nil, "", sig.Results().At(0).Type())), false)
		fnTy = p.type_(directSig, llssa.InC)
	}
	pfn.Type = p.prog.Pointer(fnTy)
	fn := b.Load(pfn)
	p.cgoRet = b.Call(fn, p.cgoArgs...)

	if isC2 {
		errnoSig := types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32])), false)
		errnoFn := b.Pkg.NewFunc("cliteErrno", errnoSig, llssa.InC)
		p.cgoErrno = b.Call(errnoFn.Expr)
		return p.cgoErrno
	}
	i32 := p.type_(types.Typ[types.Int32], llssa.InGo)
	p.cgoErrno = p.prog.Zero(i32)
	return p.cgoErrno
}

// -----------------------------------------------------------------------------

// func index(arr *T, idx int) T
func (p *context) index(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	return b.Load(p.advance(b, args))
}

// func advance(ptr *T, offset int) *T
func (p *context) advance(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		ptr := p.compileValue(b, args[0])
		offset := p.compileValue(b, args[1])
		return b.Advance(ptr, offset)
	}
	panic("advance(p ptr, offset int): invalid arguments")
}

// func alloca(size uintptr) unsafe.Pointer
func (p *context) alloca(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		n := p.compileValue(b, args[0])
		return b.Alloca(n)
	}
	panic("alloca(size uintptr): invalid arguments")
}

// func allocaCStr(s string) *int8
func (p *context) allocaCStr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		s := p.compileValue(b, args[0])
		return b.AllocaCStr(s)
	}
	panic("allocaCStr(s string): invalid arguments")
}

// func allocCStr(s string) *int8
func (p *context) allocCStr(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		s := p.compileValue(b, args[0])
		return b.AllocCStr(s)
	}
	panic("allocCStr(s string): invalid arguments")
}

// func allocaCStrs(strs []string, endWithNil bool) **int8
func (p *context) allocaCStrs(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		endWithNil, ok := constBool(args[1])
		if !ok {
			panic("allocaCStrs(strs, endWithNil): endWithNil should be constant bool")
		}
		strs := p.compileValue(b, args[0])
		return b.AllocaCStrs(strs, endWithNil)
	}
	panic("allocaCStrs(strs []string, endWithNil bool): invalid arguments")
}

// func string(cstr *int8, n ...int) *int8
func (p *context) string(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		cstr := p.compileValue(b, args[0])
		n := make([]llssa.Expr, 0, 1)
		n = p.compileVArg(n, b, args[1])
		return b.MakeString(cstr, n...)
	}
	panic("string(cstr *int8, n ...int): invalid arguments")
}

// func stringData(s string) *int8
func (p *context) stringData(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 1 {
		s := p.compileValue(b, args[0])
		return b.StringData(s)
	}
	panic("stringData(s string): invalid arguments")
}

// func funcAddr(fn any) unsafe.Pointer
func (p *context) funcAddr(b llssa.Builder, args []ssa.Value) llssa.Expr {
	if len(args) == 1 {
		if fn, ok := args[0].(*ssa.MakeInterface); ok {
			switch f := fn.X.(type) {
			case *ssa.Function:
				if aFn, _, _ := p.compileFunction(f); aFn != nil {
					return aFn.Expr
				}
			default:
				v := p.compileValue(b, f)
				if _, ok := v.Type.RawType().Underlying().(*types.Signature); ok {
					return v
				}
			}
		}
	}
	panic("funcAddr(<func>): invalid arguments")
}

// func funcPCABI0(fn any) uintptr
func (p *context) funcPCABI0(b llssa.Builder, args []ssa.Value) llssa.Expr {
	return p.funcPCABI0Value(b, args[0])
}

func (p *context) funcPCABI0Value(b llssa.Builder, v ssa.Value) llssa.Expr {
	switch v := v.(type) {
	case *ssa.MakeInterface:
		return p.funcPCABI0Value(b, v.X)
	case *ssa.Function:
		if cname := extractTrampolineCName(v.Name()); cname != "" {
			cname = p.remapTrampolineCName(cname)
			fnSig := p.syscallFnSig(len(v.Params))
			cfn := b.Pkg.NewFunc(cname, fnSig, llssa.InC)
			return b.Convert(p.type_(types.Typ[types.Uintptr], llssa.InGo), cfn.Expr)
		}
		if aFn, _, _ := p.compileFunction(v); aFn != nil {
			return b.Convert(p.type_(types.Typ[types.Uintptr], llssa.InGo), aFn.Expr)
		}
	case *ssa.MakeClosure:
		return p.funcPCABI0Value(b, v.Fn)
	default:
		if t := v.Type(); t != nil {
			if _, ok := t.Underlying().(*types.Interface); ok {
				data := b.InterfaceData(p.compileValue(b, v))
				uptr := p.type_(types.Typ[types.Uintptr], llssa.InGo)
				uptrPtr := p.prog.Pointer(uptr)
				codePtr := b.Convert(uptrPtr, data)
				return b.Load(codePtr)
			}
		}
	}
	panic("funcPCABI0(<func>): invalid arguments")
}

// zeroResult returns the zero value for the intrinsic's result tuple.
// Some intrinsics are specified as returning a tuple; in that case we need to
// materialize a typed zero value of the tuple type.
func (p *context) zeroResult(results *types.Tuple) llssa.Expr {
	if results.Len() == 1 {
		return p.prog.Zero(p.type_(results.At(0).Type(), llssa.InGo))
	}
	return p.prog.Zero(p.type_(results, llssa.InGo))
}

// syscallFnSig returns a variadic C signature with argc uintptr parameters
// followed by a varargs marker, and returning a uintptr.
func (p *context) syscallFnSig(argc int) *types.Signature {
	params := make([]*types.Var, 0, argc+1)
	for i := 0; i < argc; i++ {
		params = append(params, types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr]))
	}
	params = append(params, llssa.VArg())
	ret := types.NewVar(token.NoPos, nil, "", types.Typ[types.Uintptr])
	return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), types.NewTuple(ret), true)
}

// syscallFnSigFixed returns a non-variadic C signature with argc uintptr
// parameters and returning a uintptr.
func (p *context) syscallFnSigFixed(argc int) *types.Signature {
	params := make([]*types.Var, 0, argc)
	for i := 0; i < argc; i++ {
		params = append(params, types.NewParam(token.NoPos, nil, "", types.Typ[types.Uintptr]))
	}
	ret := types.NewVar(token.NoPos, nil, "", types.Typ[types.Uintptr])
	return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), types.NewTuple(ret), false)
}

var errnoSig = types.NewSignatureType(nil, nil, nil, nil,
	types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int32])), false)

// syscallErrno returns errno (as uintptr) if r1 is -1, otherwise 0.
// This matches the common libc syscall convention used by our llgo.syscall
// intrinsic lowering.
func (p *context) syscallErrno(b llssa.Builder, r1 llssa.Expr) llssa.Expr {
	uptr := p.type_(types.Typ[types.Uintptr], llssa.InGo)
	minus1 := p.prog.IntVal(^uint64(0), uptr)
	cond := b.BinOp(token.EQL, r1, minus1)
	errnoFn := b.Pkg.NewFunc("cliteErrno", errnoSig, llssa.InC)
	errno := b.Call(errnoFn.Expr)
	errno = b.Convert(uptr, errno)
	zero := p.prog.Zero(uptr)
	return b.SelectValue(cond, errno, zero)
}

// syscallIntrinsic implements the llgo.syscall intrinsic for libc-based syscalls.
// The first argument is treated as a function pointer, called with the remaining
// arguments (as uintptr), and it returns a (r1, r2, errno) tuple. r2 is always 0
// and errno is set iff r1 == -1.
func (p *context) syscallIntrinsic(b llssa.Builder, args []ssa.Value, results *types.Tuple) llssa.Expr {
	if len(args) < 1 {
		panic("syscall: missing arguments")
	}
	callArgs := make([]llssa.Expr, 0, len(args)-1)
	for _, arg := range args[1:] {
		callArgs = append(callArgs, p.compileValue(b, arg))
	}
	fnSig := p.syscallFnSigFixed(len(callArgs))
	fnArg := p.compileValue(b, args[0])
	fnType := p.type_(fnSig, llssa.InC)
	fnPtr := b.PtrCast(fnType, b.Convert(p.type_(types.Typ[types.UnsafePointer], llssa.InGo), fnArg))
	r1 := b.Call(fnPtr, callArgs...)
	uptr := p.type_(types.Typ[types.Uintptr], llssa.InGo)
	r2 := p.prog.Zero(uptr)
	err := p.syscallErrno(b, r1)
	tuple := p.type_(results, llssa.InGo)
	return b.Aggregate(tuple, r1, r2, err)
}

// Darwin syscall trampoline remap rationale:
// - open/openat/fcntl/ioctl: avoid varargs/ABI width mismatches when calling through llgo.syscall.
// - fdopendir/readdir_r (plus closedir for call-chain consistency): avoid darwin/amd64 INODE64 symbol-name mismatches.
// These are routed to llgo_* wrappers in runtime/internal/clite/os/_os/os_darwin.c.
var darwinTrampolineCNameMap = map[string]string{
	"open":      "llgo_open",
	"openat":    "llgo_openat",
	"fcntl":     "llgo_fcntl",
	"ioctl":     "llgo_ioctl",
	"fdopendir": "llgo_fdopendir",
	"closedir":  "llgo_closedir",
	"readdir_r": "llgo_readdir_r",
}

func (p *context) remapTrampolineCName(name string) string {
	if p.prog.Target().GOOS == "darwin" {
		if v, ok := darwinTrampolineCNameMap[name]; ok {
			return v
		}
	}
	return name
}

func (p *context) sigsetjmp(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		jb := p.compileValue(b, args[0])
		savemask := p.compileValue(b, args[1])
		return b.Sigsetjmp(jb, savemask)
	}
	panic("sigsetjmp(jb c.SigjmpBuf, savemask c.Int): invalid arguments")
}

func (p *context) siglongjmp(b llssa.Builder, args []ssa.Value) {
	if len(args) == 2 {
		jb := p.compileValue(b, args[0])
		retval := p.compileValue(b, args[1])
		b.Siglongjmp(jb, retval)
		return
	}
	panic("siglongjmp(jb c.SigjmpBuf, retval c.Int): invalid arguments")
}

func (p *context) atomic(b llssa.Builder, op llssa.AtomicOp, args []llssa.Expr) (ret llssa.Expr) {
	if len(args) == 2 {
		addr := args[0]
		val := args[1]
		return b.Atomic(op, addr, val)
	}
	panic("atomicOp(addr *T, val T) T: invalid arguments")
}

func (p *context) atomicLoad(b llssa.Builder, args []llssa.Expr) llssa.Expr {
	if len(args) == 1 {
		addr := args[0]
		return b.Load(addr).SetOrdering(llssa.OrderingSeqConsistent)
	}
	panic("atomicLoad(addr *T) T: invalid arguments")
}

func (p *context) atomicStore(b llssa.Builder, args []llssa.Expr) llssa.Expr {
	if len(args) == 2 {
		addr := args[0]
		val := args[1]
		return b.Store(addr, val).SetOrdering(llssa.OrderingSeqConsistent)
	}
	panic("atomicStore(addr *T, val T) T: invalid arguments")
}

func (p *context) atomicCmpXchg(b llssa.Builder, args []llssa.Expr) llssa.Expr {
	if len(args) == 3 {
		addr := args[0]
		old := args[1]
		new := args[2]
		return b.AtomicCmpXchg(addr, old, new)
	}
	panic("atomicCmpXchg(addr *T, old, new T) T: invalid arguments")
}

func (p *context) atomicCmpXchgOK(b llssa.Builder, args []llssa.Expr) llssa.Expr {
	ret := p.atomicCmpXchg(b, args)
	return b.Extract(ret, 1)
}

func (p *context) boolToUint8(b llssa.Builder, args []llssa.Expr) llssa.Expr {
	if len(args) == 1 {
		retType := p.type_(types.Typ[types.Uint8], llssa.InGo)
		one := p.prog.IntVal(1, retType)
		zero := p.prog.Zero(retType)
		return b.SelectValue(args[0], one, zero)
	}
	panic("boolToUint8(b bool) uint8: invalid arguments")
}

// -----------------------------------------------------------------------------

var llgoInstrs = map[string]int{
	"cstr":        llgoCstr,
	"advance":     llgoAdvance,
	"index":       llgoIndex,
	"alloca":      llgoAlloca,
	"allocCStr":   llgoAllocCStr,
	"allocaCStr":  llgoAllocaCStr,
	"allocaCStrs": llgoAllocaCStrs,
	"string":      llgoString,
	"stringData":  llgoStringData,
	"funcAddr":    llgoFuncAddr,
	"funcPCABI0":  llgoFuncPCABI0,
	"skip":        llgoSkip,
	"syscall":     llgoSyscall,
	"boolToUint8": llgoBoolToUint8,
	"pystr":       llgoPyStr,
	"pyList":      llgoPyList,
	"pyTuple":     llgoPyTuple,
	"sigjmpbuf":   llgoSigjmpbuf,
	"sigsetjmp":   llgoSigsetjmp,
	"siglongjmp":  llgoSiglongjmp,
	"deferData":   llgoDeferData,
	"unreachable": llgoUnreachable,

	"atomicLoad":         llgoAtomicLoad,
	"atomicStore":        llgoAtomicStore,
	"atomicCmpXchg":      llgoAtomicCmpXchg,
	"atomicCmpXchgOK":    llgoAtomicCmpXchgOK,
	"atomicAddReturnNew": llgoAtomicAddReturnNew,

	"atomicXchg": int(llgoAtomicXchg),
	"atomicAdd":  int(llgoAtomicAdd),
	"atomicSub":  int(llgoAtomicSub),
	"atomicAnd":  int(llgoAtomicAnd),
	"atomicNand": int(llgoAtomicNand),
	"atomicOr":   int(llgoAtomicOr),
	"atomicXor":  int(llgoAtomicXor),
	"atomicMax":  int(llgoAtomicMax),
	"atomicMin":  int(llgoAtomicMin),
	"atomicUMax": int(llgoAtomicUMax),
	"atomicUMin": int(llgoAtomicUMin),

	"_Cfunc_CString":       llgoCgoCString,
	"_Cfunc_CBytes":        llgoCgoCBytes,
	"_Cfunc_GoString":      llgoCgoGoString,
	"_Cfunc_GoStringN":     llgoCgoGoStringN,
	"_Cfunc_GoBytes":       llgoCgoGoBytes,
	"_Cfunc__CMalloc":      llgoCgoCMalloc,
	"_cgoCheckPointer":     llgoCgoCheckPointer,
	"_cgo_runtime_cgocall": llgoCgoCgocall,

	"asm":       llgoAsm,
	"stackSave": llgoStackSave,
}

// funcOf returns a function by name and set ftype = goFunc, cFunc, etc.
// or returns nil and set ftype = llgoCstr, llgoAlloca, llgoUnreachable, etc.
func (p *context) funcOf(fn *ssa.Function) (aFn llssa.Function, pyFn llssa.PyObjRef, ftype int) {
	pkgTypes, name, ftype := p.funcName(fn)
	switch ftype {
	case pyFunc:
		if kind, mod := pkgKindByScope(pkgTypes.Scope()); kind == PkgPyModule {
			pkg := p.pkg
			fnName := pysymPrefix + mod + "." + name
			if pyFn = pkg.PyObjOf(fnName); pyFn == nil {
				pyFn = pkg.PyNewFunc(fnName, fn.Signature, true)
			}
			return
		}
		ftype = ignoredFunc
	case llgoInstr:
		if ftype = llgoInstrs[name]; ftype == 0 {
			panic("unknown llgo instruction: " + name)
		}
	default:
		pkg := p.pkg
		if aFn = pkg.FuncOf(name); aFn == nil {
			if len(fn.FreeVars) > 0 {
				return nil, nil, ignoredFunc
			}
			sig := p.patchType(fn.Signature).(*types.Signature)
			aFn = pkg.NewFuncEx(name, sig, llssa.Background(ftype), false, fn.Origin() != nil)
			if disableInline {
				aFn.Inline(llssa.NoInline)
			}
		}
	}
	return
}

// -----------------------------------------------------------------------------

const (
	fnNormal = iota
	fnHasVArg
	fnIgnore
)

func (p *context) funcKind(vfn ssa.Value) int {
	if fn, ok := vfn.(*ssa.Function); ok {
		params := fn.Signature.Params()
		n := params.Len()
		if n == 0 {
			if fn.Signature.Recv() == nil {
				if fn.Name() == "init" && p.pkgNoInit(fn.Pkg.Pkg) {
					return fnIgnore
				}
			}
		} else {
			last := params.At(n - 1)
			if last.Name() == llssa.NameValist {
				return fnHasVArg
			}
		}
	}
	return fnNormal
}

func (p *context) pkgNoInit(pkg *types.Package) bool {
	p.ensureLoaded(pkg)
	if i, ok := p.loaded[pkg]; ok {
		return i.kind >= PkgNoInit
	}
	return false
}

func (p *context) offsetOfBuiltinArg(arg ssa.Value) (llssa.Expr, bool) {
	load, ok := arg.(*ssa.UnOp)
	if !ok || load.Op != token.MUL {
		return llssa.Expr{}, false
	}
	field, ok := load.X.(*ssa.FieldAddr)
	if !ok {
		return llssa.Expr{}, false
	}
	offset, ok := p.offsetOfFieldChain(field)
	if !ok {
		return llssa.Expr{}, false
	}
	return p.prog.Val(offset), true
}

func (p *context) offsetOfFieldChain(field *ssa.FieldAddr) (int, bool) {
	offset, ok := p.offsetOfFieldAddr(field)
	if !ok {
		return 0, false
	}
	for {
		parent, ok := field.X.(*ssa.FieldAddr)
		if !ok || p.isExplicitFieldAddr(parent) {
			return offset, true
		}
		parentOffset, _ := p.offsetOfFieldAddr(parent)
		offset += parentOffset
		field = parent
	}
}

func (p *context) offsetOfFieldAddr(field *ssa.FieldAddr) (int, bool) {
	ptr, _, ok := fieldAddrStruct(field)
	if !ok {
		return 0, false
	}
	typ := p.type_(ptr.Elem(), llssa.InGo)
	return int(p.prog.OffsetOf(typ, field.Field)), true
}

func (p *context) isExplicitFieldAddr(field *ssa.FieldAddr) bool {
	name, ok := fieldAddrName(field)
	if !ok {
		return true
	}
	pos := p.fset.Position(field.Pos())
	if pos.Filename == "" || pos.Line <= 0 || pos.Column <= 0 {
		return false
	}
	line, ok := p.sourceLine(pos.Filename, pos.Line)
	if !ok {
		return false
	}
	col := pos.Column - 1
	if col < 0 || col >= len(line) {
		return false
	}
	return strings.HasPrefix(line[col:], name)
}

func fieldAddrStruct(field *ssa.FieldAddr) (*types.Pointer, *types.Struct, bool) {
	if field.X == nil {
		return nil, nil, false
	}
	ptr, ok := field.X.Type().Underlying().(*types.Pointer)
	if !ok {
		return nil, nil, false
	}
	st, ok := ptr.Elem().Underlying().(*types.Struct)
	if !ok || field.Field < 0 || field.Field >= st.NumFields() {
		return nil, nil, false
	}
	return ptr, st, true
}

func fieldAddrName(field *ssa.FieldAddr) (string, bool) {
	_, st, ok := fieldAddrStruct(field)
	if !ok {
		return "", false
	}
	return st.Field(field.Field).Name(), true
}

func (p *context) sourceLine(filename string, line int) (string, bool) {
	if p.srcLines == nil {
		p.srcLines = make(map[string][]string)
	}
	lines, ok := p.srcLines[filename]
	if !ok {
		data, err := os.ReadFile(filename)
		if err != nil {
			return "", false
		}
		lines = strings.Split(string(data), "\n")
		p.srcLines[filename] = lines
	}
	if line <= 0 || line > len(lines) {
		return "", false
	}
	return lines[line-1], true
}

// -----------------------------------------------------------------------------

type explicitDeferStack struct {
	stack llssa.Expr
	owner llssa.Function
}

func (p *context) call(b llssa.Builder, act llssa.DoAction, call *ssa.CallCommon) (ret llssa.Expr) {
	return p.callEx(b, act, call, nil)
}

func (p *context) callDeferStack(b llssa.Builder, act llssa.DoAction, call *ssa.CallCommon, stack ssa.Value, fn *ssa.Function) (ret llssa.Expr) {
	return p.callEx(b, act, call, &explicitDeferStack{
		stack: p.compileValue(b, stack),
		owner: p.deferStackOwner(fn),
	})
}

// Range-over-func yield closures defer into their enclosing non-synthetic
// function frame, so walk past synthetic wrappers before selecting the owner.
// If the owner function has not been compiled yet, this helper will compile it
// lazily so the explicit defer stack has a concrete LLVM function to target.
func (p *context) deferStackOwner(fn *ssa.Function) llssa.Function {
	for fn != nil && fn.Synthetic != "" {
		fn = fn.Parent()
	}
	if fn == nil {
		return nil
	}
	if owner := p.funcs[fn]; owner != nil {
		return owner
	}
	owner, _, kind := p.compileFunction(fn)
	if kind == ignoredFunc {
		return nil
	}
	return owner
}

func (p *context) emitDo(b llssa.Builder, act llssa.DoAction, ds *explicitDeferStack, fn llssa.Expr, buildCall func(llssa.Builder, llssa.Expr, ...llssa.Expr) llssa.Expr, args ...llssa.Expr) llssa.Expr {
	if ds != nil {
		b.DeferTo(ds.owner, ds.stack, fn, buildCall, args...)
		return llssa.Nil
	}
	return b.Do(act, fn, buildCall, args...)
}

func (p *context) callEx(b llssa.Builder, act llssa.DoAction, call *ssa.CallCommon, ds *explicitDeferStack) (ret llssa.Expr) {
	p.markReflectMethodCall(call)
	cv := call.Value
	if mthd := call.Method; mthd != nil {
		o := p.compileValue(b, cv)
		fn := b.Imethod(o, mthd)
		hasVArg := fnNormal
		if llssa.HasNameValist(call.Signature()) {
			hasVArg = fnHasVArg
		}
		args := p.compileValues(b, call.Args, hasVArg)
		ret = p.emitDo(b, act, ds, fn, llssa.Builder.Call, args...)
		return
	}
	kind := p.funcKind(cv)
	if kind == fnIgnore {
		return
	}
	args := call.Args
	dbgGoSSAln(">>> Do", act, cv, args)
	switch cv := cv.(type) {
	case *ssa.Builtin:
		fn := cv.Name()
		if fn == "ssa:wrapnilchk" { // TODO(xsw): check nil ptr
			arg := args[0]
			ret = p.compileValue(b, arg)
			return
		} else if fn == "Offsetof" && act == llssa.Call {
			if offset, ok := p.offsetOfBuiltinArg(args[0]); ok {
				ret = offset
				return
			}
		}
		args := p.compileValues(b, args, kind)
		ret = p.emitDo(b, act, ds, llssa.Builtin(fn), llssa.Builder.Call, args...)
	case *ssa.Function:
		aFn, pyFn, ftype := p.compileFunction(cv)
		// TODO(xsw): check ca != llssa.Call
		switch ftype {
		case cFunc:
			p.inCFunc = true
			args := p.compileValues(b, args, kind)
			p.inCFunc = false
			ret = p.emitDo(b, act, ds, aFn.Expr, llssa.Builder.Call, args...)
		case goFunc:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, aFn.Expr, llssa.Builder.Call, args...)
		case pyFunc:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, pyFn.Expr, llssa.Builder.Call, args...)
		case llgoPyList:
			args := p.compileValues(b, args, fnHasVArg)
			ret = b.PyList(args...)
		case llgoPyTuple:
			args := p.compileValues(b, args, fnHasVArg)
			ret = b.PyTuple(args...)
		case llgoPyStr:
			ret = pystr(b, args)
		case llgoCstr:
			ret = cstr(b, args)
		case llgoAsm:
			ret = p.asm(b, args)
		case llgoCgoCString:
			ret = p.cgoCString(b, args)
		case llgoCgoCBytes:
			ret = p.cgoCBytes(b, args)
		case llgoCgoGoString:
			ret = p.cgoGoString(b, args)
		case llgoCgoGoStringN:
			ret = p.cgoGoStringN(b, args)
		case llgoCgoGoBytes:
			ret = p.cgoGoBytes(b, args)
		case llgoCgoCMalloc:
			ret = p.cgoCMalloc(b, args)
		case llgoCgoCheckPointer:
			p.cgoCheckPointer(b, args)
		case llgoCgoCgocall:
			ret = p.cgoCgocall(b, args)
		case llgoAdvance:
			ret = p.advance(b, args)
		case llgoIndex:
			ret = p.index(b, args)
		case llgoAlloca:
			ret = p.alloca(b, args)
		case llgoAllocaCStr:
			ret = p.allocaCStr(b, args)
		case llgoAllocCStr:
			ret = p.allocCStr(b, args)
		case llgoAllocaCStrs:
			ret = p.allocaCStrs(b, args)
		case llgoString:
			ret = p.string(b, args)
		case llgoStringData:
			ret = p.stringData(b, args)
		case llgoSigsetjmp:
			ret = p.sigsetjmp(b, args)
		case llgoSiglongjmp:
			p.siglongjmp(b, args)
		case llgoStackSave:
			ret = b.StackSave()
		case llgoSigjmpbuf: // func sigjmpbuf()
			ret = b.AllocaSigjmpBuf()
		case llgoDeferData: // func deferData() *Defer
			ret = b.DeferData()
		case llgoFuncAddr:
			ret = p.funcAddr(b, args)
		case llgoFuncPCABI0:
			ret = p.funcPCABI0(b, args)
		case llgoSkip:
			if results := call.Signature().Results(); results.Len() != 0 {
				ret = p.zeroResult(results)
			}
		case llgoSyscall:
			ret = p.syscallIntrinsic(b, args, call.Signature().Results())
		case llgoBoolToUint8:
			args := p.compileValues(b, args, kind)
			ret = b.Do(act, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.boolToUint8(b, args)
			}, args...)
		case llgoUnreachable: // func unreachable()
			b.Unreachable()
		case llgoAtomicLoad:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicLoad(b, args)
			}, args...)
		case llgoAtomicStore:
			args := p.compileValues(b, args, kind)
			p.emitDo(b, act, ds, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicStore(b, args)
			}, args...)
		case llgoAtomicCmpXchg:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicCmpXchg(b, args)
			}, args...)
		case llgoAtomicCmpXchgOK:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicCmpXchgOK(b, args)
			}, args...)
		case llgoAtomicAddReturnNew:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return b.BinOp(token.ADD, p.atomic(b, llssa.OpAdd, args), args[1])
			}, args...)
		default:
			if ftype >= llgoAtomicOpBase && ftype <= llgoAtomicOpLast {
				args := p.compileValues(b, args, kind)
				ret = p.emitDo(b, act, ds, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
					return p.atomic(b, llssa.AtomicOp(ftype-llgoAtomicOpBase), args)
				}, args...)
			} else {
				log.Panicf("unknown ftype: %d for %s", ftype, cv.Name())
			}
		}
	default:
		fn := p.compileValue(b, cv)
		args := p.compileValues(b, args, kind)
		ret = p.emitDo(b, act, ds, fn, llssa.Builder.Call, args...)
	}
	return
}

// -----------------------------------------------------------------------------
