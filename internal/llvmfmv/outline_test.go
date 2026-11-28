package llvmfmv

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestOutlineWideInvoke(t *testing.T) {
	const ir = `target triple = "x86_64-pc-windows-msvc"
declare i32 @personality(...)
define i32 @caller(ptr %target, ptr %data, i1 %take) personality ptr @personality {
entry:
 br i1 %take,label %yes,label %join
yes:
 %x=load <16 x i32>, ptr %data,align 4
 %v=invoke x86_vectorcallcc <16 x i32> %target(<16 x i32> %x, ptr byval(i32) align 4 %data, i16 signext 2) to label %join unwind label %unwind
join:
 %r=phi <16 x i32> [ zeroinitializer, %entry ], [ %v, %yes ]
 %first=extractelement <16 x i32> %r,i32 0
 ret i32 %first
unwind:
 %ex=landingpad {ptr,i32} cleanup
 resume {ptr,i32} %ex
}`
	m := parseModule(t, ir)
	abi := newWideABI(m)
	defer abi.data.Dispose()
	caller := m.NamedFunction("caller")
	abi.outlineBody(caller, "v1", nil, 0)
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	bridge := m.NamedFunction("caller" + symbolPrefix + "call")
	if bridge.IsNil() || targetWidth(bridge, "v1") != 512 || abi.signature(bridge.GlobalValueType()) != 0 {
		t.Fatal(m.String())
	}
	for _, s := range []string{"call x86_vectorcallcc <16 x i32> %", "ptr byval(i32) align 4", "i16 signext"} {
		if !strings.Contains(bridge.String(), s) {
			t.Fatalf("lost %q:\n%s", s, bridge.String())
		}
	}
	for _, s := range []string{"invoke void @caller.", "unwind label %unwind", "%fmv.return ]"} {
		if !strings.Contains(caller.String(), s) {
			t.Fatalf("lost %q:\n%s", s, caller.String())
		}
	}
}

func TestOutlineGuardedMask(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @q()
declare <64 x i8> @llvm.experimental.vector.compress.v64i8(<64 x i8>,<64 x i1>,<64 x i8>)
define void @caller(ptr %data,i64 %bits) {
entry:
 %g=call i1 @q()
 br i1 %g,label %yes,label %out
yes:
 %v=load <64 x i8>,ptr %data
 %k=bitcast i64 %bits to <64 x i1>
 %c=call <64 x i8> @llvm.experimental.vector.compress.v64i8(<64 x i8> %v,<64 x i1> %k,<64 x i8> zeroinitializer)
 store <64 x i8> %c,ptr %data
 br label %out
out: ret void
}`)
	abi := newWideABI(m)
	defer abi.data.Dispose()
	bit := predicateBits[X86Query("AVX512VBMI2")]
	caller := m.NamedFunction("caller")
	abi.outlineBody(caller, "v1", map[llvm.Value]uint64{m.NamedFunction("q"): bit}, 0)
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	leaf := m.NamedFunction(symbolPrefix + "op_experimental.vector.compress.v64i8")
	if leaf.IsNil() || !strings.Contains(stringAttribute(leaf, "target-features"), "+avx512vbmi2") || abi.signature(leaf.GlobalValueType()) != 0 {
		t.Fatal(m.String())
	}
	ts := leaf.GlobalValueType().ParamTypes()
	if ts[2] != m.Context().Int64Type() || !strings.Contains(leaf.String(), " to <64 x i1>") {
		t.Fatal(leaf.String())
	}
	for _, call := range []string{"@q(", "@" + leaf.Name() + "("} {
		if !strings.Contains(caller.String(), call) {
			t.Fatal(caller.String())
		}
	}
}

func TestOutlineWideFMAHasRegisterABI(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-pc-windows-msvc"
declare i1 @q()
declare <16 x float> @llvm.fma.v16f32(<16 x float>,<16 x float>,<16 x float>)
define void @caller(ptr %p) "target-features"="+avx512f" {
entry:
 %g=call i1 @q()
 br i1 %g,label %yes,label %out
yes:
 %x=load <16 x float>,ptr %p
 %v=call <16 x float> @llvm.fma.v16f32(<16 x float> %x,<16 x float> %x,<16 x float> %x)
 store <16 x float> %v,ptr %p
 br label %out
out: ret void
}`)
	abi := newWideABI(m)
	defer abi.data.Dispose()
	caller := m.NamedFunction("caller")
	abi.outlineBody(caller, "v1", map[llvm.Value]uint64{m.NamedFunction("q"): predicateBits[X86Query("FMA")]}, 0)
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	leaf := m.NamedFunction(symbolPrefix + "op_fma.v16f32")
	if leaf.IsNil() || targetWidth(leaf, "v1") != 512 || leaf.FunctionCallConv() != x86VectorCall || !strings.Contains(caller.String(), "call x86_vectorcallcc <16 x float>") {
		t.Fatal(m.String())
	}
}

func TestOutlineExpandLoadKeepsPointerAlignment(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @q()
declare <64 x i8> @llvm.masked.expandload.v64i8.p0(ptr,<64 x i1>,<64 x i8>)
define void @caller(ptr %p, ptr %out, i64 %bits) {
entry:
 %g=call i1 @q()
 br i1 %g,label %yes,label %outblk
yes:
 %mask=bitcast i64 %bits to <64 x i1>
 %r=call <64 x i8> @llvm.masked.expandload.v64i8.p0(ptr align 4 %p,<64 x i1> %mask,<64 x i8> zeroinitializer)
 store <64 x i8> %r,ptr %out
 br label %outblk
outblk: ret void
}`)
	a := newWideABI(m)
	defer a.data.Dispose()
	a.outlineBody(m.NamedFunction("caller"), "v1", map[llvm.Value]uint64{m.NamedFunction("q"): predicateBits[X86Query("AVX512VBMI2")]}, 0)
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	leaf := m.NamedFunction("caller" + symbolPrefix + "call")
	if leaf.IsNil() || !strings.Contains(leaf.String(), "@llvm.masked.expandload.v64i8.p0(ptr align 4") || !strings.Contains(leaf.String(), "<64 x i8> zeroinitializer") {
		t.Fatal(m.String())
	}
}

func TestOutlinePreservesIntrinsicImmediates(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare i1 @q()
declare <4 x float> @llvm.x86.sse41.round.ps(<4 x float>,i32 immarg)
define void @caller(ptr %p) {
entry:
 %g=call i1 @q()
 br i1 %g,label %yes,label %out
yes:
 %v=load <4 x float>,ptr %p
 %r=call <4 x float> @llvm.x86.sse41.round.ps(<4 x float> %v,i32 0), !llgo.cpu.prefer-feature !0
 store <4 x float> %r,ptr %p
 br label %out
out: ret void
}
!0=!{!"x86.avx512"}`)
	a := newWideABI(m)
	defer a.data.Dispose()
	// Call the per-instruction primitive directly: the production frontend only
	// puts its preference on known reductions; the immediate must stay local.
	caller := m.NamedFunction("caller")
	bb := llvm.NextBasicBlock(caller.FirstBasicBlock())
	call := llvm.NextInstruction(bb.FirstInstruction())
	a.outlineCall(call, 0, 0, predicateBits[X86Query("AVX512")])
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	leaf := m.NamedFunction("caller" + symbolPrefix + "call")
	if leaf.IsNil() || !strings.Contains(leaf.String(), ", i32 0)") {
		t.Fatal(m.String())
	}
}

func TestOutlineBundlePreflight(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare void @wide(<16 x i32>)
declare void @scalar()
define void @bad() {
 call void @wide(<16 x i32> zeroinitializer) [ "deopt"(i32 7) ]
 ret void
}
define void @emptyBundle() {
 call void @wide(<16 x i32> zeroinitializer) [ "deopt"() ]
 ret void
}
define void @ok() {
 call void @scalar() [ "deopt"(i32 7) ]
 call void @wide(<16 x i32> zeroinitializer)
 ret void
}`)
	a := newWideABI(m)
	defer a.data.Dispose()
	before := m.String()
	if err := a.validateOutlines(m.NamedFunction("bad"), "v1", nil, 0); err == nil || !strings.Contains(err.Error(), "operand bundle") {
		t.Fatal(err)
	}
	if err := a.validateOutlines(m.NamedFunction("emptyBundle"), "v1", nil, 0); err == nil || !strings.Contains(err.Error(), "operand bundle") {
		t.Fatal(err)
	}
	if err := a.validateOutlines(m.NamedFunction("ok"), "v1", nil, 0); err != nil {
		t.Fatal(err)
	}
	if err := a.validateOutlines(m.NamedFunction("bad"), "v4", nil, 0); err != nil {
		t.Fatal(err)
	}
	if m.String() != before {
		t.Fatal("preflight mutated bundled call")
	}
}

func TestOutlineAttributedIntrinsicIsNotShared(t *testing.T) {
	m := parseModule(t, `target triple = "x86_64-unknown-linux-gnu"
declare <4 x float> @llvm.fma.v4f32(<4 x float>,<4 x float>,<4 x float>)
define <4 x float> @caller(<4 x float> %x) {
 %a=call <4 x float> @llvm.fma.v4f32(<4 x float> noundef %x,<4 x float> %x,<4 x float> %x)
 %b=call <4 x float> @llvm.fma.v4f32(<4 x float> %a,<4 x float> %x,<4 x float> %x)
 ret <4 x float> %b
}`)
	a := newWideABI(m)
	defer a.data.Dispose()
	first := m.NamedFunction("caller").FirstBasicBlock().FirstInstruction()
	second := llvm.NextInstruction(first)
	req := predicateBits[X86Query("FMA")]
	a.outlineCall(first, 0, 0, req)
	a.outlineCall(second, 0, 0, req)
	if err := llvm.VerifyModule(m, llvm.ReturnStatusAction); err != nil {
		t.Fatal(err, m.String())
	}
	attributed := m.NamedFunction("caller" + symbolPrefix + "call")
	shared := m.NamedFunction(symbolPrefix + "op_fma.v4f32")
	if attributed.IsNil() || shared.IsNil() || !strings.Contains(attributed.String(), "noundef") || strings.Contains(shared.String(), "noundef") {
		t.Fatal(m.String())
	}
}
