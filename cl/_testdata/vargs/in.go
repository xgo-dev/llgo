// LITTEST
package main

import "github.com/goplus/lib/c"

// CHECK-LABEL: define void @"{{.*}}cl/_testdata/vargs.main"(){{.*}} {
// CHECK:   %{{.*}} = call ptr @"{{.*}}runtime/internal/runtime.AllocZ"(i64 48)
// CHECK:   store i64 1, ptr %{{.*}}, align 8
// CHECK:   %{{.*}} = insertvalue %"{{.*}}runtime/internal/runtime.eface" { ptr @_llgo_int, ptr undef }, ptr %{{.*}}, 1
// CHECK:   call void @"{{.*}}cl/_testdata/vargs.test"(%"{{.*}}runtime/internal/runtime.Slice" %{{.*}})
func main() {
	test(1, 2, 3)
}

// CHECK-LABEL: define void @"{{.*}}cl/_testdata/vargs.test"(%"{{.*}}runtime/internal/runtime.Slice" %0){{.*}} {
// CHECK:   %{{.*}} = getelementptr inbounds %"{{.*}}runtime/internal/runtime.eface", ptr %{{.*}}, i64 %{{.*}}
// CHECK:   %{{.*}} = load %"{{.*}}runtime/internal/runtime.eface", ptr %{{.*}}, align 8
// CHECK:   %{{.*}} = icmp eq ptr %{{.*}}, @_llgo_int
// CHECK:   %{{.*}} = load i64, ptr %{{.*}}, align 8
// CHECK:   %{{.*}} = call i32 (ptr, ...) @printf(ptr @{{.*}}, i64 %{{.*}})
// CHECK:   call void @"{{.*}}runtime/internal/runtime.Panic"(%"{{.*}}runtime/internal/runtime.eface" %{{.*}})
func test(a ...any) {
	for _, v := range a {
		c.Printf(c.Str("%d\n"), v.(int))
	}
}
