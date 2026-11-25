package build

import (
	"testing"
	"time"

	"github.com/xgo-dev/llgo/cmd/compiler/optlevel"
)

func TestNarrowCABI(t *testing.T) {
	for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		t.Run(level.String(), func(t *testing.T) {
			conf := NewDefaultConf(ModeTest)
			conf.OptLevel = level
			conf.RunArgs = []string{
				"-test.run=^(TestNarrowC|TestLinknameNarrowC)",
				"-test.v", "-test.count=1", "-test.timeout=90s",
			}
			conf.RunnerTimeout = 2 * time.Minute
			if _, err := Do([]string{"../../../test/cgo", "../../../test/llgoext"}, conf); err != nil {
				t.Fatal(err)
			}
		})
	}
}
