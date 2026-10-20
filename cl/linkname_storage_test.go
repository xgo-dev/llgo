package cl

import (
	"fmt"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestLinknameVariableStorageIR(t *testing.T) {
	const source = `package shared
import _ "unsafe"
//go:linkname Zero shared_zero
var Zero int32
//go:linkname Static shared_static
var Static int32 = 41
//go:linkname Callback shared_callback
var Callback func(int32) int32
//go:linkname Native C.shared_native
var Native func(int32) int32
//go:linkname Alias shared_static
var Alias int32
//go:linkname Reserved go:shared_value
var Reserved int32
//go:cgo_import_dynamic shared_import imported "libexample.so"
//go:linkname Imported shared_import
var Imported uintptr
//go:cgo_import_dynamic shared.Default imported "libexample.so"
var Default uintptr
//go:cgo_import_static static_import
//go:linkname StaticImport static_import
var StaticImport byte
//go:cgo_import_dynamic shared.Initialized imported "libexample.so"
var Initialized uintptr = 7
`
	for _, target := range []*llssa.Target{
		{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "386"}, {GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"}, {GOOS: "wasip1", GOARCH: "wasm"},
	} {
		t.Run(target.GOOS+"/"+target.GOARCH, func(t *testing.T) {
			ir := compileWithRewritesTarget(t, source, nil, target)
			for _, want := range []string{
				"@shared_zero = common global i32 0", "@shared_static = global i32 41",
				"@shared_callback = common global { ptr, ptr } zeroinitializer",
				"@shared_native = external global ptr", `@"go:shared_value" = common global i32 0`,
				"@static_import = external global i8", "@shared_import = external global i", "@shared.Default = external global i",
				fmt.Sprintf("@shared.Initialized = global i%d 7", map[bool]int{true: 32, false: 64}[target.GOARCH == "386" || target.GOARCH == "wasm"]),
			} {
				if target.GOARCH == "wasm" {
					want = strings.ReplaceAll(want, "common global", "weak global")
				}
				if !strings.Contains(ir, want) {
					t.Errorf("missing %q:\n%s", want, ir)
				}
			}
		})
	}
}
