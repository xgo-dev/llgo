//go:build !byollvm

package llvmintrinsic

// Use the same default LLVM installation as github.com/xgo-dev/llvm. With
// byollvm, the caller supplies CGO_CPPFLAGS and CGO_LDFLAGS instead.

// #cgo darwin,amd64 CPPFLAGS: -I/usr/local/opt/llvm@22/include
// #cgo darwin,arm64 CPPFLAGS: -I/opt/homebrew/opt/llvm@22/include
// #cgo linux CPPFLAGS: -I/usr/include/llvm-22 -I/usr/include/llvm-c-22
// #cgo freebsd CPPFLAGS: -I/usr/local/llvm22/include
// #cgo windows pkg-config: llvm-22
import "C"
