package foo_test

import (
	"testing"

	"github.com/xgo-dev/llgo/cmd/cl/_testgo/runextest/foo"
)

func TestFoo(t *testing.T) {
	if foo.Foo() != 1 {
		t.Fatal("Foo() != 1")
	}
}
