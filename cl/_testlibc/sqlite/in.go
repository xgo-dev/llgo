// LITTEST
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/sqlite"
)

// CHECK: define void @"{{.*}}/cl/_testlibc/sqlite.check"
func check(err sqlite.Errno) {
	if err != sqlite.OK {
		// CHECK: %2 = call ptr @sqlite3_errstr(i32 %0)
		c.Printf(c.Str("==> Error: (%d) %s\n"), err, err.Errstr())
		c.Exit(1)
	}
}

// CHECK: define void @"{{.*}}/cl/_testlibc/sqlite.main"
func main() {
	// CHECK: %0 = call { ptr, i32 } @"github.com/goplus/lib/c/sqlite.OpenV2"(ptr @1, i32 130, ptr null)
	db, err := sqlite.OpenV2(c.Str(":memory:"), sqlite.OpenReadWrite|sqlite.OpenMemory, nil)
	check(err)

	// CHECK: %3 = call i32 @sqlite3_close(ptr %1)
	db.Close()
}
