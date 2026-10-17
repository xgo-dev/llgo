package build

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/optlevel"
)

func TestCVariadicPromotedMethod(t *testing.T) {
	for _, level := range []optlevel.Level{optlevel.O0, optlevel.O2} {
		for _, fixture := range []struct {
			path string
			want string
		}{
			{"../../cl/_testrt/vamethod", "hello (100)\n(200) world\nifmt (100,200)\n"},
			{"./testdata/variadicmethod", "variadic-method-ok\n"},
		} {
			t.Run(level.String()+"/"+filepath.Base(fixture.path), func(t *testing.T) {
				conf := NewDefaultConf(ModeBuild)
				conf.OptLevel = level
				// The independent DWARF parameter-index bug is tracked by #2837.
				conf.LinkOptions.DWARF = DWARFOmit
				bin := filepath.Join(t.TempDir(), "variadic-method"+defaultAppExt(conf))
				conf.OutFile = bin
				if _, err := Do([]string{fixture.path}, conf); err != nil {
					t.Fatal(err)
				}
				if got := strings.ReplaceAll(runBinary(t, bin), "\r\n", "\n"); got != fixture.want {
					t.Fatalf("output = %q, want %q", got, fixture.want)
				}
			})
		}
	}
}
