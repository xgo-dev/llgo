package basic

import (
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/coverage/basic/testdata/dep"
	"github.com/xgo-dev/llgo/cmd/compiler/build/testdata/coverutil"
)

func TestBranch(t *testing.T) {
	if dep.Value() != 7 {
		t.Fatal("wrong testdata dependency value")
	}
	coverutil.WaitForPeer(t)
	if Initial != 1 || Branch(true) != 1 {
		t.Fatal("wrong branch")
	}
	if testing.CoverMode() != "" && testing.Coverage() == 0 {
		t.Fatal("missing coverage counters")
	}
}
