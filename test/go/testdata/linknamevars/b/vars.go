package b

import (
	"github.com/xgo-dev/llgo/test/go/testdata/linknamevars/a"
	_ "unsafe"
)

//go:linkname Zero llgo_test_shared_zero
var Zero int

//go:linkname Static llgo_test_shared_static
var Static = 41

//go:linkname Dynamic llgo_test_shared_dynamic
var Dynamic = initialize()

func initialize() int { a.Order = append(a.Order, 2); return a.Dynamic + 7 }

//go:linkname Closure llgo_test_shared_closure
var Closure = makeClosure(20)

func makeClosure(n int) func(int) int { return func(v int) int { n += v; return n } }

//go:linkname Slice llgo_test_shared_slice
var Slice = []int{2, 3, 5}

//go:linkname Text llgo_test_shared_string
var Text = "linked value"

//go:linkname Marked github.com/xgo-dev/llgo/test/go/testdata/linknamevars/a.Marked
var Marked int

//go:linkname Named audit.linkname_qualified_var
var Named = 23

//go:linkname Empty llgo_test_shared_empty
var Empty struct{}

//go:linkname Default github.com/xgo-dev/llgo/test/go/testdata/linknamevars/a.Default
var Default = 55
