//go:build !llgo
// +build !llgo

package cl

import (
	"strings"
	"testing"
)

func TestZeroSizedFieldDerefEmitsBaseNilGuard(t *testing.T) {
	const src = `package zeroderef
type T struct {
	n int
	z struct{}
}
func Eq(p, q *T) bool {
	return p.z == q.z
}
`
	ir := compileWithRewrites(t, src, nil)
	if got := strings.Count(ir, "AssertNilDeref"); got < 4 {
		t.Fatalf("zero-sized field comparison should guard field bases and loads, got %d guards:\n%s", got, ir)
	}
}
