package funcattrs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func valueTestModule(t *testing.T, ir string) llvm.Module {
	t.Helper()
	ctx := llvm.NewContext()
	t.Cleanup(ctx.Dispose)
	path := filepath.Join(t.TempDir(), "contracts.ll")
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
	t.Cleanup(mod.Dispose)
	return mod
}

func attachValueTestContracts(t *testing.T, mod llvm.Module, name, source string) {
	t.Helper()
	attrs, sig, err := parseTest(t, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(mod.Context(), mod.NamedFunction(name), sig, attrs, 0, 64); err != nil {
		t.Fatal(err)
	}
}

func optimizeValueTest(t *testing.T, mod llvm.Module) {
	t.Helper()
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid contract IR: %v\n%s", err, mod.String())
	}
	options := llvm.NewPassBuilderOptions()
	defer options.Dispose()
	if err := mod.RunPasses("default<O2>", llvm.TargetMachine{}, options); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invalid optimized contract IR: %v\n%s", err, mod.String())
	}
}

func TestValueContractsMaterializeMultipleResults(t *testing.T) {
	mod := valueTestModule(t, `
declare { ptr, i8 } @F(ptr)

define i1 @caller(ptr %p) {
entry:
  %r = call { ptr, i8 } @F(ptr %p)
  %result.p = extractvalue { ptr, i8 } %r, 0
  %result.n = extractvalue { ptr, i8 } %r, 1
  %nil = icmp eq ptr %result.p, null
  %small = icmp slt i8 %result.n, -3
  %large = icmp sge i8 %result.n, 4
  %range.bad = or i1 %small, %large
  %bad = or i1 %nil, %range.bad
  ret i1 %bad
}

define i1 @relation(ptr %p) {
entry:
  %r = call { ptr, i8 } @F(ptr %p)
  %result.p = extractvalue { ptr, i8 } %r, 0
  %same = icmp eq ptr %result.p, %p
  ret i1 %same
}
`)
	attachValueTestContracts(t, mod, "F", `
//llgo:attribute result(out) nonnull same_as(param(p))
//llgo:attribute result(n) range(-3,4)
func F(p *int) (out *int, n int8)
`)
	if err := MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	before := mod.String()
	if strings.Contains(before, valuePlanMetadata) {
		t.Fatal("ABI-local value plan escaped materialization")
	}
	if err := MaterializeValueContracts(mod); err != nil || mod.String() != before {
		t.Fatalf("materialization is not idempotent: %v", err)
	}
	optimizeValueTest(t, mod)
	for name, want := range map[string]string{"caller": "ret i1 false", "relation": "ret i1 true"} {
		body := mod.NamedFunction(name).String()
		if !strings.Contains(body, want) || !strings.Contains(body, "@F(") {
			t.Fatalf("%s failed to use the result facts while preserving the call:\n%s", name, body)
		}
	}
}

func TestValueContractsEntryIntersection(t *testing.T) {
	mod := valueTestModule(t, `
define i1 @Input(ptr %pointer, i8 %n, i16 %element) {
entry:
  %nil = icmp eq ptr %pointer, null
  %negative = icmp slt i8 %n, 0
  %large = icmp sge i8 %n, 5
  %element.bad = icmp sgt i16 %element, 9
  %a = or i1 %nil, %negative
  %c = or i1 %a, %large
  %bad = or i1 %c, %element.bad
  ret i1 %bad
}
`)
	attachValueTestContracts(t, mod, "Input", `
//llgo:attribute param(p) nonnull
//llgo:attribute param(n) range(-4,5) nonnegative
//llgo:attribute param(element) range(0,10)
func Input(p *int, n int8, element int16) bool
`)
	if err := MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	optimizeValueTest(t, mod)
	if body := mod.NamedFunction("Input").String(); !strings.Contains(body, "ret i1 false") {
		t.Fatalf("entry facts or their intersection did not reach LLVM:\n%s", body)
	}
}

func TestResultContractDoesNotRemovePanickingCall(t *testing.T) {
	mod := valueTestModule(t, `
declare ptr @Checked(ptr)
define i1 @nil_input() {
entry:
  %r = call ptr @Checked(ptr null)
  %nonnull = icmp ne ptr %r, null
  ret i1 %nonnull
}
`)
	attachValueTestContracts(t, mod, "Checked", `
//llgo:attribute result(0) nonnull same_as(param(p))
func Checked(p *int) *int
`)
	if err := MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	optimizeValueTest(t, mod)
	if body := mod.NamedFunction("nil_input").String(); !strings.Contains(body, "@Checked(ptr null)") {
		t.Fatalf("normal-result contract deleted the required call on nil input:\n%s", body)
	}
}

func TestInvokeResultFactsOnlyOnNormalEdge(t *testing.T) {
	mod := valueTestModule(t, `
declare i32 @__gxx_personality_v0(...)
declare ptr @Checked(ptr)
define ptr @caller(i1 %should.call, ptr %p) personality ptr @__gxx_personality_v0 {
entry:
  br i1 %should.call, label %try, label %join
try:
  %r = invoke ptr @Checked(ptr %p) to label %join unwind label %unwind
join:
  %result = phi ptr [ %r, %try ], [ null, %entry ]
  ret ptr %result
unwind:
  %exception = landingpad { ptr, i32 } cleanup
  resume { ptr, i32 } %exception
}
`)
	attachValueTestContracts(t, mod, "Checked", `
//llgo:attribute result(0) nonnull same_as(param(p))
func Checked(p *int) *int
`)
	if err := MaterializeValueContracts(mod); err != nil {
		t.Fatal(err)
	}
	if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
		t.Fatalf("invoke result materialization invalid: %v\n%s", err, mod.String())
	}
	fn := mod.NamedFunction("caller")
	assumptions := 0
	for bb := fn.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
			if !instr.IsACallInst().IsNil() && instr.CalledValue().Name() == "llvm.assume" {
				assumptions++
				if !strings.HasPrefix(bb.AsValue().Name(), "contract.normal") {
					t.Fatalf("postcondition escaped normal invoke edge:\n%s", fn.String())
				}
			}
		}
	}
	if assumptions != 1 || !strings.Contains(fn.String(), "[ null, %entry ]") {
		t.Fatalf("normal edge or unrelated PHI input was lost:\n%s", fn.String())
	}
}
