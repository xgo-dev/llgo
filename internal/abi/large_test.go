//go:build !llgo

package abi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestLargeAggregateThreshold(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()
	l := largeAggregateLowerer{td: td}

	if l.isLargeAggregate(ctx.Int64Type()) {
		t.Fatal("scalar type was classified as a large aggregate")
	}
	if l.isLargeAggregate(llvm.ArrayType(ctx.Int8Type(), int(MaxImplicitStackVarSize))) {
		t.Fatal("aggregate at the implicit stack limit was classified as large")
	}
	if !l.isLargeAggregate(llvm.ArrayType(ctx.Int8Type(), int(MaxImplicitStackVarSize+1))) {
		t.Fatal("aggregate above the implicit stack limit was not classified as large")
	}
}

func TestLowerLargeAggregates(t *testing.T) {
	const testIR = `
%Large = type [65537 x i8]
%Small = type [65536 x i8]

define %Large @callee(ptr nonnull %src) #1 {
entry:
  %value = load %Large, ptr %src, align 1
  %first = getelementptr inbounds %Large, ptr %src, i64 0, i64 0
  store i8 9, ptr %first, align 1
  ret %Large %value
}

define i8 @caller(ptr %src) {
entry:
  %value = call %Large @callee(ptr "llgo.reflect.methodbyname.name"="1" %src) #0
  %dst = alloca %Large, align 1
  store %Large %value, ptr %dst, align 1
  %first = getelementptr inbounds %Large, ptr %dst, i64 0, i64 0
  %result = load i8, ptr %first, align 1
  ret i8 %result
}

define %Large @wrapper(ptr %src) {
entry:
  %value = call %Large @callee(ptr %src)
  ret %Large %value
}

define %Large @indirect(ptr %fn, ptr %src) {
entry:
  %value = call %Large %fn(ptr %src)
  ret %Large %value
}

define %Large @self_copy(ptr %src) {
entry:
  %copy = load %Large, ptr %src, align 1
  store %Large %copy, ptr %src, align 1
  %result = load %Large, ptr %src, align 1
  ret %Large %result
}

define %Small @small(ptr %src) {
entry:
  %value = load %Small, ptr %src, align 1
  ret %Small %value
}

attributes #0 = { "llgo.reflect.methodbyname"="value" }
attributes #1 = { noinline }
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "large.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()

	LowerLargeAggregates(td, mod)

	callee := mod.NamedFunction("callee").String()
	if !strings.Contains(callee, "define void @callee(ptr sret([65537 x i8])") {
		t.Fatalf("large return was not lowered to sret:\n%s", callee)
	}
	if !strings.Contains(callee, "ptr nonnull %") || !strings.Contains(callee, "noinline") {
		t.Fatalf("callee attributes were not preserved:\n%s", callee)
	}
	copyAtLoad := strings.Index(callee, "call void @llvm.memcpy")
	mutation := strings.Index(callee, "store i8 9")
	if copyAtLoad < 0 || mutation < 0 || copyAtLoad >= mutation {
		t.Fatalf("return value was not copied before source mutation:\n%s", callee)
	}
	if strings.Contains(callee, "load [65537 x i8]") || strings.Contains(callee, "store [65537 x i8]") {
		t.Fatalf("callee retained a direct large aggregate copy:\n%s", callee)
	}

	caller := mod.NamedFunction("caller").String()
	for _, want := range []string{
		`call ptr @"github.com/xgo-dev/llgo/runtime/internal/runtime.AllocU"(i64 65537)`,
		"call void @callee(ptr sret([65537 x i8])",
		`ptr "llgo.reflect.methodbyname.name"="1"`,
		"call void @llvm.memcpy",
	} {
		if !strings.Contains(caller, want) {
			t.Fatalf("transformed caller missing %q:\n%s", want, caller)
		}
	}
	if strings.Contains(caller, "load [65537 x i8]") || strings.Contains(caller, "store [65537 x i8]") {
		t.Fatalf("caller reconstructed the large return as an SSA value:\n%s", caller)
	}
	if !strings.Contains(mod.String(), `"llgo.reflect.methodbyname"="value"`) {
		t.Fatalf("reflect MethodByName call marker was not preserved:\n%s", mod.String())
	}

	for _, name := range []string{"wrapper", "indirect", "self_copy"} {
		ir := mod.NamedFunction(name).String()
		if strings.Contains(ir, "load [65537 x i8]") || strings.Contains(ir, "store [65537 x i8]") {
			t.Fatalf("%s retained a direct large aggregate copy:\n%s", name, ir)
		}
	}
	if got := strings.Count(mod.NamedFunction("self_copy").String(), "call void @llvm.memcpy"); got != 1 {
		t.Fatalf("self_copy has %d memcpy calls, want 1:\n%s", got, mod.NamedFunction("self_copy").String())
	}

	small := mod.NamedFunction("small").String()
	if !strings.Contains(small, "define [65536 x i8] @small") || !strings.Contains(small, "load [65536 x i8]") {
		t.Fatalf("aggregate at the threshold was unexpectedly lowered:\n%s", small)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("transformed module is invalid: %v\n%s", err, mod.String())
	}
}

func TestLowerLargeAggregatePreservesClosureEnvAttribute(t *testing.T) {
	const testIR = `
%Large = type [65537 x i8]

define %Large @callee(ptr nest %env) {
entry:
  ret %Large zeroinitializer
}

define %Large @calleeSwift(ptr swiftself %env) {
entry:
  ret %Large zeroinitializer
}

define void @caller(ptr %env) {
entry:
  %unused = call %Large @callee(ptr nest %env)
  %unusedSwift = call %Large @calleeSwift(ptr swiftself %env)
  ret void
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "large_closure_env.ll")
	if err := os.WriteFile(path, []byte(testIR), 0o644); err != nil {
		t.Fatal(err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Dispose()
	td := llvm.NewTargetData("e-m:o-i64:64-i128:128-n32:64-S128")
	defer td.Dispose()

	LowerLargeAggregates(td, mod)

	nest := llvm.AttributeKindID("nest")
	callee := mod.NamedFunction("callee")
	if attr := callee.GetEnumAttributeAtIndex(2, nest); attr.IsNil() {
		t.Fatalf("large return lowering lost nest after inserting sret:\n%s", callee.String())
	}
	if attr := callee.GetEnumAttributeAtIndex(1, nest); !attr.IsNil() {
		t.Fatalf("large return lowering left nest on the new sret parameter:\n%s", callee.String())
	}
	swiftself := llvm.AttributeKindID("swiftself")
	calleeSwift := mod.NamedFunction("calleeSwift")
	if attr := calleeSwift.GetEnumAttributeAtIndex(2, swiftself); attr.IsNil() {
		t.Fatalf("large return lowering lost swiftself after inserting sret:\n%s", calleeSwift.String())
	}

	caller := mod.NamedFunction("caller")
	var nestedCall, swiftselfCall llvm.Value
	for block := caller.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if call := instruction.IsACallInst(); !call.IsNil() {
				switch call.CalledValue().Name() {
				case "callee":
					nestedCall = call
				case "calleeSwift":
					swiftselfCall = call
				}
			}
		}
	}
	if nestedCall.IsNil() {
		t.Fatalf("large return lowering removed the callee call:\n%s", caller.String())
	}
	if attr := nestedCall.GetCallSiteEnumAttribute(2, nest); attr.IsNil() {
		t.Fatalf("large return call lowering lost nest after inserting sret:\n%s", caller.String())
	}
	if swiftselfCall.IsNil() || swiftselfCall.GetCallSiteEnumAttribute(2, swiftself).IsNil() {
		t.Fatalf("large return call lowering lost swiftself after inserting sret:\n%s", caller.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("large-return closure-env module is invalid: %v\n%s", err, mod.String())
	}
}
