package plan9asm

import (
	"path/filepath"
	"runtime"
	"testing"

	extplan9asm "github.com/xgo-dev/plan9asm"
)

// Exercise complete original SDK files through the compiler adapter on every
// host. Successful opcode/object probes do not establish aggregate ABIInternal
// entry, private-tail or hidden-closure transport for these public Go roots.
func TestSigsForStdlibInternalBytealgAMD64(t *testing.T) {
	pkg := loadStdlibInternalBytealgForTarget(t, "linux", "amd64")
	slice := extplan9asm.LLVMType("{ ptr, i64, i64 }")
	str := extplan9asm.LLVMType("{ ptr, i64 }")
	tests := []struct {
		file string
		sigs map[string]extplan9asm.FuncSig
	}{
		{"compare_amd64.s", map[string]extplan9asm.FuncSig{
			"internal/bytealg.Compare": {Args: []extplan9asm.LLVMType{slice, slice}, Ret: extplan9asm.I64},
			"runtime.cmpstring":        {Args: []extplan9asm.LLVMType{str, str}, Ret: extplan9asm.I64},
		}},
		{"count_amd64.s", map[string]extplan9asm.FuncSig{
			"internal/bytealg.Count":       {Args: []extplan9asm.LLVMType{slice, extplan9asm.I8}, Ret: extplan9asm.I64},
			"internal/bytealg.CountString": {Args: []extplan9asm.LLVMType{str, extplan9asm.I8}, Ret: extplan9asm.I64},
		}},
		{"index_amd64.s", map[string]extplan9asm.FuncSig{
			"internal/bytealg.Index":       {Args: []extplan9asm.LLVMType{slice, slice}, Ret: extplan9asm.I64},
			"internal/bytealg.IndexString": {Args: []extplan9asm.LLVMType{str, str}, Ret: extplan9asm.I64},
		}},
		{"indexbyte_amd64.s", map[string]extplan9asm.FuncSig{
			"internal/bytealg.IndexByte":       {Args: []extplan9asm.LLVMType{slice, extplan9asm.I8}, Ret: extplan9asm.I64},
			"internal/bytealg.IndexByteString": {Args: []extplan9asm.LLVMType{str, extplan9asm.I8}, Ret: extplan9asm.I64},
		}},
		{"equal_amd64.s", map[string]extplan9asm.FuncSig{
			"runtime.memequal":        {Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr, extplan9asm.I64}, Ret: extplan9asm.I1},
			"runtime.memequal_varlen": {Args: []extplan9asm.LLVMType{extplan9asm.Ptr, extplan9asm.Ptr}, Ret: extplan9asm.I1},
		}},
	}
	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			path := filepath.Join(runtime.GOROOT(), "src", "internal", "bytealg", test.file)
			requireBytealgOriginalFileContract(t, pkg, path, "linux", "amd64", test.sigs)
		})
	}
}
