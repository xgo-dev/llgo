//go:build llgo

package gotest

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"runtime"
	"testing"
	"unsafe"
)

func compilerSizesForTest() map[string]types.Sizes {
	return map[string]types.Sizes{
		"gc":   types.SizesFor("gc", runtime.GOARCH),
		"llgo": types.SizesFor("llgo", runtime.GOARCH),
	}
}

func TestLLGoTypeSizes(t *testing.T) {
	if unsafe.Sizeof(sizeFunc(nil)) != 2*unsafe.Sizeof(uintptr(0)) {
		t.Fatal("Go funcval must contain code and environment words")
	}
	if unsafe.Sizeof(sizeNativeFunc(nil)) != unsafe.Sizeof(llgoCFunc(nil)) {
		t.Fatal("native function must occupy one native pointer")
	}
	if unsafe.Sizeof(sizePacked{}) != 11 || unsafe.Alignof(sizePacked{}) != 1 || unsafe.Sizeof(sizePacked2{}) != 12 || unsafe.Alignof(sizePacked2{}) != 2 {
		t.Fatal("packed layout")
	}
	if unsafe.Sizeof(sizePadding{}) != 2 || unsafe.Sizeof(sizeNoPadding{}) != 8 || unsafe.Sizeof(sizePaddingIf[sizeEmpty, sizeEmpty]{}) != 2 {
		t.Fatal("padding layout")
	}
	x := llgoGenericPacked[uint64]{}
	if unsafe.Sizeof(x) != 9 || unsafe.Offsetof(x.V) != 1 {
		t.Fatal("generic packed layout")
	}
	type Plain struct {
		B byte
		X uint64
		Y uint16
	}
	if reflect.TypeOf((*sizePacked)(nil)).ConvertibleTo(reflect.TypeOf((*Plain)(nil))) {
		t.Fatal("pointer conversion between different layouts")
	}
	type OtherEmpty struct{}
	mixed, base := sizePaddingMixed[sizeEmpty, OtherEmpty]{}, sizePaddingBase[sizeEmpty]{}
	if unsafe.Sizeof(mixed) != 2 || unsafe.Offsetof(mixed.Last) != 1 || unsafe.Sizeof(base) != 2 || unsafe.Offsetof(base.Value) != 1 {
		t.Fatal("conditional padding conflicts")
	}
}

//llgo:cdecl
type llgoNativeAnonymous struct {
	S struct {
		X byte
		E struct{}
	}
}
type llgoGoAnonymous struct {
	S struct {
		X byte
		E struct{}
	}
}

//llgo:cdecl
type llgoNativeCallbackField struct{ F func(int32) int32 }

func TestLLGoNativeCallbackFieldIdentity(t *testing.T) {
	v := llgoNativeCallbackField{F: llgoNativeAbs}
	if unsafe.Sizeof(v.F) != unsafe.Sizeof(llgoAbsPointer(nil)) || llgoFunctionValueSize(v.F) != unsafe.Sizeof(v.F) {
		t.Fatal("native callback field has Go funcval storage")
	}
	if v.F(-17) != 17 {
		t.Fatal("native callback field call")
	}
	rv := reflect.ValueOf(&v).Elem().Field(0)
	if rv.Type().Size() != unsafe.Sizeof(v.F) {
		t.Fatalf("native reflected callback size = %d, want %d", rv.Type().Size(), unsafe.Sizeof(v.F))
	}
	if got := rv.Call([]reflect.Value{reflect.ValueOf(int32(-23))})[0].Int(); got != 23 {
		t.Fatalf("native reflected callback call = %d", got)
	}
	converted := rv.Convert(reflect.TypeOf((func(int32) int32)(nil))).Interface().(func(int32) int32)
	if converted(-29) != 29 {
		t.Fatal("reflected native-to-Go callback conversion")
	}
	f := (func(int32) int32)(v.F)
	if unsafe.Sizeof(f) != 2*unsafe.Sizeof(uintptr(0)) || f(-19) != 19 {
		t.Fatal("native callback to Go funcval conversion")
	}
}

type llgoLayoutBox[T any] struct {
	Value T
	Tail  byte
}

//go:noinline
func llgoBoxLayout[T any](v T) llgoLayoutBox[T] { return llgoLayoutBox[T]{v, 7} }

func TestLLGoAnonymousLayoutIdentity(t *testing.T) {
	n, g := llgoNativeAnonymous{}, llgoGoAnonymous{}
	if got := llgoFunctionValueSize(n.S); got != unsafe.Sizeof(n.S) {
		t.Fatalf("native generic size = %d, want %d", got, unsafe.Sizeof(n.S))
	}
	if got := llgoFunctionValueSize(g.S); got != unsafe.Sizeof(g.S) {
		t.Fatalf("Go generic size = %d, want %d", got, unsafe.Sizeof(g.S))
	}
	nb, gb := llgoBoxLayout(n.S), llgoBoxLayout(g.S)
	if unsafe.Sizeof(nb) != 2 || unsafe.Sizeof(gb) != 3 {
		t.Fatalf("generic box sizes = %d, %d", unsafe.Sizeof(nb), unsafe.Sizeof(gb))
	}
	if got := uintptr(unsafe.Pointer(&nb.Tail)) - uintptr(unsafe.Pointer(&nb)); got != 1 {
		t.Fatalf("native box offset = %d", got)
	}
	if got := uintptr(unsafe.Pointer(&gb.Tail)) - uintptr(unsafe.Pointer(&gb)); got != 2 {
		t.Fatalf("Go box offset = %d", got)
	}
	if reflect.TypeOf(n.S) == reflect.TypeOf(g.S) {
		t.Fatal("reflection merged different layouts")
	}
	if tag := reflect.TypeOf(n.S).Field(0).Tag; tag != "" {
		t.Fatalf("compiler metadata leaked in reflection tag: %q", tag)
	}
}

type llgoGoFunc func()

//llgo:cdecl
type llgoNativeGoFields struct {
	F    llgoGoFunc
	Tail byte
}

func TestLLGoNativePreservesDeclaredFieldTypes(t *testing.T) {
	calls := 0
	v := llgoNativeGoFields{F: func() { calls++ }, Tail: 7}
	v.F()
	if calls != 1 || unsafe.Sizeof(v.F) != 2*unsafe.Sizeof(uintptr(0)) || llgoFunctionValueSize(v.F) != unsafe.Sizeof(v.F) {
		t.Fatal("native parent changed a named Go function field")
	}
	if off := uintptr(unsafe.Pointer(&v.Tail)) - uintptr(unsafe.Pointer(&v)); off != unsafe.Offsetof(v.Tail) || off != unsafe.Sizeof(v.F) {
		t.Fatalf("named Go function field offset = %d", off)
	}
	if v.Tail != 7 {
		t.Fatal("named Go function field overwrote following field")
	}
}

//llgo:cdecl
type llgoCFunc func()

//go:linkname llgoNativeAbs C.abs
func llgoNativeAbs(int32) int32

//llgo:cdecl
type llgoAbsPointer func(int32) int32

type llgoFunctionBox[T any] struct {
	F    T
	Tail uint64
}

func llgoBoxFunction[T any](f T) llgoFunctionBox[T] { return llgoFunctionBox[T]{f, 17} }
func llgoFunctionValueSize[T any](f T) uintptr      { return unsafe.Sizeof(f) }
func llgoOrdinaryFunction(x int32) int32            { return x + 19 }

// A linkname changes a declaration's call boundary, not the storage of an
// ordinary Go funcval. Marking its structural Signature as native corrupts
// otherwise identical generic instances created for ordinary Go functions.
func TestLLGoLinkedFunctionValues(t *testing.T) {
	f := llgoNativeAbs
	linked := llgoBoxFunction(f)
	ordinary := llgoBoxFunction(llgoOrdinaryFunction)
	native := llgoAbsPointer(llgoNativeAbs)
	if unsafe.Sizeof(f) != 2*unsafe.Sizeof(uintptr(0)) || llgoFunctionValueSize(f) != unsafe.Sizeof(f) {
		t.Fatal("linked declaration became a one-word Go funcval")
	}
	if unsafe.Sizeof(linked) != unsafe.Sizeof(ordinary) || unsafe.Offsetof(linked.Tail) != unsafe.Offsetof(ordinary.Tail) {
		t.Fatal("linkage changed structural generic type layout")
	}
	if actual := uintptr(unsafe.Pointer(&linked.Tail)) - uintptr(unsafe.Pointer(&linked.F)); actual != unsafe.Offsetof(linked.Tail) {
		t.Fatalf("linked generic offset = %d, constant = %d", actual, unsafe.Offsetof(linked.Tail))
	}
	if actual := uintptr(unsafe.Pointer(&ordinary.Tail)) - uintptr(unsafe.Pointer(&ordinary.F)); actual != unsafe.Offsetof(ordinary.Tail) {
		t.Fatalf("ordinary generic offset = %d, constant = %d", actual, unsafe.Offsetof(ordinary.Tail))
	}
	if llgoFunctionValueSize(native) != unsafe.Sizeof(native) || unsafe.Sizeof(native) != unsafe.Sizeof(llgoCFunc(nil)) {
		t.Fatal("explicit native function pointer layout")
	}
	if ordinary.F(0) != 19 || linked.F(-7) != 7 || native(-9) != 9 || linked.Tail != 17 {
		t.Fatal("function box contents")
	}
}

//llgo:packed
type llgoPacked struct {
	B byte
	X uint32
	Z byte
}

//llgo:pack 2
type llgoPack2 struct {
	B byte
	X uint32
	Y uint16
}

func TestLLGoSizesCompilerNames(t *testing.T) {
	if runtime.Compiler != "llgo" {
		t.Fatalf("Compiler = %q", runtime.Compiler)
	}
	for compiler, sizes := range compilerSizesForTest() {
		if sizes != types.SizesFor(compiler, runtime.GOARCH) {
			t.Fatalf("%s factory identity changes across calls", compiler)
		}
	}
	if types.SizesFor("llgo", "unknown") != nil || types.SizesFor("unknown", runtime.GOARCH) != nil {
		t.Fatal("unsupported target must return nil")
	}
}

func TestLLGoSizesDefaultAndNativeFunction(t *testing.T) {
	const source = `package p
import "unsafe"
//go:linkname cfunc C.abs
func cfunc(int32) int32
//go:linkname callback C.callback
var callback func(int32) int32
type F func(int32) int32
const C = unsafe.Sizeof(cfunc)
const V = unsafe.Sizeof(callback)
const G = unsafe.Sizeof(F(nil))
const A = unsafe.Alignof(cfunc)
`
	// The Unsafe package can be imported directly; no host export data is needed.
	for _, compiler := range []string{"", "gc", "llgo"} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "native.go", source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		conf := types.Config{Importer: llgoUnsafeImporter{}}
		if compiler != "" {
			conf.Sizes = types.SizesFor(compiler, runtime.GOARCH)
		}
		pkg, err := conf.Check("native", fset, []*ast.File{file}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for name, want := range map[string]uint64{"C": uint64(unsafe.Sizeof(llgoNativeAbs)), "V": uint64(unsafe.Sizeof(llgoCFunc(nil))), "G": uint64(unsafe.Sizeof((func() int32)(nil))), "A": uint64(unsafe.Alignof(llgoNativeAbs))} {
			got, ok := constant.Uint64Val(pkg.Scope().Lookup(name).(*types.Const).Val())
			if !ok || got != want {
				t.Errorf("compiler %q constant %s = %d, want %d", compiler, name, got, want)
			}
		}
	}
}

func TestLLGoPackedManagedReferencesRejected(t *testing.T) {
	for _, callback := range []bool{false, true} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "packed.go", "package p\n//llgo:packed\ntype T struct { B byte; P *byte }", parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		conf := types.Config{}
		called := false
		if callback {
			conf.Error = func(error) { called = true }
		}
		_, err = conf.Check("p", fset, []*ast.File{file}, nil)
		if err == nil || callback && !called {
			t.Fatalf("missing packed GC diagnostic: %v, callback=%v", err, called)
		}
	}
}

type llgoUnsafeImporter struct{}

func (llgoUnsafeImporter) Import(string) (*types.Package, error) { return types.Unsafe, nil }

var llgoPackedGlobal = llgoPacked{0x12, 0x12345678, 0x34}

//go:noinline
func llgoStoreUint32(p *uint32, v uint32) { *p = v }

func TestLLGoLayoutStorage(t *testing.T) {
	if unsafe.Sizeof(llgoPacked{}) != 6 || unsafe.Alignof(llgoPacked{}) != 1 || unsafe.Offsetof(llgoPacked{}.X) != 1 {
		t.Fatal("packed layout")
	}
	if unsafe.Sizeof(llgoPack2{}) != 8 || unsafe.Alignof(llgoPack2{}) != 2 {
		t.Fatal("pack 2 layout")
	}
	a := [2]llgoPacked{llgoPackedGlobal, {0x56, 0x87654321, 0x78}}
	llgoStoreUint32(&a[0].X, 0xaabbccdd)
	if a[0].B != 0x12 || a[0].Z != 0x34 || a[0].X != 0xaabbccdd || a[1].X != 0x87654321 {
		t.Fatalf("packed array corrupted: %v", a)
	}
	copy := a
	if copy != a {
		t.Fatal("packed copy/equality")
	}
	m := map[llgoPacked]int{a[0]: 7}
	if m[copy[0]] != 7 {
		t.Fatal("packed map key")
	}
	rt := reflect.TypeOf(a[0])
	if rt.Size() != 6 || rt.Align() != 1 || rt.NumField() != 3 || rt.Field(1).Offset != 1 {
		t.Fatalf("packed reflection: %v size %d align %d offset %d", rt, rt.Size(), rt.Align(), rt.Field(1).Offset)
	}
	rv := reflect.ValueOf(&a[0]).Elem()
	rv.Field(1).SetUint(1234)
	if a[0].X != 1234 || a[0].Z != 0x34 {
		t.Fatal("packed reflection store")
	}

}

func TestLLGoPointerLayoutConversions(t *testing.T) {
	for _, tc := range []struct {
		expression string
		allowed    bool
	}{
		{"(*Plain)(p)", false},
		{"(*Plain)(unsafe.Pointer(p))", true},
		{"Plain(*p)", true},
		{"(*Same)(p)", true},
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "pointer.go", `package p
import "unsafe"
var _ unsafe.Pointer
//llgo:packed
type Packed struct { B byte; X uint64 }
type Plain struct { B byte; X uint64 }
//llgo:packed
type Same struct { B byte; X uint64 }
func convert(p *Packed) { _ = `+tc.expression+` }
`, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		conf := types.Config{Importer: llgoUnsafeImporter{}}
		_, err = conf.Check("p", fset, []*ast.File{file}, nil)
		if (err == nil) != tc.allowed {
			t.Errorf("%s: allowed=%v, error=%v", tc.expression, tc.allowed, err)
		}
	}
}

func TestLLGoLocalPackedInstantiationRejected(t *testing.T) {
	for _, tc := range []struct {
		source  string
		allowed bool
	}{
		{"func local() { _ = Packed[*byte]{} }", false},
		{"func makePacked[T any](v T) Packed[T] { return Packed[T]{V:v} }; func local() { _ = makePacked(uint64(1)) }", true},
	} {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "local.go", `package p
//llgo:packed
type Packed[T any] struct { B byte; V T }
`+tc.source, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		conf := types.Config{}
		_, err = conf.Check("p", fset, []*ast.File{file}, nil)
		if (err == nil) != tc.allowed {
			t.Fatalf("allowed=%v, error=%v", tc.allowed, err)
		}
	}
}

func TestLLGoNativeCallbackConversionSnapshot(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		t.Skip("native MakeFunc requires a libffi trampoline")
	}
	captured := int32(7)
	fn := func(v int32) int32 { return v + captured }
	source := reflect.ValueOf(&fn).Elem()
	converted := source.Convert(reflect.TypeOf(llgoAbsPointer(nil))).Interface().(llgoAbsPointer)
	fn = func(v int32) int32 { return v + 100 }
	if got := converted(35); got != 42 {
		t.Fatalf("converted callback = %d, want 42", got)
	}
}

func TestLLGoNativeCallbackAssignment(t *testing.T) {
	if runtime.GOARCH == "wasm" {
		t.Skip("native MakeFunc requires a libffi trampoline")
	}
	captured := int32(7)
	fn := func(v int32) int32 { return v + captured }
	var native llgoAbsPointer
	reflect.ValueOf(&native).Elem().Set(reflect.ValueOf(&fn).Elem())
	fn = func(v int32) int32 { return v + 100 }
	if got := native(35); got != 42 {
		t.Fatalf("assigned callback = %d, want 42", got)
	}
	reflect.ValueOf(&fn).Elem().Set(reflect.ValueOf(native))
	if got := fn(35); got != 42 {
		t.Fatalf("assigned Go callback = %d, want 42", got)
	}
	fn = nil
	reflect.ValueOf(&native).Elem().Set(reflect.ValueOf(fn))
	if native != nil {
		t.Fatal("nil Go callback assignment became non-nil")
	}
	reflect.ValueOf(&fn).Elem().Set(reflect.ValueOf(native))
	if fn != nil {
		t.Fatal("nil native callback assignment became non-nil")
	}
	for _, pair := range [][2]reflect.Value{
		{reflect.ValueOf(fn), reflect.ValueOf(native)},
		{reflect.ValueOf(native), reflect.ValueOf(fn)},
	} {
		if !pair[0].Convert(pair[1].Type()).IsNil() {
			t.Fatal("nil callback conversion became non-nil")
		}
	}
}
