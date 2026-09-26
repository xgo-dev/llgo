package wasmworkers

import (
	"path/filepath"
	"testing"

	"github.com/xgo-dev/llgo/internal/crosscompile"
)

func TestParse(t *testing.T) {
	for _, test := range []struct {
		value string
		count int
		err   bool
	}{
		{count: 1},
		{value: "1", count: 1},
		{value: "2", count: 2},
		{value: "16", count: 16},
		{value: "0", err: true},
		{value: "17", err: true},
		{value: "two", err: true},
	} {
		got, err := Parse(test.value)
		if (err != nil) != test.err {
			t.Fatalf("Parse(%q) error = %v, want error %v", test.value, err, test.err)
		}
		if !test.err && got.Count != test.count {
			t.Fatalf("Parse(%q).Count = %d, want %d", test.value, got.Count, test.count)
		}
	}
}

func TestValidateTarget(t *testing.T) {
	if err := (Config{Count: 2}).ValidateTarget("js", "wasm", crosscompile.WasmProfileJ32, crosscompile.WasmProviderEmscripten); err != nil {
		t.Fatal(err)
	}
	if err := (Config{Count: 2}).ValidateTarget("js", "wasm", crosscompile.WasmProfileJ64, crosscompile.WasmProviderEmscripten); err != nil {
		t.Fatal(err)
	}
	for _, target := range []struct {
		goos, goarch string
		profile      crosscompile.WasmProfile
		provider     crosscompile.WasmProvider
	}{
		{"js", "wasm", crosscompile.WasmProfileJ32, crosscompile.WasmProviderGoJS},
		{"wasip1", "wasm", crosscompile.WasmProfileW32, crosscompile.WasmProviderWASI},
		{"linux", "amd64", crosscompile.WasmProfileNone, crosscompile.WasmProviderNone},
	} {
		if err := (Config{Count: 2}).ValidateTarget(target.goos, target.goarch, target.profile, target.provider); err == nil {
			t.Fatalf("ValidateTarget(%q, %q, %q, %q) succeeded", target.goos, target.goarch, target.profile, target.provider)
		}
	}
	if err := (Config{Count: 1}).ValidateTarget("linux", "amd64", crosscompile.WasmProfileNone, crosscompile.WasmProviderNone); err != nil {
		t.Fatalf("disabled config rejected native target: %v", err)
	}
}

func TestPreJSPath(t *testing.T) {
	want := filepath.Join("llgo", "internal", "wasmworkers", "worker_pre.js")
	if got := PreJSPath("llgo"); got != want {
		t.Fatalf("PreJSPath() = %q, want %q", got, want)
	}
}
