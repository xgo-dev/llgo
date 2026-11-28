// Package abitest shares IR regression checks between ABI lowering passes.
package abitest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// CheckFunctionProperties checks symbol properties and signature-dependent
// attribute remapping independently through the supplied lowering entry point.
func CheckFunctionProperties(t *testing.T, aggregate string, lower func(llvm.Module)) {
	t.Helper()
	for _, tc := range []struct {
		name, declaration string
		properties        []string
	}{
		{"definition", `define linkonce_odr hidden fastcc %Aggregate @callee(ptr nest %p) local_unnamed_addr nounwind section "example" comdat($callee) align 64 gc "shadow-stack" prefix i32 123 prologue i32 456 personality ptr @personality { ret %Aggregate zeroinitializer }`, []string{"linkonce_odr hidden fastcc", "local_unnamed_addr", `section "example"`, "comdat", "align 64", `gc "shadow-stack"`, "prefix i32 123", "prologue i32 456", "personality ptr @personality"}},
		{"declaration", `declare hidden %Aggregate @callee(ptr nest) local_unnamed_addr nounwind align 32`, []string{"hidden", "local_unnamed_addr", "align 32"}},
		{"dll-export", `define dllexport %Aggregate @callee(ptr nest %p) unnamed_addr nounwind section ".text$callee" align 32 { ret %Aggregate zeroinitializer }`, []string{"dllexport", "unnamed_addr", `section ".text$callee"`, "align 32"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := llvm.NewContext()
			defer ctx.Dispose()
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
			lower(mod)
			fn := mod.NamedFunction("callee")
			if fn.IsNil() {
				t.Fatal("lowering dropped callee")
			}
			for _, property := range tc.properties {
				if !strings.Contains(fn.String(), property) {
					t.Errorf("lost %s:\n%s", property, fn.String())
				}
			}
			if fn.GlobalValueType().ReturnType().TypeKind() != llvm.VoidTypeKind {
				t.Errorf("result was not lowered to void:\n%s", fn.String())
			}
			if got := fn.ParamsCount(); got != 2 {
				t.Errorf("parameter count = %d, want 2", got)
			}
			for _, attr := range []struct {
				index int
				name  string
			}{{1, "sret"}, {2, "nest"}, {-1, "nounwind"}} {
				if fn.GetEnumAttributeAtIndex(attr.index, llvm.AttributeKindID(attr.name)).IsNil() {
					t.Errorf("missing %s attribute at index %d:\n%s", attr.name, attr.index, fn.String())
				}
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("invalid lowered module: %v\n%s", err, mod.String())
			}
		})
	}
}
