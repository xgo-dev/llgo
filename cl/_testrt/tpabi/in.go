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

// The concrete value is boxed with its instantiated descriptor, then recovered
// by a descriptor comparison. The pointer value uses an itab and a code/env
// pair for the interface call.
// CHECK-LABEL: define void @main.main(){{.*}} {
// CHECK: %[[BOX:[0-9]+]] = call noalias dereferenceable(24) ptr @"{{.*}}/runtime/internal/runtime.AllocU"(i64 24)
// CHECK: store %"main.T[string,int]" %{{[0-9]+}}, ptr %[[BOX]], align 8
// CHECK: %[[EBOX:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"_llgo_main.T[string,int]", ptr undef }, ptr %[[BOX]], 1
// CHECK: %[[ETYPE:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.eface" %[[EBOX]], 0
// CHECK: icmp eq ptr %[[ETYPE]], @"_llgo_main.T[string,int]"
// CHECK: %[[OBJ:[0-9]+]] = call noalias dereferenceable(24) ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 24)
// CHECK: %[[ITAB:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.NewItab"(ptr {{.*}}, ptr @"*_llgo_main.T[string,int]")
// CHECK: %[[IFACE0:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" undef, ptr %[[ITAB]], 0
// CHECK: %[[IFACE:[0-9]+]] = insertvalue %"{{.*}}/runtime/internal/runtime.iface" %[[IFACE0]], ptr %[[OBJ]], 1
// CHECK: %[[DATA:[0-9]+]] = call ptr @"{{.*}}/runtime/internal/runtime.IfacePtrData"(%"{{.*}}/runtime/internal/runtime.iface" %[[IFACE]])
// CHECK: %[[VTAB:[0-9]+]] = extractvalue %"{{.*}}/runtime/internal/runtime.iface" %[[IFACE]], 0
// CHECK: %[[SLOT:[0-9]+]] = getelementptr ptr, ptr %[[VTAB]], i64 3
// CHECK: %[[METHOD:[0-9]+]] = load ptr, ptr %[[SLOT]], align 8
// CHECK: %[[METHOD0:[0-9]+]] = insertvalue { ptr, ptr } undef, ptr %[[METHOD]], 0
// CHECK: %[[METHODPAIR:[0-9]+]] = insertvalue { ptr, ptr } %[[METHOD0]], ptr %[[DATA]], 1
// CHECK: %[[METHODENV:[0-9]+]] = extractvalue { ptr, ptr } %[[METHODPAIR]], 1
// CHECK: %[[METHODCODE:[0-9]+]] = extractvalue { ptr, ptr } %[[METHODPAIR]], 0
// CHECK: call void %[[METHODCODE]](ptr %[[METHODENV]])
// CHECK: %[[ARRAY:[0-9]+]] = call noalias dereferenceable(32) ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 32)
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
