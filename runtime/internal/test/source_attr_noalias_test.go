//go:build llgo

package test

import "testing"

//go:noinline
//llgo:attr param(p) noalias
func attributeNoAliasStore(p, q *int32) int32 {
	*p = 1
	*q = 2
	return *p
}

//go:noinline
//llgo:attr param(p) noalias
func attributeNoAliasRead(p, q *int32) int32 { return *p + *q }

func TestSourceAttrNoAlias(t *testing.T) {
	var p, q int32
	if attributeNoAliasStore(&p, &q) != 1 || p != 1 || q != 2 {
		t.Fatal("noalias store changed values")
	}
	p = 9
	if attributeNoAliasRead(&p, &p) != 18 {
		t.Fatal("noalias rejected shared unmodified memory")
	}
}
