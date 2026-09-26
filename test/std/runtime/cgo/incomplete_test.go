//go:build cgo

package cgo_test

import (
	"reflect"
	"runtime/cgo"
	"testing"
)

// Incomplete is declared by runtime/cgo's import-C source and is intentionally
// absent when the selected Go build context has no cgo frontend.
func TestIncompleteTypeIdentity(t *testing.T) {
	typ := reflect.TypeOf(cgo.Incomplete{})
	if typ.Name() != "Incomplete" || typ.PkgPath() != "runtime/cgo" {
		t.Fatalf("unexpected incomplete C type marker: %v from %q", typ, typ.PkgPath())
	}
}
