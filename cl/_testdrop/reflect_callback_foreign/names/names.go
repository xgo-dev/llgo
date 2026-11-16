package names

import (
	"reflect"
	_ "unsafe"
)

// This package only reads its initializer. A whole-program proof must still
// account for the other package's linkname access to the same storage.
//
//go:linkname callbacks
var callbacks = []string{"Original"}

//go:noinline
func Invoke(value any) string {
	v := reflect.ValueOf(value)
	for _, name := range callbacks {
		return v.MethodByName(name).Call(nil)[0].String()
	}
	panic("missing callback")
}
