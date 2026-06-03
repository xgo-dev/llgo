package gotest

import (
	"reflect"
	"strings"
	"testing"
)

func genericLocalRuntimeType[T any]() reflect.Type {
	type local struct {
		value T
	}
	return reflect.TypeOf(local{})
}

func TestGenericLocalRuntimeTypesIncludeOuterArgs(t *testing.T) {
	intType := genericLocalRuntimeType[int]()
	stringType := genericLocalRuntimeType[string]()
	sameIntType := genericLocalRuntimeType[int]()
	if intType != sameIntType {
		t.Fatalf("same local generic runtime type has different identities: %v != %v", intType, sameIntType)
	}
	if intType == stringType {
		t.Fatalf("local generic runtime types are identical: %v", intType)
	}
	intName := intType.String()
	stringName := stringType.String()
	if !strings.Contains(intName, "int") {
		t.Fatalf("local generic int type name does not include type argument: %q", intName)
	}
	if !strings.Contains(stringName, "string") {
		t.Fatalf("local generic string type name does not include type argument: %q", stringName)
	}
}
