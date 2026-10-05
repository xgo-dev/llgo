package build

import (
	"slices"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestEmscriptenEHFunctionFeature(t *testing.T) {
	for _, profile := range []crosscompile.WasmProfile{
		crosscompile.WasmProfileJ32, crosscompile.WasmProfileJ64,
		crosscompile.WasmProfileW32, crosscompile.WasmProfileNone,
	} {
		for _, workers := range []bool{false, true} {
			name := string(profile)
			if workers {
				name += "/workers"
			}
			t.Run(name, func(t *testing.T) {
				mod := parseWasmAggregateIR(t, `
declare void @external()
define void @plain() { ret void }
define void @vector() "target-features"="+simd128,-atomics,-bulk-memory,-exception-handling" { ret void }
define void @threaded() "target-features"="+atomics,+bulk-memory,+exception-handling" { ret void }
`)
				ctx := &context{crossCompile: crosscompile.Export{WasmProfile: profile}}
				if workers {
					ctx.crossCompile.CCFLAGS = []string{"-pthread"}
				}
				before := mod.String()
				applyEmscriptenEHFeature(ctx, mod)
				if profile == crosscompile.WasmProfileW32 || profile == crosscompile.WasmProfileNone {
					if mod.String() != before {
						t.Fatal("changed a non-Emscripten module")
					}
					return
				}
				for _, name := range []string{"plain", "vector", "threaded"} {
					var features []string
					for _, attr := range mod.NamedFunction(name).GetFunctionAttributes() {
						if attr.IsString() && attr.GetStringKind() == "target-features" {
							features = strings.Split(attr.GetStringValue(), ",")
						}
					}
					if !slices.Contains(features, "+exception-handling") || slices.Contains(features, "-exception-handling") {
						t.Fatalf("%s lacks the native EH capability: %v", name, features)
					}
					for _, feature := range []string{"atomics", "bulk-memory"} {
						positive := workers || name == "threaded"
						negative := !workers && name == "vector"
						if slices.Contains(features, "+"+feature) != positive || slices.Contains(features, "-"+feature) != negative {
							t.Fatalf("%s has incorrect %s capability: %v", name, feature, features)
						}
					}
					if slices.Contains(features, "+simd128") != (name == "vector") {
						t.Fatalf("changed unrelated SIMD capability: %v", features)
					}
				}
				if len(mod.NamedFunction("external").GetFunctionAttributes()) != 0 {
					t.Fatal("changed an external declaration")
				}
				first := mod.String()
				applyEmscriptenEHFeature(ctx, mod)
				if mod.String() != first {
					t.Fatal("EH feature application is not idempotent")
				}
			})
		}
	}
}

func TestEmscriptenFlagsSeparatePackageFingerprints(t *testing.T) {
	for _, target := range []struct{ goos, goarch string }{
		{"js", "wasm"}, {"wasip1", "wasm"}, {"linux", "amd64"},
	} {
		t.Run(target.goos, func(t *testing.T) {
			ctx := &context{buildConf: &Config{Goos: target.goos, Goarch: target.goarch}, llvmVersion: "test"}
			fingerprint := func(flags string) string {
				t.Setenv("EMCC_CFLAGS", flags)
				manifest := newManifestBuilder()
				ctx.collectEnvInputs(manifest)
				return manifest.Fingerprint()
			}
			plain := fingerprint("")
			nativeEH := fingerprint("-fwasm-exceptions -sSUPPORT_LONGJMP=wasm")
			if (plain != nativeEH) != (target.goos == "js") {
				t.Fatal("EMCC_CFLAGS cache isolation does not match the selected compiler")
			}
			if fingerprint("") != plain {
				t.Fatal("restoring default EH did not restore its cache key")
			}
		})
	}
}

func TestEmscriptenRejectsIncompatibleSjLj(t *testing.T) {
	for _, flags := range []string{
		"-sSUPPORT_LONGJMP=emscripten", "-s SUPPORT_LONGJMP=emscripten",
		"-sSUPPORT_LONGJMP=0", "-sSUPPORT_LONGJMP=1", "-sSUPPORT_LONGJMP",
		"-sSUPPORT_LONGJMP=wasm -sSUPPORT_LONGJMP=emscripten",
		"-mllvm -enable-emscripten-sjlj",
	} {
		for _, name := range []string{"CCFLAGS", "CFLAGS", "LDFLAGS", "EMCC_CFLAGS"} {
			t.Run(name+"/"+flags, func(t *testing.T) {
				commands := commandEnv{environ: []string{name + "=" + flags}}
				for _, provider := range []crosscompile.WasmProvider{crosscompile.WasmProviderGoJS, crosscompile.WasmProviderEmscripten, crosscompile.WasmProviderWASI} {
					err := validateEmscriptenSjLj(commands, &crosscompile.Export{WasmProvider: provider})
					if provider == crosscompile.WasmProviderWASI {
						if err != nil {
							t.Fatal(err)
						}
					} else if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "panic/recover") {
						t.Fatalf("provider %s did not diagnose conflicting %s: %v", provider, name, err)
					}
				}
			})
		}
	}
	for _, flags := range []string{"", "-sSUPPORT_LONGJMP=wasm", "-s SUPPORT_LONGJMP='wasm'", "-fwasm-exceptions -sSUPPORT_LONGJMP=wasm"} {
		commands := commandEnv{environ: []string{"EMCC_CFLAGS=" + flags}}
		if err := validateEmscriptenSjLj(commands, &crosscompile.Export{WasmProvider: crosscompile.WasmProviderGoJS}); err != nil {
			t.Fatalf("valid native flags %q: %v", flags, err)
		}
	}
}

func TestEmscriptenCatchSuspensionOptions(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"-sASSERTIONS=0"}, ""},
		{[]string{"-sBINARYEN_EXTRA_PASSES=--pass-arg=asyncify-asserts"}, ""},
		{[]string{"-s", "BINARYEN_EXTRA_PASSES=--pass-arg=asyncify-asserts,--vacuum"}, ""},
		{[]string{"-sBINARYEN_EXTRA_PASSES="}, ""},
		{[]string{"-s", "BINARYEN_EXTRA_PASSES=--vacuum"}, ""},
		{[]string{"-sBINARYEN_EXTRA_PASSES=--pass-arg=asyncify-asserts,--pass-arg=asyncify-ignore-unwind-from-catch"}, "cannot ignore"},
		{[]string{"--pass-arg=asyncify-ignore-unwind-from-catch"}, "cannot ignore"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			err := validateEmscriptenEHArgs("package link arguments", tc.args)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}
