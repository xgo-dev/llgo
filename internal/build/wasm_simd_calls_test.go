package build

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
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
	t.Setenv("CCFLAGS", "")
	t.Setenv("CFLAGS", "")
	t.Setenv("LDFLAGS", "")
	for _, tc := range []struct {
		name     string
		provider crosscompile.WasmProvider
		flags    string
		bridges  int
	}{
		{"emscripten JS", crosscompile.WasmProviderEmscripten, "", 4},
		{"gojs JS", crosscompile.WasmProviderGoJS, "", 4},
		{"wasi", crosscompile.WasmProviderWASI, "", 0},
		{"emscripten native", crosscompile.WasmProviderEmscripten, "-fwasm-exceptions -sSUPPORT_LONGJMP=wasm", 0},
		{"gojs native", crosscompile.WasmProviderGoJS, "-fwasm-exceptions -sSUPPORT_LONGJMP=wasm", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &context{
				buildConf:    &Config{},
				crossCompile: crosscompile.Export{WasmProvider: tc.provider},
				commands:     commandEnv{environ: []string{"EMCC_CFLAGS=" + tc.flags}},
			}
			mod := parseWasmAggregateIR(t, source)
			before := mod.String()
			count := lowerEmscriptenSIMDCalls(ctx, mod)
			want := tc.bridges
			if count != want {
				t.Fatalf("bridges=%d want %d", count, want)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatalf("%v\n%s", err, mod.String())
			}
			if want == 0 && before != mod.String() {
				t.Fatal("changed a module without JS SjLj")
			}
			if !strings.Contains(mod.NamedFunction("ordinary").String(), "call <4 x float> @callee") {
				t.Fatal("changed ordinary vector ABI")
			}
			if want != 0 {
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
				if second := lowerEmscriptenSIMDCalls(ctx, mod); second != 0 {
					t.Fatalf("pass not idempotent: %d", second)
				}
			}
		})
	}
}

func TestEmscriptenSIMDCallBridgeMode(t *testing.T) {
	for _, tc := range []struct {
		name, cc, c, ld, emcc string
		configCC, configLD    []string
		cxx                   []string
		memory64, bridge      bool
	}{
		{name: "default", bridge: true},
		{name: "native flag", emcc: "-fwasm-exceptions"},
		{name: "native longjmp", emcc: "-s SUPPORT_LONGJMP='wasm'"},
		{name: "explicit JS", emcc: "-sSUPPORT_LONGJMP=emscripten", bridge: true},
		{name: "last longjmp wins", emcc: "-sSUPPORT_LONGJMP=wasm -s SUPPORT_LONGJMP=emscripten", bridge: true},
		{name: "default longjmp with EH", emcc: "-fwasm-exceptions -sSUPPORT_LONGJMP=1"},
		{name: "explicit EH setting", emcc: "-sWASM_EXCEPTIONS=1"},
		{name: "EH setting overrides flag", emcc: "-sWASM_EXCEPTIONS=0 -fwasm-exceptions", bridge: true},
		{name: "shared driver flags", cc: "-fwasm-exceptions"},
		{name: "compile only", c: "-fwasm-exceptions", bridge: true},
		{name: "link only", ld: "-fwasm-exceptions", bridge: true},
		{name: "config both", configCC: []string{"-fwasm-exceptions"}, configLD: []string{"-fwasm-exceptions"}},
		{name: "CXX link driver", c: "-fwasm-exceptions", cxx: []string{"-fwasm-exceptions"}},
		{name: "emcc overrides config", configCC: []string{"-sSUPPORT_LONGJMP=emscripten"}, configLD: []string{"-sSUPPORT_LONGJMP=emscripten"}, emcc: "-sSUPPORT_LONGJMP=wasm"},
		{name: "unknown response file", emcc: "-fwasm-exceptions @options.rsp", bridge: true},
		{name: "memory64 retains JS codegen", memory64: true, emcc: "-fwasm-exceptions -sSUPPORT_LONGJMP=wasm", bridge: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CCFLAGS", tc.cc)
			t.Setenv("CFLAGS", tc.c)
			t.Setenv("LDFLAGS", tc.ld)
			t.Setenv("EMCC_CFLAGS", tc.emcc)
			ctx := &context{buildConf: &Config{}, crossCompile: crosscompile.Export{
				WasmProvider: crosscompile.WasmProviderGoJS,
				CCFLAGS:      tc.configCC, LDFLAGS: tc.configLD,
			}}
			if tc.cxx != nil {
				ctx.crossCompile.CXX = "em++"
				ctx.crossCompile.CXXArgs = tc.cxx
			}
			if tc.memory64 {
				ctx.crossCompile.WasmProfile = crosscompile.WasmProfileJ64
			}
			if got := usesEmscriptenJSSjLj(ctx); got != tc.bridge {
				t.Fatalf("bridge = %v, want %v", got, tc.bridge)
			}
		})
	}
}

func TestEmscriptenSIMDCallBridgeArgumentPromotion(t *testing.T) {
	const source = `
target datalayout = "e-p:64:64-i64:64-n32:64-S128"
target triple = "wasm64-unknown-emscripten"
declare i32 @setjmp(ptr) returns_twice
declare <4 x float> @callee(<4 x float>)
define <4 x float> @withjmp(ptr %jmp, <4 x float> %x) "target-features"="+simd128" {
 %saved = call i32 @setjmp(ptr %jmp)
 %v = call <4 x float> @callee(<4 x float> %x)
 ret <4 x float> %v
}
`
	for _, pipeline := range []string{"cgscc(argpromotion)", "thinlto-pre-link<O3>", "lto-pre-link<O3>"} {
		t.Run(pipeline, func(t *testing.T) {
			mod := parseWasmAggregateIR(t, source)
			ctx := &context{crossCompile: crosscompile.Export{
				WasmProvider: crosscompile.WasmProviderEmscripten,
				WasmProfile:  crosscompile.WasmProfileJ64,
			}}
			if got := lowerEmscriptenSIMDCalls(ctx, mod); got != 1 {
				t.Fatalf("bridges = %d, want 1", got)
			}
			opts := llvm.NewPassBuilderOptions()
			defer opts.Dispose()
			if err := mod.RunPasses(pipeline, llvm.TargetMachine{}, opts); err != nil {
				t.Fatal(err)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			bridge := mod.NamedFunction("__llgo_simd_sjlj.0")
			if bridge.IsNil() || bridge.GlobalValueType().ReturnType().TypeKind() != llvm.VoidTypeKind {
				t.Fatalf("lost the memory bridge:\n%s", mod.String())
			}
			for _, param := range bridge.GlobalValueType().ParamTypes() {
				if param.TypeKind() == llvm.VectorTypeKind {
					t.Fatalf("late optimization promoted a bridge pointer to v128:\n%s", bridge.String())
				}
			}
		})
	}
}
