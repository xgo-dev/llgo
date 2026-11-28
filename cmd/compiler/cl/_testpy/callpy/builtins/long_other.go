//go:build !windows

package builtins

// C long follows the target pointer width on the supported non-Windows hosts.
type cLong = int
