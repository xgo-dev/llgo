//go:build baremetal || wasm

package runtime

const maxprocsLLGoFiles = ""

func c_maxprocs() int32 { return 1 }
