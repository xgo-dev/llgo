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
	"fmt"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func requireStdcallPanic(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil || !strings.Contains(fmt.Sprint(got), want) {
			t.Fatalf("panic = %v, want text containing %q", got, want)
		}
	}()
	fn()
}

func newStdcallType(prog Program, pkgPath, name string, sig *types.Signature) *types.Named {
	pkg := types.NewPackage(pkgPath, "p")
	named := types.NewNamed(types.NewTypeName(token.NoPos, pkg, name, nil), sig, nil)
	prog.SetTypeCallConv(pkgPath+"."+name, Stdcall)
	return named
}

func TestStdcallCallConventions(t *testing.T) {
	if !isNativeCallConv(CDecl) || !isNativeCallConv(Stdcall) || isNativeCallConv(GoDecl) {
		t.Fatal("native function CallConv classification is inconsistent")
	}
	for _, test := range []struct {
		arch string
		want llvm.CallConv
	}{
		{arch: "386", want: llvm.X86StdcallCallConv},
		{arch: "amd64", want: llvm.CCallConv},
		{arch: "arm64", want: llvm.CCallConv},
	} {
		t.Run(test.arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: test.arch})
			defer prog.Dispose()
			if got := prog.stdcallCallConv(); got != test.want {
				t.Fatalf("stdcallCallConv() = %v, want %v", got, test.want)
			}
			sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
			pkg := prog.NewPackage("p", "example.com/p")
			callee := pkg.NewFunc("Native", sig, Stdcall)
			if got := callee.impl.FunctionCallConv(); got != test.want {
				t.Fatalf("native declaration convention = %v, want %v", got, test.want)
			}
			caller := pkg.NewFunc("example.com/p.caller", sig, GoDecl)
			body := caller.MakeBody(1)
			call := body.Call(callee.Expr)
			body.Return()
			if got := call.impl.InstructionCallConv(); got != test.want {
				t.Fatalf("direct call convention = %v, want %v", got, test.want)
			}
		})
	}

	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	for _, test := range []struct {
		name string
		want string
	}{
		{name: "Undecorated", want: "Undecorated"},
		{name: "_Explicit@4", want: "\x01_Explicit@4"},
		{name: "_NotDecorated", want: "_NotDecorated"},
		{name: "_Bad@bytes", want: "_Bad@bytes"},
		{name: "_a@b@4", want: "_a@b@4"},
		{name: "_4bad@4", want: "_4bad@4"},
	} {
		if got := prog.stdcallSymbolName(test.name); got != test.want {
			t.Errorf("stdcallSymbolName(%q) = %q, want %q", test.name, got, test.want)
		}
	}
	for _, arch := range []string{"amd64", "arm64"} {
		prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
		if got := prog.stdcallSymbolName("_Explicit@4"); got != "Explicit" {
			t.Errorf("windows/%s stdcallSymbolName(_Explicit@4) = %q, want Explicit", arch, got)
		}
		prog.Dispose()
	}

	for _, test := range []struct {
		target *Target
		want   string
	}{
		{target: &Target{GOOS: "linux", GOARCH: "amd64"}, want: "only defined for Windows"},
		// LLVM's official Windows package does not build the Mips backend.
		// RISC-V exercises the same unsupported-stdcall path while remaining a
		// target that the package can instantiate.
		{target: &Target{GOOS: "windows", GOARCH: "riscv64"}, want: "not supported on windows/riscv64"},
	} {
		prog := NewProgram(test.target)
		requireStdcallPanic(t, test.want, func() { prog.stdcallCallConv() })
		prog.Dispose()
	}
}

func TestStdcallTypeValidationAndIndirectCall(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	param := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32])
	result := types.NewVar(token.NoPos, nil, "result", types.Typ[types.Int32])
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(result), false)
	callback := newStdcallType(prog, "example.com/p", "Callback", sig)
	if !prog.isStdcallType(callback) || prog.isStdcallType(sig) {
		t.Fatal("stdcall named-type recognition is inconsistent")
	}

	pkg := prog.NewPackage("p", "example.com/p")
	callerSig := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "callback", callback)),
		types.NewTuple(result), false)
	caller := pkg.NewFunc("example.com/p.indirect", callerSig, GoDecl)
	body := caller.MakeBody(1)
	call := body.Call(caller.Param(0), prog.IntVal(7, prog.Int32()))
	body.Return(call)
	if got := call.impl.InstructionCallConv(); got != llvm.X86StdcallCallConv {
		t.Fatalf("indirect call convention = %v, want x86_stdcallcc", got)
	}

	variadic := types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(token.NoPos, nil, "values", types.NewSlice(types.Typ[types.Int32]))),
		nil, true)
	requireStdcallPanic(t, "does not support variadic", func() { prog.validateStdcallSignature(variadic) })
	requireStdcallPanic(t, "requires a function type", func() { prog.validateStdcallType(types.Typ[types.Int]) })
}

func TestStdcallCallbackAdapters(t *testing.T) {
	for _, resultCount := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("results-%d", resultCount), func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
			defer prog.Dispose()
			results := make([]*types.Var, resultCount)
			for i := range results {
				results[i] = types.NewVar(token.NoPos, nil, fmt.Sprintf("result%d", i), types.Typ[types.Int32])
			}
			params := types.NewTuple(types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32]))
			sig := types.NewSignatureType(nil, nil, nil, params, types.NewTuple(results...), false)
			callback := newStdcallType(prog, "example.com/p", "Callback", sig)
			pkg := prog.NewPackage("p", "example.com/p")

			sourceName := fmt.Sprintf("example.com/p.source%d", resultCount)
			source := pkg.NewFunc(sourceName, sig, GoDecl)
			sourceBody := source.MakeBody(1)
			values := make([]Expr, resultCount)
			for i := range values {
				values[i] = prog.IntVal(uint64(i+1), prog.Int32())
			}
			sourceBody.Return(values...)

			adapterOwner := pkg.NewFunc(fmt.Sprintf("example.com/p.adapterOwner%d", resultCount), NoArgsNoRet, GoDecl)
			body := adapterOwner.MakeBody(1)
			consumerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
				types.NewVar(token.NoPos, nil, "callback", callback),
			), nil, false)
			consumer := pkg.NewFunc(fmt.Sprintf("Consume%d", resultCount), consumerSig, Stdcall)
			nativeCall := body.Call(consumer.Expr, source.Expr)
			dst := prog.Type(callback, GoDecl)
			first := body.ChangeType(dst, source.Expr)
			second := body.ChangeType(dst, source.Expr)
			body.Return()
			if first.impl != second.impl {
				t.Fatal("repeated callback conversion did not reuse its adapter")
			}
			wrapperName := pkg.Path() + ".__llgo_stdcall$" + sourceName + "$" + prog.abi.FuncName(sig)
			wrapper := pkg.FuncOf(wrapperName)
			if wrapper == nil || first.impl != wrapper.impl {
				t.Fatalf("stdcall adapter %q was not generated", wrapperName)
			}
			if !strings.Contains(nativeCall.impl.String(), wrapperName) {
				t.Fatalf("native argument conversion did not use the stdcall adapter: %s", nativeCall.impl.String())
			}
			if wrapper.impl.Linkage() != llvm.InternalLinkage || wrapper.impl.FunctionCallConv() != llvm.X86StdcallCallConv {
				t.Fatalf("adapter linkage/convention = %v/%v", wrapper.impl.Linkage(), wrapper.impl.FunctionCallConv())
			}
			var sourceCall llvm.Value
			for block := wrapper.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
				for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
					if call := instruction.IsACallInst(); !call.IsNil() && call.CalledValue() == source.impl {
						sourceCall = call
					}
				}
			}
			if sourceCall.IsNil() || sourceCall.InstructionCallConv() != llvm.CCallConv {
				t.Fatalf("adapter did not call the Go entry with its original convention:\n%s", wrapper.impl.String())
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid callback adapter module: %v\n%s", err, pkg.String())
			}
		})
	}
}

func TestStdcallCallbackConversionWithoutX86Adapter(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
			callback := newStdcallType(prog, "example.com/p", "Callback", sig)
			pkg := prog.NewPackage("p", "example.com/p")
			source := pkg.NewFunc("example.com/p.source", sig, GoDecl)
			source.MakeBody(1).Return()
			owner := pkg.NewFunc("example.com/p.owner", sig, GoDecl)
			body := owner.MakeBody(1)
			converted := body.ChangeType(prog.Type(callback, GoDecl), source.Expr)
			body.Return()
			if converted.impl != source.impl {
				t.Fatalf("windows/%s unnecessarily wrapped the callback", arch)
			}
			wrapperPrefix := pkg.Path() + ".__llgo_stdcall$" + source.Name() + "$"
			for name := range pkg.fns {
				if strings.HasPrefix(name, wrapperPrefix) {
					t.Fatalf("windows/%s emitted an x86-only adapter %q", arch, name)
				}
			}
		})
	}
}

func TestStdcallFuncvalAdapters(t *testing.T) {
	for _, resultCount := range []int{0, 1, 2} {
		t.Run(fmt.Sprintf("results-%d", resultCount), func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
			defer prog.Dispose()
			params := types.NewTuple(types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32]))
			results := make([]*types.Var, resultCount)
			for i := range results {
				results[i] = types.NewVar(token.NoPos, nil, fmt.Sprintf("result%d", i), types.Typ[types.Int32])
			}
			sig := types.NewSignatureType(nil, nil, nil, params, types.NewTuple(results...), false)
			callback := newStdcallType(prog, "example.com/p", fmt.Sprintf("Callback%d", resultCount), sig)
			pkg := prog.NewPackage("p", "example.com/p")
			ownerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
				types.NewVar(token.NoPos, nil, "callback", callback),
			), nil, false)
			owner := pkg.NewFunc(fmt.Sprintf("example.com/p.owner%d", resultCount), ownerSig, GoDecl)
			body := owner.MakeBody(1)
			goFunc := prog.Type(sig, GoDecl)
			converted := body.ChangeType(goFunc, owner.Param(0))
			body.Return()

			wrapperName := pkg.Path() + ".__llgo_stdcall_funcval$" + prog.abi.FuncName(
				goFunc.raw.Type.Underlying().(*types.Struct).Field(0).Type().(*types.Signature),
			)
			wrapper := pkg.FuncOf(wrapperName)
			if wrapper == nil || !wrapper.NeedsEnv() {
				t.Fatalf("stdcall funcval adapter %q was not generated with an environment", wrapperName)
			}
			if wrapper.impl.Linkage() != llvm.InternalLinkage {
				t.Fatalf("stdcall funcval adapter linkage = %v, want internal", wrapper.impl.Linkage())
			}
			if converted.kind != vkClosure {
				t.Fatalf("converted stdcall value kind = %v, want closure", converted.kind)
			}
			if converted.impl.Operand(1) != owner.Param(0).impl {
				t.Fatalf("funcval context does not carry the native function pointer: %s", converted.impl)
			}

			var nativeCall llvm.Value
			for block := wrapper.impl.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
				for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
					call := instruction.IsACallInst()
					if !call.IsNil() {
						nativeCall = call
						break
					}
				}
			}
			if nativeCall.IsNil() {
				t.Fatalf("adapter has no native call:\n%s", wrapper.impl.String())
			}
			if nativeCall.InstructionCallConv() != prog.stdcallCallConv() {
				t.Fatalf("adapter native call convention = %v, want %v:\n%s",
					nativeCall.InstructionCallConv(), prog.stdcallCallConv(), wrapper.impl.String())
			}
			if strings.Contains(pkg.String(), "runtime.AllocU") {
				t.Fatalf("stdcall funcval conversion unexpectedly allocated:\n%s", pkg.String())
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid stdcall funcval adapter module: %v\n%s", err, pkg.String())
			}
		})
	}
}

func TestStdcallFuncvalWithoutAdapter(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		t.Run(arch, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: arch})
			defer prog.Dispose()
			param := types.NewVar(token.NoPos, nil, "value", types.Typ[types.Int32])
			result := types.NewVar(token.NoPos, nil, "result", types.Typ[types.Int32])
			sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(param), types.NewTuple(result), false)
			callback := newStdcallType(prog, "example.com/p", "Callback", sig)
			pkg := prog.NewPackage("p", "example.com/p")
			ownerSig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
				types.NewVar(token.NoPos, nil, "callback", callback),
			), nil, false)
			owner := pkg.NewFunc("example.com/p.owner", ownerSig, GoDecl)
			body := owner.MakeBody(1)
			converted := body.ChangeType(prog.Type(sig, GoDecl), owner.Param(0))
			body.Return()

			if converted.kind != vkClosure {
				t.Fatalf("converted stdcall value kind = %v, want closure", converted.kind)
			}
			code := converted.impl.Operand(0).Operand(1)
			if code != owner.Param(0).impl {
				t.Fatalf("funcval code does not contain the native function pointer: %s", converted.impl)
			}
			env := converted.impl.Operand(1)
			if !env.IsNull() {
				t.Fatalf("funcval environment = %s, want nil", env)
			}
			wrapperPrefix := pkg.Path() + ".__llgo_stdcall_funcval$"
			for name := range pkg.fns {
				if strings.HasPrefix(name, wrapperPrefix) {
					t.Fatalf("windows/%s emitted an unnecessary stdcall funcval adapter %q", arch, name)
				}
			}
			if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid compatible stdcall funcval module: %v\n%s", err, pkg.String())
			}
		})
	}
}

func TestStdcallFuncvalAdapterSelection(t *testing.T) {
	tests := []struct {
		name  string
		arch  string
		named bool
		want  bool
	}{
		{name: "386-direct", arch: "386", want: true},
		{name: "386-named", arch: "386", named: true, want: true},
		{name: "amd64-direct", arch: "amd64"},
		{name: "amd64-named", arch: "amd64", named: true},
		{name: "arm64-direct", arch: "arm64"},
		{name: "arm64-named", arch: "arm64", named: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog := NewProgram(&Target{GOOS: "windows", GOARCH: test.arch})
			defer prog.Dispose()
			sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
			pkg := prog.NewPackage("p", "example.com/p")

			owner := pkg.NewFunc("example.com/p.owner", NoArgsNoRet, GoDecl)
			body := owner.MakeBody(1)
			var value Expr
			if test.named {
				typ := newStdcallType(prog, pkg.Path(), "Callback", sig)
				ownerSig := types.NewSignatureType(nil, nil, nil,
					types.NewTuple(types.NewVar(token.NoPos, nil, "fn", typ)), nil, false)
				owner = pkg.NewFunc("example.com/p.namedOwner", ownerSig, GoDecl)
				body = owner.MakeBody(1)
				value = owner.Param(0)
			} else {
				value = pkg.NewFunc("Native", sig, Stdcall).Expr
			}
			if got := body.needsStdcallFuncval(value); got != test.want {
				t.Fatalf("needsStdcallFuncval() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestStdcallRejectsNonDirectCallback(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	setTestRuntime(t, prog)
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	callback := newStdcallType(prog, "example.com/p", "Callback", sig)
	pkg := prog.NewPackage("p", "example.com/p")
	envStruct := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "value", types.Typ[types.Int], false),
	}, nil)
	env := types.NewVar(token.NoPos, nil, "$env", types.NewPointer(envStruct))
	source := pkg.NewEnvFunc("example.com/p.capturing", sig, GoDecl, env, false)
	source.MakeBody(1).Return()
	owner := pkg.NewFunc("example.com/p.owner", sig, GoDecl)
	body := owner.MakeBody(1)
	closure := body.MakeClosure(source.Expr, []Expr{prog.Val(1)})
	requireStdcallPanic(t, "must be a direct function reference", func() {
		body.ChangeType(prog.Type(callback, GoDecl), closure)
	})
}

func TestStdcallNativePointerAssignment(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	source := newStdcallType(prog, "p", "Source", sig)
	destination := newStdcallType(prog, "p", "Destination", sig)
	pkg := prog.NewPackage("p", "p")
	caller := pkg.NewFunc("p.assign", types.NewSignatureType(nil, nil, nil,
		types.NewTuple(types.NewVar(0, nil, "callback", source)), nil, false), GoDecl)
	body := caller.MakeBody(1)
	global := pkg.NewVar("callback", types.NewPointer(destination), GoDecl)
	body.Store(global.Expr, caller.Param(0))
	body.Store(global.Expr, prog.Nil(prog.Type(sig, GoDecl)))
	body.Return()
	ir := caller.impl.String()
	if !strings.Contains(ir, "store ptr null, ptr @callback") || strings.Contains(ir, "__llgo_stdcall$") {
		t.Fatalf("native pointer assignment emitted an adapter or lost nil:\n%s", ir)
	}
	if err := llvm.VerifyModule(pkg.Module(), llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
