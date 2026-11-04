package cl

import (
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestCExportEntryStackAlignment(t *testing.T) {
	const source = `package exportstack
//export foreign
func foreign(x uintptr) uintptr { return declaration(x) + external(x) }
func crosscall2(x uintptr) uintptr { return x + 1 }
func declaration(x uintptr) uintptr
//export external
func external(x uintptr) uintptr
`
	for _, target := range []llssa.Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "386"},
		{GOOS: "linux", GOARCH: "arm64"},
	} {
		for _, wrapped := range []bool{false, true} {
			name := target.GOOS + "/" + target.GOARCH
			if wrapped {
				name += "/wrapper-implementation"
			}
			t.Run(name, func(t *testing.T) {
				goPkg, _, files := buildGoSSAPkg(t, source)
				prog := newLLSSAProgForTarget(t, &target)
				defer prog.Dispose()
				pkg, _, err := NewPackageExWithEmbedMetaOptions(prog, nil, nil, nil, goPkg, files, nil, false, Options{CExportWrappers: wrapped})
				if err != nil {
					t.Fatal(err)
				}
				foreign := "foreign"
				external := "external"
				if wrapped {
					foreign = "exportstack.foreign"
					external = "exportstack.external"
				}
				for _, symbol := range []string{foreign, external, "exportstack.crosscall2", "exportstack.declaration"} {
					fn := pkg.Module().NamedFunction(symbol)
					if fn.IsNil() {
						t.Fatalf("missing function %q", symbol)
					}
					want := !wrapped && symbol == foreign && target.GOARCH == "amd64" && target.GOOS != "windows"
					if got := !fn.GetStringAttributeAtIndex(-1, "stackrealign").IsNil(); got != want {
						t.Errorf("%s stackrealign=%v want %v", symbol, got, want)
					}
				}
			})
		}
	}
}
