package cl

import (
	"strings"
	"testing"
)

func TestExternalCallerLocationsDoNotRootSourceStrings(t *testing.T) {
	ssapkg, files := buildCallerFrameSSAPackage(t, "example.com/external", `package external
import "runtime"
func observe() { runtime.Caller(0) }
`)
	for _, compact := range []bool{false, true} {
		prog := newLLSSAProg(t)
		prog.Target().GOOS, prog.Target().GOARCH = "wasip1", "wasm"
		prog.EnableFuncInfoMetadata(true)
		pkg, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, ssapkg, files, nil, false, Options{ShadowStack: true, CompactCallerLocations: compact})
		if err != nil {
			t.Fatal(err)
		}
		ir := pkg.Module().String()
		for _, literal := range []string{`c"example.com/external.observe`, `c"caller_frame_compile.go`} {
			if strings.Contains(ir, literal) == compact {
				t.Fatalf("compact=%v: unexpected runtime source string %s:\n%s", compact, literal, ir)
			}
		}
		if compact {
			for _, want := range []string{"PushCallerLocationFrameID", "RecordCallerLocationID", "!llgo.pcline"} {
				if !strings.Contains(ir, want) {
					t.Fatalf("missing %s:\n%s", want, ir)
				}
			}
		}
	}
}
