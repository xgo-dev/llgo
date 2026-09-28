//go:build !llgo
// +build !llgo

package cabi_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"

	"github.com/xgo-dev/llgo/internal/build"
	"github.com/xgo-dev/llgo/internal/cabi"
)

var (
	archs = []string{"amd64", "arm64", "riscv64", "armv6", "i386"}
)

func init() {
	// crosscompile
	if runtime.GOOS == "darwin" {
		archs = append(archs, "wasm32")
		archs = append(archs, "esp32")
		archs = append(archs, "esp32c3")
		archs = append(archs, "riscv64_lp64f")
		archs = append(archs, "riscv64_lp64d")
		archs = append(archs, "riscv32_ilp32")
		archs = append(archs, "riscv32_ilp32f")
		archs = append(archs, "riscv32_ilp32d")
	}
}

func buildConf(arch string) (conf *build.Config, targetAbi string) {
	conf = build.NewDefaultConf(build.ModeGen)
	conf.Goarch = arch
	conf.Goos = "linux"
	switch arch {
	case "wasm32":
		conf.Goarch = "wasm"
		conf.Goos = "wasip1"
	case "armv6":
		conf.Goarch = "arm"
	case "i386":
		conf.Goarch = "386"
	case "esp32":
		conf.Target = "esp32"
	case "esp32c3":
		conf.Target = "esp32c3"
	case "riscv64_lp64f":
		conf.Goarch = "riscv64"
		targetAbi = "lp64f"
	case "riscv64_lp64d":
		conf.Goarch = "riscv64"
		targetAbi = "lp64d"
	case "riscv32_ilp32":
		conf.Target = "riscv32"
		targetAbi = "ilp32"
	case "riscv32_ilp32f":
		conf.Target = "riscv32"
		targetAbi = "ilp32f"
	case "riscv32_ilp32d":
		conf.Target = "riscv32"
		targetAbi = "ilp32d"
	}
	return
}

func TestBoolAggregateMatchesCLayout(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeGen)
	var pack, notRet, notParam llvm.Type
	conf.ModuleHook = func(p build.Package) {
		if !strings.Contains(p.PkgPath, "boolpack") {
			return
		}
		mod := p.LPkg.Module()
		echo := mod.NamedFunction(p.PkgPath + ".Echo")
		if echo.IsNil() {
			t.Fatalf("Echo not found:\n%s", mod.String())
		}
		pack = echo.GlobalValueType().ParamTypes()[0]
		not := mod.NamedFunction(p.PkgPath + ".Not")
		if not.IsNil() {
			t.Fatalf("Not not found:\n%s", mod.String())
		}
		notRet = not.GlobalValueType().ReturnType()
		notParam = not.GlobalValueType().ParamTypes()[0]
	}
	pkgs, err := build.Do([]string{"./_testdata/boolpack"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0]
	defer pkg.LPkg.Prog.Dispose()
	if pack.C == nil {
		t.Fatal("did not observe Pack type before CABI lowering")
	}
	if pack.TypeKind() != llvm.StructTypeKind {
		t.Fatalf("Pack param kind = %v, want struct {i8,i8,i8}", pack.TypeKind())
	}
	fields := pack.StructElementTypes()
	if len(fields) != 3 {
		t.Fatalf("Pack fields = %d, want 3", len(fields))
	}
	for i, f := range fields {
		if f.TypeKind() != llvm.IntegerTypeKind || f.IntTypeWidth() != 8 {
			t.Fatalf("Pack field %d = %s, want i8 (C _Bool/char)", i, f.String())
		}
	}
	td := pkg.LPkg.Prog.TargetData()
	if got, want := td.TypeAllocSize(pack), uint64(3); got != want {
		t.Fatalf("Pack LLVM size = %d, want 3 to match C struct {_Bool; char; _Bool}", got)
	}
	if notRet.IntTypeWidth() != 1 || notParam.IntTypeWidth() != 1 {
		t.Fatalf("Not should be i1(i1) like Clang _Bool, got ret=%s param=%s", notRet.String(), notParam.String())
	}
	tr := cabi.NewTransformer(pkg.LPkg.Prog, "", "", false)
	info := tr.GetTypeInfo(pack.Context(), llvm.FunctionType(pack.Context().VoidType(), nil, false), pack, 1)
	if info.Size != 3 {
		t.Fatalf("cabi Sizeof(Pack) = %d, want 3 (C layout); llvm=%s", info.Size, pack.String())
	}
}

func TestBuild(t *testing.T) {
	for _, arch := range archs {
		conf, _ := buildConf(arch)
		pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
		if err != nil {
			t.Fatalf("build error: %v %v", arch, err)
		}
		pkgs[0].LPkg.Prog.Dispose()
	}
}

func TestABI(t *testing.T) {
	dirs, err := os.ReadDir("./_testdata/demo")
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, f := range dirs {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".go") {
			if f.Name() == "demo.go" {
				continue
			}
			files = append(files, f.Name())
		}
	}
	for _, arch := range archs {
		t.Run(arch, func(t *testing.T) {
			testArch(t, arch, arch, files)
		})
	}
}

func testArch(t *testing.T, arch string, archDir string, files []string) {
	conf, targetAbi := buildConf(arch)
	preABI := make(map[string]string)
	if targetAbi != "" {
		conf.ModuleHook = func(pkg build.Package) {
			preABI[pkg.PkgPath] = pkg.LPkg.String()
		}
	}
	for _, file := range files {
		pkgs, err := build.Do([]string{filepath.Join("./_testdata/demo", file)}, conf)
		if err != nil {
			t.Fatalf("build error: %v %v", arch, err)
		}
		ctx := llvm.NewContext()
		llfile := filepath.Join("./_testdata/arch", archDir, file[:len(file)-3]+".ll")
		buf, err := llvm.NewMemoryBufferFromFile(llfile)
		if err != nil {
			t.Fatalf("bad file: %v %v", llfile, err)
		}
		m, err := ctx.ParseIR(buf)
		if err != nil {
			t.Fatalf("parser IR error %v", arch)
		}
		pkg := pkgs[0].LPkg
		actual := pkg.Module()
		customABI := false
		if targetAbi != "" {
			pre, ok := preABI[pkgs[0].PkgPath]
			if !ok {
				t.Fatalf("pre-ABI module for %q was not captured", pkgs[0].PkgPath)
			}
			preFile := filepath.Join(t.TempDir(), "pre.ll")
			if err := os.WriteFile(preFile, []byte(pre), 0o644); err != nil {
				t.Fatal(err)
			}
			preBuf, err := llvm.NewMemoryBufferFromFile(preFile)
			if err != nil {
				t.Fatal(err)
			}
			actual, err = ctx.ParseIR(preBuf)
			if err != nil {
				t.Fatalf("parse pre-ABI IR error: %v", err)
			}
			customABI = true
			tr := cabi.NewTransformer(pkg.Prog, conf.Target, targetAbi, false)
			tr.TransformModule(file, actual)
		}
		testModule(t, context{arch: arch, file: file}, pkg.Prog.TargetData(), actual, m)
		if customABI {
			actual.Dispose()
		}
		m.Dispose()
		ctx.Dispose()
		pkg.Prog.Dispose()
	}
}

type context struct {
	arch string
	file string
}

func testModule(t *testing.T, ctx context, td llvm.TargetData, m llvm.Module, c llvm.Module) {
	var fns []llvm.Value
	fn := c.FirstFunction()
	for !fn.IsNil() {
		if !fn.IsDeclaration() {
			fns = append(fns, fn)
		}
		fn = llvm.NextFunction(fn)
	}
	for _, fn := range fns {
		// check c linkname
		testFunc(t, ctx, td, m.NamedFunction(fn.Name()), fn)
		// check Go symbol
		testFunc(t, ctx, td, m.NamedFunction("main."+fn.Name()), fn)
	}
}

func testFunc(t *testing.T, ctx context, td llvm.TargetData, fn llvm.Value, cfn llvm.Value) {
	if fn.IsNil() {
		t.Fatalf("%v: function %q not found", ctx, cfn.Name())
	}
	ft := fn.GlobalValueType()
	cft := cfn.GlobalValueType()
	pts := ft.ParamTypes()
	cpts := cft.ParamTypes()
	if len(pts) != len(cpts) {
		t.Logf("%v %v: bad param type %v != %v", ctx, fn.Name(), ft, cft)
		return
	}
	for i, pt := range pts {
		if !checkType(td, pt, cpts[i], false) {
			t.Fatalf("%v %v: bad param type %v != %v", ctx, fn.Name(), ft, cft)
		}
		if i == 0 {
			if fn.GetStringAttributeAtIndex(1, "sret") != cfn.GetStringAttributeAtIndex(1, "sret") {
				t.Fatalf("%v %v: bad param attr type %v != %v", ctx, fn.Name(), ft, cft)
			}
		}
		if fn.GetStringAttributeAtIndex(i+1, "byval") != cfn.GetStringAttributeAtIndex(i+1, "byval") {
			t.Fatalf("%v %v: bad param attr type %v != %v", ctx, fn.Name(), ft, cft)
		}
	}
	if !checkType(td, ft.ReturnType(), cft.ReturnType(), true) {
		t.Fatalf("%v %v: bad return type %v != %v", ctx, fn.Name(), ft, cft)
	}
}

func checkType(td llvm.TargetData, ft llvm.Type, cft llvm.Type, bret bool) bool {
	if ft == cft {
		return true
	}
	if bret {
		if ft.TypeKind() == llvm.VoidTypeKind && (cft.TypeKind() == llvm.VoidTypeKind || td.TypeAllocSize(cft) == 0) {
			return true
		} else if cft.TypeKind() == llvm.VoidTypeKind && (ft.TypeKind() == llvm.VoidTypeKind || td.TypeAllocSize(ft) == 0) {
			return true
		}
	} else if ft.TypeKind() == llvm.VoidTypeKind && cft.TypeKind() == llvm.VoidTypeKind {
		return true
	}
	if ft.TypeKind() == llvm.VoidTypeKind || cft.TypeKind() == llvm.VoidTypeKind {
		return false
	}
	if td.ABITypeAlignment(ft) != td.ABITypeAlignment(cft) {
		return false
	}
	if td.TypeAllocSize(ft) != td.TypeAllocSize(cft) {
		return false
	}
	et := elementTypes(td, ft)
	cet := elementTypes(td, cft)
	if len(et) != len(cet) {
		return false
	}
	if len(et) == 1 {
		return true
	}
	for i, t := range et {
		if !checkType(td, t, cet[i], bret) {
			return false
		}
	}
	return true
}

func elementTypes(td llvm.TargetData, typ llvm.Type) (types []llvm.Type) {
	switch typ.TypeKind() {
	case llvm.VoidTypeKind:
	case llvm.StructTypeKind:
		for _, t := range typ.StructElementTypes() {
			types = append(types, elementTypes(td, t)...)
		}
	case llvm.ArrayTypeKind:
		sub := elementTypes(td, typ.ElementType())
		n := typ.ArrayLength()
		for i := 0; i < n; i++ {
			types = append(types, sub...)
		}
	default:
		types = append(types, typ)
	}
	return
}

func byvalAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	id := llvm.AttributeKindID("byval")
	return ctx.CreateTypeAttribute(id, typ)
}

func sretAttribute(ctx llvm.Context, typ llvm.Type) llvm.Attribute {
	id := llvm.AttributeKindID("sret")
	return ctx.CreateTypeAttribute(id, typ)
}

// TestIssue1608_AttrPointerReturnNoMemcpy is a regression test for
// https://github.com/xgo-dev/llgo/issues/1608
//
// When transforming functions with sret (structure return), the optimizer
// previously used memcpy from the load source address. However, if the source
// address content is modified between load and ret, the memcpy would copy the
// wrong value. The fix is to always use store instead of memcpy.
func TestIssue1608_AttrPointerReturnNoMemcpy(t *testing.T) {
	// Create test IR that mimics the bug scenario:
	// 1. Load a slice from struct field
	// 2. Modify the struct field
	// 3. Return the originally loaded slice
	testIR := `; ModuleID = 'test'
source_filename = "test"

%Slice = type { ptr, i64, i64 }
%T = type { %Slice }

define %Slice @testfunc() {
entry:
  %0 = alloca %T, align 8
  %1 = getelementptr inbounds %T, ptr %0, i32 0, i32 0

  ; Store first slice {ptr, 2, 2}
  %2 = insertvalue %Slice undef, ptr null, 0
  %3 = insertvalue %Slice %2, i64 2, 1
  %4 = insertvalue %Slice %3, i64 2, 2
  store %Slice %4, ptr %1, align 8

  ; Load the slice (should return this value)
  %5 = getelementptr inbounds %T, ptr %0, i32 0, i32 0
  %6 = load %Slice, ptr %5, align 8

  ; Modify the source - THIS IS THE KEY!
  %7 = insertvalue %Slice undef, ptr null, 0
  %8 = insertvalue %Slice %7, i64 3, 1
  %9 = insertvalue %Slice %8, i64 3, 2
  %10 = getelementptr inbounds %T, ptr %0, i32 0, i32 0
  store %Slice %9, ptr %10, align 8

  ; Return the originally loaded value
  ret %Slice %6
}
`

	// Expected IR after CABI transformation:
	// - Function signature changed to use sret
	// - Return value stored via sret pointer (NOT memcpy!)
	// - Returns void
	expectedFuncIR := `define void @testfunc(ptr sret(%Slice) %0) {
entry:
  %1 = alloca %T, align 8
  %2 = getelementptr inbounds %T, ptr %1, i32 0, i32 0
  %3 = insertvalue %Slice undef, ptr null, 0
  %4 = insertvalue %Slice %3, i64 2, 1
  %5 = insertvalue %Slice %4, i64 2, 2
  store %Slice %5, ptr %2, align 8
  %6 = getelementptr inbounds %T, ptr %1, i32 0, i32 0
  %7 = load %Slice, ptr %6, align 8
  %8 = insertvalue %Slice undef, ptr null, 0
  %9 = insertvalue %Slice %8, i64 3, 1
  %10 = insertvalue %Slice %9, i64 3, 2
  %11 = getelementptr inbounds %T, ptr %1, i32 0, i32 0
  store %Slice %10, ptr %11, align 8
  store %Slice %7, ptr %0, align 8
  ret void
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()

	// Write test IR to temporary file
	tmpfile := filepath.Join(t.TempDir(), "test.ll")
	if err := os.WriteFile(tmpfile, []byte(testIR), 0644); err != nil {
		t.Fatalf("Failed to write test IR: %v", err)
	}

	// Parse the test IR
	buf, err := llvm.NewMemoryBufferFromFile(tmpfile)
	if err != nil {
		t.Fatalf("Failed to read test IR: %v", err)
	}

	mod, err := ctx.ParseIR(buf)
	if err != nil {
		t.Fatalf("Failed to parse test IR: %v", err)
	}
	defer mod.Dispose()

	// Get the function before transformation
	fn := mod.NamedFunction("testfunc")
	if fn.IsNil() {
		t.Fatal("Test function not found")
	}

	// Build minimal config for CABI transformation
	conf, _ := buildConf("arm64")
	pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
	if err != nil {
		t.Fatalf("Failed to build demo: %v", err)
	}

	prog := pkgs[0].LPkg.Prog
	defer prog.Dispose()

	// Apply CABI transformation with optimize=true
	// This tests the code path that previously had the memcpy bug
	tr := cabi.NewTransformer(prog, "", "", true)
	tr.TransformModule("test", mod)

	// Get transformed function IR
	transformedFn := mod.NamedFunction("testfunc")
	if transformedFn.IsNil() {
		t.Fatal("Transformed function not found")
	}
	actualFuncIR := strings.TrimSpace(transformedFn.String())
	expectedFuncIRTrimmed := strings.TrimSpace(expectedFuncIR)

	// Compare IR
	if actualFuncIR != expectedFuncIRTrimmed {
		t.Errorf("Transformed IR mismatch!\n\nExpected:\n%s\n\nActual:\n%s", expectedFuncIRTrimmed, actualFuncIR)
	}
}

func TestAttrPointerCallParamNoLoadSourceReuse(t *testing.T) {
	for _, arch := range []string{"arm64", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			testAttrPointerCallParamNoLoadSourceReuse(t, arch)
		})
	}
}

func testAttrPointerCallParamNoLoadSourceReuse(t *testing.T, arch string) {
	testIR := `; ModuleID = 'test'
source_filename = "test"

%Slice = type { ptr, i64, i64 }

declare void @sink(%Slice)
declare void @llvm.memset(ptr, i8, i64, i1)

define void @caller() {
entry:
  %src = alloca %Slice, align 8
  %first0 = insertvalue %Slice undef, ptr null, 0
  %first1 = insertvalue %Slice %first0, i64 3, 1
  %first2 = insertvalue %Slice %first1, i64 3, 2
  store %Slice %first2, ptr %src, align 8
  %val = load %Slice, ptr %src, align 8
  call void @llvm.memset(ptr %src, i8 0, i64 24, i1 false)
  call void @sink(%Slice %val)
  ret void
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tmpfile := filepath.Join(t.TempDir(), "test.ll")
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

	conf, _ := buildConf(arch)
	pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
	if err != nil {
		t.Fatalf("Failed to build demo: %v", err)
	}
	defer pkgs[0].LPkg.Prog.Dispose()
	tr := cabi.NewTransformer(pkgs[0].LPkg.Prog, "", "", true)
	tr.TransformModule("test", mod)

	caller := mod.NamedFunction("caller")
	if caller.IsNil() {
		t.Fatal("caller not found")
	}
	ir := caller.String()
	if strings.Contains(ir, "call void @sink(ptr byval(%Slice) %src)") {
		t.Fatalf("call reused load source that is later cleared:\n%s", ir)
	}
	if strings.Contains(ir, "call void @sink(ptr %src)") {
		t.Fatalf("call reused load source that is later cleared:\n%s", ir)
	}
	if !strings.Contains(ir, "store %Slice %val, ptr") {
		t.Fatalf("call argument should be materialized from loaded value:\n%s", ir)
	}
	if !strings.Contains(ir, "call void @sink(ptr") {
		t.Fatalf("sink call was not rewritten to pointer argument:\n%s", ir)
	}
}

// TestSkipFuncs verifies function-level opt-out from C ABI transformation.
func TestSkipFuncs(t *testing.T) {
	testIR := `; ModuleID = 'test'
source_filename = "test"

%Big = type { i64, i64, i64 }

declare i64 @"pkg.asm"(i64, %Big)

define i64 @"pkg.caller"(i64 %0) {
entry:
  %1 = call i64 @"pkg.asm"(i64 %0, %Big zeroinitializer)
  ret i64 %1
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tmpfile := filepath.Join(t.TempDir(), "skipfuncs.ll")
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

	conf, _ := buildConf("arm64")
	pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
	if err != nil {
		t.Fatalf("Failed to build demo: %v", err)
	}
	prog := pkgs[0].LPkg.Prog
	defer prog.Dispose()

	tr := cabi.NewTransformer(prog, "", "", true)
	tr.SetSkipFuncs([]string{"pkg.asm"})
	tr.TransformModule("test", mod)

	asm := mod.NamedFunction("pkg.asm")
	if asm.IsNil() {
		t.Fatal("pkg.asm not found")
	}
	asmHead := strings.SplitN(asm.String(), "\n", 2)[0]
	if strings.Contains(asmHead, "byval(") || strings.Contains(asmHead, "sret(") {
		t.Fatalf("pkg.asm should not be wrapped:\n%s", asm.String())
	}
	if !strings.Contains(asmHead, "%Big") {
		t.Fatalf("pkg.asm signature unexpectedly changed:\n%s", asm.String())
	}

	caller := mod.NamedFunction("pkg.caller")
	if caller.IsNil() {
		t.Fatal("pkg.caller not found")
	}
	ir := caller.String()
	if strings.Contains(ir, "alloca %Big") {
		t.Fatalf("call to skipped function was unexpectedly rewritten:\n%s", ir)
	}
	if !strings.Contains(ir, "call i64 @pkg.asm(i64 %0, %Big") {
		t.Fatalf("call to skipped function signature changed unexpectedly:\n%s", ir)
	}
}

func TestSkipTypedslicecopy(t *testing.T) {
	testIR := `; ModuleID = 'test'
source_filename = "test"

%reflect.unsafeheaderSlice = type { ptr, i64, i64 }
%Big = type { i64, i64, i64 }

declare i64 @"github.com/xgo-dev/llgo/runtime/internal/runtime.Typedslicecopy"(ptr, %reflect.unsafeheaderSlice, %reflect.unsafeheaderSlice)

define i64 @"pkg.copy"(%reflect.unsafeheaderSlice %0, %reflect.unsafeheaderSlice %1, %Big %2) {
entry:
  %3 = call i64 @"github.com/xgo-dev/llgo/runtime/internal/runtime.Typedslicecopy"(ptr null, %reflect.unsafeheaderSlice %0, %reflect.unsafeheaderSlice %1)
  ret i64 %3
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tmpfile := filepath.Join(t.TempDir(), "skip_typedslicecopy.ll")
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

	conf, _ := buildConf("arm64")
	pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
	if err != nil {
		t.Fatalf("Failed to build demo: %v", err)
	}
	prog := pkgs[0].LPkg.Prog
	defer prog.Dispose()

	tr := cabi.NewTransformer(prog, "", "", true)
	tr.SetSkipFuncs([]string{"github.com/xgo-dev/llgo/runtime/internal/runtime.Typedslicecopy"})
	tr.TransformModule("test", mod)

	callee := mod.NamedFunction("github.com/xgo-dev/llgo/runtime/internal/runtime.Typedslicecopy")
	if callee.IsNil() {
		t.Fatal("Typedslicecopy declaration not found")
	}
	head := strings.SplitN(callee.String(), "\n", 2)[0]
	if !strings.Contains(head, "%reflect.unsafeheaderSlice") {
		t.Fatalf("Typedslicecopy declaration unexpectedly rewritten:\n%s", callee.String())
	}
	if strings.Contains(head, "byval(") || strings.Contains(head, "ptr, ptr, ptr") {
		t.Fatalf("Typedslicecopy declaration should keep original struct args:\n%s", callee.String())
	}

	copyFn := mod.NamedFunction("pkg.copy")
	if copyFn.IsNil() {
		t.Fatal("pkg.copy not found")
	}
	ir := copyFn.String()
	if !strings.Contains(ir, `call i64 @"github.com/xgo-dev/llgo/runtime/internal/runtime.Typedslicecopy"(ptr`) {
		t.Fatalf("call to Typedslicecopy missing:\n%s", ir)
	}
	if !strings.Contains(ir, "%reflect.unsafeheaderSlice") {
		t.Fatalf("call to Typedslicecopy unexpectedly rewritten to pointer args:\n%s", ir)
	}
}

func TestKeepInlineAsmCallSignature(t *testing.T) {
	testIR := `; ModuleID = 'test'
source_filename = "test"

define { i32, i32, i32, i32 } @"internal/cpu.cpuid"(i32 %0, i32 %1) {
entry:
  %2 = call { i32, i32, i32, i32 } asm sideeffect "cpuid", "={ax},={bx},={cx},={dx},{ax},{cx},~{dirflag},~{fpsr},~{flags}"(i32 %0, i32 %1)
  ret { i32, i32, i32, i32 } %2
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tmpfile := filepath.Join(t.TempDir(), "inline_asm.ll")
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

	conf, _ := buildConf("amd64")
	pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
	if err != nil {
		t.Fatalf("Failed to build demo: %v", err)
	}
	prog := pkgs[0].LPkg.Prog
	defer prog.Dispose()

	tr := cabi.NewTransformer(prog, "", "", true)
	tr.TransformModule("test", mod)

	fn := mod.NamedFunction("internal/cpu.cpuid")
	if fn.IsNil() {
		t.Fatal("internal/cpu.cpuid not found")
	}
	head := strings.SplitN(fn.String(), "\n", 2)[0]
	if !strings.Contains(head, "{ i64, i64 }") {
		t.Fatalf("function signature should still be cabi-rewritten:\n%s", fn.String())
	}
	ir := fn.String()
	if strings.Contains(ir, `call { i64, i64 } asm sideeffect "cpuid"`) {
		t.Fatalf("inline asm call should keep original constraint-matching signature:\n%s", ir)
	}
	if !strings.Contains(ir, `call { i32, i32, i32, i32 } asm sideeffect "cpuid"`) {
		t.Fatalf("inline asm call signature unexpectedly changed:\n%s", ir)
	}
}

// TestRuntimeSliceWrap verifies that runtime Slice follows C ABI rewriting,
// while unrelated large aggregates keep normal wrapping.
func TestRuntimeSliceWrap(t *testing.T) {
	testIR := `; ModuleID = 'test'
source_filename = "test"

%"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" = type { ptr, i64, i64 }
%Big = type { i64, i64, i64 }

define %"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" @"pkg.keep"(%"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" %0) {
entry:
  ret %"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" %0
}

define i64 @"pkg.mixed"(%"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice" %0, %Big %1) {
entry:
  ret i64 0
}
`

	ctx := llvm.NewContext()
	defer ctx.Dispose()

	tmpfile := filepath.Join(t.TempDir(), "runtime_slice_wrap.ll")
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

	conf, _ := buildConf("amd64")
	pkgs, err := build.Do([]string{"./_testdata/demo/demo.go"}, conf)
	if err != nil {
		t.Fatalf("Failed to build demo: %v", err)
	}
	prog := pkgs[0].LPkg.Prog
	defer prog.Dispose()

	tr := cabi.NewTransformer(prog, "", "", true)
	tr.TransformModule("test", mod)

	keep := mod.NamedFunction("pkg.keep")
	if keep.IsNil() {
		t.Fatal("pkg.keep not found")
	}
	keepHead := strings.SplitN(keep.String(), "\n", 2)[0]
	if !strings.Contains(keepHead, "sret(%\"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice\")") {
		t.Fatalf("runtime slice return should use sret after rewrite:\n%s", keep.String())
	}
	if !strings.Contains(keepHead, "byval(%\"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice\")") {
		t.Fatalf("runtime slice param should use byval after rewrite:\n%s", keep.String())
	}

	mixed := mod.NamedFunction("pkg.mixed")
	if mixed.IsNil() {
		t.Fatal("pkg.mixed not found")
	}
	mixedHead := strings.SplitN(mixed.String(), "\n", 2)[0]
	if !strings.Contains(mixedHead, "byval(%\"github.com/xgo-dev/llgo/runtime/internal/runtime.Slice\")") {
		t.Fatalf("runtime slice param should be rewritten in mixed function:\n%s", mixed.String())
	}
	if !strings.Contains(mixedHead, "byval(%Big)") {
		t.Fatalf("non-runtime large aggregate should still be wrapped:\n%s", mixed.String())
	}
}
