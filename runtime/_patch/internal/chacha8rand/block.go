// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSES/Go-BSD-3-Clause.txt
// file at this module root for license terms.

package chacha8rand

//llgo:skip block

// block is the chacha8rand block function. Every LLGo target already runs
// the generic entry (the selected arch assembly only tail-jumps to
// block_generic), so provide the body in Go and let the same skip rule drop
// the bodyless assembly-backed declaration from chacha8.go.
func block(seed *[4]uint64, blocks *[32]uint64, counter uint32) {
	block_generic(seed, blocks, counter)
}
