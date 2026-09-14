package ssa

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestAssertNilDerefZeroExprNoPanic(t *testing.T) {
	var b Builder
	b.AssertNilDeref(Expr{})
}

func TestAssertNilDerefColdCall(t *testing.T) {
	Initialize(InitAllTargets | InitAllTargetInfos | InitAllTargetMCs | InitAllAsmPrinters)
	for _, target := range []*Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux"},
		{GOOS: "darwin", GOARCH: "arm64", LLVMTarget: "aarch64-apple-darwin"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "arm64", LLVMTarget: "aarch64-pc-windows-msvc"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-emscripten"},
		{GOOS: "js", GOARCH: "wasm", LLVMTarget: "wasm64-unknown-emscripten"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasip1"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := NewProgram(target)
			defer prog.Dispose()
			prog.SetRuntime(func() *types.Package {
				rt := types.NewPackage(PkgRuntime, "runtime")
				params := types.NewTuple(types.NewVar(token.NoPos, nil, "nil", types.Typ[types.Bool]))
				sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
				rt.Scope().Insert(types.NewFunc(token.NoPos, rt, "AssertNilDeref", sig))
				return rt
			})
			pkg := prog.NewPackage("p", "example.com/nilcheck")
			params := types.NewTuple(types.NewVar(token.NoPos, nil, "p", types.Typ[types.UnsafePointer]))
			sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
			for _, name := range []string{"dynamic", "nil", "allocated", "constant", "value"} {
				fnSig := sig
				if name == "value" {
					results := types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Int]))
					fnSig = types.NewSignatureType(nil, nil, nil, params, results, false)
				}
				fn := pkg.NewFunc(name, fnSig, InGo)
				b := fn.MakeBody(1)
				var panicBlock llvm.BasicBlock
				b.PanicSite = func(b Builder) { panicBlock = b.impl.GetInsertBlock() }
				ptr := fn.Param(0)
				switch name {
				case "nil":
					ptr = prog.Nil(prog.VoidPtr())
				case "allocated":
					allocator := pkg.NewFunc(PkgRuntime+".AllocU", prog.tyMalloc(), InGo)
					ptr = b.Call(allocator.Expr, prog.IntVal(8, prog.Uintptr()))
				case "constant":
					ptr = b.Convert(prog.VoidPtr(), prog.IntVal(1, prog.Uintptr()))
				}
				b.AssertNilDeref(ptr)
				if name == "value" {
					b.Return(prog.IntVal(42, prog.Int()))
				} else {
					b.Return()
				}
				b.EndBuild()
				if name == "dynamic" {
					entry := fn.impl.EntryBasicBlock()
					branch := entry.LastInstruction()
					if branch.InstructionOpcode() != llvm.Br || branch.SuccessorsCount() != 2 {
						t.Fatalf("dynamic nil check must branch before calling runtime:\n%s", fn.impl.String())
					}
					for inst := entry.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
						if inst.InstructionOpcode() == llvm.Call {
							t.Fatalf("nil check calls runtime unconditionally:\n%s", fn.impl.String())
						}
					}
					failure := branch.Successor(0)
					if panicBlock != failure {
						t.Fatal("panic-site callback did not run in the failure block")
					}
					next := failure.LastInstruction()
					if prog.NeedsFramePointer() {
						if next.InstructionOpcode() != llvm.Ret {
							t.Fatalf("native nil failure must retain a separate formal return:\n%s", fn.impl.String())
						}
					} else if next.InstructionOpcode() != llvm.Br || next.SuccessorsCount() != 1 || next.Successor(0) != failure {
						t.Fatalf("non-native nil failure must not rejoin the success path:\n%s", fn.impl.String())
					}
				}
			}
			mod := pkg.Module()
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := mod.RunPasses("function(instcombine,simplifycfg)", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"dynamic", "nil", "allocated", "constant", "value"} {
				ir := mod.NamedFunction(name).String()
				if got := strings.Contains(ir, "AssertNilDeref"); got != (name == "dynamic" || name == "nil" || name == "value") {
					t.Fatalf("unexpected optimized %s check:\n%s", name, ir)
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if prog.NeedsFramePointer() {
				// Model LTO seeing the helper body and all of its call sites.
				// It must not propagate a constant true into this noinline
				// helper and rediscover noreturn through its callers.
				helper := mod.NamedFunction(PkgRuntime + ".AssertNilDeref")
				helper.SetLinkage(llvm.InternalLinkage)
				helper.AddFunctionAttr(prog.ctx.CreateEnumAttribute(llvm.AttributeKindID("noinline"), 0))
				entry := prog.ctx.AddBasicBlock(helper, "entry")
				failure := prog.ctx.AddBasicBlock(helper, "failure")
				success := prog.ctx.AddBasicBlock(helper, "success")
				hb := prog.ctx.NewBuilder()
				hb.SetInsertPointAtEnd(entry)
				boolTy := prog.Bool().ll
				barrierTy := llvm.FunctionType(boolTy, []llvm.Type{boolTy}, false)
				barrier := llvm.InlineAsm(barrierTy, "", "=r,0", true, false, llvm.InlineAsmDialectATT, false)
				condition := hb.CreateCall(barrierTy, barrier, []llvm.Value{helper.Param(0)}, "")
				hb.CreateCondBr(condition, failure, success)
				hb.SetInsertPointAtEnd(failure)
				hb.CreateBr(failure)
				hb.SetInsertPointAtEnd(success)
				hb.CreateRetVoid()
				hb.Dispose()
				if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
					t.Fatal(err)
				}
				fn := mod.NamedFunction("nil")
				if attr := fn.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("noreturn")); attr.C != nil {
					t.Fatalf("static nil caller must retain a return-PC continuation after optimization:\n%s", fn.String())
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				obj, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.ObjectFile)
				if err != nil {
					t.Fatal(err)
				}
				obj.Dispose()
			}
		})
	}
}
