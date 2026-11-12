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
	if testing.Short() {
		t.Skip("builds and runs the O3/full-LTO conversion regression")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LLGO_ROOT", root)
	conf := NewDefaultConf(ModeTest)
	conf.OptLevel, conf.LTO = optlevel.O3, lto.Full
	// Disable Go global DCE so a dev-tag run matches the release profile.
	// Keep LTOPlugin and DeadcodeDrop disabled; those modes route T2I through NewItab.
	conf.DisableGoGlobalDCE = true
	conf.CompileOnly = true
	conf.OutFile = filepath.Join(t.TempDir(), "static-itab.test")
	if runtime.GOOS == "windows" {
		conf.OutFile += ".exe"
	}
	var conversion string
	// ModuleHook observes frontend lowering before optimization. The final
	// O3/full-LTO binary is exercised by runBinary below.
	conf.ModuleHook = func(pkg Package) {
		if pkg.Name != "gotest" {
			return
		}
		ir := pkg.LPkg.String()
		if strings.Contains(ir, ".staticItabConvert\"(") || strings.Contains(ir, ".staticItabConvert(") {
			conversion = llvmFunctionBody(t, ir, "staticItabConvert")
		}
	}
	if _, err := Do([]string{"../../test/go/static_itab_test.go"}, conf); err != nil {
		t.Fatal(err)
	}
	if conversion == "" {
		t.Fatal("frontend conversion fixture was not captured")
	}
	if !strings.Contains(conversion, "_llgo_itab$") {
		t.Fatalf("frontend static pointer conversion did not load a static itab:\n%s", conversion)
	}
	if strings.Contains(conversion, "runtime.NewItab") {
		t.Fatalf("frontend static pointer conversion retained runtime lookup:\n%s", conversion)
	}
	if output := runBinary(t, conf.OutFile, "-test.run=^TestStaticPointerInterfaceConversion$"); !strings.Contains(output, "PASS") {
		t.Fatalf("full-LTO conversion regression failed:\n%s", output)
	}
}
