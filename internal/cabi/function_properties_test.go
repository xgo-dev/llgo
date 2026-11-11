package cabi

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/abi"
	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

func TestAggregateLoweringFunctionProperties(t *testing.T) {
	llvm.InitializeAllTargets()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllTargetInfos()
	for _, lowering := range []string{"large", "cabi"} {
		for _, tc := range []struct {
			name, declaration string
			properties        []string
		}{
			{"definition", `define linkonce_odr hidden fastcc %Aggregate @callee(ptr nest %p) local_unnamed_addr nounwind section "example" comdat($callee) align 64 gc "shadow-stack" prefix i32 123 prologue i32 456 personality ptr @personality { ret %Aggregate zeroinitializer }`, []string{"linkonce_odr hidden fastcc", "local_unnamed_addr", `section "example"`, "comdat", "align 64", `gc "shadow-stack"`, "prefix i32 123", "prologue i32 456", "personality ptr @personality"}},
			{"declaration", `declare hidden %Aggregate @callee(ptr nest) local_unnamed_addr nounwind align 32`, []string{"hidden", "local_unnamed_addr", "align 32"}},
			{"dll-export", `define dllexport %Aggregate @callee(ptr nest %p) unnamed_addr nounwind section ".text$callee" align 32 { ret %Aggregate zeroinitializer }`, []string{"dllexport", "unnamed_addr", `section ".text$callee"`, "align 32"}},
		} {
			t.Run(lowering+"/"+tc.name, func(t *testing.T) {
				ctx := llvm.NewContext()
				defer ctx.Dispose()
				aggregate := "{ i64, i64, i64 }"
				if lowering == "large" {
					aggregate = "[65537 x i8]"
				}
				source := fmt.Sprintf("%%Aggregate = type %s\n$callee = comdat any\ndeclare i32 @personality(...)\n%s\n", aggregate, tc.declaration)
				path := filepath.Join(t.TempDir(), "properties.ll")
				if err := os.WriteFile(path, []byte(source), 0600); err != nil {
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
					t.Fatalf("invalid input: %v", err)
				}
				if lowering == "large" {
					td := llvm.NewTargetData("e-p:64:64-i64:64-n8:16:32:64-S128")
					defer td.Dispose()
					abi.LowerLargeAggregates(td, mod, abi.AggregateLoweringConfig{GoWordSize: 8})
				} else {
					prog := llssa.NewProgram(&llssa.Target{GOOS: "linux", GOARCH: "amd64"})
					defer prog.Dispose()
					NewTransformer(prog, "x86_64-unknown-linux-gnu", "", true).TransformModule("test", mod)
				}
				fn := mod.NamedFunction("callee")
				for _, property := range tc.properties {
					if !strings.Contains(fn.String(), property) {
						t.Errorf("lost %s:\n%s", property, fn.String())
					}
				}
				if fn.GlobalValueType().ReturnType().TypeKind() != llvm.VoidTypeKind || fn.ParamsCount() != 2 || fn.GetEnumAttributeAtIndex(1, llvm.AttributeKindID("sret")).IsNil() || fn.GetEnumAttributeAtIndex(2, llvm.AttributeKindID("nest")).IsNil() || fn.GetEnumFunctionAttribute(llvm.AttributeKindID("nounwind")).IsNil() {
					t.Fatalf("signature or attribute remapping changed:\n%s", fn.String())
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
				}
			})
		}
	}
}
