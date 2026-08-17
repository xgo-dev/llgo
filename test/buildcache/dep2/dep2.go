//go:build llgo

// Package dep2 provides utility functions that depend on dep1.
package dep2

import "github.com/xgo-dev/llgo/test/buildcache/dep1"

// Double returns double of the sum.
func Double(a, b int) int {
	return dep1.Add(a, b) * 2
}

// Msg returns a greeting message.
func Msg() string {
	return "dep2:" + dep1.Msg()
}
