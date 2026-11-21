package build

import (
	"testing"

	"github.com/xgo-dev/llgo/cl"
	"github.com/xgo-dev/llgo/internal/buildenv"
	"github.com/xgo-dev/llgo/internal/crosscompile"
	"github.com/xgo-dev/llgo/internal/lto"
	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestEmbeddedSizePruningConfig(t *testing.T) {
	conf := Config{Target: "esp32c3-flash", BuildMode: BuildModeExe, Mode: ModeBuild,
		OptLevel: optlevel.Oz, LTO: lto.Full, DeadcodeDrop: true, DisableGoGlobalDCE: true}
	target := crosscompile.Export{ExtraFiles: []string{"targets/device/esp/esp32c3.S"}}
	if got := embeddedSizePruningConfig(&conf, target, cl.Options{}); got != buildenv.Dev {
		t.Fatalf("bundled boot file: pruning = %v", got)
	}
	target.ExtraFiles = append(target.ExtraFiles, "foreign.c")
	if embeddedSizePruningConfig(&conf, target, cl.Options{}) {
		t.Fatal("foreign boot file did not disable pruning")
	}
}
