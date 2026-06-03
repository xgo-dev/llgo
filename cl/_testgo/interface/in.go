// LITTEST
package main

import (
	"github.com/goplus/llgo/cl/_testdata/foo"
)

type Game1 struct {
	*foo.Game
}

type Game2 struct {
}

// CHECK-LABEL: define void @"{{.*}}interface.(*Game2).initGame"(ptr %0){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
func (p *Game2) initGame() {
}

// CHECK-LABEL: define void @"{{.*}}/cl/_testgo/interface.main"(){{.*}} {
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.AllocZ"(i64 8)
// CHECK: insertvalue %"{{.*}}/runtime/internal/runtime.eface" { ptr @"*_llgo_{{.*}}/cl/_testgo/interface.Game1"
// CHECK: call i1 @"{{.*}}/runtime/internal/runtime.Implements"
// CHECK: call ptr @"{{.*}}/runtime/internal/runtime.NewItab"
// CHECK: call void @"{{.*}}/runtime/internal/runtime.PrintIface"
// CHECK: ret void
func main() {
	var g1 any = &Game1{&foo.Game{}}

	var g2 any = &Game2{}

	v1, ok := g1.(foo.Gamer)

	println("OK", v1, ok)

	if ok {
		v1.Load()
	}

	v2, ok := g2.(foo.Gamer)

	println("FAIL", v2, ok)
}
