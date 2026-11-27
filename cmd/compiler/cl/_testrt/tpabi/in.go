// LITTEST
// Scope: common
package main

type integer interface {
	~int | ~uint | ~uintptr | ~int32 | ~uint32 | ~int64 | ~uint64
}

// llgo:link advance llgo.advance
func advance[PtrT any, I integer](ptr PtrT, offset I) PtrT { return ptr }

// Generic instantiation keeps the concrete type in interface metadata and
// preserves both value and pointer method ABIs. llgo.advance must lower to the
// same array GEP for its C helper and linked method spellings.

type T[M, N any] struct {
	m M
	n N
}

func (t *T[M, N]) Demo() {
	println(t.m, t.n)
}

func (t T[M, N]) Info() {
	println(t.m, t.n)
}

type I interface {
	Demo()
}

type K[N any] [4]N

//llgo:link (*K).Advance llgo.advance
func (t *K[N]) Advance(n int) *K[N] {
	return nil
}

func main() {
	var a any = T[string, int]{"a", 1}
	println(a.(T[string, int]).m)
	var i I = &T[string, int]{"hello", 100}
	i.Demo()

	k := &K[int]{1, 2, 3, 4}
	println(advance(k, 1))
	println(k.Advance(1))
}

// The concrete value is boxed with its instantiated descriptor and recovered
// directly when the dynamic type is statically known. The pointer value uses
// an itab and devirtualizes the known Demo receiver call.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %{{[0-9]+}} = alloca %"main.T[string,int]", align 8
// CHECK: store %"main.T[string,int]" %{{[0-9]+}}, ptr %[[BOX:[0-9]+]], align 8
// CHECK: %[[EBOX:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_main.T[string,int]", ptr undef }, ptr %[[BOX]], 1
// CHECK: %[[EDATA:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %[[EBOX]], 1
// CHECK: load %"main.T[string,int]", ptr %[[EDATA]]
// CHECK: %[[IFACE:[0-9]+]] = insertvalue %"{{.*}}iface" { ptr @"_llgo_itab${{.*}}", ptr undef }, ptr %[[OBJ:[0-9]+]], 1
// CHECK: call void @"main.(*T[string,int]).Demo"(ptr %[[OBJ]])
// CHECK: %[[ARRAY:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.AllocAtomicU"(i64 32)
// CHECK-NEXT: call void @llvm.memset.p0.i64(ptr %[[ARRAY]], i8 0, i64 32, i1 false)
// CHECK: getelementptr [4 x i64], ptr %[[ARRAY]], i64 1
// CHECK: getelementptr [4 x i64], ptr %[[ARRAY]], i64 1

// Value, pointer, and generated wrapper methods retain distinct receiver ABIs.
// CHECK-LABEL: define linkonce void @"main.T[string,int].Info"(
// CHECK-SAME: %"main.T[string,int]" %[[VALUE:[0-9]+]]){{.*}} {
// CHECK: store %"main.T[string,int]" %[[VALUE]], ptr %{{[0-9]+}}, align 8
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"

// CHECK-LABEL: define linkonce void @"main.(*T[string,int]).Demo"(
// CHECK-SAME: ptr %[[PTR:[0-9]+]]){{.*}} {
// CHECK: getelementptr inbounds nuw %"main.T[string,int]", ptr %[[PTR]], i32 0, i32 0
// CHECK: getelementptr inbounds nuw %"main.T[string,int]", ptr %[[PTR]], i32 0, i32 1
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintString"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintInt"

// CHECK-LABEL: define linkonce void @"main.(*T[string,int]).Info"(
// CHECK-SAME: ptr %[[WRAPPED:[0-9]+]]){{.*}} {
// CHECK: %[[ISNIL:[0-9]+]] = icmp eq ptr %[[WRAPPED]], null
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PanicWrapNilPointer"(i1 %[[ISNIL]], {{.*}})
// CHECK: %[[LOADED:[0-9]+]] = load %"main.T[string,int]", ptr %[[WRAPPED]], align 8
// CHECK: call void @"main.T[string,int].Info"(%"main.T[string,int]" %[[LOADED]])
