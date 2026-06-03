// LITTEST
//go:build llgo
// +build llgo

package llgotag

// CHECK-LABEL: define void @"{{.*}}.Foo"(){{.*}} {
// CHECK-NEXT: _llgo_0:
// CHECK-NEXT:   ret void
// CHECK-NEXT: }
func Foo() {
}
