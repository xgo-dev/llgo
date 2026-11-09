//go:build llgo && !baremetal && (!wasm || (js && llgo.wasm.reflect.libffi))

package gotest

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestExplicitAlignmentReflectionCall(t *testing.T) {
	defer func() {
		if v := recover(); v == nil || !strings.Contains(fmt.Sprint(v), "alignment above 8 bytes") {
			t.Fatalf("libffi diagnostic: %v", v)
		}
	}()
	reflect.ValueOf(explicitAlignedIdentity).Call([]reflect.Value{reflect.ValueOf(aligned64{})})
}
