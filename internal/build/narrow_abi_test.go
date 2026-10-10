package build

import (
	"flag"
	"testing"
	"time"
)

var narrowCABIHostedWasm = flag.Bool("narrow-cabi-wasm", false, "execute narrow C ABI regressions on all hosted Wasm profiles")

func runNarrowCABI(t *testing.T, conf *Config, packages ...string) {
	t.Helper()
	conf.RunArgs = []string{
		"-test.run=^(TestNarrowC|TestLinkname.*CVariadicCall|TestLinknameNarrowC)",
		"-test.v", "-test.count=1", "-test.timeout=90s",
	}
	conf.RunnerTimeout = 2 * time.Minute
	if _, err := Do(packages, conf); err != nil {
		t.Fatal(err)
	}
}

func TestNarrowCABI(t *testing.T) {
	runNarrowCABI(t, NewDefaultConf(ModeTest), "../../test/cgo", "../../test/llgoext")
}

func TestNarrowCABIHostedWasm(t *testing.T) {
	if !*narrowCABIHostedWasm {
		t.Skip("enable -narrow-cabi-wasm with the hosted Wasm toolchains installed")
	}
	for _, profile := range []string{"emscripten", "emscripten-memory64", "wasi", "gojs"} {
		t.Run(profile, func(t *testing.T) {
			conf := NewDefaultConf(ModeTest)
			conf.Emulator = true
			if profile == "gojs" {
				conf.Goos, conf.Goarch = "js", "wasm"
			} else {
				conf.Target = profile
			}
			runNarrowCABI(t, conf, "../../test/llgoext")
		})
	}
}
