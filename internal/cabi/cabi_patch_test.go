//go:build !llgo
// +build !llgo

package cabi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestDevLTOGlobalDCETargetArchAndNewTransformerArchSelection(t *testing.T) {
	if got := targetArch("riscv64-unknown-linux-gnu"); got != "riscv64" {
		t.Fatalf("targetArch(triple) = %q, want riscv64", got)
	}
	if got := targetArch("wasm"); got != "wasm" {
		t.Fatalf("targetArch(single arch) = %q, want wasm", got)
	}

	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	tests := []struct {
		target string
		abi    string
		arch   string
		check  func(TypeInfoSys) bool
	}{
		{"xtensa-esp32-none-elf", "", "xtensa", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoEsp32); return ok }},
		{"riscv32-unknown-elf", "ilp32f", "riscv32", func(sys TypeInfoSys) bool {
			rv, ok := sys.(*TypeInfoRiscv32)
			return ok && rv.mabi == "ilp32f"
		}},
		{"amd64-unknown-linux-gnu", "", "amd64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoAmd64); return ok }},
		{"arm64-apple-darwin", "", "arm64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoArm64); return ok }},
		{"arm-unknown-linux-gnueabihf", "", "arm", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoArm); return ok }},
		{"wasm-unknown-wasip1", "", "wasm", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWasm); return ok }},
		{"riscv64-unknown-linux-gnu", "lp64d", "riscv64", func(sys TypeInfoSys) bool {
			rv, ok := sys.(*TypeInfoRiscv64)
			return ok && rv.mabi == "lp64d"
		}},
		{"386-unknown-linux-gnu", "", "386", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfo386); return ok }},
	}
	for _, tc := range tests {
		tr := NewTransformer(prog, tc.target, tc.abi, ModeCFunc, true)
		if tr.arch != tc.arch {
			t.Fatalf("NewTransformer(%q).arch = %q, want %q", tc.target, tr.arch, tc.arch)
		}
		if tr.mode != ModeCFunc || !tr.optimize {
			t.Fatalf("NewTransformer did not preserve mode/optimize: mode=%v optimize=%v", tr.mode, tr.optimize)
		}
		if !tc.check(tr.sys) {
			t.Fatalf("NewTransformer(%q) selected unexpected sys implementation %T", tc.target, tr.sys)
		}
	}
}

func TestDevLTOGlobalDCEFuncNoUnwindCreatesNounwindAttribute(t *testing.T) {
	ctx := llvm.NewContext()
	attr := funcNoUnwind(ctx)
	if attr.IsNil() {
		t.Fatal("funcNoUnwind returned nil attribute")
	}
	if got, want := attr.GetEnumKind(), int(llvm.AttributeKindID("nounwind")); got != want {
		t.Fatalf("funcNoUnwind kind = %d, want %d", got, want)
	}
	if got := attr.GetEnumValue(); got != 0 {
		t.Fatalf("funcNoUnwind value = %d, want 0", got)
	}
}

func TestClosureEnvAttributeRemappedByCABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Value = type { ptr, ptr, i64 }

define %Value @callee(ptr %g, ptr %out, ptr nest %env, %Value %value) {
entry:
  ret %Value %value
}

define %Value @caller(ptr %g, ptr %out, ptr nest %env, %Value %value) {
entry:
  %result = call %Value @callee(ptr %g, ptr %out, ptr nest %env, %Value %value)
  ret %Value %result
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "closure_env.ll")
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

	prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	tr := NewTransformer(prog, "amd64-unknown-linux-gnu", "", ModeAllFunc, true)
	tr.TransformModule("test", mod)

	nest := llvm.AttributeKindID("nest")
	callee := mod.NamedFunction("callee")
	if attr := callee.GetEnumAttributeAtIndex(4, nest); attr.IsNil() {
		t.Fatalf("C ABI lowering lost/remapped nest on the definition:\n%s", callee.String())
	}
	if attr := callee.GetEnumAttributeAtIndex(3, nest); !attr.IsNil() {
		t.Fatalf("C ABI lowering left nest on the old definition parameter:\n%s", callee.String())
	}

	caller := mod.NamedFunction("caller")
	var nestedCall llvm.Value
	for block := caller.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if call := instruction.IsACallInst(); !call.IsNil() && call.CalledValue().Name() == "callee" {
				nestedCall = call
			}
		}
	}
	if nestedCall.IsNil() {
		t.Fatalf("transformed caller has no callee call:\n%s", caller.String())
	}
	if attr := nestedCall.GetCallSiteEnumAttribute(4, nest); attr.IsNil() {
		t.Fatalf("C ABI lowering lost/remapped nest on the call:\n%s", caller.String())
	}
	if attr := nestedCall.GetCallSiteEnumAttribute(3, nest); !attr.IsNil() {
		t.Fatalf("C ABI lowering left nest on the old call parameter:\n%s", caller.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("C ABI closure-env module is invalid: %v\n%s", err, mod.String())
	}
}

func TestClosureEnvAttributePreservedByCallbackWrapper(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Value = type { ptr, ptr, i64 }

define RETURN @callback(ptr ATTR %env, %Value %value) {
entry:
  RET
}
`
	returnCases := []struct {
		name string
		typ  string
		ret  string
	}{
		{name: "aggregate", typ: "%Value", ret: "ret %Value %value"},
		{name: "void", typ: "void", ret: "ret void"},
		{name: "scalar", typ: "i64", ret: "ret i64 7"},
	}
	for _, returnCase := range returnCases {
		t.Run(returnCase.name, func(t *testing.T) {
			for _, attrName := range []string{"nest", "swiftself"} {
				t.Run(attrName, func(t *testing.T) {
					ctx := llvm.NewContext()
					defer ctx.Dispose()
					path := filepath.Join(t.TempDir(), "closure_env_callback.ll")
					ir := strings.NewReplacer(
						"RETURN", returnCase.typ,
						"RET", returnCase.ret,
						"ATTR", attrName,
					).Replace(testIR)
					if err := os.WriteFile(path, []byte(ir), 0o644); err != nil {
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

					prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
					defer prog.Dispose()
					tr := NewTransformer(prog, "amd64-unknown-linux-gnu", "", ModeAllFunc, true)
					callback := mod.NamedFunction("callback")
					wrapper, ok := tr.transformCallbackFunc(mod, callback)
					if !ok {
						t.Fatalf("callback wrapper was not required:\n%s", mod.String())
					}

					kind := llvm.AttributeKindID(attrName)
					var wrapperHasAttr bool
					for i := 1; i <= wrapper.GlobalValueType().ParamTypesCount(); i++ {
						if !wrapper.GetEnumAttributeAtIndex(i, kind).IsNil() {
							wrapperHasAttr = true
							break
						}
					}
					if !wrapperHasAttr {
						t.Fatalf("callback wrapper lost/remapped %s:\n%s", attrName, wrapper.String())
					}
					var callbackCall llvm.Value
					for block := wrapper.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
						for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
							if call := instruction.IsACallInst(); !call.IsNil() && call.CalledValue() == callback {
								callbackCall = call
							}
						}
					}
					if callbackCall.IsNil() || callbackCall.GetCallSiteEnumAttribute(1, kind).IsNil() {
						t.Fatalf("callback wrapper call lost %s:\n%s", attrName, wrapper.String())
					}
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatalf("C ABI callback closure-env module is invalid: %v\n%s", err, mod.String())
					}
				})
			}
		})
	}
}

func TestSetSkipFuncsAndShouldSkipCall(t *testing.T) {
	tr := &Transformer{}
	tr.SetSkipFuncs([]string{" foo ", "", "bar"})

	if !tr.shouldSkipFunc("foo") {
		t.Fatalf("shouldSkipFunc(foo) = false, want true")
	}
	if !tr.shouldSkipFunc("bar") {
		t.Fatalf("shouldSkipFunc(bar) = false, want true")
	}
	if tr.shouldSkipFunc("") {
		t.Fatalf("shouldSkipFunc(\"\") = true, want false")
	}
	if tr.shouldSkipFunc("baz") {
		t.Fatalf("shouldSkipFunc(baz) = true, want false")
	}

	ctx := llvm.NewContext()
	mod := ctx.NewModule("m")
	fty := llvm.FunctionType(ctx.VoidType(), nil, false)

	callee := llvm.AddFunction(mod, "foo", fty)
	caller := llvm.AddFunction(mod, "caller", fty)
	b := ctx.NewBuilder()
	entry := ctx.AddBasicBlock(caller, "entry")
	b.SetInsertPointAtEnd(entry)
	directCall := llvm.CreateCall(b, fty, callee, nil)
	b.CreateRetVoid()
	if !tr.shouldSkipCall(directCall) {
		t.Fatalf("shouldSkipCall(direct call to foo) = false, want true")
	}

	ptrTy := llvm.PointerType(fty, 0)
	caller2Ty := llvm.FunctionType(ctx.VoidType(), []llvm.Type{ptrTy}, false)
	caller2 := llvm.AddFunction(mod, "caller2", caller2Ty)
	b2 := ctx.NewBuilder()
	entry2 := ctx.AddBasicBlock(caller2, "entry")
	b2.SetInsertPointAtEnd(entry2)
	indirectCall := b2.CreateCall(fty, caller2.Param(0), nil, "")
	b2.CreateRetVoid()
	if tr.shouldSkipCall(indirectCall) {
		t.Fatalf("shouldSkipCall(indirect call) = true, want false")
	}
}

func TestRuntimeHeaderWrapAndTypeInfo(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	tr := NewTransformer(prog, "", "", ModeAllFunc, false)

	ctx := llvm.NewContext()
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	sliceTy := ctx.StructCreateNamed("github.com/xgo-dev/llgo/runtime/internal/runtime.Slice")
	sliceTy.StructSetBody([]llvm.Type{ptr, ctx.Int64Type(), ctx.Int64Type()}, false)

	if !tr.IsWrapType(ctx, llvm.FunctionType(ctx.VoidType(), nil, false), sliceTy, 1) {
		t.Fatalf("IsWrapType should be true for runtime Slice header")
	}
	info := tr.GetTypeInfo(ctx, llvm.FunctionType(ctx.VoidType(), nil, false), sliceTy, 1)
	if info.Kind == AttrNone {
		t.Fatalf("GetTypeInfo should not keep AttrNone for runtime Slice")
	}
	if info.Size == 0 || info.Align == 0 {
		t.Fatalf("GetTypeInfo size/align should be non-zero, got size=%d align=%d", info.Size, info.Align)
	}
}

func TestReflectMethodByNameNameArgAttributeRemapped(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%String = type { ptr, i64 }
%Value = type { ptr, ptr, i64 }

declare void @callee(%Value, %String)

define void @caller(%Value %v, %String %name) {
entry:
  call void @callee(%Value %v, %String "llgo.reflect.methodbyname.name"="1" %name) #0
  ret void
}

attributes #0 = { "llgo.reflect.methodbyname"="value" }
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tmpfile := filepath.Join(t.TempDir(), "reflect_methodbyname_attr.ll")
	if err := os.WriteFile(tmpfile, []byte(testIR), 0644); err != nil {
		t.Fatalf("Failed to write test IR: %v", err)
	}
	buf, err := llvm.NewMemoryBufferFromFile(tmpfile)
	if err != nil {
		t.Fatalf("Failed to read test IR: %v", err)
	}
	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatalf("Failed to parse test IR: %v", err)
	}
	defer mod.Dispose()

	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	tr := NewTransformer(prog, "amd64-unknown-linux-gnu", "", ModeAllFunc, true)
	tr.TransformModule("test", mod)

	caller := mod.NamedFunction("caller")
	if caller.IsNil() {
		t.Fatal("caller function not found")
	}
	ir := caller.String()
	if !strings.Contains(mod.String(), `"llgo.reflect.methodbyname"="value"`) {
		t.Fatalf("reflect MethodByName call marker was not preserved:\n%s", mod.String())
	}
	if !strings.Contains(ir, `ptr "llgo.reflect.methodbyname.name"="1"`) {
		t.Fatalf("reflect MethodByName name marker was not remapped to string data pointer:\n%s", ir)
	}
	if strings.Contains(ir, `i64 "llgo.reflect.methodbyname.name"="1"`) {
		t.Fatalf("reflect MethodByName name marker should not be remapped to string length:\n%s", ir)
	}
}

func TestPreloweredSRetAttributePreserved(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Large = type [65537 x i8]
%Param = type { i64, i64, i64 }

define void @callee(ptr sret(%Large) %result, %Param %param) {
entry:
  ret void
}

define void @caller(ptr %result, %Param %param) {
entry:
  call void @callee(ptr sret(%Large) %result, %Param %param)
  ret void
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "prelowered_sret.ll")
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

	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	tr := NewTransformer(prog, "arm64-apple-darwin", "", ModeAllFunc, true)
	tr.TransformModule("test", mod)

	callee := mod.NamedFunction("callee").String()
	if !strings.Contains(callee, "define void @callee(ptr sret([65537 x i8])") {
		t.Fatalf("pre-lowered function lost its sret attribute:\n%s", callee)
	}
	caller := mod.NamedFunction("caller").String()
	if !strings.Contains(caller, "call void @callee(ptr sret([65537 x i8])") {
		t.Fatalf("pre-lowered call lost its sret attribute:\n%s", caller)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("transformed module is invalid: %v\n%s", err, mod.String())
	}
}
