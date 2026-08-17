//go:build llgo

// Package main tests build cache functionality.
package main

import (
	"github.com/goplus/lib/c"

	"github.com/xgo-dev/llgo/test/buildcache/dep1"
	"github.com/xgo-dev/llgo/test/buildcache/dep2"
	"github.com/xgo-dev/llgo/test/buildcache/dep3"
)

func main() {
	verifyRecoverCache()

	c.Printf(c.Str("dep1.Add(1,2) = %d\n"), dep1.Add(1, 2))
	c.Printf(c.Str("dep2.Double(1,2) = %d\n"), dep2.Double(1, 2))
	c.Printf(c.Str("dep3.Triple(1,2) = %d\n"), dep3.Triple(1, 2))
	c.Printf(c.Str("Messages: %s, %s, %s\n"),
		c.AllocaCStr(dep1.Msg()),
		c.AllocaCStr(dep2.Msg()),
		c.AllocaCStr(dep3.Msg()))
}
