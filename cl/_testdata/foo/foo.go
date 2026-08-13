// LITTEST
package foo

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testdata/foo.Bar"(){{.*}} {
// CHECK: [[BAR_STACK:%[0-9]+]] = alloca { i64 }
// CHECK: [[BAR_FIELD:%[0-9]+]] = getelementptr inbounds { i64 }, ptr [[BAR_STACK]], i32 0, i32 0
// CHECK-NEXT: store i64 1, ptr [[BAR_FIELD]]
// CHECK-NEXT: [[BAR_VALUE:%[0-9]+]] = load { i64 }, ptr [[BAR_STACK]]
// CHECK-NEXT: [[BAR_BOX:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: store { i64 } [[BAR_VALUE]], ptr [[BAR_BOX]]
// CHECK-NEXT: [[BAR_EFACE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_struct${{.*}}", ptr undef }, ptr [[BAR_BOX]], 1
// CHECK-NEXT: ret %"{{.*}}/runtime/internal/runtime.eface" [[BAR_EFACE]]

func Bar() any {
	return struct{ V int }{1}
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testdata/foo.F"(){{.*}} {
// CHECK: [[F_STACK:%[0-9]+]] = alloca { i64 }
// CHECK: [[F_FIELD:%[0-9]+]] = getelementptr inbounds { i64 }, ptr [[F_STACK]], i32 0, i32 0
// CHECK-NEXT: store i64 1, ptr [[F_FIELD]]
// CHECK-NEXT: [[F_VALUE:%[0-9]+]] = load { i64 }, ptr [[F_STACK]]
// CHECK-NEXT: [[F_BOX:%[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT: store { i64 } [[F_VALUE]], ptr [[F_BOX]]
// CHECK-NEXT: [[F_EFACE:%[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testdata/foo.struct${{.*}}", ptr undef }, ptr [[F_BOX]], 1
// CHECK-NEXT: ret %"{{.*}}/runtime/internal/runtime.eface" [[F_EFACE]]

func F() any {
	return struct{ v int }{1}
}

type Foo struct {
	pb *byte
	F  float32
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testdata/foo.Foo.Pb"(%"{{.*}}/cl/_testdata/foo.Foo" %0){{.*}} {
// CHECK: [[PB_STACK:%[0-9]+]] = alloca %"{{.*}}/cl/_testdata/foo.Foo"
// CHECK: store %"{{.*}}/cl/_testdata/foo.Foo" %0, ptr [[PB_STACK]]
// CHECK-NEXT: [[PB_FIELD:%[0-9]+]] = getelementptr inbounds %"{{.*}}/cl/_testdata/foo.Foo", ptr [[PB_STACK]], i32 0, i32 0
// CHECK-NEXT: [[PB_VALUE:%[0-9]+]] = load ptr, ptr [[PB_FIELD]]
// CHECK-NEXT: ret ptr [[PB_VALUE]]

func (v Foo) Pb() *byte {
	return v.pb
}

type Gamer interface {
	initGame()
	Load()
}

type Game struct {
}

func (g *Game) initGame() {
}

func (g *Game) Load() {
	println("load")
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testdata/foo.(*Foo).Pb"(ptr %0){{.*}} {
// CHECK: [[PB_NIL:%[0-9]+]] = icmp eq ptr %0, null
// CHECK-NEXT: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 [[PB_NIL]], %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @{{[0-9]+}}, i64 2 })
// CHECK-NEXT: [[PB_RECEIVER:%[0-9]+]] = load %"{{.*}}/cl/_testdata/foo.Foo", ptr %0
// CHECK-NEXT: [[PB_RESULT:%[0-9]+]] = call ptr @"{{.*}}/cl/_testdata/foo.Foo.Pb"(%"{{.*}}/cl/_testdata/foo.Foo" [[PB_RECEIVER]])
// CHECK-NEXT: ret ptr [[PB_RESULT]]
