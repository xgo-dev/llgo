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
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/xgo-dev/llgo/internal/genmethod"
	llssa "github.com/xgo-dev/llgo/ssa"
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

func constInt(v ssa.Value) (ret int, ok bool) {
	if c, ok := v.(*ssa.Const); ok {
		if iv, exact := constant.Int64Val(c.Value); exact {
			return int(iv), true
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
		return b.GoStringN(p.compileValue(b, args[0]), b.FitIntSize(p.compileValue(b, args[1])))
	}
	panic("cgoGoStringN(<cstr>, n int): invalid arguments")
}

// func _Cfunc_GoBytes(s *int8, n int) []byte
func (p *context) cgoGoBytes(b llssa.Builder, args []ssa.Value) (ret llssa.Expr) {
	if len(args) == 2 {
		return b.GoBytes(p.compileValue(b, args[0]), b.FitIntSize(p.compileValue(b, args[1])))
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

// syscallFnSigFixed returns a non-variadic C signature with the given parameter
// types and returning a uintptr.
func (p *context) syscallFnSigFixed(paramTypes []types.Type) *types.Signature {
	params := make([]*types.Var, 0, len(paramTypes))
	for _, typ := range paramTypes {
		params = append(params, types.NewParam(token.NoPos, nil, "", typ))
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
// arguments, and it returns a (r1, r2, errno) tuple. r2 is always 0 and errno is
// set iff r1 == -1.
func (p *context) syscallIntrinsic(b llssa.Builder, args []ssa.Value, results *types.Tuple) llssa.Expr {
	if len(args) < 1 {
		panic("syscall: missing arguments")
	}
	callArgs := make([]llssa.Expr, 0, len(args)-1)
	for _, arg := range args[1:] {
		callArgs = append(callArgs, p.compileValue(b, arg))
	}
	callArgTypes := make([]types.Type, 0, len(callArgs))
	for _, arg := range callArgs {
		callArgTypes = append(callArgTypes, arg.RawType())
	}
	fnSig := p.syscallFnSigFixed(callArgTypes)
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
	"open":   "llgo_open",
	"openat": "llgo_openat",
	"fcntl":  "llgo_fcntl",
	"ioctl":  "llgo_ioctl",
}

var darwinTrampolineCNameAmd64Map = map[string]string{
	"fdopendir": "fdopendir$INODE64",
	"readdir_r": "readdir_r$INODE64",
	"getfsstat": "getfsstat$INODE64",
}

func (p *context) remapTrampolineCName(name string) string {
	if p.prog.Target().GOOS == "darwin" {
		if v, ok := darwinTrampolineCNameMap[name]; ok {
			return v
		}
		if p.prog.Target().GOARCH == "amd64" {
			if v, ok := darwinTrampolineCNameAmd64Map[name]; ok {
				return v
			}
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

func (p *context) setjmp(b llssa.Builder, args []ssa.Value) llssa.Expr {
	if len(args) == 1 {
		return b.Setjmp(p.compileValue(b, args[0]))
	}
	panic("setjmp(jb c.JmpBuf): invalid arguments")
}

func (p *context) longjmp(b llssa.Builder, args []ssa.Value) {
	if len(args) == 2 {
		b.Longjmp(p.compileValue(b, args[0]), p.compileValue(b, args[1]))
		return
	}
	panic("longjmp(jb c.JmpBuf, retval c.Int): invalid arguments")
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
	"closureEnv":  llgoClosureEnv,
	"pystr":       llgoPyStr,
	"pyList":      llgoPyList,
	"pyTuple":     llgoPyTuple,
	"sigjmpbuf":   llgoSigjmpbuf,
	"sigsetjmp":   llgoSigsetjmp,
	"siglongjmp":  llgoSiglongjmp,
	"setjmp":      llgoSetjmp,
	"longjmp":     llgoLongjmp,
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
			// Source env-bearing bodies are created by compileFuncDecl before
			// lowering. Imported declarations cannot reconstruct //llgo:env.
			aFn = pkg.NewFuncEx(name, sig, llssa.Background(ftype), false, p.needsLinkOnce(fn))
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
		return PkgSkipsInit(i.kind)
	}
	return false
}

func (p *context) offsetOfBuiltinArg(arg ssa.Value) (llssa.Expr, bool) {
	if field, ok := arg.(*ssa.Field); ok {
		st, ok := field.X.Type().Underlying().(*types.Struct)
		if !ok || field.Field < 0 || field.Field >= st.NumFields() {
			return llssa.Expr{}, false
		}
		typ := p.type_(field.X.Type(), llssa.InGo)
		return p.prog.Val(int(p.prog.OffsetOf(typ, field.Field))), true
	}
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

func (p *context) isAddressOfFieldAddr(field *ssa.FieldAddr) bool {
	if field == nil || p.addrOfFieldAddrs == nil {
		return false
	}
	_, ok := p.addrOfFieldAddrs[field.Pos()]
	return ok
}

func collectAddrOfFieldSelectors(files []*ast.File) map[token.Pos]none {
	ret := make(map[token.Pos]none)
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			unary, ok := n.(*ast.UnaryExpr)
			if !ok || unary.Op != token.AND {
				return true
			}
			collectFieldSelectorChain(ret, unary.X)
			return true
		})
	}
	if len(ret) == 0 {
		return nil
	}
	return ret
}

func collectFieldSelectorChain(ret map[token.Pos]none, expr ast.Expr) {
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.SelectorExpr:
			ret[e.Sel.Pos()] = none{}
			expr = e.X
		default:
			return
		}
	}
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

func (p *context) shouldTrackCallerFrames() bool {
	if p == nil || p.pkg == nil || p.fn == nil || p.goFn == nil || !p.trackCallerFrames {
		return false
	}
	if !p.runtimeCallerFuncs[p.goFn] {
		return false
	}
	if target := p.prog.Target(); target != nil && (target.Target != "" || target.GOARCH == "wasm") {
		return false
	}
	return canTrackCallerFramesForPackage(p.pkg.Path())
}

// canTrackCallerFramesForPackage excludes only the runtime core, whose
// frames are unwinder plumbing rather than user code. Everything else —
// stdlib, third-party, user packages — goes through the same per-package
// analysis: functions that (transitively, within the package) reach a
// runtime.Caller/Callers call must keep physical frames (log.Output,
// slog's Logger.log, testing's decorate chains qualify exactly this way),
// and packages that never read caller pcs track nothing and pay nothing.
func canTrackCallerFramesForPackage(pkgPath string) bool {
	return pkgPath != llssa.PkgRuntime &&
		pkgPath != "runtime" &&
		!strings.HasPrefix(pkgPath, "github.com/xgo-dev/llgo/runtime/internal/")
}

func packageUsesRuntimeCaller(c *CallerTracking, pkg *ssa.Package) bool {
	return len(runtimeCallerFuncSet(c, pkg)) != 0
}

func fnUsesRuntimeCaller(c *CallerTracking, fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	if fn.Pkg == nil {
		return fnHasDirectRuntimeCaller(fn)
	}
	return runtimeCallerFuncSet(c, fn.Pkg)[fn]
}

// runtimeCallerFuncSet is the per-package tracking set: functions that
// must keep physical frames (noinline, no tail calls) and get statement
// anchors at their call sites. Five criteria feed it:
//
//  1. the function (transitively, within the package) reaches a
//     runtime.Caller/Callers call — it consumes caller pcs itself;
//  2. the function statically calls another package's pc-consuming
//     function (log.Println, slog methods, t.Errorf, ...) — its frame is
//     what the callee's fixed Caller depth attributes, so inlining it
//     would both mis-attribute the location and, on ELF, drop the
//     function symbol its pcline sections are link-ordered to.
//  3. program-unique functions (main.main and package init functions)
//     already have one logical instance, so retaining their frame is free;
//  4. //go:noinline functions already retain their frame, so emitting
//     statement anchors adds no further inlining cost;
//  5. the function can run below a defer that consumes panic pcs — recover
//     exposes the panicked call chain after longjmp has removed those physical
//     frames, so the compiler must keep and annotate the possible callees.
//
// Criterion 2 tests membership against the callee package's *base* set
// (criterion 1 alone), so tracking extends exactly one call level past a
// pc-consuming package and does not cascade through arbitrary wrapper
// layers; multi-package wrapper chains remain the P4 inline-tree's job.
func runtimeCallerFuncSet(c *CallerTracking, pkg *ssa.Package) map[*ssa.Function]bool {
	return callerTrackingFuncSetsForPackage(c, pkg).frames
}

// recoverPanicSiteFuncSet is the subset whose implicit panic instructions
// need exact PC-line anchors. Caller consumers, program-unique functions, and
// //go:noinline functions can require stable frames without needing an anchor
// at every potentially panicking instruction.
func recoverPanicSiteFuncSet(c *CallerTracking, pkg *ssa.Package) map[*ssa.Function]bool {
	return callerTrackingFuncSetsForPackage(c, pkg).recoverPanicSites
}

func callerTrackingFuncSetsForPackage(c *CallerTracking, pkg *ssa.Package) callerTrackingFuncSets {
	if pkg == nil {
		return callerTrackingFuncSets{}
	}
	if sets, ok := c.extended[pkg]; ok {
		return sets
	}
	if c.precomputed {
		panic("caller-tracking function set was not precomputed")
	}
	base := runtimeCallerBaseSet(c, pkg)
	funcs, trackable := collectRuntimeCallerFunctions(pkg)
	sets := computeRuntimeCallerFuncSets(c.recoverAnalysis(), pkg, funcs, base, trackable, func(dep *ssa.Package) map[*ssa.Function]bool {
		return runtimeCallerBaseSet(c, dep)
	})
	c.extended[pkg] = sets
	return sets
}

type callerTrackingFuncSets struct {
	frames            map[*ssa.Function]bool
	recoverPanicSites map[*ssa.Function]bool
}

func computeRuntimeCallerFuncSets(recover *recoverFacts, pkg *ssa.Package, funcs, base, trackable map[*ssa.Function]bool, baseSet func(*ssa.Package) map[*ssa.Function]bool) callerTrackingFuncSets {
	frames := make(map[*ssa.Function]bool, len(base))
	for fn := range base {
		frames[fn] = true
	}
	for fn := range trackable {
		if frames[fn] {
			continue
		}
		// Criterion 3: pin program-unique frames. main.main and package
		// init functions run once, so noinline is free — and they are the
		// bottom frames of almost every panic traceback, where an
		// approximate declaration-adjacent line is most visible.
		if isProgramUniqueFrame(pkg, fn) {
			frames[fn] = true
			continue
		}
		// Criterion 4: //go:noinline functions already keep their frames,
		// so statement anchors are free of the usual inlining cost — their
		// panic-traceback lines become exact instead of
		// declaration-adjacent.
		if hasNoInlineDirective(fn) {
			frames[fn] = true
			continue
		}
		forEachCall(fn, func(call *ssa.CallCommon) {
			callee := call.StaticCallee()
			if callee == nil || callee.Pkg == nil || callee.Pkg == pkg {
				return
			}
			if !canTrackCallerFramesForPackage(callee.Pkg.Pkg.Path()) {
				return
			}
			if baseSet(callee.Pkg)[callee] {
				frames[fn] = true
			}
		})
	}
	recoverPanicSites := addRecoverObservableCallees(recover, pkg, funcs, base, frames, trackable, baseSet)
	if len(frames) == 0 {
		frames = nil
	}
	return callerTrackingFuncSets{frames: frames, recoverPanicSites: recoverPanicSites}
}

// addRecoverObservableCallees keeps the same-package synchronous call/defer
// subtree below a defer that can inspect caller pcs. These frames are no
// longer physically live when the deferred function runs after recover;
// runtime reconstructs them from the panic snapshot, so allowing LLVM to
// inline them would lose the function identity and its panic-site line. These
// functions are added to frames and also returned as a distinct set: only the
// returned set needs
// anchors at implicit panic instructions.
func addRecoverObservableCallees(recover *recoverFacts, pkg *ssa.Package, funcs, base, frames, trackable map[*ssa.Function]bool, baseSet func(*ssa.Package) map[*ssa.Function]bool) map[*ssa.Function]bool {
	if pkg == nil || len(trackable) == 0 {
		return nil
	}
	analysis := &runtimeCallerAnalysis{
		pkg:       pkg,
		funcs:     funcs,
		trackable: trackable,
		callsites: collectRuntimeCallerCallsites(funcs),
	}
	queue := make([]*ssa.Function, 0)
	seen := make(map[*ssa.Function]bool)
	isRecoverObserver := func(target *ssa.Function) bool {
		if target == nil || !recover.needsRecoverScope(target) {
			return false
		}
		if isRuntimeCallerFrameFunc(target) {
			return true
		}
		targetBase := base
		if target.Pkg != nil && target.Pkg != pkg {
			targetBase = baseSet(target.Pkg)
		}
		return targetBase[target]
	}
	add := func(fn *ssa.Function) {
		if !trackable[fn] || seen[fn] {
			return
		}
		seen[fn] = true
		frames[fn] = true
		queue = append(queue, fn)
	}
	addCallee := func(fn *ssa.Function, deferred bool) {
		// A recovering defer is the observer at the edge of this subtree,
		// not another possible panic site below itself.
		if deferred && isRecoverObserver(fn) {
			return
		}
		add(fn)
	}
	for fn := range trackable {
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				deferInstr, ok := instr.(*ssa.Defer)
				if !ok {
					continue
				}
				targets, resolved := analysis.callTargets(fn, &deferInstr.Call)
				if !resolved {
					continue
				}
				for target := range targets {
					if isRecoverObserver(target) {
						add(fn)
						break
					}
				}
			}
		}
	}
	if len(queue) == 0 {
		return nil
	}

	candidateIndex := newRecoverPanicCandidateIndex(trackable)
	for len(queue) != 0 {
		fn := queue[0]
		queue = queue[1:]
		for _, block := range fn.Blocks {
			for _, instr := range block.Instrs {
				var call *ssa.CallCommon
				isDefer := false
				switch instr := instr.(type) {
				case *ssa.Call:
					call = &instr.Call
				case *ssa.Defer:
					// Non-observing deferred calls still run synchronously in
					// this goroutine while the panic unwinds. Their panic sites
					// can therefore be seen by a later recovering defer.
					call = &instr.Call
					isDefer = true
				case *ssa.Go:
					// A go edge starts an independent goroutine with its own
					// panic/recover chain and cannot be observed by this defer.
					continue
				default:
					continue
				}
				if _, builtin := call.Value.(*ssa.Builtin); builtin {
					continue
				}
				targets, resolved := analysis.callTargets(fn, call)
				if !resolved {
					// The call can target any same-package function compatible
					// with the value. Retaining the package-local candidates is
					// conservative; cross-package callees keep their own policy.
					for _, candidate := range candidateIndex.compatible(call.Signature()) {
						addCallee(candidate, isDefer)
					}
					continue
				}
				for target := range targets {
					addCallee(target, isDefer)
				}
			}
		}
	}
	return seen
}

type recoverPanicCandidateGroup struct {
	signature  *types.Signature
	candidates []*ssa.Function
}

type recoverPanicCandidateIndex map[string][]recoverPanicCandidateGroup

// newRecoverPanicCandidateIndex scans trackable once and buckets candidates by
// fully qualified structural signature. The identical-signature groups retain
// a collision guard without making each unresolved call rescan the package.
func newRecoverPanicCandidateIndex(trackable map[*ssa.Function]bool) recoverPanicCandidateIndex {
	index := make(recoverPanicCandidateIndex)
	for candidate := range trackable {
		signature := candidate.Signature
		key := recoverPanicSignatureKey(signature)
		groups := index[key]
		matched := false
		for i := range groups {
			if types.Identical(signature, groups[i].signature) {
				groups[i].candidates = append(groups[i].candidates, candidate)
				matched = true
				break
			}
		}
		if !matched {
			groups = append(groups, recoverPanicCandidateGroup{
				signature:  signature,
				candidates: []*ssa.Function{candidate},
			})
		}
		index[key] = groups
	}
	return index
}

func (index recoverPanicCandidateIndex) compatible(signature *types.Signature) []*ssa.Function {
	for _, group := range index[recoverPanicSignatureKey(signature)] {
		if types.Identical(signature, group.signature) {
			return group.candidates
		}
	}
	return nil
}

func recoverPanicSignatureKey(signature *types.Signature) string {
	return types.TypeString(signature, func(pkg *types.Package) string {
		return pkg.Path()
	})
}

func (a *runtimeCallerAnalysis) callTargets(fn *ssa.Function, call *ssa.CallCommon) (map[*ssa.Function]bool, bool) {
	if call == nil {
		return nil, false
	}
	if callee := call.StaticCallee(); callee != nil {
		return map[*ssa.Function]bool{callee: true}, true
	}
	if call.Method != nil {
		return a.interfaceMethodTargets(fn, call.Value, call.Method)
	}
	return a.functionValueTargets(fn, call.Value)
}

// CallerTracking memoizes frontend analyses for one compilation. Like Patches,
// it is compilation-scoped state owned by the driver: create one per
// compilation and pass it to every NewPackageExWithEmbed call of that
// compilation, so cross-package queries hit the same caller-tracking and
// recover facts. It must not outlive the compilation — the caches are keyed by
// *ssa.Package and *ssa.Function, so anything longer-lived would pin every
// compiled package's go/types and go/ssa graphs. Concurrent drivers call
// Precompute before workers start; recover facts also synchronize lazy queries
// for nested and synthetic functions that are not package members.
type CallerTracking struct {
	base        map[*ssa.Package]map[*ssa.Function]bool
	extended    map[*ssa.Package]callerTrackingFuncSets
	recover     *recoverFacts
	precomputed bool
}

// Precompute resolves caller-tracking and recover data before package backends
// start. Once it returns, callers may share c for concurrent lookups as long as
// pkgs contains every package that can be passed to this compilation. A later
// caller-tracking lookup miss panics instead of mutating the shared maps;
// recover facts retain their own synchronization for nested and synthetic
// functions that are not package members.
func (c *CallerTracking) Precompute(pkgs []*ssa.Package) {
	if c == nil {
		return
	}
	pkgs = uniqueCallerTrackingPackages(pkgs)
	if c.precomputed {
		for _, pkg := range pkgs {
			if _, ok := c.base[pkg]; !ok {
				panic("caller-tracking base set was not precomputed")
			}
			if _, ok := c.extended[pkg]; !ok {
				panic("caller-tracking function set was not precomputed")
			}
		}
		return
	}
	if len(pkgs) == 0 {
		c.precomputed = true
		return
	}
	c.recoverAnalysis().precompute(pkgs)

	// RuntimeTypes and MethodValue are Program-wide and may lazily populate
	// shared SSA state. Snapshot and materialize them once in stable package
	// order before computing package sets. The remaining analysis is kept
	// serial: interface target resolution may still consult MethodValue, and
	// scheduling those calls concurrently makes later module layout depend on
	// worker completion order.
	runtimeTypes := callerTrackingRuntimeTypes(pkgs)
	methods := callerTrackingMethods(pkgs, runtimeTypes)
	analyses := make([]callerTrackingPackageAnalysis, len(pkgs))
	base := make([]map[*ssa.Function]bool, len(pkgs))
	extended := make([]callerTrackingFuncSets, len(pkgs))
	index := make(map[*ssa.Package]int, len(pkgs))
	for i, pkg := range pkgs {
		index[pkg] = i
	}

	for i := range pkgs {
		analyses[i] = analyzeCallerTrackingPackage(pkgs[i], methods[pkgs[i]])
		base[i] = analyses[i].base
	}
	for i := range pkgs {
		extended[i] = computeRuntimeCallerFuncSets(c.recoverAnalysis(), pkgs[i], analyses[i].funcs, base[i], analyses[i].trackable, func(dep *ssa.Package) map[*ssa.Function]bool {
			j, ok := index[dep]
			if !ok {
				panic("caller-tracking dependency was not precomputed")
			}
			return base[j]
		})
	}

	for i, pkg := range pkgs {
		c.base[pkg] = base[i]
	}
	for i, pkg := range pkgs {
		c.extended[pkg] = extended[i]
	}
	c.precomputed = true
}

type callerTrackingPackageAnalysis struct {
	funcs     map[*ssa.Function]bool
	base      map[*ssa.Function]bool
	trackable map[*ssa.Function]bool
}

func analyzeCallerTrackingPackage(pkg *ssa.Package, methods []*ssa.Function) callerTrackingPackageAnalysis {
	funcs, trackable := collectRuntimeCallerFunctionsWithMethods(pkg, methods)
	analysis := &runtimeCallerAnalysis{
		pkg:       pkg,
		funcs:     funcs,
		trackable: trackable,
		callsites: collectRuntimeCallerCallsites(funcs),
		memo:      make(map[*ssa.Function]bool),
		visiting:  make(map[*ssa.Function]bool),
	}
	return callerTrackingPackageAnalysis{
		funcs:     funcs,
		base:      computeRuntimeCallerBaseSetFromAnalysis(analysis),
		trackable: trackable,
	}
}

// callerTrackingMethods materializes method wrappers in stable package order
// before analysis workers start. Program.MethodValue is thread-safe but may
// lazily create wrappers; doing that in workers would make Program mutation
// order depend on scheduling and could change later module emission order.
func callerTrackingMethods(pkgs []*ssa.Package, runtimeTypes map[*ssa.Package][]types.Type) map[*ssa.Package][]*ssa.Function {
	result := make(map[*ssa.Package][]*ssa.Function, len(pkgs))
	for _, pkg := range pkgs {
		result[pkg] = collectRuntimeCallerMethods(pkg, runtimeTypes[pkg])
	}
	return result
}

// callerTrackingRuntimeTypes snapshots Program-wide runtime types once and
// partitions them by declaring package before workers start. RuntimeTypes
// computes derived types under Program and method-set locks; calling it from
// every package worker would serialize the supposedly parallel phase and would
// repeatedly scan the same whole-program data.
func callerTrackingRuntimeTypes(pkgs []*ssa.Package) map[*ssa.Package][]types.Type {
	byProgram := make(map[*ssa.Program]map[*types.Package][]types.Type)
	result := make(map[*ssa.Package][]types.Type, len(pkgs))
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Prog == nil || pkg.Pkg == nil {
			continue
		}
		byPackage, ok := byProgram[pkg.Prog]
		if !ok {
			byPackage = make(map[*types.Package][]types.Type)
			for _, typ := range pkg.Prog.RuntimeTypes() {
				if owner := typeOwnerPackage(typ); owner != nil {
					byPackage[owner] = append(byPackage[owner], typ)
				}
			}
			byProgram[pkg.Prog] = byPackage
		}
		result[pkg] = byPackage[pkg.Pkg]
	}
	return result
}

func uniqueCallerTrackingPackages(pkgs []*ssa.Package) []*ssa.Package {
	unique := make([]*ssa.Package, 0, len(pkgs))
	seen := make(map[*ssa.Package]bool, len(pkgs))
	for _, pkg := range pkgs {
		if pkg != nil && !seen[pkg] {
			seen[pkg] = true
			unique = append(unique, pkg)
		}
	}
	return unique
}

// NewCallerTracking creates the frontend-analysis caches for one compilation.
func NewCallerTracking() *CallerTracking {
	return &CallerTracking{
		base:     make(map[*ssa.Package]map[*ssa.Function]bool),
		extended: make(map[*ssa.Package]callerTrackingFuncSets),
		recover:  newRecoverFacts(),
	}
}

func isProgramUniqueFrame(pkg *ssa.Package, fn *ssa.Function) bool {
	if fn == nil || fn.Parent() != nil {
		return false
	}
	name := fn.Name()
	if name == "init" || strings.HasPrefix(name, "init#") {
		return true
	}
	return name == "main" && pkg.Pkg != nil && pkg.Pkg.Name() == "main"
}

func runtimeCallerBaseSet(c *CallerTracking, pkg *ssa.Package) map[*ssa.Function]bool {
	if pkg == nil {
		return nil
	}
	if set, ok := c.base[pkg]; ok {
		return set
	}
	if c.precomputed {
		panic("caller-tracking base set was not precomputed")
	}
	set := computeRuntimeCallerBaseSet(pkg)
	c.base[pkg] = set
	return set
}

func computeRuntimeCallerBaseSet(pkg *ssa.Package) map[*ssa.Function]bool {
	var runtimeTypes []types.Type
	if pkg != nil && pkg.Prog != nil {
		for _, typ := range pkg.Prog.RuntimeTypes() {
			if typeBelongsToPackage(typ, pkg.Pkg) {
				runtimeTypes = append(runtimeTypes, typ)
			}
		}
	}
	return analyzeCallerTrackingPackage(pkg, collectRuntimeCallerMethods(pkg, runtimeTypes)).base
}

func computeRuntimeCallerBaseSetFromAnalysis(analysis *runtimeCallerAnalysis) map[*ssa.Function]bool {
	if !analysis.packageHasRuntimeCaller() {
		return nil
	}
	out := make(map[*ssa.Function]bool)
	for {
		ntrack := len(analysis.trackable)
		for fn := range analysis.trackable {
			if analysis.fnMayReachRuntimeCaller(fn) {
				out[fn] = true
			}
		}
		if len(analysis.trackable) == ntrack {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type runtimeCallerAnalysis struct {
	pkg       *ssa.Package
	funcs     map[*ssa.Function]bool
	trackable map[*ssa.Function]bool
	callsites map[*ssa.Function][]*ssa.CallCommon
	memo      map[*ssa.Function]bool
	visiting  map[*ssa.Function]bool
}

func collectRuntimeCallerFunctions(pkg *ssa.Package) (funcs, trackable map[*ssa.Function]bool) {
	var runtimeTypes []types.Type
	if pkg != nil && pkg.Prog != nil {
		runtimeTypes = pkg.Prog.RuntimeTypes()
	}
	return collectRuntimeCallerFunctionsWithMethods(pkg, collectRuntimeCallerMethods(pkg, runtimeTypes))
}

func collectRuntimeCallerMethods(pkg *ssa.Package, runtimeTypes []types.Type) []*ssa.Function {
	if pkg == nil || pkg.Prog == nil {
		return nil
	}
	type methodType struct {
		key string
		typ types.Type
	}
	typesToVisit := make([]methodType, 0)
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
	for _, typ := range runtimeTypes {
		if typeBelongsToPackage(typ, pkg.Pkg) {
			addType(typ)
		}
	}
	sort.Slice(typesToVisit, func(i, j int) bool {
		return typesToVisit[i].key < typesToVisit[j].key
	})

	methods := make([]*ssa.Function, 0)
	seen := make(map[*ssa.Function]bool)
	addMethods := func(typ types.Type) {
		methodSet := pkg.Prog.MethodSets.MethodSet(typ)
		for i := 0; i < methodSet.Len(); i++ {
			meth := methodSet.At(i)
			if genmethod.SupportsGenericMethods && genmethod.IsGenericMethod(meth.Type()) {
				continue
			}
			fn := pkg.Prog.MethodValue(meth)
			if fn != nil && !seen[fn] {
				seen[fn] = true
				methods = append(methods, fn)
			}
		}
	}
	for _, candidate := range typesToVisit {
		addMethods(candidate.typ)
	}
	return methods
}

func collectRuntimeCallerFunctionsWithMethods(pkg *ssa.Package, methods []*ssa.Function) (funcs, trackable map[*ssa.Function]bool) {
	funcs = make(map[*ssa.Function]bool)
	trackable = make(map[*ssa.Function]bool)
	var add func(*ssa.Function, bool) bool
	add = func(fn *ssa.Function, track bool) bool {
		if fn == nil || !functionBelongsToPackage(pkg, fn) {
			return false
		}
		if track {
			trackable[fn] = true
		}
		if funcs[fn] {
			return false
		}
		funcs[fn] = true
		for _, anon := range fn.AnonFuncs {
			// Anonymous functions inherit trackability: one that reaches
			// runtime.Caller needs its own frame (noinline + no tail calls)
			// or physical unwinding loses it.
			add(anon, track)
		}
		return true
	}
	for _, member := range pkg.Members {
		if fn, ok := member.(*ssa.Function); ok {
			add(fn, true)
		}
	}
	// Methods are as trackable as package-level functions: one that
	// (transitively) calls runtime.Caller needs frames and pcline labels of its
	// own. Method wrappers were materialized serially before workers started.
	for _, method := range methods {
		add(method, true)
	}
	for changed := true; changed; {
		changed = false
		for fn := range funcs {
			forEachCall(fn, func(call *ssa.CallCommon) {
				if add(call.StaticCallee(), trackable[fn]) {
					changed = true
				}
			})
		}
	}
	return funcs, trackable
}

func collectRuntimeCallerCallsites(funcs map[*ssa.Function]bool) map[*ssa.Function][]*ssa.CallCommon {
	callsites := make(map[*ssa.Function][]*ssa.CallCommon)
	for fn := range funcs {
		forEachCall(fn, func(call *ssa.CallCommon) {
			callee := call.StaticCallee()
			if funcs[callee] {
				callsites[callee] = append(callsites[callee], call)
			}
		})
	}
	return callsites
}

func functionBelongsToPackage(pkg *ssa.Package, fn *ssa.Function) bool {
	if pkg == nil || fn == nil {
		return false
	}
	if fn.Pkg == pkg {
		return true
	}
	return fn.Pkg == nil && fn.Parent() != nil && functionBelongsToPackage(pkg, fn.Parent())
}

func typeBelongsToPackage(typ types.Type, pkg *types.Package) bool {
	return pkg != nil && typeOwnerPackage(typ) == pkg
}

func typeOwnerPackage(typ types.Type) *types.Package {
	for {
		if ptr, ok := types.Unalias(typ).(*types.Pointer); ok {
			typ = ptr.Elem()
			continue
		}
		break
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj() == nil {
		return nil
	}
	return named.Obj().Pkg()
}

func (a *runtimeCallerAnalysis) packageHasRuntimeCaller() bool {
	for fn := range a.funcs {
		if fnHasDirectRuntimeCaller(fn) {
			return true
		}
	}
	return false
}

func fnHasDirectRuntimeCaller(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			call, ok := instr.(ssa.CallInstruction)
			if !ok {
				continue
			}
			if isRuntimeCallerFrameFunc(call.Common().StaticCallee()) {
				return true
			}
		}
	}
	for _, anon := range fn.AnonFuncs {
		if fnHasDirectRuntimeCaller(anon) {
			return true
		}
	}
	return false
}

func (a *runtimeCallerAnalysis) fnMayReachRuntimeCaller(fn *ssa.Function) bool {
	if fn == nil {
		return false
	}
	if isRuntimeCallerFrameFunc(fn) {
		return true
	}
	if !a.funcs[fn] {
		return false
	}
	if ok, done := a.memo[fn]; done {
		return ok
	}
	if a.visiting[fn] {
		return false
	}
	a.visiting[fn] = true
	defer delete(a.visiting, fn)
	reaches := false
	forEachCall(fn, func(call *ssa.CallCommon) {
		if reaches {
			return
		}
		callee := call.StaticCallee()
		switch {
		case isRuntimeCallerFrameFunc(callee):
			reaches = true
		case callee != nil:
			reaches = a.fnMayReachRuntimeCaller(callee)
		case call.Method != nil:
			reaches = a.interfaceInvokeMayReachRuntimeCaller(fn, call)
		default:
			reaches = a.functionValueCallMayReachRuntimeCaller(fn, call.Value)
		}
	})
	if !reaches {
		for _, anon := range fn.AnonFuncs {
			if a.fnMayReachRuntimeCaller(anon) {
				if a.trackable[fn] {
					a.trackable[anon] = true
				}
				reaches = true
				break
			}
		}
	}
	a.memo[fn] = reaches
	return reaches
}

func (a *runtimeCallerAnalysis) functionValueCallMayReachRuntimeCaller(fn *ssa.Function, value ssa.Value) bool {
	targets, ok := a.functionValueTargets(fn, value)
	if !ok {
		return true
	}
	for target := range targets {
		if a.fnMayReachRuntimeCaller(target) {
			return true
		}
	}
	return false
}

func (a *runtimeCallerAnalysis) functionValueTargets(fn *ssa.Function, value ssa.Value) (map[*ssa.Function]bool, bool) {
	if targets, ok := staticFunctionTargets(value); ok {
		return targets, true
	}
	param, ok := value.(*ssa.Parameter)
	if !ok || param.Parent() != fn {
		return nil, false
	}
	idx, ok := parameterIndex(fn, param)
	if !ok {
		return nil, false
	}
	return a.functionParamTargets(fn, idx)
}

func (a *runtimeCallerAnalysis) functionParamTargets(fn *ssa.Function, idx int) (map[*ssa.Function]bool, bool) {
	callsites := a.callsites[fn]
	if len(callsites) == 0 {
		return nil, false
	}
	targets := make(map[*ssa.Function]bool)
	for _, call := range callsites {
		args := call.Args
		if idx >= len(args) {
			return nil, false
		}
		argTargets, ok := staticFunctionTargets(args[idx])
		if !ok {
			return nil, false
		}
		for target := range argTargets {
			targets[target] = true
		}
	}
	return targets, true
}

func staticFunctionTargets(value ssa.Value) (map[*ssa.Function]bool, bool) {
	switch v := value.(type) {
	case *ssa.Function:
		return map[*ssa.Function]bool{v: true}, true
	case *ssa.MakeClosure:
		if fn, ok := v.Fn.(*ssa.Function); ok {
			return map[*ssa.Function]bool{fn: true}, true
		}
	}
	return nil, false
}

func (a *runtimeCallerAnalysis) interfaceInvokeMayReachRuntimeCaller(fn *ssa.Function, call *ssa.CallCommon) bool {
	targets, ok := a.interfaceMethodTargets(fn, call.Value, call.Method)
	if !ok {
		return true
	}
	for target := range targets {
		if a.fnMayReachRuntimeCaller(target) {
			return true
		}
	}
	return false
}

func (a *runtimeCallerAnalysis) interfaceMethodTargets(fn *ssa.Function, value ssa.Value, method *types.Func) (map[*ssa.Function]bool, bool) {
	if targets, ok := a.staticInterfaceMethodTargets(value, method); ok {
		return targets, true
	}
	param, ok := value.(*ssa.Parameter)
	if !ok || param.Parent() != fn {
		return nil, false
	}
	idx, ok := parameterIndex(fn, param)
	if !ok {
		return nil, false
	}
	callsites := a.callsites[fn]
	if len(callsites) == 0 {
		return nil, false
	}
	targets := make(map[*ssa.Function]bool)
	for _, call := range callsites {
		args := call.Args
		if idx >= len(args) {
			return nil, false
		}
		argTargets, ok := a.staticInterfaceMethodTargets(args[idx], method)
		if !ok {
			return nil, false
		}
		for target := range argTargets {
			targets[target] = true
		}
	}
	return targets, true
}

func (a *runtimeCallerAnalysis) staticInterfaceMethodTargets(value ssa.Value, method *types.Func) (map[*ssa.Function]bool, bool) {
	switch v := value.(type) {
	case *ssa.MakeInterface:
		return a.methodTargetsForType(v.X.Type(), method)
	case *ssa.ChangeInterface:
		return a.staticInterfaceMethodTargets(v.X, method)
	}
	return nil, false
}

func (a *runtimeCallerAnalysis) methodTargetsForType(typ types.Type, method *types.Func) (map[*ssa.Function]bool, bool) {
	if a.pkg == nil || a.pkg.Prog == nil || method == nil {
		return nil, false
	}
	methods := a.pkg.Prog.MethodSets.MethodSet(typ)
	for i := 0; i < methods.Len(); i++ {
		sel := methods.At(i)
		if sel.Obj().Name() != method.Name() {
			continue
		}
		if genmethod.SupportsGenericMethods && genmethod.IsGenericMethod(sel.Type()) {
			continue
		}
		fn := a.pkg.Prog.MethodValue(sel)
		if fn == nil {
			return nil, false
		}
		return map[*ssa.Function]bool{fn: true}, true
	}
	return nil, false
}

func parameterIndex(fn *ssa.Function, param *ssa.Parameter) (int, bool) {
	for i, candidate := range fn.Params {
		if candidate == param {
			return i, true
		}
	}
	return 0, false
}

func forEachCall(fn *ssa.Function, do func(*ssa.CallCommon)) {
	if fn == nil {
		return
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if call, ok := instr.(ssa.CallInstruction); ok {
				do(call.Common())
			}
		}
	}
}

func isRuntimeCallerFunc(fn *ssa.Function) bool {
	if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	switch fn.Pkg.Pkg.Path() {
	case "runtime", "github.com/xgo-dev/llgo/runtime/internal/lib/runtime":
		return isRuntimeCallerName(fn.Name())
	case "runtime/debug":
		return fn.Name() == "Stack"
	default:
		return false
	}
}

func isRuntimeCallerFrameFunc(fn *ssa.Function) bool {
	if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	switch fn.Pkg.Pkg.Path() {
	case "runtime", "github.com/xgo-dev/llgo/runtime/internal/lib/runtime":
		return isRuntimeCallerFrameName(fn.Name())
	case "runtime/debug":
		return fn.Name() == "Stack"
	default:
		return false
	}
}

func isRuntimeCallerLookupFunc(fn *ssa.Function) bool {
	if fn == nil || fn.Pkg == nil || fn.Pkg.Pkg == nil {
		return false
	}
	switch fn.Pkg.Pkg.Path() {
	case "runtime", "github.com/xgo-dev/llgo/runtime/internal/lib/runtime":
		switch fn.Name() {
		case "Caller", "Callers", "Stack":
			return true
		}
	case "runtime/debug":
		return fn.Name() == "Stack"
	}
	return false
}

func isRuntimeCallerName(name string) bool {
	switch name {
	case "Caller", "Callers", "CallersFrames", "FuncForPC", "Stack":
		return true
	default:
		return false
	}
}

func isRuntimeCallerFrameName(name string) bool {
	switch name {
	case "Caller", "Callers", "CallersFrames", "Stack":
		return true
	default:
		return false
	}
}

func (p *context) runtimeCallerFrameName() string {
	if p == nil {
		return ""
	}
	if p.goFn != nil && p.goFn.Pkg != nil && p.goFn.Pkg.Pkg != nil {
		return runtimeFrameName(funcName(p.goFn.Pkg.Pkg, p.goFn, false))
	}
	if p.fn != nil {
		return runtimeFrameName(p.fn.Name())
	}
	return ""
}

func (p *context) pushCallerLocationFrame(b llssa.Builder, fn *ssa.Function) {
	if !p.options.ShadowStack {
		return
	}
	if fn == nil {
		return
	}
	pos := p.fset.Position(fn.Pos())
	pos.Filename = runtimeSourceFilename(
		p.prog.Target(),
		directiveFilename(p.fset, fn.Pos(), pos.Filename, p.sourceLine),
	)
	entry := b.Convert(p.prog.Uintptr(), p.fn.Expr)
	p.callerFrameMark = b.Call(
		p.runtimeFunc("PushCallerLocationFrame", pushCallerLocationFrameSig()),
		entry,
		b.Str(p.runtimeCallerFrameName()),
		b.Str(pos.Filename),
		p.prog.IntVal(uint64(pos.Line), p.prog.Int()),
	)
}

func (p *context) recordCallerLocation(b llssa.Builder, pos token.Pos) {
	p.recordRuntimeLocation(b, pos, "RecordCallerLocation")
}

func (p *context) recordPanicLocation(b llssa.Builder, pos token.Pos) {
	p.recordRuntimeLocation(b, pos, "RecordPanicLocation")
}

func (p *context) recordPanicSite(b llssa.Builder, pos token.Pos) {
	p.recordPanicLocation(b, pos)
	if p.panicSiteFuncs[p.goFn] {
		p.emitPCLineLabel(b, pos)
	}
}

func (p *context) recordRuntimeLocation(b llssa.Builder, pos token.Pos, fn string) {
	if !p.options.ShadowStack || !p.shouldTrackCallerFrames() {
		return
	}
	position := p.fset.Position(pos)
	position.Filename = runtimeSourceFilename(
		p.prog.Target(),
		directiveFilename(p.fset, pos, position.Filename, p.sourceLine),
	)
	if position.Line <= 0 || position.Filename == "" {
		return
	}
	b.Call(
		p.runtimeFunc(fn, recordRuntimeLocationSig()),
		b.Convert(p.prog.Uintptr(), p.fn.Expr),
		b.Str(p.runtimeCallerFrameName()),
		b.Str(position.Filename),
		p.prog.IntVal(uint64(position.Line), p.prog.Int()),
	)
}

func (p *context) recordCallerLocationForCall(b llssa.Builder, call *ssa.CallCommon) {
	if !p.shouldTrackCallerFrames() {
		return
	}
	callee := call.StaticCallee()
	if isRuntimeCallerLookupFunc(callee) {
		p.recordCallerLocation(b, call.Pos())
		return
	}
	p.recordPanicLocation(b, call.Pos())
}

func (p *context) emitPCLineLabel(b llssa.Builder, pos token.Pos) {
	if p == nil || p.pkg == nil || p.fn == nil || !p.prog.FuncInfoSitesEnabled() || !p.shouldTrackCallerFrames() {
		return
	}
	target := p.prog.Target()
	if !canEmitPCLineLabelsForTarget(target) {
		return
	}
	position := p.fset.Position(pos)
	// Normalize before the emptiness check: an empty //line directive
	// filename must anchor as "??" (gc's spelling), not lose its anchor.
	position.Filename = runtimeSourceFilename(
		target,
		directiveFilename(p.fset, pos, position.Filename, p.sourceLine),
	)
	if position.Line <= 0 || position.Filename == "" {
		return
	}
	// Lookup uses the nearest preceding PC anchor. Within one basic block,
	// another anchor for the same runtime-visible file and line cannot change
	// the result, even when several implicit panic checks came from one Go
	// expression.
	if position.Filename == p.lastPCLineFile && position.Line == p.lastPCLineLine {
		return
	}
	p.lastPCLineFile = position.Filename
	p.lastPCLineLine = position.Line
	p.pcLineSeq++
	id := pcLineID(p.fn.Name(), p.pcLineSeq)
	label := pcLineLabelName(id)
	if target.GOOS == "darwin" {
		// Mach-O subsections-via-symbols treats every non-local symbol as an
		// atom boundary; a visible label in the middle of a function body
		// lets the linker split and reorder the function. The "L" prefix
		// keeps the label assembler-local so the function stays one atom.
		label = "L" + label
	}
	asmLabel := label + "_${:uid}"
	ptrDirective := ".quad"
	align := "3"
	if p.prog.PointerSize() == 4 {
		ptrDirective = ".long"
		align = "2"
	}
	// Keep section names in sync with internal/build/funcinfo_table.go
	// (pcLineSiteSectionInfo). ELF ties the record to its final code location
	// via SHF_LINK_ORDER (honored by --gc-sections). Use the local asm label
	// rather than the source function symbol: Full LTO can inline or remove the
	// latter without rewriting symbol names embedded in inline-asm strings.
	// Mach-O uses a live_support section plus one linker-private atom symbol per
	// record so -dead_strip keeps a record exactly when the function containing
	// its label is live. COFF uses an associative COMDAT tied to the function
	// section containing the local anchor, so /OPT:REF has the same behavior.
	pushSection := ".pushsection llgo_pcline,\"awo\",@progbits," + asmLabel
	recordSymbol := ""
	switch target.GOOS {
	case "darwin":
		pushSection = ".pushsection __DATA,__llgo_pcl,regular,live_support"
		recordSymbol = "l_llgo_pcline_rec_${:uid}:\n"
	case "windows":
		// '$' is an inline-asm escape. '$$m' reaches the COFF assembler as
		// the '$m' subsection suffix used for lexicographic merging.
		pushSection = ".pushsection .llgopcl$$m,\"dr\",associative," + asmLabel
	}
	b.InlineAsm(
		asmLabel + ":\n" +
			pushSection + "\n" +
			".p2align " + align + "\n" +
			recordSymbol +
			ptrDirective + " " + asmLabel + "\n" +
			".quad " + uint64Hex(id) + "\n" +
			".popsection",
	)
	p.pkg.EmitPCLineInfo(id, p.fn.Name(), position.Filename, position.Line, position.Column)
}

func canEmitPCLineLabelsForTarget(target *llssa.Target) bool {
	if target == nil {
		return false
	}
	if target.Target != "" || target.GOARCH == "wasm" {
		return false
	}
	// ELF uses SHF_LINK_ORDER associated sections; Mach-O uses live_support;
	// COFF uses associative COMDAT sections.
	switch target.GOOS {
	case "linux", "darwin", "windows":
		return true
	}
	return false
}

func pcLineID(symbol string, seq uint64) uint64 {
	const (
		offset = uint64(14695981039346656037)
		prime  = uint64(1099511628211)
	)
	h := offset
	for i := 0; i < len(symbol); i++ {
		h ^= uint64(symbol[i])
		h *= prime
	}
	for i := 0; i < 8; i++ {
		h ^= byteOfUint64(seq, uint(i*8))
		h *= prime
	}
	if h == 0 {
		return 1
	}
	return h
}

func byteOfUint64(v uint64, shift uint) uint64 {
	return (v >> shift) & 0xff
}

func pcLineLabelName(id uint64) string {
	const hexdigits = "0123456789abcdef"
	var buf [16]byte
	for i := len(buf) - 1; i >= 0; i-- {
		buf[i] = hexdigits[id&0xf]
		id >>= 4
	}
	return "__llgo_pcsite_" + string(buf[:])
}

func uint64Hex(v uint64) string {
	const hexdigits = "0123456789abcdef"
	var buf [18]byte
	buf[0] = '0'
	buf[1] = 'x'
	for i := len(buf) - 1; i >= 2; i-- {
		buf[i] = hexdigits[v&0xf]
		v >>= 4
	}
	return string(buf[:])
}

func asmQuoteSymbol(symbol string) string {
	var b strings.Builder
	b.Grow(len(symbol) + 2)
	b.WriteByte('"')
	for i := 0; i < len(symbol); i++ {
		switch symbol[i] {
		case '\\', '"':
			b.WriteByte('\\')
		case '$':
			b.WriteByte('$')
		}
		b.WriteByte(symbol[i])
	}
	b.WriteByte('"')
	return b.String()
}

func (p *context) popCallerLocationFrame(b llssa.Builder) {
	if p.callerFrameMark.IsNil() {
		return
	}
	b.Call(p.runtimeFunc("PopCallerLocationFrame", popCallerLocationFrameSig()), p.callerFrameMark)
}

func (p *context) runtimeFunc(name string, sig *types.Signature) llssa.Expr {
	p.pkg.NeedRuntime = true
	fullName := llssa.PkgRuntime + "." + name
	if fn := p.pkg.FuncOf(fullName); fn != nil {
		return fn.Expr
	}
	return p.pkg.NewFuncEx(fullName, sig, llssa.InGo, false, false).Expr
}

func pushCallerLocationFrameSig() *types.Signature {
	return types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "entry", types.Typ[types.Uintptr]),
			types.NewVar(token.NoPos, nil, "name", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "file", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "startLine", types.Typ[types.Int]),
		),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int])),
		false,
	)
}

func recordRuntimeLocationSig() *types.Signature {
	return types.NewSignatureType(nil, nil, nil,
		types.NewTuple(
			types.NewVar(token.NoPos, nil, "entry", types.Typ[types.Uintptr]),
			types.NewVar(token.NoPos, nil, "name", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "file", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "line", types.Typ[types.Int]),
		),
		nil,
		false,
	)
}

func popCallerLocationFrameSig() *types.Signature {
	return types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "mark", types.Typ[types.Int])),
		nil,
		false,
	)
}

func runtimeFrameName(name string) string {
	const commandLineArguments = "command-line-arguments."
	if strings.HasPrefix(name, commandLineArguments) {
		name = "main." + name[len(commandLineArguments):]
	}
	return normalizeRuntimeAnonFuncName(name)
}

func normalizeRuntimeAnonFuncName(name string) string {
	dollar := strings.LastIndexByte(name, '$')
	if dollar < 0 || dollar == len(name)-1 {
		return name
	}
	for i := dollar + 1; i < len(name); i++ {
		if name[i] < '0' || name[i] > '9' {
			return name
		}
	}
	return name[:dollar] + ".func" + name[dollar+1:]
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

func (p *context) emitDo(b llssa.Builder, act llssa.DoAction, ds *explicitDeferStack, mayRecover bool, fn llssa.Expr, buildCall func(llssa.Builder, llssa.Expr, ...llssa.Expr) llssa.Expr, args ...llssa.Expr) llssa.Expr {
	if ds != nil {
		b.DeferToRecover(ds.owner, ds.stack, mayRecover, fn, buildCall, args...)
		return llssa.Nil
	}
	switch act {
	case llssa.Call, llssa.Go:
		if act == llssa.Call && isRecoverTransparentWrapper(p.goFn) {
			return b.CallRecoverAlias(p.fn.Expr, mayRecover, fn, buildCall, args...)
		}
		return b.Do(act, fn, buildCall, args...)
	default:
		b.DeferRecover(act, mayRecover, fn, buildCall, args...)
		return llssa.Nil
	}
}

func (p *context) callMayRecover(v ssa.Value) bool {
	switch v := v.(type) {
	case *ssa.Builtin:
		return false
	case *ssa.Function:
		return p.needsRecoverScope(v)
	case *ssa.MakeClosure:
		if fn, ok := v.Fn.(*ssa.Function); ok {
			return p.needsRecoverScope(fn)
		}
		return true
	case *ssa.Call:
		// The deferred callee is the call result, not the factory function.
		return true
	}
	return true
}

func (p *context) staticArrayLenBuiltinArg(b llssa.Builder, arg ssa.Value) (llssa.Expr, bool) {
	var arr *types.Array
	var sideEffect ssa.Value
	if load, ok := arg.(*ssa.UnOp); ok && load.Op == token.MUL {
		if t, ok := types.Unalias(load.Type()).Underlying().(*types.Array); ok {
			arr = t
			sideEffect = load.X
		}
	} else if ptr, ok := types.Unalias(arg.Type()).Underlying().(*types.Pointer); ok {
		if t, ok := types.Unalias(ptr.Elem()).Underlying().(*types.Array); ok {
			arr = t
			sideEffect = arg
		}
	}
	if arr == nil {
		return llssa.Expr{}, false
	}
	p.compileValue(b, sideEffect)
	return b.Const(constant.MakeInt64(arr.Len()), p.type_(types.Typ[types.Int], llssa.InGo)), true
}

func isPointerGoType(t types.Type) bool {
	_, ok := types.Unalias(t).Underlying().(*types.Pointer)
	return ok
}

func isKnownNonNilAddr(v ssa.Value) bool {
	switch v := v.(type) {
	case *ssa.Alloc, *ssa.Global:
		return true
	case *ssa.FieldAddr:
		return isKnownNonNilAddr(v.X)
	case *ssa.IndexAddr:
		return isKnownNonNilAddr(v.X)
	}
	return false
}

func isWrapNilCheckCall(v ssa.Value) bool {
	call, ok := v.(*ssa.Call)
	if !ok {
		return false
	}
	builtin, ok := call.Call.Value.(*ssa.Builtin)
	return ok && builtin.Name() == "ssa:wrapnilchk"
}

func (p *context) emitNilDerefBaseCheck(b llssa.Builder, addr ssa.Value) {
	switch addr := addr.(type) {
	case *ssa.UnOp:
		if addr.Op != token.MUL || isKnownNonNilAddr(addr.X) || isWrapNilCheckCall(addr.X) {
			return
		}
		p.emitCheckedDerefCheck(b, addr)
	case *ssa.FieldAddr:
		if isKnownNonNilAddr(addr.X) || isWrapNilCheckCall(addr.X) {
			return
		}
		p.emitNilDerefBaseCheck(b, addr.X)
		if isPointerGoType(addr.X.Type()) {
			base := p.compileValue(b, addr.X)
			b.AssertNilDeref(base)
		}
	}
}

func (p *context) emitCheckedDerefCheck(b llssa.Builder, arg *ssa.UnOp) {
	p.emitNilDerefBaseCheck(b, arg.X)
	ptr := p.compileValue(b, arg.X)
	b.AssertNilDeref(ptr)
}

func (p *context) compileCheckedDeref(b llssa.Builder, arg *ssa.UnOp) llssa.Expr {
	p.emitNilDerefBaseCheck(b, arg.X)
	ptr := p.compileValue(b, arg.X)
	checked := b.NilDerefCheck(ptr)
	ret := b.UnOp(token.MUL, checked)
	p.bvals[arg] = ret
	return ret
}

func valueReceiverNilDerefArg(fn *ssa.Function, args []ssa.Value) (*ssa.UnOp, bool) {
	if fn == nil || len(args) == 0 {
		return nil, false
	}
	recv := fn.Signature.Recv()
	if recv == nil || isPointerGoType(recv.Type()) {
		return nil, false
	}
	arg, ok := args[0].(*ssa.UnOp)
	if !ok || arg.Op != token.MUL || isKnownNonNilAddr(arg.X) || isWrapNilCheckCall(arg.X) {
		return nil, false
	}
	return arg, true
}

func boundValueReceiverNilDerefArg(fn *ssa.Function, bindings []ssa.Value) (*ssa.UnOp, bool) {
	if fn == nil || len(fn.FreeVars) == 0 || len(bindings) == 0 {
		return nil, false
	}
	if !strings.HasPrefix(fn.Synthetic, "bound method wrapper for ") {
		return nil, false
	}
	if isPointerGoType(fn.FreeVars[0].Type()) {
		return nil, false
	}
	arg, ok := bindings[0].(*ssa.UnOp)
	if !ok || arg.Op != token.MUL || isKnownNonNilAddr(arg.X) || isWrapNilCheckCall(arg.X) {
		return nil, false
	}
	return arg, true
}

func collectMethodNilDerefChecks(fn *ssa.Function) map[*ssa.UnOp]none {
	var checks map[*ssa.UnOp]none
	mark := func(arg *ssa.UnOp, ok bool) {
		if !ok {
			return
		}
		if checks == nil {
			checks = make(map[*ssa.UnOp]none)
		}
		checks[arg] = none{}
	}
	markCall := func(call *ssa.CallCommon) {
		if fn, ok := call.Value.(*ssa.Function); ok {
			mark(valueReceiverNilDerefArg(fn, call.Args))
		}
	}
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch instr := instr.(type) {
			case *ssa.Call:
				markCall(&instr.Call)
			case *ssa.Defer:
				markCall(&instr.Call)
			case *ssa.Go:
				markCall(&instr.Call)
			case *ssa.MakeClosure:
				if bound, ok := instr.Fn.(*ssa.Function); ok {
					mark(boundValueReceiverNilDerefArg(bound, instr.Bindings))
				}
			}
		}
	}
	return checks
}

func (p *context) callEx(b llssa.Builder, act llssa.DoAction, call *ssa.CallCommon, ds *explicitDeferStack) (ret llssa.Expr) {
	p.recordCallerLocationForCall(b, call)
	p.emitPCLineLabel(b, call.Pos())
	cv := call.Value
	mayRecover := p.callMayRecover(cv)
	if mthd := call.Method; mthd != nil {
		reflectCheck := p.reflectTypeMethodCheck(call, mthd)
		o := p.compileValue(b, cv)
		var fn llssa.Expr
		needsRecoverToken := act != llssa.Call && act != llssa.Go
		if act == llssa.Call && isRecoverTransparentWrapper(p.goFn) {
			needsRecoverToken = true
		}
		if needsRecoverToken {
			fn = b.ImethodWithRecoverToken(o, mthd)
		} else {
			fn = b.Imethod(o, mthd)
		}
		hasVArg := fnNormal
		if llssa.HasNameValist(call.Signature()) {
			hasVArg = fnHasVArg
		}
		args := p.compileValues(b, call.Args, hasVArg)
		ret = p.emitDo(b, act, ds, true, fn, llssa.Builder.Call, args...)
		if reflectCheck.Kind&llssa.ReflectTypeMethodByName != 0 && reflectCheck.Name == "" {
			b.MarkReflectTypeMethodByNameExpr(ret, 1)
		}
		b.EmitReflectTypeMethodCheckedLoad(ret, reflectCheck)
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
		if fn == "ssa:wrapnilchk" {
			ptr := p.compileValue(b, args[0])
			recvType := p.compileValue(b, args[1])
			methodName := p.compileValue(b, args[2])
			ret = b.WrapNilCheck(ptr, recvType, methodName)
			return
		} else if (fn == "len" || fn == "cap") && len(args) == 1 && act == llssa.Call {
			if n, ok := p.staticArrayLenBuiltinArg(b, args[0]); ok {
				ret = n
				return
			}
		} else if fn == "Offsetof" && act == llssa.Call {
			if offset, ok := p.offsetOfBuiltinArg(args[0]); ok {
				ret = offset
				return
			}
		}
		args := p.compileValues(b, args, kind)
		ret = p.emitDo(b, act, ds, false, llssa.Builtin(fn), llssa.Builder.Call, args...)
	case *ssa.Function:
		aFn, pyFn, ftype := p.compileFunction(cv)
		// TODO(xsw): check ca != llssa.Call
		switch ftype {
		case cFunc:
			p.inCFunc = true
			args := p.compileValues(b, args, kind)
			p.inCFunc = false
			ret = p.emitDo(b, act, ds, mayRecover, aFn.Expr, llssa.Builder.Call, args...)
		case goFunc:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, mayRecover, aFn.Expr, llssa.Builder.Call, args...)
		case pyFunc:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, mayRecover, pyFn.Expr, llssa.Builder.Call, args...)
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
		case llgoSetjmp:
			ret = p.setjmp(b, args)
		case llgoLongjmp:
			p.longjmp(b, args)
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
		case llgoClosureEnv:
			if len(args) != 0 || p.fn == nil || !p.fn.NeedsEnv() {
				panic("closureEnv(): called outside an env-bearing function")
			}
			ret = p.fn.Env()
		case llgoUnreachable: // func unreachable()
			b.Unreachable()
		case llgoAtomicLoad:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, false, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicLoad(b, args)
			}, args...)
		case llgoAtomicStore:
			args := p.compileValues(b, args, kind)
			p.emitDo(b, act, ds, false, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicStore(b, args)
			}, args...)
		case llgoAtomicCmpXchg:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, false, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicCmpXchg(b, args)
			}, args...)
		case llgoAtomicCmpXchgOK:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, false, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return p.atomicCmpXchgOK(b, args)
			}, args...)
		case llgoAtomicAddReturnNew:
			args := p.compileValues(b, args, kind)
			ret = p.emitDo(b, act, ds, false, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
				return b.BinOp(token.ADD, p.atomic(b, llssa.OpAdd, args), args[1])
			}, args...)
		default:
			if ftype >= llgoAtomicOpBase && ftype <= llgoAtomicOpLast {
				args := p.compileValues(b, args, kind)
				ret = p.emitDo(b, act, ds, false, llssa.Nil, func(b llssa.Builder, _ llssa.Expr, args ...llssa.Expr) llssa.Expr {
					return p.atomic(b, llssa.AtomicOp(ftype-llgoAtomicOpBase), args)
				}, args...)
			} else {
				log.Panicf("unknown ftype: %d for %s", ftype, cv.Name())
			}
		}
	default:
		fn := p.compileValue(b, cv)
		args := p.compileDynamicCallValues(b, call, kind)
		ret = p.emitDo(b, act, ds, mayRecover, fn, llssa.Builder.Call, args...)
	}
	return
}

func (p *context) compileDynamicCallValues(b llssa.Builder, call *ssa.CallCommon, hasVArg int) []llssa.Expr {
	args := p.compileValues(b, call.Args, hasVArg)
	params := call.Signature().Params()
	n := min(len(call.Args)-hasVArg, params.Len())
	for i, arg := range call.Args[:n] {
		want := params.At(i).Type()
		if needsNamedClosureChange(arg.Type(), want) {
			args[i] = b.ChangeType(p.type_(want, llssa.InGo), args[i])
		}
	}
	return args
}

func needsNamedClosureChange(got, want types.Type) bool {
	if types.Identical(got, want) {
		return false
	}
	_, gotIsFunc := got.Underlying().(*types.Signature)
	_, wantIsFunc := want.Underlying().(*types.Signature)
	return gotIsFunc && wantIsFunc && types.Identical(got.Underlying(), want.Underlying())
}

func (p *context) reflectTypeMethodCheck(call *ssa.CallCommon, method *types.Func) (check llssa.ReflectMethodCheck) {
	if !isReflectType(call.Value.Type()) {
		return
	}
	if pkg := method.Pkg(); pkg == nil || pkg.Path() != "reflect" {
		return
	}
	switch method.Name() {
	case "Method":
		if len(call.Args) != 1 {
			return
		}
		if index, ok := constInt(call.Args[0]); ok {
			p.pkg.RecordReflectMethodByIndex(p.fn.Name(), index)
			check.Kind = llssa.ReflectTypeMethodByIndex
			break
		}
		p.pkg.MarkReflectMethod(p.fn.Name())
		check.Kind = llssa.ReflectTypeMethodDynamic
	case "MethodByName":
		if len(call.Args) != 1 {
			return
		}
		if name, ok := constStr(call.Args[0]); ok {
			p.pkg.RecordReflectMethodByName(p.fn.Name(), name)
			check.Kind = llssa.ReflectTypeMethodByName
			check.Name = name
			break
		}
		p.pkg.MarkReflectMethod(p.fn.Name())
		check.Kind = llssa.ReflectTypeMethodDynamic | llssa.ReflectTypeMethodByName
	}
	p.pkg.NeedAbiInit |= check.Kind
	return
}

func isReflectType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "Type" && obj.Pkg() != nil && obj.Pkg().Path() == "reflect"
}

// -----------------------------------------------------------------------------
