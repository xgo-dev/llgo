package dcepass

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestRegularTypeFieldOverrides(t *testing.T) {
	const text = `
%common = type { i64, i64, i32, i8, ptr }
%"github.com/xgo-dev/llgo/runtime/abi.StructType" = type { %common, { ptr, i64 }, { ptr, i64, i64 } }
%"github.com/xgo-dev/llgo/runtime/abi.UncommonType" = type { { ptr, i64 }, i16, i16, i32 }
%"github.com/xgo-dev/llgo/runtime/abi.Method" = type { { ptr, i64 }, ptr, ptr, ptr }
%descriptor = type { %"github.com/xgo-dev/llgo/runtime/abi.StructType", %"github.com/xgo-dev/llgo/runtime/abi.UncommonType", [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] }
@fields = private constant [1 x ptr] [ptr @child]
@child = external constant i8
@hidden = weak_odr hidden constant %descriptor {
 %"github.com/xgo-dev/llgo/runtime/abi.StructType" { %common { i64 24, i64 0, i32 42, i8 9, ptr @equal }, { ptr, i64 } zeroinitializer, { ptr, i64, i64 } { ptr @fields, i64 1, i64 1 } },
 %"github.com/xgo-dev/llgo/runtime/abi.UncommonType" { { ptr, i64 } zeroinitializer, i16 1, i16 1, i32 24 },
 [1 x %"github.com/xgo-dev/llgo/runtime/abi.Method"] [%"github.com/xgo-dev/llgo/runtime/abi.Method" { { ptr, i64 } zeroinitializer, ptr null, ptr @method, ptr @method }]
}, align 32
@visible = weak_odr constant %"github.com/xgo-dev/llgo/runtime/abi.StructType" { %common { i64 24, i64 0, i32 42, i8 8, ptr @equal }, { ptr, i64 } zeroinitializer, { ptr, i64, i64 } { ptr @fields, i64 1, i64 1 } }
@irregular = weak_odr constant %"github.com/xgo-dev/llgo/runtime/abi.StructType" { %common { i64 24, i64 0, i32 42, i8 0, ptr @equal }, { ptr, i64 } zeroinitializer, { ptr, i64, i64 } { ptr @fields, i64 1, i64 1 } }
@mutable = global i64 42
declare void @equal()
declare void @method()
`
	for _, mode := range []string{"new", "declaration", "in-place", "compacted", "nil", "keep"} {
		t.Run(mode, func(t *testing.T) {
			srcCtx, dstCtx := llvm.NewContext(), llvm.NewContext()
			defer srcCtx.Dispose()
			defer dstCtx.Dispose()
			path := filepath.Join(t.TempDir(), "fields.ll")
			if err := os.WriteFile(path, []byte(text), 0600); err != nil {
				t.Fatal(err)
			}
			src := parseModule(t, &srcCtx, path)
			defer src.Dispose()
			before := src.String()
			dst := src
			if mode != "in-place" {
				dst = dstCtx.NewModule("dst")
				defer dst.Dispose()
			}
			if mode == "declaration" {
				llvm.AddGlobal(dst, newOverrideEmitter(dst).cloneType(src.NamedGlobal("hidden").GlobalValueType()), "hidden")
			}
			if mode == "compacted" {
				EmitStrongTypeOverrides(dst, []llvm.Module{src}, nil, func(string) bool { return true }, false)
			}
			drop := func(name string) bool { return name != "visible" && mode != "keep" }
			if mode == "nil" {
				drop = nil
			}
			EmitRegularTypeFieldOverrides(dst, []llvm.Module{src, src}, drop)
			if mode == "nil" || mode == "keep" {
				if !dst.FirstGlobal().IsNil() {
					t.Fatal("disabled pruning emitted globals")
				}
				return
			}
			g := dst.NamedGlobal("hidden")
			if g.IsNil() {
				t.Fatal("missing strong descriptor")
			}
			value := g.Initializer()
			fields := value.Operand(0)
			if !fields.Operand(2).IsNull() {
				t.Fatal("field table retained")
			}
			if fields.Operand(0).Operand(2).ZExtValue() != 42 || fields.Operand(0).Operand(4) != dst.NamedFunction("equal") {
				t.Fatal("descriptor prefix changed")
			}
			if g.Visibility() != llvm.HiddenVisibility || g.Alignment() != 32 || g.Linkage() != llvm.ExternalLinkage {
				t.Fatal("descriptor attributes changed")
			}
			if mode == "compacted" && value.Operand(2).Type().ArrayLength() != 0 {
				t.Fatal("method compaction undone")
			}
			for _, name := range []string{"visible", "irregular"} {
				other := dst.NamedGlobal(name)
				if mode == "in-place" {
					if other.Initializer().Operand(2).IsNull() {
						t.Fatalf("%s lost fields", name)
					}
				} else if !other.IsNil() {
					t.Fatalf("unexpected override for %s", name)
				}
			}
			if mode != "in-place" && src.String() != before {
				t.Fatal("source module mutated")
			}
			stable := dst.String()
			EmitRegularTypeFieldOverrides(dst, []llvm.Module{src, dst}, drop)
			if dst.String() != stable {
				t.Fatal("repeated pruning changed output")
			}
			if err := llvm.VerifyModule(dst, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(dst.String()), 0600); err != nil {
				t.Fatal(err)
			}
			roundCtx := llvm.NewContext()
			defer roundCtx.Dispose()
			parsed := parseModule(t, &roundCtx, path)
			parsed.Dispose()
		})
	}
}

func TestTrimRegularTypeFieldsGuards(t *testing.T) {
	for _, tc := range []struct{ common, value, fields string }{
		{"i64", "i64 1", ""},
		{"{ i64 }", "{ i64 } { i64 1 }", ", i64 1, i64 1"},
		{"{ i64, i64, i64, ptr }", "{ i64, i64, i64, ptr } { i64 1, i64 1, i64 1, ptr null }", ", i64 1, i64 1"},
		{"{ i64, i64, i64, i8 }", "{ i64, i64, i64, i8 } { i64 1, i64 1, i64 1, i8 8 }", ", i64 1, i64 0"},
	} {
		ctx := llvm.NewContext()
		defer ctx.Dispose()
		rest := strings.Repeat(", i64", strings.Count(tc.fields, ","))
		text := fmt.Sprintf(`%%"github.com/xgo-dev/llgo/runtime/abi.StructType" = type { %s%s }
@guard = constant %%"github.com/xgo-dev/llgo/runtime/abi.StructType" { %s%s }
@scalar = constant i64 1
@empty = constant {} zeroinitializer
`, tc.common, rest, tc.value, tc.fields)
		path := filepath.Join(t.TempDir(), "guard.ll")
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		mod := parseModule(t, &ctx, path)
		defer mod.Dispose()
		values := []llvm.Value{{}, mod.NamedGlobal("guard").Initializer(), mod.NamedGlobal("scalar").Initializer(), mod.NamedGlobal("empty").Initializer()}
		for _, value := range values {
			if _, changed := trimRegularTypeFields(value); changed {
				t.Fatalf("malformed or empty descriptor changed: %s", value)
			}
		}
	}
}
