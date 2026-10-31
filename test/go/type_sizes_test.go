package gotest

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"testing"
	"unsafe"
)

type sizeFunc func(int) int

//llgo:cdecl
type sizeNativeFunc func(int) int
type sizeAlias = sizeFunc
type sizeDefined sizeFunc
type sizeEmpty struct{}
type sizeFuncRecord struct {
	B byte
	F [2]sizeFunc
	Z sizeEmpty
}
type sizeContainer[T any] struct {
	B byte
	V T
	Z [0]uint64
}

//llgo:packed
type sizePacked struct {
	B byte
	X uint64
	Y uint16
}

func (p sizePacked) Bump(n uint64) sizePacked { p.X += n; return p }

//llgo:pack 2
type sizePacked2 struct {
	B byte
	X uint64
	Y uint16
}

type sizePadding struct {
	//llgo:padding
	E sizeEmpty
	X byte
}

type sizeNoPadding struct {
	X uint64
	//llgo:no-padding
	E sizeEmpty
}

type sizePaddingIf[A, B any] struct {
	//llgo:padding-if
	A A
	//llgo:padding-if
	B B
}

//llgo:cdecl
type sizeNativeNested struct {
	A [2]struct {
		X byte
		E struct{}
	}
}
type sizeOrdinaryNested struct {
	A [2]struct {
		X byte
		E struct{}
	}
}

type sizePaddingMixed[A, B any] struct {
	First  A
	Middle B
	//llgo:padding-if
	Last A
}

type sizePaddingBase[T any] struct {
	sizeEmpty
	//llgo:padding-if
	Value T
}

// These directives are ordinary comments to the standard Go compiler. Both
// compilers must agree with their own types.Sizes model and physical storage.
const sizeCommonSource = `package p
type F func(int) int
//llgo:cdecl
type NativeF func(int) int
type Alias = F
type Defined F
type Empty struct{}
type FuncRecord struct { B byte; F [2]F; Z Empty }
type Container[T any] struct { B byte; V T; Z [0]uint64 }
type Generic = Container[[2]F]
//llgo:packed
type Packed struct { B byte; X uint64; Y uint16 }
//llgo:pack 2
type Packed2 struct { B byte; X uint64; Y uint16 }
type Padding struct {
 //llgo:padding
 E Empty
 X byte
}
type NoPadding struct {
 X uint64
 //llgo:no-padding
 E Empty
}
type PaddingIf[A, B any] struct {
 //llgo:padding-if
 A A
 //llgo:padding-if
 B B
}
type Same = PaddingIf[Empty,Empty]
type Ptr *byte
type Slice []F
type Array [3]F
type Interface interface{ M() }
type Map map[int]F
type Chan chan F
type Complex complex128
type Str string
`

func TestCompilerTypeSizes(t *testing.T) {
	// Each compiler-specific file supplies implementations of the same public
	// interface. All common checks compare that interface with real storage.
	for compiler, sizes := range compilerSizesForTest() {
		t.Run(compiler, func(t *testing.T) {
			testCompilerTypeSizes(t, sizes)
		})
	}
}

type sizeTestImports map[string]*types.Package

func (p sizeTestImports) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}
	if pkg := p[path]; pkg != nil {
		return pkg, nil
	}
	return nil, fmt.Errorf("missing import %q", path)
}

func TestCompilerImportedTypeSizes(t *testing.T) {
	for compiler, sizes := range compilerSizesForTest() {
		t.Run(compiler, func(t *testing.T) {
			imports := sizeTestImports{}
			check := func(path, source string) *types.Package {
				t.Helper()
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "sizes.go", source, parser.ParseComments)
				if err != nil {
					t.Fatal(err)
				}
				conf := types.Config{Sizes: sizes, Importer: imports}
				pkg, err := conf.Check(path, fset, []*ast.File{file}, nil)
				if err != nil {
					t.Fatal(err)
				}
				return pkg
			}
			imports["a"] = check("a", "package a\n//llgo:cdecl\ntype NativeA func(int) int")
			imports["b"] = check("b", "package b\n//llgo:cdecl\ntype NativeB func(int) int")
			pkg := check("copies", `package copies
import . "a"
import renamed "a"
import . "b"
import "unsafe"
type CopyA NativeA
type CopyB NativeB
var _ renamed.NativeA
const SizeA = unsafe.Sizeof(CopyA(nil))
const SizeB = unsafe.Sizeof(CopyB(nil))
`)
			for _, suffix := range []string{"A", "B"} {
				typ := pkg.Scope().Lookup("Copy" + suffix).Type()
				want := int64(unsafe.Sizeof(sizeNativeFunc(nil)))
				constantSize, ok := constant.Int64Val(pkg.Scope().Lookup("Size" + suffix).(*types.Const).Val())
				if got := sizes.Sizeof(typ); got != want || !ok || constantSize != want {
					t.Errorf("Copy%s: size=%d, constant=%d, want %d", suffix, got, constantSize, want)
				}
			}
		})
	}
}

func testCompilerTypeSizes(t *testing.T, sizes types.Sizes) {
	t.Helper()
	if sizes == nil {
		t.Fatal("nil types.Sizes")
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sizes.go", sizeCommonSource, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	conf := types.Config{Sizes: sizes}
	pkg, err := conf.Check("sizes", fset, []*ast.File{file}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		size, align uintptr
		offsets     []uintptr
	}{
		{"F", unsafe.Sizeof(sizeFunc(nil)), unsafe.Alignof(sizeFunc(nil)), nil},
		{"NativeF", unsafe.Sizeof(sizeNativeFunc(nil)), unsafe.Alignof(sizeNativeFunc(nil)), nil},
		{"Alias", unsafe.Sizeof(sizeAlias(nil)), unsafe.Alignof(sizeAlias(nil)), nil},
		{"Defined", unsafe.Sizeof(sizeDefined(nil)), unsafe.Alignof(sizeDefined(nil)), nil},
		{"FuncRecord", unsafe.Sizeof(sizeFuncRecord{}), unsafe.Alignof(sizeFuncRecord{}), []uintptr{0, unsafe.Offsetof(sizeFuncRecord{}.F), unsafe.Offsetof(sizeFuncRecord{}.Z)}},
		{"Generic", unsafe.Sizeof(sizeContainer[[2]sizeFunc]{}), unsafe.Alignof(sizeContainer[[2]sizeFunc]{}), []uintptr{0, unsafe.Offsetof(sizeContainer[[2]sizeFunc]{}.V), unsafe.Offsetof(sizeContainer[[2]sizeFunc]{}.Z)}},
		{"Packed", unsafe.Sizeof(sizePacked{}), unsafe.Alignof(sizePacked{}), []uintptr{0, unsafe.Offsetof(sizePacked{}.X), unsafe.Offsetof(sizePacked{}.Y)}},
		{"Packed2", unsafe.Sizeof(sizePacked2{}), unsafe.Alignof(sizePacked2{}), []uintptr{0, unsafe.Offsetof(sizePacked2{}.X), unsafe.Offsetof(sizePacked2{}.Y)}},
		{"Padding", unsafe.Sizeof(sizePadding{}), unsafe.Alignof(sizePadding{}), []uintptr{0, unsafe.Offsetof(sizePadding{}.X)}},
		{"NoPadding", unsafe.Sizeof(sizeNoPadding{}), unsafe.Alignof(sizeNoPadding{}), []uintptr{0, unsafe.Offsetof(sizeNoPadding{}.E)}},
		{"Same", unsafe.Sizeof(sizePaddingIf[sizeEmpty, sizeEmpty]{}), unsafe.Alignof(sizePaddingIf[sizeEmpty, sizeEmpty]{}), []uintptr{0, unsafe.Offsetof(sizePaddingIf[sizeEmpty, sizeEmpty]{}.B)}},
		{"Ptr", unsafe.Sizeof((*byte)(nil)), unsafe.Alignof((*byte)(nil)), nil},
		{"Slice", unsafe.Sizeof([]sizeFunc(nil)), unsafe.Alignof([]sizeFunc(nil)), nil},
		{"Array", unsafe.Sizeof([3]sizeFunc{}), unsafe.Alignof([3]sizeFunc{}), nil},
		{"Interface", unsafe.Sizeof((interface{ M() })(nil)), unsafe.Alignof((interface{ M() })(nil)), nil},
		{"Map", unsafe.Sizeof(map[int]sizeFunc(nil)), unsafe.Alignof(map[int]sizeFunc(nil)), nil},
		{"Chan", unsafe.Sizeof((chan sizeFunc)(nil)), unsafe.Alignof((chan sizeFunc)(nil)), nil},
		{"Complex", unsafe.Sizeof(complex128(0)), unsafe.Alignof(complex128(0)), nil},
		{"Str", unsafe.Sizeof(""), unsafe.Alignof(""), nil},
	} {
		typ := pkg.Scope().Lookup(tc.name).Type()
		if got := sizes.Sizeof(typ); got != int64(tc.size) {
			t.Errorf("%s size %d != unsafe %d", tc.name, got, tc.size)
		}
		if got := sizes.Alignof(typ); got != int64(tc.align) {
			t.Errorf("%s align %d != unsafe %d", tc.name, got, tc.align)
		}
		if tc.offsets != nil {
			st := typ.Underlying().(*types.Struct)
			fields := make([]*types.Var, st.NumFields())
			for i := range fields {
				fields[i] = st.Field(i)
			}
			for i, offset := range sizes.Offsetsof(fields) {
				if offset != int64(tc.offsets[i]) {
					t.Errorf("%s field %d offset %d != unsafe %d", tc.name, i, offset, tc.offsets[i])
				}
			}
		}
	}
}

func sizeGeneric[T any]() (uintptr, uintptr) { var v T; return unsafe.Sizeof(v), unsafe.Alignof(v) }

func TestCompilerLayoutStorage(t *testing.T) {
	if size, align := sizeGeneric[sizePacked](); size != unsafe.Sizeof(sizePacked{}) || align != unsafe.Alignof(sizePacked{}) {
		t.Fatal("generic packed layout differs from constant layout")
	}
	if size, align := sizeGeneric[sizeContainer[[2]sizeFunc]](); size != unsafe.Sizeof(sizeContainer[[2]sizeFunc]{}) || align != unsafe.Alignof(sizeContainer[[2]sizeFunc]{}) {
		t.Fatal("generic function-containing layout differs from constant layout")
	}
	packed := [2]sizePacked{{1, 2, 3}, {4, 5, 6}}
	if stride := uintptr(unsafe.Pointer(&packed[1])) - uintptr(unsafe.Pointer(&packed[0])); stride != unsafe.Sizeof(sizePacked{}) {
		t.Fatal("packed array stride")
	}
	copy := packed
	if copy != packed {
		t.Fatal("packed array copy/equality")
	}
	m := map[sizePacked]int{packed[0]: 7}
	if m[copy[0]] != 7 {
		t.Fatal("packed map keys")
	}
	value := reflect.ValueOf(&packed[0]).Elem()
	value.Field(1).SetUint(1234567)
	if packed[0].X != 1234567 || packed[0].B != 1 || packed[0].Y != 3 || packed[1] != copy[1] {
		t.Fatal("packed reflection store corrupted adjacent bytes")
	}
	for _, tc := range []struct {
		typ                 reflect.Type
		size, align, offset uintptr
	}{
		{reflect.TypeOf(sizePacked{}), unsafe.Sizeof(sizePacked{}), unsafe.Alignof(sizePacked{}), unsafe.Offsetof(sizePacked{}.X)},
		{reflect.TypeOf(sizeNoPadding{}), unsafe.Sizeof(sizeNoPadding{}), unsafe.Alignof(sizeNoPadding{}), unsafe.Offsetof(sizeNoPadding{}.E)},
		{reflect.TypeOf(sizeFuncRecord{}), unsafe.Sizeof(sizeFuncRecord{}), unsafe.Alignof(sizeFuncRecord{}), unsafe.Offsetof(sizeFuncRecord{}.F)},
	} {
		if tc.typ.NumField() < 2 || tc.typ.Size() != tc.size || uintptr(tc.typ.Align()) != tc.align || tc.typ.Field(1).Offset != tc.offset {
			t.Errorf("reflection differs from unsafe for %v", tc.typ)
		}
	}
	env := 17
	record := sizeFuncRecord{B: 1, F: [2]sizeFunc{func(x int) int { return x + env }, func(x int) int { return x * env }}}
	recordCopy := record
	if recordCopy.F[0](3) != 20 || recordCopy.F[1](3) != 51 {
		t.Fatal("function value copy lost closure environment")
	}
}

func TestCompilerLayoutReflectCalls(t *testing.T) {
	input := sizePacked{1, 2, 3}
	bias := uint64(11)
	closure := func(p sizePacked) sizePacked { p.X += bias; return p }
	call := reflect.ValueOf(closure)
	got := call.Call([]reflect.Value{reflect.ValueOf(input)})[0].Interface().(sizePacked)
	if got != (sizePacked{1, 13, 3}) {
		t.Fatalf("packed closure call = %v", got)
	}
	generated := reflect.MakeFunc(call.Type(), func(args []reflect.Value) []reflect.Value {
		p := args[0].Interface().(sizePacked)
		p.X *= 3
		return []reflect.Value{reflect.ValueOf(p)}
	})
	typed := generated.Interface().(func(sizePacked) sizePacked)
	if got := typed(input); got != (sizePacked{1, 6, 3}) {
		t.Fatalf("packed MakeFunc = %v", got)
	}
	if got := generated.Call([]reflect.Value{reflect.ValueOf(input)})[0].Interface().(sizePacked); got != (sizePacked{1, 6, 3}) {
		t.Fatalf("packed MakeFunc reflection = %v", got)
	}
	method := reflect.ValueOf(input).MethodByName("Bump")
	if got := method.Call([]reflect.Value{reflect.ValueOf(uint64(7))})[0].Interface().(sizePacked); got != (sizePacked{1, 9, 3}) {
		t.Fatalf("packed method = %v", got)
	}
	m, ok := reflect.TypeOf(input).MethodByName("Bump")
	if !ok {
		t.Fatal("missing packed method")
	}
	if got := m.Func.Call([]reflect.Value{reflect.ValueOf(input), reflect.ValueOf(uint64(7))})[0].Interface().(sizePacked); got != (sizePacked{1, 9, 3}) {
		t.Fatalf("packed method expression = %v", got)
	}
}

func TestCompilerLayoutConversions(t *testing.T) {
	type Plain struct {
		B byte
		X uint64
		Y uint16
	}
	p := sizePacked{1, 0x1234567890abcdef, 3}
	plain := Plain(p)
	if plain.B != p.B || plain.X != p.X || plain.Y != p.Y || sizePacked(plain) != p {
		t.Fatal("conversion between packed and natural layouts")
	}
	if sizePacked(sizePacked2(p)) != p {
		t.Fatal("conversion between packing alignments")
	}
	if got := reflect.ValueOf(p).Convert(reflect.TypeOf(Plain{})).Interface().(Plain); got != plain {
		t.Fatal("reflection packed conversion")
	}
	if got := reflect.ValueOf(plain).Convert(reflect.TypeOf(p)).Interface().(sizePacked); got != p {
		t.Fatal("reflection natural conversion")
	}
	native := sizeNativeNested{}
	native.A[0].X, native.A[1].X = 31, 37
	ordinary := sizeOrdinaryNested(native)
	if ordinary.A[0].X != 31 || ordinary.A[1].X != 37 || sizeNativeNested(ordinary) != native {
		t.Fatal("nested array layout conversion")
	}
	converted := reflect.ValueOf(native).Convert(reflect.TypeOf(ordinary)).Interface().(sizeOrdinaryNested)
	if converted != ordinary {
		t.Fatal("reflected nested array layout conversion")
	}
}

func TestCompilerConditionalPaddingConflicts(t *testing.T) {
	type OtherEmpty struct{}
	mixed := sizePaddingMixed[sizeEmpty, OtherEmpty]{}
	base := sizePaddingBase[sizeEmpty]{}
	// The compiler-specific files check the expected differences. Here the
	// public reflection interface and actual addresses must agree with unsafe.
	if reflect.TypeOf(mixed).Size() != unsafe.Sizeof(mixed) || reflect.TypeOf(base).Size() != unsafe.Sizeof(base) {
		t.Fatal("conditional padding reflection size")
	}
	wantOffset := unsafe.Offsetof(mixed.Last)
	if actual := uintptr(unsafe.Pointer(&mixed.Last)) - uintptr(unsafe.Pointer(&mixed)); actual != wantOffset {
		t.Fatal("non-adjacent field address")
	}
	if actual := uintptr(unsafe.Pointer(&base.Value)) - uintptr(unsafe.Pointer(&base)); actual != unsafe.Offsetof(base.Value) {
		t.Fatal("embedded base field address")
	}
}

func TestCompilerLayoutDynamicReflectCall(t *testing.T) {
	packed := reflect.TypeOf(sizePacked{})
	dynamic := reflect.StructOf([]reflect.StructField{{Name: "Packed", Type: packed, Tag: `layout:"dynamic"`}})
	signature := reflect.FuncOf([]reflect.Type{dynamic}, []reflect.Type{dynamic}, false)
	fn := reflect.MakeFunc(signature, func(in []reflect.Value) []reflect.Value { return in })
	value := reflect.New(dynamic).Elem()
	input := sizePacked{7, 0x1234567890abcdef, 9}
	value.Field(0).Set(reflect.ValueOf(input))
	got := fn.Call([]reflect.Value{value})[0].Field(0).Interface().(sizePacked)
	if got != input {
		t.Fatalf("dynamic packed callback = %v, want %v", got, input)
	}
}

//llgo:packed
type llgoGenericPacked[T ~uint32 | ~uint64] struct {
	B byte
	V T
}

func llgoMakeGenericPacked[T ~uint32 | ~uint64](v T) llgoGenericPacked[T] {
	return llgoGenericPacked[T]{B: 7, V: v}
}

func TestCompilerGenericPacked(t *testing.T) {
	x := llgoMakeGenericPacked(uint64(42))
	if x.B != 7 || x.V != 42 {
		t.Fatal(x)
	}
	if size, align := sizeGeneric[llgoGenericPacked[uint64]](); size != unsafe.Sizeof(x) || align != unsafe.Alignof(x) {
		t.Fatal("generic packed layout differs from constant layout")
	}
}

func TestCompilerFunctionReflectionSize(t *testing.T) {
	plain := reflect.TypeOf((func(int) int)(nil))
	for _, tc := range []struct {
		typ         reflect.Type
		size, align uintptr
	}{
		{plain, unsafe.Sizeof((func(int) int)(nil)), unsafe.Alignof((func(int) int)(nil))},
		{reflect.TypeOf(sizeFunc(nil)), unsafe.Sizeof(sizeFunc(nil)), unsafe.Alignof(sizeFunc(nil))},
		{reflect.TypeOf(sizeDefined(nil)), unsafe.Sizeof(sizeDefined(nil)), unsafe.Alignof(sizeDefined(nil))},
		{reflect.FuncOf([]reflect.Type{reflect.TypeOf(0)}, []reflect.Type{reflect.TypeOf(0)}, false), unsafe.Sizeof((func(int) int)(nil)), unsafe.Alignof((func(int) int)(nil))},
		{reflect.TypeOf(sizeFuncRecord{}).Field(1).Type.Elem(), unsafe.Sizeof(sizeFunc(nil)), unsafe.Alignof(sizeFunc(nil))},
	} {
		if tc.typ.Size() != tc.size || tc.typ.Align() != int(tc.align) || tc.typ.FieldAlign() != int(tc.align) {
			t.Errorf("%s: reflect size/align = %d/%d, unsafe = %d/%d", tc.typ, tc.typ.Size(), tc.typ.Align(), tc.size, tc.align)
		}
	}
}
