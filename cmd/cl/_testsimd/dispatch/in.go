// LITTEST linux/amd64 windows/amd64 darwin/arm64 wasip1/wasm
// Scope: common
package main

import (
	"simd"
	"simd/archsimd"
)

// The scalar dispatcher chooses emulation, then the optional wide variant,
// then the common 128-bit fallback. The bridge types must match that choice.
// CHECK-LABEL: define void @main.add(
// CHECK: call i1 @simd.Emulated()
// CHECK: call void @"main.add@simd0"(
// AMD64: call i64 @simd.VectorBitSize()
// AMD64: icmp eq i64 {{.*}}, 256
// AMD64: call void @"main.add@simd256"(
// CHECK: call void @"main.add@simd128"(
// CHECK-LABEL: define <4 x i32> @main.add128(
// CHECK: add <4 x i32>
// CHECK: ret <4 x i32>
// AMD64-LABEL: define <8 x i32> @main.add256(
// AMD64: add <8 x i32>
// AMD64: ret <8 x i32>
// CHECK-LABEL: define void @"main.add@simd0"(
// CHECK: @"simd/internal/bridge.Uint32s.Add"(
// CHECK-LABEL: define void @"main.add@simd128"(
// CHECK: @"simd/internal/bridge.Uint32x4.Add"(
// AMD64-LABEL: define void @"main.add@simd256"(
// AMD64: @"simd/internal/bridge.Uint32x8.Add"(

// Forwarding must alias the caller's recover frame for every variant, while
// each body containing recover binds that alias to its own frame.
// CHECK-LABEL: define void @main.catch(
// CHECK-NOT: BindRecoverFrame
// CHECK: StartRecoverFrameAlias"(ptr @main.catch, ptr @"main.catch@simd0")
// CHECK-NEXT: call void @"main.catch@simd0"(
// CHECK-NEXT: {{.*}}EndRecoverFrameAlias"(
// AMD64: StartRecoverFrameAlias"(ptr @main.catch, ptr @"main.catch@simd256")
// AMD64-NEXT: call void @"main.catch@simd256"(
// AMD64-NEXT: {{.*}}EndRecoverFrameAlias"(
// CHECK: StartRecoverFrameAlias"(ptr @main.catch, ptr @"main.catch@simd128")
// CHECK-NEXT: call void @"main.catch@simd128"(
// CHECK-NEXT: {{.*}}EndRecoverFrameAlias"(
// CHECK-LABEL: define void @"main.catch@simd0"(
// CHECK: BindRecoverFrame"(ptr @"main.catch@simd0",
// CHECK: {{.*}}.Recover"(
// CHECK-LABEL: define void @"main.catch@simd128"(
// CHECK: BindRecoverFrame"(ptr @"main.catch@simd128",
// CHECK: {{.*}}.Recover"(
// AMD64-LABEL: define void @"main.catch@simd256"(
// AMD64: BindRecoverFrame"(ptr @"main.catch@simd256",
// AMD64: {{.*}}.Recover"(
// CHECK-LABEL: define void @main.init()
// CHECK: call {{i32|i64}} @main.width()
// CHECK-LABEL: define void @main.recoverThroughDispatcher(
// CHECK: StartRecoverFrame"(ptr @main.catch)
// CHECK: call void @main.catch(
// CHECK: EndRecoverFrame"(
// CHECK-LABEL: define {{i32|i64}} @main.width()
// CHECK: call {{i32|i64}} @"main.width@simd0"()
// AMD64: call i64 @"main.width@simd256"()
// CHECK: call {{i32|i64}} @"main.width@simd128"()
// CHECK-LABEL: define {{i32|i64}} @"main.width@simd0"()
// CHECK: @"simd/internal/bridge.Uint32s.Len"(
// CHECK-LABEL: define {{i32|i64}} @"main.width@simd128"()
// CHECK: @"simd/internal/bridge.Uint32x4.Len"(
// AMD64-LABEL: define i64 @"main.width@simd256"()
// AMD64: @"simd/internal/bridge.Uint32x8.Len"(

func width() int { return (simd.Uint32s{}).Len() }

func add(x, y []uint32) {
	simd.LoadUint32s(x).Add(simd.LoadUint32s(y)).Store(x)
}

func catch(out *any) {
	var v simd.Uint32s
	*out = recover()
	if v.Len() == 0 {
		panic("zero lanes")
	}
}

func recoverThroughDispatcher(out *any) {
	defer catch(out)
	panic("caught")
}

var initial = width()

func main() {
	var out any
	recoverThroughDispatcher(&out)
	x := make([]uint32, initial)
	add(x, x)
	println(initial, out, x[0])
}

func add128(x, y archsimd.Uint32x4) archsimd.Uint32x4 { return x.Add(y) }
