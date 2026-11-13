package cabi

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestTransformModuleSkipsLLVMScalableIntrinsic(t *testing.T) {
	const childEnv = "LLGO_CABI_SCALABLE_INTRINSIC_TEST_CHILD"
	if os.Getenv(childEnv) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestTransformModuleSkipsLLVMScalableIntrinsic$")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("C ABI transformer crashed on an LLVM scalable intrinsic: %v\n%s", err, output)
		}
		return
	}

	const testIR = `
declare void @llvm.test.scalable(<vscale x 2 x i64>)
declare void @llvm.test.metadata(metadata)

define void @probe() {
entry:
  call void @llvm.test.scalable(<vscale x 2 x i64> poison)
  call void @llvm.test.metadata(metadata !0)
  ret void
}

!0 = !{}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "scalable-intrinsic.ll")
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

	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "darwin", GOARCH: "arm64"})
	defer prog.Dispose()
	NewTransformer(prog, "aarch64-apple-darwin", "", true).TransformModule("test", mod)
	if mod.NamedFunction("probe").IsNil() {
		t.Fatal("probe function disappeared")
	}
}

func TestShouldSkipLLVMIntrinsic(t *testing.T) {
	transformer := &Transformer{}
	for _, name := range []string{"llvm.test.scalable", "llvm.test.metadata"} {
		if !transformer.shouldSkipFunc(name) {
			t.Fatalf("LLVM intrinsic %q must bypass C ABI sizing", name)
		}
	}
	for _, name := range []string{"ordinary.function", "example.com/myllvm.Helper"} {
		if transformer.shouldSkipFunc(name) {
			t.Fatalf("ordinary function %q was skipped", name)
		}
	}
}

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
		{"x86_64-w64-windows-gnu", "", "amd64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWindowsAmd64); return ok }},
		{"aarch64-w64-windows-gnu", "", "arm64", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWindowsArm64); return ok }},
		{"i686-w64-windows-gnu", "", "386", func(sys TypeInfoSys) bool { _, ok := sys.(*TypeInfoWindows386); return ok }},
	}
	for _, tc := range tests {
		tr := NewTransformer(prog, tc.target, tc.abi, true)
		if tr.arch != tc.arch {
			t.Fatalf("NewTransformer(%q).arch = %q, want %q", tc.target, tr.arch, tc.arch)
		}
		if !tr.optimize {
			t.Fatal("NewTransformer did not preserve optimize")
		}
		if !tc.check(tr.sys) {
			t.Fatalf("NewTransformer(%q) selected unexpected sys implementation %T", tc.target, tr.sys)
		}
	}
	windowsProg := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "amd64"})
	defer windowsProg.Dispose()
	if tr := NewTransformer(windowsProg, "", "", true); tr.arch != "amd64" {
		t.Fatalf("implicit Windows transformer arch = %q, want amd64", tr.arch)
	} else if _, ok := tr.sys.(*TypeInfoWindowsAmd64); !ok {
		t.Fatalf("implicit Windows transformer selected %T, want *TypeInfoWindowsAmd64", tr.sys)
	}
}

func TestWindowsComdatPreservedByCABILowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	mod := ctx.NewModule("comdat")
	defer mod.Dispose()
	large := ctx.StructType([]llvm.Type{ctx.Int64Type(), ctx.Int64Type(), ctx.Int64Type()}, false)
	ft := llvm.FunctionType(ctx.VoidType(), []llvm.Type{large}, false)
	fn := llvm.AddFunction(mod, "generic", ft)
	fn.SetLinkage(llvm.LinkOnceAnyLinkage)
	comdat := mod.Comdat("generic")
	comdat.SetSelectionKind(llvm.AnyComdatSelectionKind)
	fn.SetComdat(comdat)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))
	b.CreateRetVoid()

	prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "amd64"})
	defer prog.Dispose()
	NewTransformer(prog, "x86_64-pc-windows-msvc", "", true).TransformModule("test", mod)

	lowered := mod.NamedFunction("generic")
	if lowered.IsNil() {
		t.Fatalf("lowered function not found:\n%s", mod.String())
	}
	gotComdat := lowered.Comdat()
	if gotComdat.C == nil {
		t.Fatalf("C ABI lowering dropped the function COMDAT:\n%s", lowered.String())
	}
	if got := gotComdat.SelectionKind(); got != llvm.AnyComdatSelectionKind {
		t.Fatalf("lowered function COMDAT selection = %v, want any", got)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("lowered COMDAT module is invalid: %v\n%s", err, mod.String())
	}
}

func TestWindowsARM64VoidAggregateReturnLowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Empty = type {}
%Large = type { i64, i64, i64 }

define %Empty @callee(%Large %value) {
entry:
  ret %Empty zeroinitializer
}

define %Empty @caller(%Large %value) {
entry:
  %result = call %Empty @callee(%Large %value)
  %slot = alloca %Empty
  store %Empty %result, ptr %slot
  ret %Empty %result
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "empty_return.ll")
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

	prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "arm64"})
	defer prog.Dispose()
	NewTransformer(prog, "aarch64-pc-windows-msvc", "", true).TransformModule("test", mod)

	ir := mod.String()
	for _, function := range []string{"callee", "caller"} {
		if got := mod.NamedFunction(function).GlobalValueType().ReturnType().TypeKind(); got != llvm.VoidTypeKind {
			t.Fatalf("lowered %s return kind = %v, want void:\n%s", function, got, ir)
		}
	}
	if strings.Contains(ir, "<badref>") || strings.Contains(ir, "store void") || strings.Contains(ir, "ret %Empty") {
		t.Fatalf("void aggregate result left invalid value uses:\n%s", ir)
	}
	if !strings.Contains(mod.NamedFunction("caller").String(), "call void @callee(") {
		t.Fatalf("caller did not use the lowered void ABI:\n%s", mod.NamedFunction("caller").String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("void aggregate return module is invalid: %v\n%s", err, ir)
	}
}

func TestWindowsCABITargetDetection(t *testing.T) {
	tests := []struct {
		name   string
		target *llssa.Target
		triple string
		want   bool
	}{
		{"explicit msvc", nil, "x86_64-pc-windows-msvc", true},
		{"versioned msvc", nil, "x86_64-pc-windows-msvc19.40", true},
		{"windows default environment", nil, "x86_64-pc-windows", true},
		{"mingw", nil, "x86_64-w64-windows-gnu", true},
		{"mingw short triple", nil, "x86_64-w64-mingw32", true},
		{"cygwin", nil, "x86_64-pc-windows-cygnus", false},
		{"cygwin short triple", nil, "x86_64-pc-cygwin", false},
		{"msys", nil, "x86_64-pc-msys", false},
		{"linux", nil, "x86_64-unknown-linux-gnu", false},
		{"implicit windows", &llssa.Target{GOOS: "windows"}, "", true},
		{"arch-only windows", &llssa.Target{GOOS: "windows"}, "x86_64", true},
		{"implicit linux", &llssa.Target{GOOS: "linux"}, "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := usesWindowsCABI(test.target, test.triple); got != test.want {
				t.Fatalf("usesWindowsCABI(%q) = %v, want %v", test.triple, got, test.want)
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
				checkTypeInfo(t, tr, hfa, 1, AttrWidthType, "[4 x float]")
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
			tr := NewTransformer(prog, test.triple, "", true)
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

func TestWindows386CABILowersGoAggregateToNativeLayout(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	// This is the private SSA representation of Go struct { X int8; Y int64 }:
	// Y is at offset 4 and the whole value has four-byte alignment. Native MSVC
	// C instead lays out the equivalent structure as { i8, i64 }, with Y at 8.
	bytePadding := llvm.ArrayType(ctx.Int8Type(), 3)
	goType := ctx.StructType([]llvm.Type{
		ctx.StructType([]llvm.Type{ctx.Int8Type(), bytePadding}, true),
		ctx.StructType([]llvm.Type{ctx.Int64Type()}, true),
		llvm.ArrayType(ctx.Int32Type(), 0),
	}, false)
	nativeType, changed := windows386NativeAggregateType(ctx, goType)
	if !changed {
		t.Fatal("Go/386 aggregate layout was not recognized")
	}
	if got, want := nativeType.String(), "{ i8, i64 }"; got != want {
		t.Fatalf("native aggregate type = %s, want %s", got, want)
	}
	tr := NewTransformer(prog, "i686-pc-windows-msvc", "", true)
	if got, want := tr.td.ElementOffset(goType, 1), uint64(4); got != want {
		t.Fatalf("Go aggregate second field offset = %d, want %d", got, want)
	}
	if got, want := tr.td.ElementOffset(nativeType, 1), uint64(8); got != want {
		t.Fatalf("native aggregate second field offset = %d, want %d", got, want)
	}

	mod := ctx.NewModule("windows-386-go-c-layout")
	defer mod.Dispose()
	ft := llvm.FunctionType(goType, []llvm.Type{goType}, false)
	callee := llvm.AddFunction(mod, "cRoundTrip", ft)
	caller := llvm.AddFunction(mod, "example.com/p.call", ft)
	b := ctx.NewBuilder()
	defer b.Dispose()
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(caller, "entry"))
	b.CreateRet(llvm.CreateCall(b, ft, callee, []llvm.Value{caller.Param(0)}))
	exported := llvm.AddFunction(mod, "exportRoundTrip", ft)
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(exported, "entry"))
	b.CreateRet(exported.Param(0))
	callback := llvm.AddFunction(mod, "example.com/p.callback", ft)
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(callback, "entry"))
	b.CreateRet(callback.Param(0))
	registerType := llvm.FunctionType(ctx.VoidType(), []llvm.Type{llvm.PointerType(ft, 0)}, false)
	register := llvm.AddFunction(mod, "registerRoundTrip", registerType)
	useCallback := llvm.AddFunction(mod, "example.com/p.useCallback", llvm.FunctionType(ctx.VoidType(), nil, false))
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(useCallback, "entry"))
	b.CreateCall(registerType, register, []llvm.Value{callback}, "")
	b.CreateRetVoid()
	goArray := llvm.ArrayType(goType, 2)
	arrayFT := llvm.FunctionType(goArray, []llvm.Type{goArray}, false)
	arrayCallee := llvm.AddFunction(mod, "cArrayRoundTrip", arrayFT)
	arrayCaller := llvm.AddFunction(mod, "example.com/p.callArray", arrayFT)
	b.SetInsertPointAtEnd(ctx.AddBasicBlock(arrayCaller, "entry"))
	b.CreateRet(llvm.CreateCall(b, arrayFT, arrayCallee, []llvm.Value{arrayCaller.Param(0)}))

	tr.TransformModule("test", mod)
	lowered := mod.NamedFunction("cRoundTrip").String()
	for _, want := range []string{"sret({ i8, i64 })", "byval({ i8, i64 }) align 4"} {
		if !strings.Contains(lowered, want) {
			t.Fatalf("native aggregate declaration does not contain %q:\n%s", want, lowered)
		}
	}
	arrayLowered := mod.NamedFunction("cArrayRoundTrip").String()
	for _, want := range []string{"sret([2 x { i8, i64 }])", "byval([2 x { i8, i64 }]) align 4"} {
		if !strings.Contains(arrayLowered, want) {
			t.Fatalf("native aggregate array declaration does not contain %q:\n%s", want, arrayLowered)
		}
	}
	for _, name := range []string{"exportRoundTrip", "example.com/p.callback"} {
		fn := mod.NamedFunction(name)
		if fn.IsNil() {
			t.Fatalf("missing lowered native aggregate function %s:\n%s", name, mod.String())
		}
		for _, want := range []string{"sret({ i8, i64 })", "byval({ i8, i64 }) align 4"} {
			if got := fn.String(); !strings.Contains(got, want) {
				t.Fatalf("lowered native aggregate function %s does not contain %q:\n%s", name, want, got)
			}
		}
	}
	if got := useCallback.String(); !strings.Contains(got, `ptr @"example.com/p.callback"`) {
		t.Fatalf("lowered callback was not passed directly:\n%s", got)
	}
	if legacy := mod.NamedFunction("__llgo_cdecl$example.com/p.callback"); !legacy.IsNil() {
		t.Fatalf("unexpected legacy callback wrapper:\n%s", legacy.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid Go/native aggregate bridge: %v\n%s", err, mod.String())
	}
}

func TestWindows386HostLayoutTailReturnUsesRegisterABI(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	prog := llssa.NewProgram(&llssa.Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	// SSA storage for `struct { _ structs.HostLayout; P *int; Z struct{} }`
	// keeps Go's extra tail byte (size/align 8/4). Windows/386 CABI strips
	// that wrapper, so the callable type is a 4-byte pointer and
	// (HostTail, byte) is an 8-byte register return, not a 12-byte sret.
	empty := ctx.StructType(nil, false)
	ptr := llvm.PointerType(ctx.Int8Type(), 0)
	tailPad := ctx.StructType([]llvm.Type{empty, llvm.ArrayType(ctx.Int8Type(), 4)}, true)
	hostTail := ctx.StructType([]llvm.Type{empty, ptr, tailPad}, false)
	resultPad := ctx.StructType([]llvm.Type{ctx.Int8Type(), llvm.ArrayType(ctx.Int8Type(), 3)}, true)
	pair := ctx.StructType([]llvm.Type{hostTail, resultPad}, false)

	tr := NewTransformer(prog, "i686-pc-windows-msvc", "", true)
	if got, want := tr.Sizeof(hostTail), 8; got != want {
		t.Fatalf("HostTail storage size = %d, want %d", got, want)
	}
	if got, want := tr.Sizeof(pair), 12; got != want {
		t.Fatalf("(HostTail, byte) storage size = %d, want %d", got, want)
	}
	nativeTail, changed := windows386NativeAggregateType(ctx, hostTail)
	if !changed {
		t.Fatal("HostTail Go/386 wrappers were not stripped")
	}
	if got, want := tr.Sizeof(nativeTail), 4; got != want {
		t.Fatalf("HostTail callable size = %d, want %d (%s)", got, want, nativeTail)
	}
	checkTypeInfo(t, tr, pair, 0, AttrWidthType, "i64")
}

func TestWindows386AggregateLayoutMismatchPanics(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	one := ctx.StructType([]llvm.Type{ctx.Int32Type()}, false)
	two := ctx.StructType([]llvm.Type{ctx.Int32Type(), ctx.Int32Type()}, false)
	marker := llvm.ArrayType(ctx.Int32Type(), 0)
	if !windows386GoAlignmentMarker(marker) {
		t.Fatal("zero-length integer array was not recognized as an alignment marker")
	}
	if windows386GoAlignmentMarker(llvm.ArrayType(ctx.Int32Type(), 1)) {
		t.Fatal("non-empty integer array was recognized as an alignment marker")
	}

	tests := []struct {
		name       string
		source     llvm.Type
		native     llvm.Type
		fromNative bool
		want       string
	}{
		{name: "extra Go field to native", source: two, native: one, want: "aggregate field"},
		{name: "extra Go field from native", source: two, native: one, fromNative: true, want: "aggregate field"},
		{name: "missing Go field to native", source: one, native: two, want: "aggregate field"},
		{name: "missing Go field from native", source: one, native: two, fromNative: true, want: "aggregate field"},
		{name: "unsupported kind to native", source: ctx.Int32Type(), native: ctx.Int64Type(), want: "i32"},
		{name: "unsupported kind from native", source: ctx.Int32Type(), native: ctx.Int64Type(), fromNative: true, want: "i32"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mod := ctx.NewModule(test.name)
			defer mod.Dispose()
			input := test.source
			if test.fromNative {
				input = test.native
			}
			fn := llvm.AddFunction(mod, "mismatch", llvm.FunctionType(ctx.VoidType(), []llvm.Type{input}, false))
			b := ctx.NewBuilder()
			defer b.Dispose()
			b.SetInsertPointAtEnd(ctx.AddBasicBlock(fn, "entry"))

			defer func() {
				got := recover()
				if got == nil {
					t.Fatal("layout mismatch did not panic")
				}
				message, ok := got.(string)
				if !ok || !strings.Contains(message, test.want) {
					t.Fatalf("layout mismatch panic = %v, want diagnostic containing %q", got, test.want)
				}
			}()
			if test.fromNative {
				windows386AggregateFromNative(b, fn.Param(0), test.source, test.native)
			} else {
				windows386AggregateToNative(b, fn.Param(0), test.source, test.native)
			}
		})
	}
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
		callback    []string
	}{
		{
			name: "amd64", goarch: "amd64", triple: "x86_64-pc-windows-msvc",
			declaration: []string{"declare void @cOdd(ptr sret(%Odd)", "ptr)"},
			callback:    []string{"define void @main.callback(ptr sret(%Odd)", "ptr %"},
		},
		{
			name: "arm64", goarch: "arm64", triple: "aarch64-pc-windows-msvc",
			declaration: []string{"declare i24 @cOdd(i64)"},
			callback:    []string{"define i24 @main.callback(i64 %"},
		},
		{
			name: "386", goarch: "386", triple: "i686-pc-windows-msvc",
			declaration: []string{"declare void @cOdd(ptr sret(%Odd)", "ptr byval(%Odd) align 4"},
			callback:    []string{"define void @main.callback(ptr sret(%Odd)", "ptr byval(%Odd) align 4"},
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
			tr := NewTransformer(prog, test.triple, "", true)
			tr.TransformModule("test", mod)

			for _, want := range test.declaration {
				if got := mod.NamedFunction("cOdd").String(); !strings.Contains(got, want) {
					t.Fatalf("lowered declaration does not contain %q:\n%s", want, got)
				}
			}
			callback := mod.NamedFunction("main.callback")
			if callback.IsNil() {
				t.Fatalf("lowered callback was not found:\n%s", mod.String())
			}
			for _, want := range test.callback {
				if got := callback.String(); !strings.Contains(got, want) {
					t.Fatalf("lowered callback does not contain %q:\n%s", want, got)
				}
			}
			if legacy := mod.NamedFunction("__llgo_cdecl$main.callback"); !legacy.IsNil() {
				t.Fatalf("unexpected legacy callback wrapper:\n%s", legacy.String())
			}
			if got := mod.NamedFunction("main.passCallback").String(); !strings.Contains(got, "ptr @main.callback") {
				t.Fatalf("lowered callback was not passed directly:\n%s", got)
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
	llvm.InitializeAllAsmPrinters()

	const testIR = `
%Odd = type { i8, i8, i8 }

declare x86_stdcallcc void @consume(%Odd)
declare void @registerCallback(ptr)

define x86_stdcallcc %Odd @"main.callback"(%Odd %value) {
entry:
  ret %Odd %value
}

define void @"main.call"(%Odd %value) {
entry:
  call x86_stdcallcc void @consume(%Odd %value)
  call void @registerCallback(ptr @"main.callback")
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
	NewTransformer(prog, "i686-pc-windows-msvc", "", true).TransformModule("test", mod)

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
	callback := mod.NamedFunction("main.callback")
	if callback.IsNil() {
		t.Fatalf("lowered stdcall callback not found:\n%s", mod.String())
	}
	if got := callback.FunctionCallConv(); got != llvm.X86StdcallCallConv {
		t.Fatalf("callback calling convention = %v, want x86_stdcallcc", got)
	}
	for _, want := range []string{"ptr sret(%Odd)", "ptr byval(%Odd) align 4"} {
		if got := callback.String(); !strings.Contains(got, want) {
			t.Fatalf("lowered callback does not contain %q:\n%s", want, got)
		}
	}
	if legacy := mod.NamedFunction("__llgo_stdcall$main.callback"); !legacy.IsNil() {
		t.Fatalf("unexpected legacy stdcall callback wrapper:\n%s", legacy.String())
	}
	if got := caller.String(); !strings.Contains(got, "call void @registerCallback(ptr @main.callback)") {
		t.Fatalf("lowered stdcall callback was not passed directly:\n%s", got)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("MSVC stdcall module is invalid: %v\n%s", err, mod.String())
	}
	assembly, err := prog.TargetMachine().EmitToMemoryBuffer(mod, llvm.AssemblyFile)
	if err != nil {
		t.Fatalf("emit MSVC x86 assembly: %v\n%s", err, mod.String())
	}
	defer assembly.Dispose()
	if got := string(assembly.Bytes()); !strings.Contains(got, "_consume@4") {
		t.Fatalf("stdcall declaration did not use MSVC x86 symbol decoration:\n%s", got)
	}
}

func TestMSVC386RegisterAggregateReturnPreservesLoadedValue(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Pair = type { ptr, ptr }
%Holder = type { %Pair }

define %Pair @read(ptr %holder) {
entry:
  %field = getelementptr inbounds %Holder, ptr %holder, i32 0, i32 0
  %saved = load %Pair, ptr %field, align 4
  store %Pair zeroinitializer, ptr %field, align 4
  ret %Pair %saved
}

define %Pair @readDirect(ptr %holder) {
entry:
  %field = getelementptr inbounds %Holder, ptr %holder, i32 0, i32 0
  %saved = load %Pair, ptr %field, align 4
  ret %Pair %saved
}
`
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "register_return.ll")
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
	NewTransformer(prog, "i686-pc-windows-msvc", "", true).TransformModule("test", mod)

	fn := mod.NamedFunction("read")
	if got := fn.GlobalValueType().ReturnType().String(); got != "i64" {
		t.Fatalf("lowered return type = %s, want i64:\n%s", got, fn.String())
	}
	var ret llvm.Value
	for block := fn.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if !instruction.IsAReturnInst().IsNil() {
				ret = instruction
			}
		}
	}
	if ret.IsNil() || ret.OperandsCount() != 1 {
		t.Fatalf("lowered return instruction not found:\n%s", fn.String())
	}
	value := ret.Operand(0).IsALoadInst()
	if value.IsNil() || value.Operand(0).IsAAllocaInst().IsNil() {
		t.Fatalf("lowered return reloads the mutated source instead of the saved value:\n%s", fn.String())
	}
	direct := mod.NamedFunction("readDirect")
	var directReturn llvm.Value
	for block := direct.FirstBasicBlock(); !block.IsNil(); block = llvm.NextBasicBlock(block) {
		for instruction := block.FirstInstruction(); !instruction.IsNil(); instruction = llvm.NextInstruction(instruction) {
			if !instruction.IsAReturnInst().IsNil() {
				directReturn = instruction
			}
		}
	}
	if directReturn.IsNil() || directReturn.OperandsCount() != 1 {
		t.Fatalf("lowered direct return instruction not found:\n%s", direct.String())
	}
	directValue := directReturn.Operand(0).IsALoadInst()
	if directValue.IsNil() || !directValue.Operand(0).IsAAllocaInst().IsNil() {
		t.Fatalf("adjacent load/return did not retain the direct return path:\n%s", direct.String())
	}
	if got := directValue.Alignment(); got != 4 {
		t.Fatalf("direct aggregate return alignment = %d, want 4:\n%s", got, direct.String())
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("lowered module is invalid: %v\n%s", err, mod.String())
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
	tr := NewTransformer(prog, "amd64-unknown-linux-gnu", "", true)
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

func TestClosureEnvAttributesPreservedByCABI(t *testing.T) {
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
					path := filepath.Join(t.TempDir(), "closure_env.ll")
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
					NewTransformer(prog, "amd64-unknown-linux-gnu", "", true).TransformModule("test", mod)

					callback := mod.NamedFunction("callback")
					kind := llvm.AttributeKindID(attrName)
					count := 0
					for i := 1; i <= callback.GlobalValueType().ParamTypesCount(); i++ {
						if !callback.GetEnumAttributeAtIndex(i, kind).IsNil() {
							count++
						}
					}
					if count != 1 {
						t.Fatalf("C ABI lowering retained %d %s attributes, want 1:\n%s", count, attrName, callback.String())
					}
					if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
						t.Fatalf("C ABI closure-env module is invalid: %v\n%s", err, mod.String())
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
	tr := NewTransformer(prog, "", "", false)

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
	tr := NewTransformer(prog, "amd64-unknown-linux-gnu", "", true)
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
	tr := NewTransformer(prog, "arm64-apple-darwin", "", true)
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

func TestParamHomeReusePreservesObjectIdentity(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Aggregate = type { i64, i64, i64 }

declare void @capture(ptr)
declare void @llvm.memset.p0.i64(ptr nocapture writeonly, i8, i64, i1 immarg)

define i64 @copy_then_mutate(%Aggregate %value) {
entry:
  %mutable = alloca %Aggregate, align 8
  %original = alloca %Aggregate, align 8
  store %Aggregate %value, ptr %mutable, align 8
  %mutable.field = getelementptr inbounds %Aggregate, ptr %mutable, i32 0, i32 1
  store i64 99, ptr %mutable.field, align 8
  store %Aggregate %value, ptr %original, align 8
  %original.field = getelementptr inbounds %Aggregate, ptr %original, i32 0, i32 1
  %result = load i64, ptr %original.field, align 8
  ret i64 %result
}

define i64 @rematerialize_one_copy(%Aggregate %value) {
entry:
  %copy = alloca %Aggregate, align 8
  store %Aggregate %value, ptr %copy, align 8
  store %Aggregate %value, ptr %copy, align 8
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reuse_after_memset(%Aggregate %value) {
entry:
  %copy = alloca %Aggregate, align 8
  call void @llvm.memset.p0.i64(ptr %copy, i8 0, i64 24, i1 false)
  store %Aggregate %value, ptr %copy, align 8
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reuse_natural_alignment(%Aggregate %value) {
entry:
  %copy = alloca %Aggregate
  store %Aggregate %value, ptr %copy
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field
  ret i64 %result
}

define i64 @reject_preinit_escape(%Aggregate %value) {
entry:
  %copy = alloca %Aggregate, align 8
  call void @capture(ptr %copy)
  store %Aggregate %value, ptr %copy, align 8
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reject_conditional_init(%Aggregate %value, i1 %cond) {
entry:
  %copy = alloca %Aggregate, align 8
  br i1 %cond, label %left, label %right
left:
  store %Aggregate %value, ptr %copy, align 8
  br label %done
right:
  store %Aggregate %value, ptr %copy, align 8
  br label %done
done:
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reject_stronger_alignment(%Aggregate %value) {
entry:
  %copy = alloca %Aggregate, align 32
  store %Aggregate %value, ptr %copy, align 32
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reject_wrong_allocated_type(%Aggregate %value) {
entry:
  %copy = alloca i8, align 8
  store %Aggregate %value, ptr %copy, align 8
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reject_multiple_elements(%Aggregate %value) {
entry:
  %copy = alloca %Aggregate, i64 2, align 8
  store %Aggregate %value, ptr %copy, align 8
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}

define i64 @reject_dynamic_elements(%Aggregate %value, i64 %count) {
entry:
  %copy = alloca %Aggregate, i64 %count, align 8
  store %Aggregate %value, ptr %copy, align 8
  %copy.field = getelementptr inbounds %Aggregate, ptr %copy, i32 0, i32 1
  %result = load i64, ptr %copy.field, align 8
  ret i64 %result
}
`

	for _, test := range []struct {
		name   string
		goos   string
		goarch string
		triple string
	}{
		{name: "amd64", goos: "linux", goarch: "amd64", triple: "x86_64-unknown-linux-gnu"},
		{name: "arm64", goos: "darwin", goarch: "arm64", triple: "arm64-apple-darwin"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "param_copies.ll")
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

			prog := llssa.NewProgram(&llssa.Target{GOOS: test.goos, GOARCH: test.goarch})
			defer prog.Dispose()
			NewTransformer(prog, test.triple, "", true).TransformModule("test", mod)

			got := mod.NamedFunction("copy_then_mutate").String()
			if strings.Contains(got, "getelementptr inbounds %Aggregate, ptr %mutable") {
				t.Fatalf("C ABI lowering did not reuse the first proven parameter home:\n%s", got)
			}
			if !strings.Contains(got, "getelementptr inbounds %Aggregate, ptr %original") {
				t.Fatalf("C ABI lowering aliased the independent parameter copy:\n%s", got)
			}
			singleCopy := mod.NamedFunction("rematerialize_one_copy").String()
			if strings.Contains(singleCopy, "getelementptr inbounds %Aggregate, ptr %copy") {
				t.Fatalf("C ABI lowering did not reuse the indirect parameter for one alloca:\n%s", singleCopy)
			}
			memsetCopy := mod.NamedFunction("reuse_after_memset").String()
			if !strings.Contains(memsetCopy, "call void @llvm.memset.p0.i64(ptr %copy") {
				t.Fatalf("C ABI lowering moved the pre-initialization memset to the parameter home:\n%s", memsetCopy)
			}
			if strings.Contains(memsetCopy, "getelementptr inbounds %Aggregate, ptr %copy") {
				t.Fatalf("C ABI lowering did not reuse storage after a direct memset:\n%s", memsetCopy)
			}
			naturalAlignment := mod.NamedFunction("reuse_natural_alignment").String()
			if strings.Contains(naturalAlignment, "getelementptr inbounds %Aggregate, ptr %copy") {
				t.Fatalf("C ABI lowering did not use the alloca type's natural alignment:\n%s", naturalAlignment)
			}
			for _, name := range []string{
				"reject_preinit_escape",
				"reject_conditional_init",
				"reject_stronger_alignment",
				"reject_wrong_allocated_type",
				"reject_multiple_elements",
				"reject_dynamic_elements",
			} {
				got := mod.NamedFunction(name).String()
				if !strings.Contains(got, "getelementptr inbounds %Aggregate, ptr %copy") {
					t.Fatalf("C ABI lowering reused an unproven parameter home in %s:\n%s", name, got)
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("lowered parameter-copy module is invalid: %v\n%s", err, mod.String())
			}
		})
	}
}

func TestCanReuseParamHomeRequiresEntryBlockOrder(t *testing.T) {
	const testIR = `
%Aggregate = type { i64, i64, i64 }

define void @non_entry_alloca(%Aggregate %value) {
entry:
  br label %body
body:
  %copy = alloca %Aggregate, align 8
  store %Aggregate %value, ptr %copy, align 8
  ret void
}

define void @init_before_alloca(%Aggregate %value) {
entry:
  %source = alloca %Aggregate, align 8
  store %Aggregate %value, ptr %source, align 8
  %copy = alloca %Aggregate, align 8
  ret void
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "param_home_order.ll")
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

	nonEntry := mod.NamedFunction("non_entry_alloca")
	entry := nonEntry.EntryBasicBlock()
	body := llvm.NextBasicBlock(entry)
	alloc := body.FirstInstruction()
	initStore := llvm.NextInstruction(alloc)
	if canReuseParamHome(alloc, initStore, entry, nonEntry.Param(0).Type(), 8, 8) {
		t.Fatal("parameter home outside the entry block was accepted")
	}

	earlierInit := mod.NamedFunction("init_before_alloca")
	entry = earlierInit.EntryBasicBlock()
	initStore = llvm.NextInstruction(entry.FirstInstruction())
	alloc = llvm.NextInstruction(initStore)
	alloc.SetAlignment(0)
	if alloc.Alignment() != 0 {
		t.Fatalf("failed to clear candidate alloca alignment: got %d", alloc.Alignment())
	}
	if canReuseParamHome(alloc, initStore, entry, earlierInit.Param(0).Type(), 8, 8) {
		t.Fatal("parameter initialization before its candidate alloca was accepted")
	}
}

func TestWidthReturnUsesEvaluatedValue(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()

	const testIR = `
%Small = type { i32, i32 }
%Pair = type { i64, i64 }

define %Small @return_loaded_snapshot(ptr %src) {
entry:
  %snapshot = load %Small, ptr %src, align 4
  store %Small { i32 9, i32 10 }, ptr %src, align 4
  ret %Small %snapshot
}

define i32 @width_param_copies(%Small %value) {
entry:
  %first = alloca %Small, align 4
  %second = alloca %Small, align 4
  store %Small %value, ptr %first, align 4
  %first.field = getelementptr inbounds %Small, ptr %first, i32 0, i32 0
  store i32 99, ptr %first.field, align 4
  store %Small %value, ptr %second, align 4
  %second.field = getelementptr inbounds %Small, ptr %second, i32 0, i32 0
  %result = load i32, ptr %second.field, align 4
  ret i32 %result
}

define i64 @two_width_param_copies(%Pair %value) {
entry:
  %first = alloca %Pair, align 8
  %second = alloca %Pair, align 8
  store %Pair %value, ptr %first, align 8
  %first.field = getelementptr inbounds %Pair, ptr %first, i32 0, i32 0
  store i64 99, ptr %first.field, align 8
  store %Pair %value, ptr %second, align 8
  %second.field = getelementptr inbounds %Pair, ptr %second, i32 0, i32 0
  %result = load i64, ptr %second.field, align 8
  ret i64 %result
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()
	path := filepath.Join(t.TempDir(), "width_return.ll")
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
	NewTransformer(prog, "x86_64-unknown-linux-gnu", "", true).TransformModule("test", mod)

	got := mod.NamedFunction("return_loaded_snapshot").String()
	if !strings.Contains(got, "ret i64") {
		t.Fatalf("small aggregate return was not width-lowered:\n%s", got)
	}
	if strings.Contains(got, "load i64, ptr %src") {
		t.Fatalf("C ABI lowering re-read a return load's modified source:\n%s", got)
	}
	for _, test := range []struct {
		name string
		typ  string
	}{
		{name: "width_param_copies", typ: "%Small"},
		{name: "two_width_param_copies", typ: "%Pair"},
	} {
		got := mod.NamedFunction(test.name).String()
		if strings.Contains(got, "getelementptr inbounds "+test.typ+", ptr %first") {
			t.Fatalf("C ABI lowering did not reuse the first %s parameter home:\n%s", test.name, got)
		}
		if !strings.Contains(got, "getelementptr inbounds "+test.typ+", ptr %second") {
			t.Fatalf("C ABI lowering aliased an independent %s parameter copy:\n%s", test.name, got)
		}
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("lowered width-return module is invalid: %v\n%s", err, mod.String())
	}
}
