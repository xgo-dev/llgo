package cl_test

import (
	"testing"

	"github.com/xgo-dev/llgo/cl/cltest"
	"github.com/xgo-dev/llgo/internal/build"
)

func TestSIMDMidwayCodegen(t *testing.T) {
	conf := build.NewDefaultConf(build.ModeGen)
	conf.Goos, conf.Goarch, conf.GOAMD64, conf.GOEXPERIMENT = "linux", "amd64", "v1", "simd"
	cltest.RunAndTestFromDir(t, "", "./_testsimd", nil, cltest.WithRunConfig(conf), cltest.WithOutputCheck(false))
}
