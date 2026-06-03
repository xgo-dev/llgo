// LITTEST
package main

import "fmt"

var a = 0

func main() {
	defer func() {
		err := recover()
		fmt.Println(err)
	}()
	// CHECK: call ptr @"github.com/goplus/llgo/runtime/internal/runtime.MapAccess1"(ptr @"map[_llgo_int]_llgo_int", ptr null, ptr {{%.*}})
	m := [0]map[int]int{}[a][0]
	print(m)
}
