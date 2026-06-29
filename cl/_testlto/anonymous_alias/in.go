// LITTEST
package main

// CHECK-DAG: !"Virtual Function Elim"
// CHECK-DAG: call { ptr, i1 } @llvm.type.checked.load(ptr %{{[0-9]+}}, i32 0, metadata !"go.method.NewPoint:func(float64, float64) *struct{x float64; y float64}")
// CHECK-DAG: !"go.method.NewPoint:func(float64, float64) *struct{x float64; y float64}"
// CHECK-DAG: !"go.method.value.reflect"
// CHECK-DAG: !"go.method.type.reflect"
// CHECK-DAG: !vcall_visibility

type MyPoint = struct {
	x float64
	y float64
}

type IPoint = struct {
	x float64
	y float64
}

type S struct{}

type I interface {
	NewPoint(dx, dy float64) *IPoint
}

func (S) NewPoint(dx, dy float64) *MyPoint {
	p := &MyPoint{}
	p.x += dx
	p.y += dy
	return p
}

func main() {
	var s I = S{}
	pt := s.NewPoint(float64(3), float64(4))
	println(pt.x, pt.y)
}
