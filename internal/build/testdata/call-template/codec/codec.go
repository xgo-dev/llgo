// Package codec exercises same-package candidates selected by an importer.
package codec

import "reflect"

//llgo:template fmtInt
//llgo:template fmtPair
func Stringify(a, b any) string {
	switch x := a.(type) {
	case int:
		if y, ok := b.(int); ok {
			return fmtPair(x, y)
		}
		return fmtInt(x, b)
	case string:
		return x
	}
	return "unknown"
}

func fmtInt(a int, b any) string {
	if b == nil {
		return "one integer"
	}
	return "integer and value"
}

func fmtPair(a, b int) string {
	return "two integers"
}

//llgo:template equalSlices
func Equal(a, b any) bool {
	x, y := reflect.ValueOf(a), reflect.ValueOf(b)
	if x.Kind() != reflect.Slice || x.Type() != y.Type() ||
		!x.Type().Elem().Comparable() || x.Len() != y.Len() {
		return false
	}
	for i := 0; i < x.Len(); i++ {
		if x.Index(i).Interface() != y.Index(i).Interface() {
			return false
		}
	}
	return true
}

func equalSlices[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

//llgo:template checkTrue if enabled == true
func Check(enabled bool, n any) int {
	if !enabled {
		return 0
	}
	if v, ok := n.(int); ok {
		return checkTrue(enabled, v)
	}
	return 0
}

func checkTrue(enabled bool, n int) int {
	return n
}
