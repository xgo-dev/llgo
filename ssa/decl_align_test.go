package ssa

import (
	"go/types"
	"testing"
)

func TestMemProfileShardsGlobalAlignment(t *testing.T) {
	Initialize(InitAll)
	prog := NewProgram(&Target{GOOS: "darwin", GOARCH: "arm64"})
	defer prog.Dispose()

	pkg := prog.NewPackage("runtime", PkgRuntime)
	// Synthetic type: this test only verifies that the name-based alignment
	// override fires for runtimeMemProfileShardsSym. Real memProfileShard
	// stride matching is verified by internal/build.TestMemProfileCounterAlignment.
	arrTy := types.NewPointer(types.NewArray(types.Typ[types.Uint64], 32))

	shards := pkg.NewVar(runtimeMemProfileShardsSym, arrTy, GoDecl)
	if got := shards.impl.Alignment(); got != runtimeMemProfileShardsAlign {
		t.Fatalf("memProfileShardsData alignment = %d, want %d", got, runtimeMemProfileShardsAlign)
	}

	other := pkg.NewVar(PkgRuntime+".otherGlobal", arrTy, GoDecl)
	if got := other.impl.Alignment(); got != 8 {
		t.Fatalf("otherGlobal alignment = %d, want 8", got)
	}
}
