package ssa

import (
	"go/types"
	"testing"
)

func TestNativeExportStackAlignmentDefaultTarget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target *Target
	}{
		{"arch-only", &Target{GOARCH: "amd64"}},
		{"os-only", &Target{GOOS: "linux"}},
		{"empty", &Target{}},
		{"nil", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog := NewProgram(tc.target)
			defer prog.Dispose()
			pkg := prog.NewPackage("exportstack", "exportstack")
			fn := pkg.NewFunc("entry", types.NewSignature(nil, nil, nil, false), GoDecl)
			fn.MakeBody(1).Return()
			fn.AlignNativeStack()
			target := prog.Target()
			want := target.effectiveGOARCH() == "amd64" && (target.effectiveGOOS() == "linux" || target.effectiveGOOS() == "darwin")
			if got := !pkg.Module().NamedFunction("entry").GetStringAttributeAtIndex(-1, "stackrealign").IsNil(); got != want {
				t.Fatalf("effective %s/%s stackrealign=%v want %v", target.effectiveGOOS(), target.effectiveGOARCH(), got, want)
			}
		})
	}
}
