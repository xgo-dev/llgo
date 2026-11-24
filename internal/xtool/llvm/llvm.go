package llvm

import "github.com/xgo-dev/llgo/target"

// IsDarwinTarget reports whether a physical LLVM target uses Apple's ABI.
func IsDarwinTarget(triple string) bool {
	return target.IsDarwinTriple(triple)
}

// IsWindowsTarget recognizes native Windows and MinGW triples.
func IsWindowsTarget(triple string) bool {
	return target.IsWindowsTriple(triple)
}

// GetTargetTriple returns the default LLVM target triple for goos/goarch.
func GetTargetTriple(goos, goarch string) string {
	return target.Triple(goos, goarch)
}

// GetTargetTripleWithGOARM returns the LLVM target triple for goos/goarch/goarm.
func GetTargetTripleWithGOARM(goos, goarch, goarm string) string {
	return target.TripleWithGOARM(goos, goarch, goarm)
}
