package gotest

import (
	"reflect"
	"testing"

	methods "github.com/goplus/llgo/test/go/kubelinkmethods"
	slicesa "github.com/goplus/llgo/test/go/kubelinkslicesa"
	slicesb "github.com/goplus/llgo/test/go/kubelinkslicesb"
)

type kubeLinkQueue interface {
	Add(any)
	Len() int
}

type kubeLinkPromoted interface {
	M()
	N() int
}

func TestKubeLinkGenericMethodTableSymbols(t *testing.T) {
	var q kubeLinkQueue = methods.NewQueue()
	q.Add("item")
	if got := q.Len(); got != 1 {
		t.Fatalf("Len() = %d, want 1", got)
	}

	var p kubeLinkPromoted = methods.NewOuter()
	p.M()
	if got := p.N(); got != 1 {
		t.Fatalf("N() = %d, want 1", got)
	}
}

func TestKubeLinkGenericInstanceClosureLinkOnce(t *testing.T) {
	got := append(slicesa.A(), slicesb.B()...)
	want := []string{"a", "aa", "b", "bb"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("combined slices = %v, want %v", got, want)
	}
}
