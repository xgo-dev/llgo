package cabi

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestTargetArchAndNewTransformerArchSelection(t *testing.T) {
	if got := targetArch("riscv64-unknown-linux-gnu"); got != "riscv64" {
		t.Fatalf("targetArch(triple) = %q, want riscv64", got)
	}
	if got := targetArch("x86_64-pc-windows-msvc"); got != "amd64" {
		t.Fatalf("targetArch(x86_64 triple) = %q, want amd64", got)
	}
	if got := targetArch("aarch64-pc-windows-msvc"); got != "arm64" {
		t.Fatalf("targetArch(aarch64 triple) = %q, want arm64", got)
	}
	if got := targetArch("i686-pc-windows-msvc"); got != "386" {
		t.Fatalf("targetArch(i686 triple) = %q, want 386", got)
	}
	if got := targetArch("thumbv7em-none-eabi"); got != "arm" {
		t.Fatalf("targetArch(thumb triple) = %q, want arm", got)
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
		{"x86_64-unknown-linux-gnu", "", "amd64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoAmd64); return ok }},
		{"aarch64-apple-darwin", "", "arm64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoArm64); return ok }},
		{"arm-unknown-linux-gnueabihf", "", "arm", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoArm); return ok }},
		{"wasm32-unknown-wasip1", "", "wasm", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWasm); return ok }},
		{"riscv64-unknown-linux-gnu", "lp64d", "riscv64", func(sys TypeInfoSys) bool {
			rv, ok := sys.(*TypeInfoRiscv64)
			return ok && rv.mabi == "lp64d"
		}},
		{"i386-unknown-linux-gnu", "", "386", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfo386); return ok }},
		{"x86_64-pc-windows-msvc", "", "amd64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWindowsAmd64); return ok }},
		{"aarch64-pc-windows-msvc", "", "arm64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWindowsArm64); return ok }},
		{"i686-pc-windows-msvc", "", "386", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWindows386); return ok }},
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
	windowsProg := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "amd64"})
	defer windowsProg.Dispose()
	if tr := NewTransformer(windowsProg, "", "", ModeCFunc, true); tr.arch != "amd64" {
		t.Fatalf("implicit Windows transformer arch = %q, want amd64", tr.arch)
	} else if _, ok := tr.sys.(*TypeInfoWindowsAmd64); !ok {
		t.Fatalf("implicit Windows transformer selected %T, want *TypeInfoWindowsAmd64", tr.sys)
	}
}

func TestMSVCTargetDetection(t *testing.T) {
	tests := []struct {
		name   string
		target *llssa.Target
		triple string
		want   bool
	}{
		{"explicit msvc", nil, "x86_64-pc-windows-msvc", true},
		{"versioned msvc", nil, "x86_64-pc-windows-msvc19.40", true},
		{"windows default environment", nil, "x86_64-pc-windows", true},
		{"mingw", nil, "x86_64-w64-windows-gnu", false},
		{"mingw short triple", nil, "x86_64-w64-mingw32", false},
		{"cygwin", nil, "x86_64-pc-windows-cygnus", false},
		{"linux", nil, "x86_64-unknown-linux-gnu", false},
		{"implicit windows", &llssa.Target{GOOS: "windows"}, "", true},
		{"arch-only windows", &llssa.Target{GOOS: "windows"}, "x86_64", true},
		{"implicit linux", &llssa.Target{GOOS: "linux"}, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isMSVCTarget(test.target, test.triple); got != test.want {
				t.Fatalf("isMSVCTarget(%q) = %v, want %v", test.triple, got, test.want)
			}
		})
	}
}

func TestMSVCAggregateClassification(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	tests := []struct {
		name   string
		goarch string
		triple string
		check  func(t *testing.T, ctx llvm.Context, tr *Transformer)
	}{
		{
			name:   "amd64",
			goarch: "amd64",
			triple: "x86_64-pc-windows-msvc",
			check: func(t *testing.T, ctx llvm.Context, tr *Transformer) {
				checkTypeInfo(t, tr, ctx.VoidType(), 0, AttrVoid, "void")
				checkTypeInfo(t, tr, ctx.Int32Type(), 1, AttrNone, "i32")
				checkTypeInfo(t, tr, ctx.StructType(nil, false), 0, AttrWidthType, "i32")
				checkTypeInfo(t, tr, ctx.StructType(nil, false), 1, AttrWidthType, "i32")
				for _, width := range []int{1, 2, 4, 8} {
					aggregate := ctx.StructType([]llvm.Type{ctx.IntType(width * 8)}, false)
					checkTypeInfo(t, tr, aggregate, 0, AttrWidthType, "i"+strconv.Itoa(width*8))
					checkTypeInfo(t, tr, aggregate, 1, AttrWidthType, "i"+strconv.Itoa(width*8))
				}
				for _, width := range []int{3, 5, 16} {
					aggregate := ctx.StructType([]llvm.Type{llvm.ArrayType(ctx.Int8Type(), width)}, false)
					checkTypeInfo(t, tr, aggregate, 0, AttrPointer, "ptr")
					checkTypeInfo(t, tr, aggregate, 1, AttrPointer, "ptr")
				}
				checkTypeInfo(t, tr, ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.Int64Type()}, false), 0, AttrPointer, "ptr")
				if tr.sys.SupportByVal() {
					t.Fatal("Microsoft x64 indirect aggregates must not use byval")
				}
			},
		},
		{
			name:   "arm64",
			goarch: "arm64",
			triple: "aarch64-pc-windows-msvc",
			check: func(t *testing.T, ctx llvm.Context, tr *Transformer) {
				checkTypeInfo(t, tr, ctx.StructType(nil, false), 0, AttrVoid, "void")
				checkTypeInfo(t, tr, ctx.StructType(nil, false), 1, AttrVoid, "void")
				odd := ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
				checkTypeInfo(t, tr, odd, 0, AttrWidthType, "i24")
				checkTypeInfo(t, tr, odd, 1, AttrWidthType, "i64")
				hfa := ctx.StructType([]llvm.Type{ctx.FloatType(), ctx.FloatType(), ctx.FloatType(), ctx.FloatType()}, false)
				checkTypeInfo(t, tr, hfa, 0, AttrNone, hfa.String())
				checkTypeInfo(t, tr, hfa, 1, AttrNone, hfa.String())
				large := ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.Int64Type(), ctx.Int64Type()}, false)
				checkTypeInfo(t, tr, large, 0, AttrPointer, "ptr")
				checkTypeInfo(t, tr, large, 1, AttrPointer, "ptr")
			},
		},
		{
			name:   "386",
			goarch: "386",
			triple: "i686-pc-windows-msvc",
			check: func(t *testing.T, ctx llvm.Context, tr *Transformer) {
				checkTypeInfo(t, tr, ctx.VoidType(), 0, AttrVoid, "void")
				checkTypeInfo(t, tr, ctx.Int32Type(), 1, AttrNone, "i32")
				checkTypeInfo(t, tr, ctx.StructType(nil, false), 0, AttrVoid, "void")
				checkTypeInfo(t, tr, ctx.StructType(nil, false), 1, AttrPointer, "ptr")
				odd := ctx.StructType([]llvm.Type{ctx.Int8Type(), ctx.Int8Type(), ctx.Int8Type()}, false)
				checkTypeInfo(t, tr, odd, 0, AttrPointer, "ptr")
				checkTypeInfo(t, tr, odd, 1, AttrPointer, "ptr")
				pair := ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int32Type()}, false)
				checkTypeInfo(t, tr, pair, 0, AttrWidthType, "i64")
				checkTypeInfo(t, tr, pair, 1, AttrExtract, pair.String())
				// Clang 19 expands an unpadded pair of 64-bit scalar fields,
				// but passes padded Win32 aggregates byval at stack alignment 4.
				unpadded := ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.DoubleType()}, false)
				if info := checkTypeInfo(t, tr, unpadded, 1, AttrExtract, unpadded.String()); info.ByValAlign != 0 {
					t.Fatalf("unpadded aggregate byval alignment = %d, want 0", info.ByValAlign)
				}
				internallyPadded := ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int64Type()}, false)
				if info := checkTypeInfo(t, tr, internallyPadded, 1, AttrPointer, "ptr"); info.ByValAlign != 4 {
					t.Fatalf("internally padded aggregate byval alignment = %d, want 4", info.ByValAlign)
				}
				trailingPadded := ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.Int32Type()}, false)
				if info := checkTypeInfo(t, tr, trailingPadded, 1, AttrPointer, "ptr"); info.ByValAlign != 4 {
					t.Fatalf("trailing padded aggregate byval alignment = %d, want 4", info.ByValAlign)
				}
				pointer := ctx.StructType([]llvm.Type{llvm.PointerType(ctx.Int8Type(), 0)}, false)
				checkTypeInfo(t, tr, pointer, 0, AttrWidthType, "ptr")
				checkTypeInfo(t, tr, pointer, 1, AttrWidthType, "ptr")
				if windows386CanExtract(nil) {
					t.Fatal("empty structure element list must not be expanded")
				}
				if windows386CanExtract([]llvm.Type{ctx.StructType([]llvm.Type{ctx.Int32Type()}, false)}) {
					t.Fatal("nested structures must not be expanded")
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: test.goarch})
			defer prog.Dispose()
			tr := NewTransformer(prog, test.triple, "", ModeCFunc, true)
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			test.check(t, ctx, tr)
		})
	}
}

func checkTypeInfo(t *testing.T, tr *Transformer, typ llvm.Type, index int, kind AttrKind, type1 string) *TypeInfo {
	t.Helper()
	ftyp := llvm.FunctionType(typ.Context().VoidType(), nil, false)
	info := tr.GetTypeInfo(typ.Context(), ftyp, typ, index)
	if info.Kind != kind || info.Type1.String() != type1 {
		t.Fatalf("GetTypeInfo(%s, index %d) = kind %v, type %s; want kind %v, type %s",
			typ, index, info.Kind, info.Type1, kind, type1)
	}
	return info
}

func TestMSVCCallAndCallbackLowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Odd = type { i8, i8, i8 }
%Padded = type { i32, i64 }

declare %Odd @cOdd(%Odd)
declare void @registerCallback(ptr)
declare void @cVararg(%Padded, ...)

define %Padded @cPadded(%Padded %value) {
entry:
  %slot = alloca %Padded, align 8
  store %Padded %value, ptr %slot, align 8
  %loaded = load %Padded, ptr %slot, align 8
  ret %Padded %loaded
}

define %Odd @"main.call"(%Odd %value) {
entry:
  %result = call %Odd @cOdd(%Odd %value)
  ret %Odd %result
}

define %Odd @"main.callback"(%Odd %value) {
entry:
  ret %Odd %value
}

define void @"main.passCallback"() {
entry:
  call void @registerCallback(ptr @"main.callback")
  ret void
}

define void @"main.vararg"(%Padded %value) {
entry:
  call void (%Padded, ...) @cVararg(%Padded %value, i32 17, double 2.500000e+00)
  ret void
}
`
	tests := []struct {
		name        string
		goarch      string
		triple      string
		declaration []string
		wrapper     []string
	}{
		{
			name: "amd64", goarch: "amd64", triple: "x86_64-pc-windows-msvc",
			declaration: []string{"declare void @cOdd(ptr sret(%Odd)", "ptr)"},
			wrapper:     []string{"define linkonce void @\"__llgo_cdecl$main.callback\"(ptr sret(%Odd)", "ptr %"},
		},
		{
			name: "arm64", goarch: "arm64", triple: "aarch64-pc-windows-msvc",
			declaration: []string{"declare i24 @cOdd(i64)"},
			wrapper:     []string{"define linkonce i24 @\"__llgo_cdecl$main.callback\"(i64 %"},
		},
		{
			name: "386", goarch: "386", triple: "i686-pc-windows-msvc",
			declaration: []string{"declare void @cOdd(ptr sret(%Odd)", "ptr byval(%Odd) align 4"},
			wrapper:     []string{"define linkonce void @\"__llgo_cdecl$main.callback\"(ptr sret(%Odd)", "ptr byval(%Odd) align 4"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "msvc.ll")
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

			prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: test.goarch})
			defer prog.Dispose()
			tr := NewTransformer(prog, test.triple, "", ModeCFunc, true)
			tr.TransformModule("test", mod)

			for _, want := range test.declaration {
				if got := mod.NamedFunction("cOdd").String(); !strings.Contains(got, want) {
					t.Fatalf("lowered declaration does not contain %q:\n%s", want, got)
				}
			}
			wrapper := mod.NamedFunction("__llgo_cdecl$main.callback")
			if wrapper.IsNil() {
				t.Fatalf("callback wrapper was not generated:\n%s", mod.String())
			}
			for _, want := range test.wrapper {
				if got := wrapper.String(); !strings.Contains(got, want) {
					t.Fatalf("lowered callback wrapper does not contain %q:\n%s", want, got)
				}
			}
			if got := mod.NamedFunction("main.call").String(); !strings.Contains(got, "call ") || !strings.Contains(got, "@cOdd(") {
				t.Fatalf("call site was not preserved and lowered:\n%s", got)
			}
			if got := mod.NamedFunction("main.vararg").String(); !strings.Contains(got, "i32 17") || !strings.Contains(got, "double 2.500000e+00") {
				t.Fatalf("lowering dropped C variadic operands:\n%s", got)
			}
			if test.goarch == "386" {
				paddedFn := mod.NamedFunction("cPadded")
				padded := paddedFn.String()
				if !strings.Contains(padded, "ptr byval(%Padded) align 4") {
					t.Fatalf("lowered padded x86 aggregate lost its byval alignment:\n%s", padded)
				}
				if !strings.Contains(padded, "ptr %slot, align 8") {
					t.Fatalf("lowering redirected naturally aligned local accesses to the 4-byte-aligned byval pointer:\n%s", padded)
				}
				alignedLoad := false
				for block := paddedFn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
					for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
						if load := instruction.IsALoadInst(); !load.IsNil() && strings.Contains(load.String(), "load %Padded") && load.Alignment() == 4 {
							alignedLoad = true
						}
					}
				}
				if !alignedLoad {
					t.Fatalf("lowered padded x86 aggregate has no 4-byte-aligned incoming load:\n%s", padded)
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("MSVC C ABI module is invalid: %v\n%s", err, mod.String())
			}
		})
	}
}

func TestMSVC386CallingConventionLowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Odd = type { i8, i8, i8 }

declare x86_stdcallcc void @consume(%Odd)

define void @"main.call"(%Odd %value) {
entry:
  call x86_stdcallcc void @consume(%Odd %value)
  ret void
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "msvc_stdcall.ll")
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

	prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	NewTransformer(prog, "i686-pc-windows-msvc", "", ModeCFunc, true).TransformModule("test", mod)

	consume := mod.NamedFunction("consume")
	if got := consume.FunctionCallConv(); got != llvm.X86StdcallCallConv {
		t.Fatalf("lowered declaration calling convention = %v, want x86_stdcallcc", got)
	}
	var loweredCall llvm.Value
	caller := mod.NamedFunction("main.call")
	for block := caller.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if call := instruction.IsACallInst(); !call.IsNil() && call.CalledValue() == consume {
				loweredCall = call
			}
		}
	}
	if loweredCall.IsNil() {
		t.Fatalf("lowered stdcall call not found:\n%s", caller.String())
	}
	if got := loweredCall.InstructionCallConv(); got != llvm.X86StdcallCallConv {
		t.Fatalf("lowered call calling convention = %v, want x86_stdcallcc", got)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("MSVC stdcall module is invalid: %v\n%s", err, mod.String())
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
