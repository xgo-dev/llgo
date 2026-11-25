package cabi_test

import (
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/build"
	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
)

func TestWasmBoundaryHeaderProjections(t *testing.T) {
	runWasmHeaderConfigs(t, checkWasmHeaderProjections)
}

func checkWasmHeaderProjections(t *testing.T, tags string, level optlevel.Level) {
	t.Helper()
	scalarReaders := []string{
		"FirstSliceLen", "LastSliceCap", "FirstStringLen", "NestedSliceLen", "SliceTag",
		"ResultSliceLen", "ResultSliceCap", "ResultStringLen", "ResultSliceTag",
		"FirstSliceData", "ResultSliceData",
	}
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Target, conf.Tags, conf.OptLevel = "wasi", tags, level
	pkgs, err := build.Do([]string{"./_testdata/wasmheaderboundary/projections/main.go"}, conf)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs[0].LPkg
	defer pkg.Prog.Dispose()
	mod := pkg.Module()
	for _, group := range []struct {
		names          []string
		wantRangeClear bool
	}{
		{scalarReaders, false},
		// A typed load already clears the one header being returned.
		{[]string{"FirstSliceValue", "ResultSliceValue"}, false},
		{[]string{
			"FirstSliceSlot", "ResultSliceSlot", "ForwardSlices", "ForwardResult",
			"SliceLenOrSlot", "ResultLenOrSlot", "StoreSlices", "StoreResult",
		}, true},
	} {
		for _, name := range group.names {
			fn := mod.NamedFunction(name)
			if fn.IsNil() {
				t.Fatalf("missing %s", name)
			}
			if got := strings.Contains(fn.String(), "@__llgo_clear_header_range"); got != group.wantRangeClear {
				t.Errorf("%s has header range clear=%v, want %v", name, got, group.wantRangeClear)
			}
		}
	}
	optimizeWasmModule(t, pkg, level)
	if level != optlevel.O2 {
		return
	}
	// Observe the high word of a typed copy after optimization. Its value must
	// be zero even if the incoming C header held arbitrary padding.
	fn := mod.NamedFunction("TypedSlicePadding")
	if fn.IsNil() {
		t.Fatal("missing TypedSlicePadding")
	}
	if !strings.Contains(fn.String(), "ret i32 0") {
		t.Errorf("typed slice padding was not zeroed:\n%s", fn)
	}
	if tags != "nogc" {
		// Publishing GC roots independently writes to the shadow stack.
		return
	}
	text := wasmAssembly(t, pkg)
	for _, name := range scalarReaders {
		body := wasmAssemblyBody(t, text, name)
		// Keep the runtime context calls opaque. Allow their scalar stack
		// accesses, but catch aggregate copies even when LLVM unrolls them.
		loads, stores := strings.Count(body, ".load"), strings.Count(body, ".store")
		if loads > 4 || stores > 4 || strings.Contains(body, "\tloop") ||
			strings.Contains(body, "memory.copy") || strings.Contains(body, "memory.fill") {
			t.Errorf("%s materializes an aggregate for a scalar projection (%d loads, %d stores)", name, loads, stores)
		}
	}
}
