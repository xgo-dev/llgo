package compile

import (
	"go/ast"
	"go/types"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgo-dev/llgo/internal/build/typesynth"
	"github.com/xgo-dev/llgo/internal/packages"
	gopackages "golang.org/x/tools/go/packages"
)

func TestBinaryCallEligibility(t *testing.T) {
	dir := t.TempDir()
	source := `package main
import "encoding/binary"

type record struct{ N uint16 }
type private struct{ n uint16 }
var buf []byte
var value record
var ptr *record
var order binary.ByteOrder
var boxed any = value
var len int
var array [8]byte
var pointer *[8]byte
var low int
type userOrder struct{ binary.ByteOrder }


func goodEncode() { binary.Encode(buf, binary.BigEndian, &value) }
func goodDecode() { binary.Decode(buf, binary.LittleEndian, new(record)) }
func goodPrivate() { binary.Encode(buf, binary.NativeEndian, private{}) }
func goodNilSize() { binary.Size(ptr) }
func goodFixedBounds() { binary.Encode(pointer[1:7], binary.BigEndian, value) }
func goodDynamicBounds() { binary.Encode(array[low:], binary.BigEndian, value) }
func goodSliceBounds() { binary.Encode(buf[1:7], binary.BigEndian, value) }
func noOrder() { binary.Encode(buf, order, value) }
func noBoxed() { binary.Encode(buf, binary.BigEndian, boxed) }
func noLocalOrder() { binary.Encode(buf, userOrder{binary.BigEndian}, value) }
func noNilDecode() { binary.Decode(buf, binary.BigEndian, ptr) }
func noValueDecode() { binary.Decode(buf, binary.BigEndian, value) }
func noPrivateDecode() { binary.Decode(buf, binary.BigEndian, new(private)) }
func noBasic() { binary.Encode(buf, binary.BigEndian, uint16(0)) }
func noMap() { binary.Size(map[int]record(nil)) }
func noIntField() { binary.Size(struct{ N int }{}) }
func noAppend() { binary.Append(buf, binary.BigEndian, value) }
func main() {}
`
	for name, contents := range map[string]string{"go.mod": "module example.com/codecs\ngo 1.23\n", "main.go": source} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	initial := loadInput(t, dir)
	initial.TypesSizes = types.SizesFor("gc", "386")
	if !shadowsGeneratedBuiltins(initial.Types) {
		t.Fatal("package-level len must suppress generated codecs")
	}
	for _, decl := range initial.Syntax[0].Decls {
		f, ok := decl.(*ast.FuncDecl)
		if !ok || len(f.Body.List) == 0 {
			continue
		}
		call := f.Body.List[0].(*ast.ExprStmt).X.(*ast.CallExpr)
		fn, _ := typesynth.Callee(initial.TypesInfo, call.Fun)
		plan, got := planBinaryCall(initial, call, strings.ToLower(fn.Name()), initial.Imports["encoding/binary"].Types, math.MaxInt32)
		if want := strings.HasPrefix(f.Name.Name, "good"); got != want {
			t.Errorf("%s: eligible = %v, want %v", f.Name, got, want)
		}
		if want := f.Name.Name == "goodFixedBounds"; plan.unchecked != want {
			t.Errorf("%s: unchecked = %v, want %v", f.Name, plan.unchecked, want)
		}
	}
}

func loadInput(t *testing.T, dir string) *packages.Package {
	t.Helper()
	cfg := gopackages.Config{Mode: gopackages.LoadAllSyntax, Dir: dir}
	initial, err := gopackages.Load(&cfg, ".")
	if err != nil || len(initial) != 1 {
		t.Fatalf("load %s: %v", dir, err)
	}
	if gopackages.PrintErrors(initial) != 0 {
		t.Fatalf("load %s: type errors", dir)
	}
	return initial[0]
}
