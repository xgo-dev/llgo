package hooks

import "reflect"

var names = []string{"BeforeCreate", "AfterFind"}

//go:noinline
func Invoke(value any) {
	v := reflect.ValueOf(value)
	for _, name := range names {
		m := v.MethodByName(name)
		if m.IsValid() {
			println(m.Call(nil)[0].String())
		}
	}
}
