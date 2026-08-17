package main

import "github.com/xgo-dev/llgo/xtool/env/llvm"

// LLVM is part of the llgo process environment. Prepare PATH before command
// dispatch so build requests and their workers only need an environment
// snapshot; they do not carry or reselect an LLVM installation.
func init() {
	llvm.SetupPath()
}
