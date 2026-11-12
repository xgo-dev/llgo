package build

import (
	"go/types"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestCExportWrapperStackAlignment(t *testing.T) {
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm64"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			prog := llssa.NewProgram(&target)
			defer prog.Dispose()
			pkg := prog.NewPackage("exportstack", "exportstack")
			sig := newSignature([]types.Type{types.Typ[types.Uintptr]}, []types.Type{types.Typ[types.Uintptr]})
			defineCExportWrappers(pkg, []cExport{{goName: "exportstack.impl", cName: "foreign", sig: sig}}, nil)
			for _, name := range []string{"foreign", "exportstack.impl"} {
				fn := pkg.Module().NamedFunction(name)
				if fn.IsNil() {
					t.Fatalf("missing function %q", name)
				}
				want := name == "foreign" && target.GOARCH == "amd64" && target.GOOS != "windows"
				if got := !fn.GetStringAttributeAtIndex(-1, "stackrealign").IsNil(); got != want {
					t.Errorf("%s stackrealign=%v want %v", name, got, want)
				}
			}
		})
	}
}
