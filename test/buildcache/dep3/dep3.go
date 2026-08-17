//go:build llgo

// Package dep3 provides utility functions that depend on dep1.
package dep3

import "github.com/xgo-dev/llgo/test/buildcache/dep1"

// Triple returns triple of the sum.
func Triple(a, b int) int {
	return dep1.Add(a, b) * 3
}

// Msg returns a greeting message.
func Msg() string {
	return "dep3:" + dep1.Msg()
}
