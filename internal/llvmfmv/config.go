//go:build !byollvm

package llvmfmv

// Use the same LLVM installation as github.com/xgo-dev/llvm. Releases using
// byollvm supply their headers and libraries through the existing CGO flags.

/*
#cgo darwin,amd64 CPPFLAGS: -I/usr/local/opt/llvm@22/include
#cgo darwin,arm64 CPPFLAGS: -I/opt/homebrew/opt/llvm@22/include
#cgo linux CPPFLAGS: -I/usr/include/llvm-22 -I/usr/lib/llvm-22/include -I/usr/lib64/llvm22/include
#cgo freebsd CPPFLAGS: -I/usr/local/llvm22/include
#cgo windows pkg-config: llvm-22
*/
import "C"
