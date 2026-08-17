//go:build !llgo

package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestFloatToUint32ConversionMode(t *testing.T) {
	const src = `package floatconvert
func Convert(x float32) uint32 { return uint32(x) }
`
	tests := []struct {
		name       string
		saturating bool
		want       string
		notWant    string
	}{
		{name: "legacy", want: "fptosi float", notWant: "fptoui float"},
		{name: "saturating", saturating: true, want: "fptoui float", notWant: "fptosi float"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ssaPkg, _, files := buildGoSSAPkg(t, src)
			prog := newLLSSAProgForTarget(t, &llssa.Target{
				GOOS:                    "linux",
				GOARCH:                  "amd64",
				SaturatingFloatToUint32: tt.saturating,
			})
			pkg, err := NewPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			ir := mustNamedFunction(t, pkg.Module(), "floatconvert.Convert").String()
			if !strings.Contains(ir, tt.want) || strings.Contains(ir, tt.notWant) {
				t.Fatalf("conversion IR does not use %s mode:\n%s", tt.name, ir)
			}
		})
	}
}
