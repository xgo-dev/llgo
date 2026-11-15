//go:build llgo && llgo_pclntab_external && wasm

package pcln

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "unsafe"
)

type runtimeFuncInfoRecord struct {
	symbolPkg  uint32
	symbolName uint32
	namePkg    uint32
	nameName   uint32
	fileRoot   uint32
	fileName   uint32
	line       uint32
}

const (
	runtimeFuncInfoLineWrapper = uint32(1 << 31)
	runtimeFuncInfoLineMask    = runtimeFuncInfoLineWrapper - 1
)

//go:linkname runtimeFuncInfoTable C.__llgo_funcinfo_table
var runtimeFuncInfoTable *runtimeFuncInfoRecord

//go:linkname runtimeFuncInfoStrings C.__llgo_funcinfo_strings
var runtimeFuncInfoStrings *c.Char

//go:linkname runtimeFuncInfoStringOffsets C.__llgo_funcinfo_string_offsets
var runtimeFuncInfoStringOffsets *uint32

//go:linkname runtimeFuncInfoStringCount C.__llgo_funcinfo_string_count
var runtimeFuncInfoStringCount uintptr

//go:linkname runtimeFuncInfoHash C.__llgo_funcinfo_hash
var runtimeFuncInfoHash *uint16

//go:linkname runtimeFuncInfoCount C.__llgo_funcinfo_count
var runtimeFuncInfoCount uintptr

//go:linkname runtimeFuncInfoHashMask C.__llgo_funcinfo_hash_mask
var runtimeFuncInfoHashMask uintptr

//go:linkname runtimeFuncInfoSymbolIndex C.__llgo_funcinfo_symbol_index
var runtimeFuncInfoSymbolIndex *runtimeFuncInfoSymbolIndexRecord

//go:linkname runtimeFuncInfoSymbolIndexCount C.__llgo_funcinfo_symbol_index_count
var runtimeFuncInfoSymbolIndexCount uintptr

type runtimePCLineRecord struct {
	id        uint64
	funcIndex uint32
	fileRoot  uint32
	fileName  uint32
	line      uint32
}

//go:linkname runtimePCLineTable C.__llgo_pcline_table
var runtimePCLineTable *runtimePCLineRecord

//go:linkname runtimePCLineCount C.__llgo_pcline_count
var runtimePCLineCount uintptr

type runtimeFuncInfoSymbolIndexRecord struct {
	symbolID  uint64
	funcIndex uint32
}
