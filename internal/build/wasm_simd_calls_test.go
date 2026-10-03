package build

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestEmscriptenSIMDCallBridge(t *testing.T) {
	const source = `
declare i32 @setjmp(ptr) returns_twice
declare <4 x float> @callee(<4 x float>, ptr)
declare float @scalar(float)
declare float @consumevec(<4 x float>)
declare void @sink(<4 x float>)
declare <4 x float> @llvm.sqrt.v4f32(<4 x float>)
define <4 x float> @withjmp(ptr %jmp, ptr %fn, ptr %env, <4 x float> %x) {
 %saved = call i32 @setjmp(ptr %jmp)
 %a = call <4 x float> @callee(<4 x float> %x, ptr %env)
 %b = call <4 x float> %fn(<4 x float> %a, ptr %env)
 %s = call float @scalar(float 1.0)
 %late = call float @late()
 call void @sink(<4 x float> %a)
 %u = call float @consumevec(<4 x float> %a)
 %c = call <4 x float> @llvm.sqrt.v4f32(<4 x float> %b)
 ret <4 x float> %c
}
define float @late() alwaysinline {
 %v = call <4 x float> @callee(<4 x float> zeroinitializer, ptr null)
 %f = extractelement <4 x float> %v, i32 0
 ret float %f
}
define <4 x float> @ordinary(<4 x float> %x, ptr %env) {
 %v = call <4 x float> @callee(<4 x float> %x, ptr %env)
 ret <4 x float> %v
}
`
	for _, provider := range []string{"emscripten", "wasi", "gojs"} {
		t.Run(provider, func(t *testing.T) {
			mod := parseWasmAggregateIR(t, source)
			before := mod.NamedFunction("ordinary").String()
			count := lowerEmscriptenSIMDCalls(provider, mod)
			want := 0
			if provider == "emscripten" {
				want = 4
			}
			if count != want {
				t.Fatalf("bridges=%d want %d", count, want)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v\n%s", err, mod.String())
			}
			if provider != "emscripten" && before != mod.NamedFunction("ordinary").String() {
				t.Fatal("changed other provider")
			}
			if !strings.Contains(mod.NamedFunction("ordinary").String(), "call <4 x float> @callee") {
				t.Fatal("changed ordinary vector ABI")
			}
			if provider == "emscripten" {
				ir := mod.NamedFunction("withjmp").String()
				if strings.Contains(ir, "call <4 x float> @callee") || strings.Contains(ir, "call <4 x float> %fn") {
					t.Fatalf("v128 crosses JS SjLj boundary:\n%s", ir)
				}
				for _, name := range []string{"__llgo_simd_sjlj.0", "__llgo_simd_sjlj.1"} {
					fn := mod.NamedFunction(name)
					if fn.IsNil() || fn.GlobalValueType().ReturnType().TypeKind() != llvm.VoidTypeKind {
						t.Fatal("missing memory bridge")
					}
					if strings.Contains(fn.String(), "setjmp") || !strings.Contains(fn.String(), "call <4 x float>") {
						t.Fatalf("bad bridge:\n%s", fn.String())
					}
				}
				opts := llvm.NewPassBuilderOptions()
				defer opts.Dispose()
				if err := mod.RunPasses("default<O2>", llvm.TargetMachine{}, opts); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(mod.NamedFunction("withjmp").String(), "call float @late") {
					t.Fatalf("late optimization imported an unbridged vector call:\n%s", mod.String())
				}
				if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
					t.Fatal(err)
				}
				if second := lowerEmscriptenSIMDCalls(provider, mod); second != 0 {
					t.Fatalf("pass not idempotent: %d", second)
				}
			}
		})
	}
}
