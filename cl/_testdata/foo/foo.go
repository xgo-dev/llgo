// LITTEST
package foo

// CHECK: {{^}}@6 = private unnamed_addr constant [43 x i8] c"{{.*}}/cl/_testdata/foo.Foo", align 1{{$}}
// CHECK: {{^}}@7 = private unnamed_addr constant [2 x i8] c"Pb", align 1{{$}}
// CHECK: {{^}}@8 = private unnamed_addr constant [4 x i8] c"load", align 1{{$}}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testdata/foo.Bar"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = load { i64 }, ptr %0, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } %2, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_struct$K-dZ9QotZfVPz2a0YdRa9vmZUuDXPTqZOlMShKEDJtk", ptr undef }, ptr %3, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.eface" %4
// CHECK-NEXT: }

func Bar() any {
	return struct{ V int }{1}
}

// CHECK-LABEL: define %"{{.*}}/runtime/internal/runtime.eface" @"{{.*}}/cl/_testdata/foo.F"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = alloca { i64 }, align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %0, i8 0, i64 8, i1 false)
// CHECK-NEXT:   %1 = getelementptr inbounds { i64 }, ptr %0, i32 0, i32 0
// CHECK-NEXT:   store i64 1, ptr %1, align 8
// CHECK-NEXT:   %2 = load { i64 }, ptr %0, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 8)
// CHECK-NEXT:   store { i64 } %2, ptr %3, align 8
// CHECK-NEXT:   %4 = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"{{.*}}/cl/_testdata/foo.struct$MYpsoM99ZwFY087IpUOkIw1zjBA_sgFXVodmn1m-G88", ptr undef }, ptr %3, 1
// CHECK-NEXT:   ret %"{{.*}}/runtime/internal/runtime.eface" %4
// CHECK-NEXT: }

func F() any {
	return struct{ v int }{1}
}

type Foo struct {
	pb *byte
	F  float32
}

// CHECK-LABEL: define ptr @"{{.*}}/cl/_testdata/foo.Foo.Pb"(%"{{.*}}/cl/_testdata/foo.Foo" %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = alloca %"{{.*}}/cl/_testdata/foo.Foo", align 8
// CHECK-NEXT:   call void @llvm.memset.p0.i64(ptr %1, i8 0, i64 16, i1 false)
// CHECK-NEXT:   store %"{{.*}}/cl/_testdata/foo.Foo" %0, ptr %1, align 8
// CHECK-NEXT:   %2 = getelementptr inbounds %"{{.*}}/cl/_testdata/foo.Foo", ptr %1, i32 0, i32 0
// CHECK-NEXT:   %3 = load ptr, ptr %2, align 8
// CHECK-NEXT:   ret ptr %3
// CHECK-NEXT: }

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
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %1 = icmp eq ptr %0, null
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %1, %"{{.*}}/runtime/internal/runtime.String" { ptr @6, i64 43 }, %"{{.*}}/runtime/internal/runtime.String" { ptr @7, i64 2 })
// CHECK-NEXT:   %2 = load %"{{.*}}/cl/_testdata/foo.Foo", ptr %0, align 8
// CHECK-NEXT:   %3 = call ptr @"{{.*}}/cl/_testdata/foo.Foo.Pb"(%"{{.*}}/cl/_testdata/foo.Foo" %2)
// CHECK-NEXT:   ret ptr %3
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/foo.(*Game).Load"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintString"(%"{{.*}}/runtime/internal/runtime.String" { ptr @8, i64 4 })
// CHECK-NEXT:   call void @"{{.*}}/runtime/internal/runtime.PrintByte"(i8 10)
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/foo.(*Game).initGame"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define void @"{{.*}}/cl/_testdata/foo.init"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %0 = load i1, ptr @"{{.*}}/cl/_testdata/foo.init$guard", align 1
// CHECK-NEXT:   br i1 %0, label %_llgo_2, label %_llgo_1
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_1:                                          ; preds = %_llgo_0
// CHECK-NEXT:   store i1 true, ptr @"{{.*}}/cl/_testdata/foo.init$guard", align 1
// CHECK-NEXT:   br label %_llgo_2
// CHECK-EMPTY:
// CHECK-NEXT: _llgo_2:                                          ; preds = %_llgo_1, %_llgo_0
// CHECK-NEXT:   ret void
// CHECK-NEXT: }

// CHECK-LABEL: define linkonce i1 @"__llgo_stub.{{.*}}/runtime/internal/runtime.memequal64"(ptr %0, ptr %1, ptr %2){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   %3 = tail call i1 @"{{.*}}/runtime/internal/runtime.memequal64"(ptr %1, ptr %2)
// CHECK-NEXT:   ret i1 %3
// CHECK-NEXT: }
