package ssa

import (
	"fmt"
	"strings"
	"testing"
)

func TestRuntimeSymbolRejectsUnregisteredBranch(t *testing.T) {
	if RuntimeSymbol("MapAccess2", "MapAccess1", "MapAccess2") != "MapAccess2" {
		t.Fatal("changed the selected symbol")
	}
	defer func() {
		if got := recover(); got == nil || !strings.Contains(fmt.Sprint(got), "MapAccessFuture") {
			t.Fatalf("new selector branch was not rejected: %v", got)
		}
	}()
	RuntimeSymbol("MapAccessFuture", "MapAccess1", "MapAccess2")
}
