package cl

import (
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
)

func TestHostLayoutWasmFieldAccess(t *testing.T) {
	const src = `package hostaccess
import "structs"
type Inner struct { P *byte; W uintptr }
type Record struct { _ structs.HostLayout; Words [2]uintptr; P *byte; Inner Inner }
func Append(items []Record, value Record) []Record { return append(items, value) }
func Element(items []Record, i int) *Record { return &items[i] }
func Native(p *Record) uintptr { p.Words[1] = 9; return p.Words[0] }
func GoNested(p *Record) uintptr { p.Inner.W = 9; return p.Inner.W }
func NativePointer(p *Record, q *byte) *byte { p.P = q; return p.P }
func GoPointer(p *Record, q *byte) *byte { p.Inner.P = q; return p.Inner.P }
`
	ir := compileWithRewritesTarget(t, src, nil, &llssa.Target{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"})
	for _, forbidden := range []string{"inttoptr { ptr, i32 }", "getelementptr inbounds %hostaccess.Record, { ptr, i32 }"} {
		if strings.Contains(ir, forbidden) {
			t.Fatalf("host layout leaked into a Go slice header: %s\n%s", forbidden, ir)
		}
	}
	for name, checks := range map[string][]string{
		"Native":        {"store i64 9", "load i64"},
		"GoNested":      {"store i64 9", "load i64"},
		"NativePointer": {"store i64", "load i64"},
		"GoPointer":     {"store i64", "load i64"},
	} {
		start := strings.Index(ir, "@hostaccess."+name+"(")
		if start < 0 {
			t.Fatalf("missing %s:\n%s", name, ir)
		}
		body := ir[start:]
		end := strings.Index(body, "\n}")
		if end < 0 {
			t.Fatal("missing function body")
		}
		body = body[:end]
		for _, check := range checks {
			if !strings.Contains(body, check) {
				t.Fatalf("%s missing %q:\n%s", name, check, body)
			}
		}
	}
}

func TestHostLayoutLegacySizeConstants(t *testing.T) {
	const src = `package legacysize
import "structs"
import "unsafe"
type H struct { _ structs.HostLayout; B byte }
//DIRECTIVE
type Control struct { B byte; Fn func(); Tail byte }
//DIRECTIVE
type Nested struct { H H; Fn func(); Tail byte }
//DIRECTIVE
type Direct struct { _ structs.HostLayout; B byte; Fn func(); Tail byte }
//DIRECTIVE
type Alias = struct { _ structs.HostLayout; B byte; Fn func(); Tail byte }
const control = int(unsafe.Sizeof(Control{}))
var _ [control - int(unsafe.Sizeof(Nested{}))]byte
var _ [int(unsafe.Sizeof(Nested{})) - control]byte
var _ [control - int(unsafe.Sizeof(Direct{}))]byte
var _ [int(unsafe.Sizeof(Direct{})) - control]byte
var _ [control - int(unsafe.Sizeof(Alias{}))]byte
var _ [int(unsafe.Sizeof(Alias{})) - control]byte
var _ [2*control - int(unsafe.Sizeof([2]Nested{}))]byte
var _ [int(unsafe.Sizeof([2]Nested{})) - 2*control]byte
func Size() uintptr { return unsafe.Sizeof(Nested{}) }
`
	for _, directive := range []string{"//llgo:cdecl", "//llgo:type C"} {
		for _, target := range []*llssa.Target{
			{GOOS: "linux", GOARCH: "amd64"},
			{GOOS: "wasip1", GOARCH: "wasm", WasmProfile: "w32"},
		} {
			t.Run(directive+"/"+target.GOARCH, func(t *testing.T) {
				compileWithRewritesTarget(t, strings.ReplaceAll(src, "//DIRECTIVE", directive), nil, target)
			})
		}
	}
}
