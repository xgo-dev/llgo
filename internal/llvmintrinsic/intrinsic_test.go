package llvmintrinsic

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestResolveWithoutSyntheticCall(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i32 := ctx.Int32Type()
	good := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	binding, err := (Intrinsic{Name: "llvm.ctpop"}).Resolve(good)
	if err != nil {
		t.Fatal(err)
	}
	if binding.Name() != "llvm.ctpop.i32" || binding.Signature() != good || binding.ID() != llvm.LookupIntrinsicID("llvm.ctpop") {
		t.Fatal("incorrect resolved binding")
	}
	for _, tc := range []struct {
		name      string
		signature llvm.Type
		pattern   Type
		want      string
	}{
		{"llvm.not.real", good, nil, "unknown LLVM intrinsic"},
		{"llvm.ctpop", i32, nil, "function signature"},
		{"llvm.ctpop", llvm.Type{}, nil, "function signature"},
		{"llvm.ctpop.i16", good, nil, "name does not match"},
		{"llvm.ctpop", llvm.FunctionType(i32, []llvm.Type{ctx.FloatType()}, false), nil, "does not match"},
		{"llvm.ctpop", good, Func(I64, I64), "expected i64"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := (Intrinsic{Name: tc.name, Type: tc.pattern}).Resolve(tc.signature)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %s", err, tc.want)
			}
		})
	}
	// A valid LLVM NVVM immediate is 128, not zero. Declaration resolution
	// must not manufacture zero operands that violate LLVM's range attribute.
	id := llvm.LookupIntrinsicID("llvm.nvvm.fence.proxy.tensormap_generic.acquire.cta")
	if id == 0 {
		t.Fatal("missing linked NVVM intrinsic")
	}
	if _, err := (Intrinsic{Name: baseName(id)}).Resolve(sampleSignature(ctx, id, 0)); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryRefinementsAndImmarg(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	sig := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr, ptr, ctx.Int64Type(), ctx.Int1Type()}, false)
	spec := Intrinsic{Name: "llvm.memcpy.inline", Type: Func(Void, AnyPointer, AnyPointer, Const(Integer), I1)}
	binding, err := spec.Resolve(sig)
	if err != nil {
		t.Fatal(err)
	}
	if len(binding.plan.rules) != 2 {
		t.Fatalf("constant checks=%v", binding.plan.rules)
	}
	mod := ctx.NewModule("operands")
	defer mod.Dispose()
	fn := llvm.AddFunction(mod, "caller", sig)
	args := []llvm.Value{fn.Param(0), fn.Param(1), llvm.ConstInt(ctx.Int64Type(), 8, false), llvm.ConstInt(ctx.Int1Type(), 0, false)}
	if err := binding.CheckArguments(args...); err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{2, 3} {
		bad := append([]llvm.Value(nil), args...)
		bad[index] = fn.Param(index)
		if err := binding.CheckArguments(bad...); err == nil || !strings.Contains(err.Error(), "immediate constant") {
			t.Fatalf("dynamic operand %d accepted: %v", index, err)
		}
	}
	bad := append([]llvm.Value(nil), args...)
	bad[2] = llvm.Undef(ctx.Int64Type())
	if err := binding.CheckArguments(bad...); err == nil {
		t.Fatal("undef accepted as immediate")
	}
	if err := binding.CheckArguments(args[:3]...); err == nil {
		t.Fatal("missing operand accepted")
	}
	if err := binding.CheckArguments(append(args, args[0])...); err == nil {
		t.Fatal("extra operand accepted")
	}
	bad = append([]llvm.Value(nil), args...)
	bad[0] = llvm.Value{}
	if err := binding.CheckArguments(bad...); err == nil {
		t.Fatal("nil operand accepted")
	}
	bad = append([]llvm.Value(nil), args...)
	bad[2] = llvm.ConstInt(ctx.Int32Type(), 8, false)
	if err := binding.CheckArguments(bad...); err == nil {
		t.Fatal("wrong operand width accepted")
	}
	for _, name := range []string{"llvm.memcpy", "llvm.memcpy.inline", "llvm.memmove", "llvm.memset", "llvm.memset.inline"} {
		params := []llvm.Type{ptr, ptr, ctx.Int64Type(), ctx.Int1Type()}
		patterns := []Type{AnyPointer, AnyPointer, Integer, I1}
		if strings.Contains(name, "memset") {
			params[1] = ctx.Int8Type()
			patterns[1] = I8
		}
		if strings.HasSuffix(name, ".inline") {
			patterns[2] = Const(Integer)
		}
		ft := llvm.FunctionType(ctx.VoidType(), params, false)
		if _, err := (Intrinsic{Name: name, Type: Func(Void, patterns...)}).Resolve(ft); err != nil {
			t.Fatal(err)
		}
		params[2] = llvm.VectorType(ctx.Int64Type(), 2)
		if _, err := (Intrinsic{Name: name, Type: Func(Void, patterns...)}).Resolve(llvm.FunctionType(ctx.VoidType(), params, false)); err == nil {
			t.Fatalf("%s accepted vector length", name)
		}
	}
}

func TestSupplementalShapeRestrictions(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i32, f32 := ctx.Int32Type(), ctx.FloatType()
	for _, name := range []string{"llvm.fptosi.sat", "llvm.fptoui.sat"} {
		spec := Intrinsic{Name: name, Type: Func(Bind("result", Integers), Shape(Ref("result"), Floating))}
		for _, tc := range []struct {
			dst, src llvm.Type
			ok       bool
		}{
			{i32, f32, true}, {llvm.VectorType(i32, 4), llvm.VectorType(f32, 4), true},
			{llvm.VectorType(i32, 2), llvm.VectorType(f32, 4), false},
			{i32, llvm.VectorType(f32, 1), false},
			{llvm.VectorType(i32, 1), f32, false},
		} {
			_, err := spec.Resolve(llvm.FunctionType(tc.dst, []llvm.Type{tc.src}, false))
			if (err == nil) != tc.ok {
				t.Fatalf("%s %v <- %v: %v", name, tc.dst, tc.src, err)
			}
		}
	}
	spec := Intrinsic{Name: "llvm.powi", Type: Func(nil, nil, Integer)}
	if _, err := spec.Resolve(llvm.FunctionType(f32, []llvm.Type{f32, i32}, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := spec.Resolve(llvm.FunctionType(llvm.VectorType(f32, 4), []llvm.Type{llvm.VectorType(f32, 4), llvm.VectorType(i32, 4)}, false)); err == nil {
		t.Fatal("accepted vector exponent")
	}
}

func TestConstantAndRangePatterns(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i8, i32, f32 := ctx.Int8Type(), ctx.Int32Type(), ctx.FloatType()
	for _, tc := range []struct {
		typ     llvm.Type
		pattern Type
		value   llvm.Value
		ok      bool
	}{
		{i8, Range(Integer, -1, 2), llvm.ConstInt(i8, 255, false), true},
		{i8, Range(Integer, -1, 2), llvm.ConstInt(i8, 2, false), false},
		{i8, UnsignedRange(Integer, 0, 256), llvm.ConstInt(i8, 255, false), true},
		{ctx.Int1Type(), UnsignedRange(Integer, 0, 2), llvm.ConstInt(ctx.Int1Type(), 1, false), true},
		{i8, UnsignedRange(Integer, 0, 4), llvm.ConstInt(i8, 255, false), false},
		{i32, Range(Integer, 0, 4), llvm.ConstInt(i32, 3, false), true},
		{i32, Range(Integer, 0, 4), llvm.Undef(i32), false},
		{ctx.IntType(128), Range(Integer, 0, 4), llvm.ConstInt(ctx.IntType(128), 3, false), true},
		{f32, Const(Floating), llvm.ConstFloat(f32, 1.5), true},
	} {
		plan, err := Match(Func(Void, tc.pattern), llvm.FunctionType(ctx.VoidType(), []llvm.Type{tc.typ}, false))
		if err != nil {
			t.Fatal(err)
		}
		err = plan.CheckArguments(tc.value)
		if (err == nil) != tc.ok {
			t.Fatalf("%v: %v", tc.pattern, err)
		}
	}
	ptr := llvm.PointerType(i8, 0)
	for _, tc := range []struct {
		pattern   Type
		signature llvm.Type
	}{
		{Func(Const(Integer)), llvm.FunctionType(i32, nil, false)},
		{Func(Void, Struct(Const(Integer))), llvm.FunctionType(ctx.VoidType(), []llvm.Type{ctx.StructType([]llvm.Type{i32}, false)}, false)},
		{Func(Void, Const(AnyPointer)), llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptr}, false)},
		{Func(Void, Range(Integer, 4, 0)), llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, false)},
		{Func(Void, UnsignedRange(Integer, 4, 0)), llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32}, false)},
		{Func(Void, Range(Floating, 0, 4)), llvm.FunctionType(ctx.VoidType(), []llvm.Type{f32}, false)},
		{Func(Void, VarArgs(Any), I32), llvm.FunctionType(ctx.VoidType(), []llvm.Type{i32, i32}, false)},
		{Func(I32, I32), i32},
	} {
		if _, err := Match(tc.pattern, tc.signature); err == nil {
			t.Fatalf("accepted %v", tc.pattern)
		}
	}
	plan, err := Match(Any, i32)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.CheckArguments(); err == nil {
		t.Fatal("operand checks on non-function")
	}
}

func TestCallsArraysStructsAndVarargs(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	arr := llvm.ArrayType(ctx.Int8Type(), 3)
	st := ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.DoubleType()}, false)
	for _, name := range []string{"llvm.ssa.copy", "llvm.is.constant"} {
		for _, typ := range []llvm.Type{arr, st} {
			result := typ
			if name == "llvm.is.constant" {
				result = ctx.Int1Type()
			}
			sig := llvm.FunctionType(result, []llvm.Type{typ}, false)
			binding, err := (Intrinsic{Name: name}).Resolve(sig)
			if err != nil {
				t.Fatal(err)
			}
			mod := ctx.NewModule("aggregate")
			fn := llvm.AddFunction(mod, "caller", sig)
			builder := ctx.NewBuilder()
			builder.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
			value, err := binding.Call(builder, mod, fn.Param(0))
			if err != nil {
				t.Fatal(err)
			}
			builder.CreateRet(value)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			builder.Dispose()
			mod.Dispose()
		}
	}
	sig := llvm.FunctionType(ctx.VoidType(), nil, true)
	binding, err := (Intrinsic{Name: "llvm.fake.use", Type: Func(Void, VarArgs(Any))}).Resolve(sig)
	if err != nil {
		t.Fatal(err)
	}
	mod := ctx.NewModule("varargs")
	defer mod.Dispose()
	caller := llvm.AddFunction(mod, "caller", llvm.FunctionType(ctx.VoidType(), []llvm.Type{ctx.Int8Type(), ctx.FloatType(), arr, st}, false))
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
	args := []llvm.Value{caller.Param(0), caller.Param(1), caller.Param(2), caller.Param(3)}
	call, err := binding.Call(builder, mod, args...)
	if err != nil {
		t.Fatal(err)
	}
	if call.Operand(0).Type() != ctx.Int8Type() || call.Operand(1).Type() != ctx.FloatType() {
		t.Fatal("C default promotions applied to intrinsic varargs")
	}
	builder.CreateRetVoid()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
	plan, err := Match(Func(Void, VarArgs(Const(Integer))), sig)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.CheckArguments(llvm.ConstInt(ctx.Int32Type(), 1, false)); err != nil {
		t.Fatal(err)
	}
	if err := plan.CheckArguments(caller.Param(0)); err == nil {
		t.Fatal("dynamic variadic immediate accepted")
	}
	if err := plan.CheckArguments(llvm.ConstFloat(ctx.FloatType(), 1)); err == nil {
		t.Fatal("floating tail accepted as integer")
	}
}

func TestDeclarationConflictsAndRejectedCall(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i32 := ctx.Int32Type()
	sig := llvm.FunctionType(i32, []llvm.Type{i32}, false)
	binding, err := (Intrinsic{Name: "llvm.ctpop"}).Resolve(sig)
	if err != nil {
		t.Fatal(err)
	}
	for _, conflict := range []string{"type", "calling-convention", "definition", "global"} {
		t.Run(conflict, func(t *testing.T) {
			mod := ctx.NewModule("conflict")
			defer mod.Dispose()
			if conflict == "global" {
				llvm.AddGlobal(mod, i32, binding.Name())
			} else {
				typ := sig
				if conflict == "type" {
					typ = llvm.FunctionType(ctx.Int16Type(), nil, false)
				}
				fn := llvm.AddFunction(mod, binding.Name(), typ)
				if conflict == "calling-convention" {
					fn.SetFunctionCallConv(llvm.FastCallConv)
				}
				if conflict == "definition" {
					b := ctx.NewBuilder()
					b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
					b.CreateRet(fn.Param(0))
					b.Dispose()
				}
			}
			if _, err := binding.Declare(mod); err == nil || !strings.Contains(err.Error(), "conflicting") {
				t.Fatalf("got %v", err)
			}
		})
	}
	other := llvm.NewContext()
	defer other.Dispose()
	otherMod := other.NewModule("other")
	defer otherMod.Dispose()
	if _, err := binding.Declare(otherMod); err == nil {
		t.Fatal("cross-context declaration accepted")
	}
	if _, err := binding.Declare(llvm.Module{}); err == nil {
		t.Fatal("nil module accepted")
	}
	mod := ctx.NewModule("recovery")
	defer mod.Dispose()
	caller := llvm.AddFunction(mod, "caller", sig)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
	if _, err := binding.Call(b, mod); err == nil {
		t.Fatal("missing argument accepted")
	}
	if !mod.NamedFunction(binding.Name()).IsNil() {
		t.Fatal("rejected call inserted declaration")
	}
	value, err := binding.Call(b, mod, caller.Param(0))
	if err != nil {
		t.Fatal(err)
	}
	b.CreateRet(value)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}
}

func TestUnnamedStructNamesAreModuleLocal(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	a, b := ctx.StructCreateNamed(""), ctx.StructCreateNamed("")
	a.StructSetBody([]llvm.Type{ctx.Int32Type()}, false)
	b.StructSetBody([]llvm.Type{ctx.Int64Type()}, false)
	resolve := func(typ llvm.Type) *Binding {
		t.Helper()
		binding, err := (Intrinsic{Name: "llvm.ssa.copy"}).Resolve(llvm.FunctionType(typ, []llvm.Type{typ}, false))
		if err != nil {
			t.Fatal(err)
		}
		return binding
	}
	first, second := resolve(a), resolve(b)
	mod := ctx.NewModule("names")
	defer mod.Dispose()
	other, err := second.Declare(mod)
	if err != nil {
		t.Fatal(err)
	}
	actual, err := first.Declare(mod)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Name() == other.Name() || actual.GlobalValueType() != first.Signature() {
		t.Fatal("module-local names conflated distinct identified structs")
	}
	caller := llvm.AddFunction(mod, "caller", first.Signature())
	builder := ctx.NewBuilder()
	defer builder.Dispose()
	builder.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
	value, err := first.Call(builder, mod, caller.Param(0))
	if err != nil {
		t.Fatal(err)
	}
	builder.CreateRet(value)
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err)
	}

	// Explicit overloaded names must be checked in the destination module
	// before a declaration is inserted, even when LLVM chooses another suffix.
	explicit, err := (Intrinsic{Name: first.Name()}).Resolve(first.Signature())
	if err != nil {
		t.Fatal(err)
	}
	otherMod := ctx.NewModule("explicit")
	defer otherMod.Dispose()
	if _, err := second.Declare(otherMod); err != nil {
		t.Fatal(err)
	}
	if _, err := explicit.Declare(otherMod); err == nil {
		t.Fatal("explicit name changed silently")
	}
	if !otherMod.NamedFunction(actual.Name()).IsNil() {
		t.Fatal("rejected explicit name inserted a declaration")
	}

	ordinary, err := (Intrinsic{Name: "llvm.ctpop.i32"}).Resolve(llvm.FunctionType(ctx.Int32Type(), []llvm.Type{ctx.Int32Type()}, false))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ordinary.Declare(otherMod); err != nil {
		t.Fatal(err)
	}
}
