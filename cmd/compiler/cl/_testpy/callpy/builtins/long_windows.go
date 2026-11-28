//go:build windows

package builtins

// Windows uses a 32-bit C long on both 32- and 64-bit targets.
type cLong = int32
