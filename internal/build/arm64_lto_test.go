package build

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestDarwinARM64LTOFeatures(t *testing.T) {
	llssa.Initialize(llssa.InitAll)
	const triple = "arm64-apple-macosx11.0.0"
	backend, err := llvm.GetTargetFromTriple(triple)
	if err != nil {
		t.Fatal(err)
	}
	machine := backend.CreateTargetMachine(triple, "generic", "", llvm.CodeGenLevelDefault, llvm.RelocPIC, llvm.CodeModelDefault)
	defer machine.Dispose()

	for _, test := range []struct {
		name   string
		goos   string
		goarm  string
		lto    bool
		want   string
		opcode string
	}{
		{"baseline", "darwin", "v8.0", true, "", "ldaxr"},
		{"no LTO", "darwin", "v8.0,lse", false, "", "ldaxr"},
		{"other OS", "linux", "v8.0,lse", true, "", "ldaxr"},
		{"explicit LSE", "darwin", "v8.0,lse", true, "+neon,+lse", "ldaddal"},
		{"implied LSE", "darwin", "v8.1,lse", true, "+v8.1a,+neon,+lse", "ldaddal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			lc := llvm.NewContext()
			defer lc.Dispose()
			mod := lc.NewModule("features")
			defer mod.Dispose()
			mod.SetTarget(triple)
			fnType := llvm.FunctionType(lc.Int64Type(), []llvm.Type{llvm.PointerType(lc.Int64Type(), 0)}, false)
			fn := llvm.AddFunction(mod, "increment", fnType)
			guarded := llvm.AddFunction(mod, "guarded", fnType)
			declaration := llvm.AddFunction(mod, "declaration", fnType)
			builder := lc.NewBuilder()
			for _, def := range []llvm.Value{fn, guarded} {
				builder.SetInsertPointAtEnd(lc.AddBasicBlock(def, "entry"))
				result := builder.CreateAtomicRMW(llvm.AtomicRMWBinOpAdd, def.Param(0),
					llvm.ConstInt(lc.Int64Type(), 1, false), llvm.AtomicOrderingSequentiallyConsistent, false)
				builder.CreateRet(result)
			}
			builder.Dispose()

			const attrName = "target-features"
			guarded.AddFunctionAttr(lc.CreateStringAttribute(attrName, "+aes,-lse"))
			target := &llssa.Target{GOOS: test.goos, GOARCH: "arm64", GOARM64: test.goarm}
			applyDarwinARM64LTOFeatures(test.lto, target, mod)
			features := func(fn llvm.Value) string {
				if attr := fn.GetStringAttributeAtIndex(-1, attrName); !attr.IsNil() {
					return attr.GetStringValue()
				}
				return ""
			}
			if got := features(fn); got != test.want {
				t.Fatalf("target-features = %q, want %q", got, test.want)
			}
			if got := features(guarded); !strings.HasSuffix(got, "+aes,-lse") || strings.Contains(got, "+lse") {
				t.Fatalf("feature override lost: %q", got)
			}
			if got := features(declaration); got != "" {
				t.Fatalf("declaration carries target features: %q", got)
			}
			before := mod.String()
			applyDarwinARM64LTOFeatures(test.lto, target, mod)
			if mod.String() != before {
				t.Fatal("applying the target features twice changed the module")
			}
			asm, err := machine.EmitToMemoryBuffer(mod, llvm.AssemblyFile)
			if err != nil {
				t.Fatal(err)
			}
			defer asm.Dispose()
			text := string(asm.Bytes())
			body, _, _ := strings.Cut(text, "_guarded:")
			if !strings.Contains(body, test.opcode) {
				t.Fatalf("want %s in increment:\n%s", test.opcode, text)
			}
		})
	}
}
