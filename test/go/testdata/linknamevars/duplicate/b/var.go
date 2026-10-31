package b

import _ "unsafe"

//go:linkname Value llgo_test_duplicate_data
var Value = 41
