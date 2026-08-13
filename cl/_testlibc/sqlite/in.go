// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/sqlite"
)

// CHECK-LABEL: define void @main.check(i32 %0){{.*}} {
// CHECK: %[[IS_ERROR:[0-9]+]] = icmp ne i32 %0, 0
// CHECK: br i1 %[[IS_ERROR]]
func check(err sqlite.Errno) {
	if err != sqlite.OK {
		// CHECK: %[[ERRSTR:[0-9]+]] = call ptr @sqlite3_errstr(i32 %0)
		// CHECK-NEXT: call i32 (ptr, ...) @printf(ptr @{{[0-9]+}}, i32 %0, ptr %[[ERRSTR]])
		c.Printf(c.Str("==> Error: (%d) %s\n"), err, err.Errstr())
		c.Exit(1)
	}
}

// CHECK-LABEL: define void @main.main(){{.*}} {
func main() {
	// CHECK: %[[OPEN:[0-9]+]] = call { ptr, i32 } @"github.com/goplus/lib/c/sqlite.OpenV2"(ptr @{{[0-9]+}}, i32 130, ptr null)
	// CHECK: %[[DB:[0-9]+]] = extractvalue { ptr, i32 } %[[OPEN]], 0
	// CHECK: %[[OPEN_ERR:[0-9]+]] = extractvalue { ptr, i32 } %[[OPEN]], 1
	// CHECK: call void @main.check(i32 %[[OPEN_ERR]])
	db, err := sqlite.OpenV2(c.Str(":memory:"), sqlite.OpenReadWrite|sqlite.OpenMemory, nil)
	check(err)

	// CHECK: call i32 @sqlite3_close(ptr %[[DB]])
	db.Close()
}
