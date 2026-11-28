package genericlocalcache

import (
	"reflect"
	"testing"
)

func localRuntimeType[T any]() reflect.Type {
	type local struct {
		value T
	}
	return reflect.TypeOf(local{})
}

func TestLocalRuntimeType(t *testing.T) {
	intType := localRuntimeType[int]()
	stringType := localRuntimeType[string]()
	if intType == stringType {
		t.Fatalf("generic local runtime types are identical: %v", intType)
	}
}
