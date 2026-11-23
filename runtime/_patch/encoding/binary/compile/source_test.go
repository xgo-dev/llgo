package compile

import (
	"go/types"
	"testing"
)

func TestShortBufferAccessorContract(t *testing.T) {
	pkg := types.NewPackage("encoding/binary", "binary")
	if name := shortBufferHelper(pkg); name != "" {
		t.Fatalf("missing helper returned %q", name)
	}
	bad := types.NewSignatureType(nil, nil, nil, nil,
		types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.String])), false)
	pkg.Scope().Insert(types.NewFunc(0, pkg, "LLGoBinaryShortBuffer", bad))
	if name := shortBufferHelper(pkg); name != "" {
		t.Fatalf("incompatible helper returned %q", name)
	}
}
