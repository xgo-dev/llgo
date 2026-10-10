package cabi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestIntegerExtensionAttributesSurviveLowering(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	const testIR = `
%Large = type { i64, i64, i64 }
%Split = type { i64, double }
%Empty = type {}

define %Large @large(i8 signext %a, i16 zeroext %b) {
  ret %Large zeroinitializer
}

define i8 @small(%Empty %empty, %Split %split, i8 signext %a, i16 zeroext %b) {
  ret i8 %a
}

define void @caller(ptr %largePtr, ptr %smallPtr, i8 %a, i16 %b) {
  %largeDirect = call %Large @large(i8 signext %a, i16 zeroext %b)
  %largeIndirect = call %Large %largePtr(i8 signext %a, i16 zeroext %b)
  %smallDirect = call i8 @small(%Empty zeroinitializer, %Split zeroinitializer, i8 signext %a, i16 zeroext %b)
  %smallIndirect = call i8 %smallPtr(%Empty zeroinitializer, %Split zeroinitializer, i8 zeroext %a, i16 signext %b)
  ret void
}
`
	for _, target := range []ssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "linux", GOARCH: "amd64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := ssa.NewProgram(&target)
			defer prog.Dispose()
			ctx := llvm.NewContext()
			defer ctx.Dispose()
			path := filepath.Join(t.TempDir(), "narrow.ll")
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
			NewTransformer(prog, target.Spec().Triple, "", true).TransformModule("test", mod)
			// The large return inserts sret. The small function removes an empty
			// parameter and expands the pair on amd64.
			largeWant := []string{"", "", "signext", "zeroext"}
			smallWant := []string{"", "", "signext", "zeroext"}
			indirectWant := []string{"", "", "zeroext", "signext"}
			if target.GOARCH == "amd64" {
				smallWant = []string{"", "", "", "signext", "zeroext"}
				indirectWant = []string{"", "", "", "zeroext", "signext"}
			}
			checkIntegerAttrs(t, mod.NamedFunction("large"), false, largeWant)
			checkIntegerAttrs(t, mod.NamedFunction("small"), false, smallWant)
			caller := mod.NamedFunction("caller")
			var calls []llvm.Value
			for bb := caller.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
				for instr := bb.FirstInstruction(); !instr.IsNil(); instr = llvm.NextInstruction(instr) {
					if call := instr.IsACallInst(); !call.IsNil() {
						calls = append(calls, call)
					}
				}
			}
			if len(calls) != 4 {
				t.Fatalf("got %d calls, want 4\n%s", len(calls), caller.String())
			}
			for i, want := range [][]string{largeWant, largeWant, smallWant, indirectWant} {
				checkIntegerAttrs(t, calls[i], true, want)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid integer extension ABI: %v\n%s", err, mod.String())
			}
		})
	}
}

func checkIntegerAttrs(t *testing.T, value llvm.Value, call bool, want []string) {
	t.Helper()
	for i, expected := range want {
		for _, name := range []string{"signext", "zeroext"} {
			var attr llvm.Attribute
			if call {
				attr = value.GetCallSiteEnumAttribute(i, llvm.AttributeKindID(name))
			} else {
				attr = value.GetEnumAttributeAtIndex(i, llvm.AttributeKindID(name))
			}
			if !attr.IsNil() != (expected == name) {
				t.Errorf("attribute %s at index %d: want %q\n%s", name, i, expected, value.String())
			}
		}
	}
}
