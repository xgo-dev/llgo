//go:build !windows && !wasm

package syscall

type Errno uintptr

type Signal int
