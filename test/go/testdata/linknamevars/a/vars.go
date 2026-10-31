package a

import _ "unsafe"

//go:linkname Zero llgo_test_shared_zero
var Zero int

//go:linkname Static llgo_test_shared_static
var Static int

//go:linkname Dynamic llgo_test_shared_dynamic
var Dynamic = initialize()

var Order []int

func initialize() int { Order = append(Order, 1); return 11 }

//go:linkname Closure llgo_test_shared_closure
var Closure func(int) int

//go:linkname Slice llgo_test_shared_slice
var Slice []int

//go:linkname Text llgo_test_shared_string
var Text string
var Copied = Text

//go:linkname Marked
var Marked = 19

//go:linkname Named audit.linkname_qualified_var
var Named int

//go:linkname Empty llgo_test_shared_empty
var Empty struct{}

// This definition is deliberately unannotated.
var Default int
var DefaultCopied = Default

//go:linkname GoNamespace go:llgo_test_reserved_storage
var GoNamespace int32
