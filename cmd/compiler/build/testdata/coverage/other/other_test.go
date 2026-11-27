package other_test

import (
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/coverage/other"
	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/coverutil"
)

func TestValue(t *testing.T) {
	coverutil.WaitForPeer(t)
	if other.Value() != 7 {
		t.Fatal("wrong value")
	}
}
