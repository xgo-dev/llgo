package cabi

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/funcattrs"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestSourceScalarAttributesSurviveAggregateArgumentABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargetMCs()
	for _, target := range []llssa.Target{{GOOS: "linux", GOARCH: "amd64"}, {GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "windows", GOARCH: "386"}, {GOOS: "wasip1", GOARCH: "wasm64"}} {
		t.Run(target.GOARCH, func(t *testing.T) {
			prog := llssa.NewProgram(&target)
			defer prog.Dispose()
			mod := prog.NewPackage("attrs", "attrs").Module()
			ctx := mod.Context()
			ptr := llvm.PointerType(ctx.Int8Type(), 0)
			agg := ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.Int64Type(), ctx.Int64Type()}, false)
			ft := llvm.FunctionType(ptr, []llvm.Type{agg, ptr}, false)
			fn := llvm.AddFunction(mod, "copy_with_aggregate", ft)
			fn.AddAttributeAtIndex(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0))
			fn.AddAttributeAtIndex(2, ctx.CreateEnumAttribute(llvm.AttributeKindID("returned"), 0))
			fn.AddFunctionAttr(ctx.CreateStringAttribute(funcattrs.Metadata, "[]"))
			ranged := llvm.AddFunction(mod, "range_with_aggregate", llvm.FunctionType(ctx.Int32Type(), []llvm.Type{agg}, false))
			ranged.AddAttributeAtIndex(0, ctx.CreateConstantRangeAttribute(llvm.AttributeKindID("range"), 32, []uint64{0}, []uint64{64}))
			ranged.AddFunctionAttr(ctx.CreateStringAttribute(funcattrs.Metadata, "[]"))
			caller := llvm.AddFunction(mod, "caller", ft)
			b := ctx.NewBuilder()
			defer b.Dispose()
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
			call := llvm.CreateCall(b, ft, fn, []llvm.Value{caller.Param(0), caller.Param(1)})
			call.AddCallSiteAttribute(0, ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0))
			b.CreateRet(call)
			tr := NewTransformer(prog, mod.Target(), "", false)
			tr.TransformModule("attrs", mod)
			lowered := mod.NamedFunction("copy_with_aggregate")
			if lowered.GlobalValueType() == ft {
				t.Fatal("test did not exercise ABI rewriting")
			}
			if lowered.GetEnumAttributeAtIndex(0, llvm.AttributeKindID("nonnull")).IsNil() {
				t.Fatalf("return nonnull lost:\n%s", mod.String())
			}
			params := lowered.GlobalValueType().ParamTypesCount()
			if lowered.GetEnumAttributeAtIndex(params, llvm.AttributeKindID("returned")).IsNil() {
				t.Fatalf("parameter relation lost:\n%s", mod.String())
			}
			if !strings.Contains(mod.String(), "call nonnull ptr @copy_with_aggregate") {
				t.Fatalf("call-site return attr lost:\n%s", mod.String())
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(fmt.Errorf("%s: %w", target.GOARCH, err))
			}
			if mod.NamedFunction("range_with_aggregate").GetEnumAttributeAtIndex(0, llvm.AttributeKindID("range")).IsNil() {
				t.Fatal("constant range lost during ABI rewrite")
			}
		})
	}
}

func TestSourceABIWidensIndirectMemoryContract(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargetMCs()
	p := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer p.Dispose()
	mod := p.NewPackage("bad", "bad").Module()
	ctx := mod.Context()
	agg := llvm.ArrayType(ctx.Int64Type(), 32)
	fn := llvm.AddFunction(mod, "indirect_readonly", llvm.FunctionType(agg, nil, false))
	fn.AddFunctionAttr(ctx.CreateEnumAttribute(llvm.AttributeKindID("memory"), 0x555))
	fn.AddFunctionAttr(ctx.CreateStringAttribute(funcattrs.Metadata, "[]"))
	NewTransformer(p, mod.Target(), "", false).TransformModule("bad", mod)
	physical := mod.NamedFunction("indirect_readonly")
	if physical.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("sret")).IsNil() {
		t.Fatal("test did not exercise an indirect return")
	}
	if got := physical.GetEnumAttributeAtIndex(-1, llvm.AttributeKindID("memory")).GetEnumValue(); got != 0x557 {
		t.Fatalf("sret storage write is missing from the physical memory summary: %#x", got)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}
