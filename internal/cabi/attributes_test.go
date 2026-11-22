package cabi

import (
	"go/importer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmattr"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestABIAttributeTransport(t *testing.T) {
	llssa.Initialize(llssa.InitAllTargets | llssa.InitAllTargetInfos | llssa.InitAllTargetMCs | llssa.InitAllAsmPrinters)
	for _, target := range []*llssa.Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"},
		{GOOS: "darwin", GOARCH: "arm64", LLVMTarget: "aarch64-apple-darwin"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "windows", GOARCH: "386", LLVMTarget: "i686-pc-windows-msvc"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasi"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := llssa.NewProgram(target)
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			const ir = `
%Large = type { i64, i64, i64, i64 }
%Empty = type {}
declare nonnull ptr @allocate(%Large, %Empty, i32, ptr returned) allocsize(2) nounwind
declare ptr @allocate_count(%Large, %Empty, i32, i64) allocsize(2, 3) allockind("alloc,zeroed") "alloc-family"="test"
declare %Large @produce(%Large, ptr readonly captures(none)) memory(none) nounwind willreturn

define i64 @writable_home(%Large %v) memory(none) {
  %home = alloca %Large, align 1
  store %Large %v, ptr %home, align 1
  %old = load i64, ptr %home, align 1
  store i64 41, ptr %home, align 1
  ret i64 %old
}
declare [8 x ptr] @capture(ptr captures(ret: address, provenance))
define [8 x ptr] @forward_capture(ptr captures(ret: address, provenance) %p) {
  %r = call [8 x ptr] @capture(ptr captures(ret: address, provenance) %p)
  ret [8 x ptr] %r
}

define nonnull ptr @forward(%Large %v, %Empty %empty, i32 %n, ptr returned %p) allocsize(2) nounwind {
  %r = call nonnull ptr @allocate(%Large %v, %Empty %empty, i32 %n, ptr returned %p) allocsize(2) nounwind
  ret ptr %r
}
define ptr @forward_count(%Large %v, %Empty %empty, i32 %size, i64 %count) allocsize(2, 3) {
  %r = call ptr @allocate_count(%Large %v, %Empty %empty, i32 %size, i64 %count) allocsize(2, 3) allockind("alloc,zeroed") "alloc-family"="test"
  ret ptr %r
}
define %Large @roundtrip(%Large %v, ptr readonly captures(none) %p) memory(none) nounwind willreturn {
  %r = call %Large @produce(%Large %v, ptr readonly captures(none) %p) memory(none) nounwind willreturn
  ret %Large %r
}
`
			path := filepath.Join(t.TempDir(), "attrs.ll")
			if err := os.WriteFile(path, []byte(ir), 0600); err != nil {
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
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			tr := NewTransformer(prog, target.LLVMTarget, "", true)
			tr.TransformModule("test", mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v\n%s", err, mod.String())
			}
			home := mod.NamedFunction("writable_home")
			if !strings.Contains(home.String(), "store i64 41, ptr %0") {
				t.Fatalf("fixture did not write through the reused parameter home: %s", home.String())
			}
			effects := requireABIAttr(t, home.GetEnumAttributeAtIndex, -1, "memory")
			if effects.GetEnumValue() != llvmattr.MemoryArgRead|llvmattr.MemoryArgWrite {
				t.Fatalf("writable parameter home has wrong effects: %s", home.String())
			}
			for _, name := range []string{"capture", "forward_capture"} {
				fn := mod.NamedFunction(name)
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, 1, "sret")
				if !fn.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("captures")).IsNil() {
					t.Fatalf("sret parameter retained return-only capture: %s", fn.String())
				}
				if name == "forward_capture" {
					call := findAttributeCall(t, fn, "capture")
					if !call.GetCallSiteEnumAttribute(2, llvm.AttributeKindID("captures")).IsNil() {
						t.Fatalf("sret call retained return-only capture: %s", call.String())
					}
				}
			}
			for _, name := range []string{"allocate", "forward"} {
				fn := mod.NamedFunction(name)
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, 0, "nonnull")
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, -1, "nounwind")
				types := fn.GlobalValueType().ParamTypes()
				var sizeIndex int
				for i, typ := range types {
					if typ == ctx.Int32Type() {
						sizeIndex = i
					}
				}
				attr := requireABIAttr(t, fn.GetEnumAttributeAtIndex, -1, "allocsize")
				if attr.GetEnumValue() != llvmattr.AllocSize(uint32(sizeIndex), llvmattr.AllocSizeNoCount) {
					t.Fatalf("bad allocsize: %s", fn.String())
				}
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, len(types), "returned")
				if name == "forward" {
					call := findAttributeCall(t, fn, "allocate")
					requireABIAttr(t, call.GetCallSiteEnumAttribute, 0, "nonnull")
					requireABIAttr(t, call.GetCallSiteEnumAttribute, -1, "nounwind")
					if requireABIAttr(t, call.GetCallSiteEnumAttribute, -1, "allocsize").GetEnumValue() != attr.GetEnumValue() {
						t.Fatal("call allocsize differs")
					}
					requireABIAttr(t, call.GetCallSiteEnumAttribute, len(types), "returned")
				}
			}
			// Both calloc-style indices move when ABI lowering removes the empty
			// argument. Check declarations, definitions and rewritten calls.
			for _, name := range []string{"allocate_count", "forward_count"} {
				fn := mod.NamedFunction(name)
				params := fn.GlobalValueType().ParamTypes()
				want := llvmattr.AllocSize(uint32(len(params)-2), uint32(len(params)-1))
				if requireABIAttr(t, fn.GetEnumAttributeAtIndex, -1, "allocsize").GetEnumValue() != want {
					t.Fatalf("bad allocation size/count indices: %s", fn.String())
				}
				if name == "forward_count" {
					call := findAttributeCall(t, fn, "allocate_count")
					requireABIAttr(t, call.GetCallSiteEnumAttribute, -1, "allockind")
					if call.GetCallSiteStringAttribute(-1, "alloc-family").IsNil() {
						t.Fatal("call lost its allocation family")
					}
					if requireABIAttr(t, call.GetCallSiteEnumAttribute, -1, "allocsize").GetEnumValue() != want {
						t.Fatalf("bad call allocation size/count indices: %s", call.String())
					}
				}
			}
			for _, name := range []string{"produce", "roundtrip"} {
				fn := mod.NamedFunction(name)
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, 1, "sret")
				last := len(fn.GlobalValueType().ParamTypes())
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, last, "readonly")
				requireABIAttr(t, fn.GetEnumAttributeAtIndex, last, "captures")
				attr := requireABIAttr(t, fn.GetEnumAttributeAtIndex, -1, "memory")
				if attr.GetEnumValue() != llvmattr.MemoryArgRead|llvmattr.MemoryArgWrite {
					t.Fatalf("memory must include ABI reads/writes: %s", fn.String())
				}
				if name == "roundtrip" {
					call := findAttributeCall(t, fn, "produce")
					requireABIAttr(t, call.GetCallSiteEnumAttribute, last, "readonly")
					requireABIAttr(t, call.GetCallSiteEnumAttribute, last, "captures")
					if requireABIAttr(t, call.GetCallSiteEnumAttribute, -1, "memory").GetEnumValue() != llvmattr.MemoryArgRead|llvmattr.MemoryArgWrite {
						t.Fatal("call memory must include ABI reads/writes")
					}
				}
			}
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := mod.RunPasses("default<O2>", prog.TargetMachine(), opts); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			// A stale memory(none) contract would allow DCE to delete the producer,
			// leaving the returned sret storage uninitialized.
			if !strings.Contains(mod.NamedFunction("roundtrip").String(), "@produce") {
				t.Fatalf("producer incorrectly eliminated:\n%s", mod.String())
			}
		})
	}
}

func requireABIAttr(t *testing.T, get func(int, uint) llvm.Attribute, index int, name string) llvm.Attribute {
	t.Helper()
	attr := get(index, llvm.AttributeKindID(name))
	if attr.IsNil() {
		t.Fatalf("missing %s at index %d", name, index)
	}
	return attr
}

func findAttributeCall(t *testing.T, fn llvm.Value, name string) llvm.Value {
	t.Helper()
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if call := inst.IsACallInst(); !call.IsNil() && call.CalledValue().Name() == name {
				return call
			}
		}
	}
	t.Fatalf("missing call to %s", name)
	return llvm.Value{}
}

// Exercise real runtime declarations rather than a synthetic ABI signature.
// CStrDup keeps a pointer return while its String input is expanded or indirect;
// StringFrom gains sret and must move the source pointer's readonly attribute.
func TestRuntimeABIAttributes(t *testing.T) {
	llssa.Initialize(llssa.InitAllTargets | llssa.InitAllTargetInfos | llssa.InitAllTargetMCs | llssa.InitAllAsmPrinters)
	rt, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(llssa.PkgRuntime)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []*llssa.Target{
		{GOOS: "linux", GOARCH: "amd64", LLVMTarget: "x86_64-unknown-linux-gnu"},
		{GOOS: "darwin", GOARCH: "arm64", LLVMTarget: "aarch64-apple-darwin"},
		{GOOS: "windows", GOARCH: "amd64", LLVMTarget: "x86_64-pc-windows-msvc"},
		{GOOS: "wasip1", GOARCH: "wasm", LLVMTarget: "wasm32-unknown-wasi"},
	} {
		t.Run(target.LLVMTarget, func(t *testing.T) {
			prog := llssa.NewProgram(target)
			defer prog.Dispose()
			prog.SetRuntime(rt)
			pkg := prog.NewPackage("test", "example.com/test")
			mod := pkg.Module()
			names := []string{"CStrDup", "CStrCopy", "StringFrom", "StringEqual", "Complex128Div", "PanicErrorString"}
			for _, name := range names {
				pkg.RuntimeFunc(name)
			}
			tr := NewTransformer(prog, target.LLVMTarget, "", false)
			tr.TransformModule(pkg.Path(), mod)
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v\n%s", err, mod.String())
			}
			function := func(name string) llvm.Value { return mod.NamedFunction(llssa.PkgRuntime + "." + name) }
			for _, attr := range []string{"nonnull", "noalias", "noundef"} {
				requireABIAttr(t, function("CStrDup").GetEnumAttributeAtIndex, 0, attr)
			}
			requireABIAttr(t, function("CStrCopy").GetEnumAttributeAtIndex, 1, "returned")
			from := function("StringFrom")
			source := 1
			if !from.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("sret")).IsNil() {
				source++
				if !from.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("readonly")).IsNil() {
					t.Fatal("source readonly leaked to sret storage")
				}
			}
			requireABIAttr(t, from.GetEnumAttributeAtIndex, source, "readonly")
			requireABIAttr(t, function("PanicErrorString").GetEnumAttributeAtIndex, -1, "noreturn")
			for _, name := range []string{"StringEqual", "Complex128Div"} {
				fn := function(name)
				attr := requireABIAttr(t, fn.GetEnumAttributeAtIndex, -1, "memory")
				want := llvmattr.MemoryRead
				// These targets pass the two-word aggregates indirectly.
				if target.GOOS == "windows" || target.GOARCH == "wasm" {
					want |= llvmattr.MemoryArgWrite
				}
				if attr.GetEnumValue() != want {
					t.Fatalf("bad memory effects: %s", fn.String())
				}
			}
		})
	}
}
