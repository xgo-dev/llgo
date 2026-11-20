//go:build llgo && baremetal

package chacha8rand

// The GOROOT assembly stub forwards to the same generic Go implementation.
//
//llgo:skip block
func block(seed *[4]uint64, blocks *[32]uint64, counter uint32) {
	block_generic(seed, blocks, counter)
}
