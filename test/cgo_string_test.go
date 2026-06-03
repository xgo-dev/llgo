//go:build llgo
// +build llgo

package test

import (
	"testing"

	"github.com/goplus/lib/c"
)

func TestCGoStringNil(t *testing.T) {
	var p *c.Char
	if got := c.GoString(p); got != "" {
		t.Fatalf("GoString(nil) = %q, want empty string", got)
	}
	if got := c.GoString(p, 0); got != "" {
		t.Fatalf("GoStringN(nil, 0) = %q, want empty string", got)
	}
}
