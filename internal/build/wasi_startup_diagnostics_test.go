package build

import (
	"go/token"
	"go/types"
	"strings"
	"testing"

	"golang.org/x/tools/go/ssa"
)

func TestWASIStartupDiagnosticsSourceContract(t *testing.T) {
	for _, test := range []struct {
		name, old, replacement string
		legacy, want           bool
	}{
		{name: "typed WASI imports", want: true},
		{name: "legacy unsafe WASI imports", legacy: true, want: true},
		{name: "unexpected stdio setup", old: "SetNonblock(1, true)", replacement: "SetNonblock(1, false)"},
		{name: "unexpected nonblocking flag lookup", old: "&stat)", replacement: "nil)"},
		{name: "unexpected preopen error", old: "fd_prestat: ", replacement: "preopen: "},
		{name: "different errno table", old: "[2]string", replacement: "[2]int"},
		{name: "missing errno table", old: "var errorstr [2]string", replacement: ""},
		{name: "missing caller location", old: "var dependency = initDependency()", replacement: ""},
		{name: "missing path guard", old: "if llgoWASIPathStateNeeded", replacement: "if true"},
	} {
		t.Run(test.name, func(t *testing.T) {
			preopen, flags := wasiCompactPreopenBody, wasiCompactGetFlagsBody
			if test.legacy {
				preopen = strings.NewReplacer(
					"&prestat)", "unsafe.Pointer(&prestat))",
					"&dirNameBuf[0],", "unsafe.Pointer(&dirNameBuf[0]),",
				).Replace(preopen)
				flags = strings.Replace(flags, "&stat", "unsafe.Pointer(&stat)", 1)
			}
			source := `package syscall
import "unsafe"
var _ unsafe.Pointer
type errno uint32
func (errno) Error() string { return "" }
const EBADF = 8
const ENOTDIR = 54
const preopentypeDir = 0
const FDFLAG_NONBLOCK = 4
type prestat struct { typ uint8; dir struct { prNameLen uint32 } }
type fdstat struct { fdflags uint16 }
type opendir struct { fd int32; name string }
var llgoWASIPathStateNeeded bool
var errorstr [2]string
var preopens []opendir
var cwd string
var dependency = initDependency()
func initDependency() int { return 0 }
func Getenv(string) (string, bool) { return "", false }
func joinPath(a, b string) string { return a+b }
func fd_prestat_get(int32, any) errno { return EBADF }
func fd_prestat_dir_name(int32, any, uint32) errno { return 0 }
func fd_fdstat_get(int32, any) errno { return 0 }
func fd_fdstat_set_flags(int32, uint32) errno { return 0 }
func errnoErr(errno) error { return nil }
func SetNonblock(fd int, nonblocking bool) error ` + wasiCompactSetNonblockBody +
				"\nfunc fd_fdstat_get_flags(fd int) (uint32, error) " + flags +
				"\nfunc init() " + wasiCompactStdioBody + "\nfunc init() " + preopen
			if test.old != "" {
				if !strings.Contains(source, test.old) {
					t.Fatalf("fixture missing %q", test.old)
				}
				source = strings.ReplaceAll(source, test.old, test.replacement)
			}
			prog := ssa.NewProgram(token.NewFileSet(), ssa.SanityCheckFunctions)
			prog.CreatePackage(types.Unsafe, nil, nil, true)
			buildFormatTestPackage(t, prog, "syscall", source, types.Unsafe).Build()
			got := wasiProcessDiagnostics(&context{progSSA: prog})
			if (got != nil) != test.want {
				t.Fatalf("startup diagnostics available = %t, want %t", got != nil, test.want)
			}
			if got != nil && (got.errnoTable.Len() != 2 ||
				!strings.Contains(got.prestat, "syscall.init#2(...)") ||
				!strings.Contains(got.prestatName, "syscall.init(...)") ||
				got.prestat == got.prestatName) {
				t.Fatalf("startup diagnostics = %+v, want separate preopen locations and two errno entries", got)
			}
		})
	}
}
