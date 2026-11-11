package build

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestStaticPointerInterfaceFullLTO(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeTest)
	conf.OptLevel, conf.LTO = optlevel.O3, lto.Full
	// Exercise the ordinary release profile even in a -tags=dev test run.
	// The optional method-drop/plugin profile deliberately keeps NewItab.
	conf.DisableGoGlobalDCE = true
	conf.CompileOnly = true
	conf.OutFile = filepath.Join(t.TempDir(), "static-itab.test")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	var conversion string
	conf.ModuleHook = func(pkg Package) {
		ir := pkg.LPkg.String()
		if strings.Contains(ir, ".staticItabConvert\"(") || strings.Contains(ir, ".staticItabConvert(") {
			conversion = llvmFunctionBody(t, ir, "staticItabConvert")
		}
	}
	if _, err := Do([]string{"../../test/go/static_itab_test.go"}, conf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(conversion, "_llgo_itab$") {
		t.Fatalf("static pointer conversion did not load a static itab:\n%s", conversion)
	}
	if strings.Contains(conversion, "runtime.NewItab") || strings.Contains(conversion, "runtime.Alloc") {
		t.Fatalf("static pointer conversion retained runtime lookup or boxing:\n%s", conversion)
	}
	if output := runBinary(t, conf.OutFile, "-test.run=^TestStaticPointerInterfaceConversion$"); !strings.Contains(output, "PASS") {
		t.Fatalf("full-LTO conversion regression failed:\n%s", output)
	}
}
