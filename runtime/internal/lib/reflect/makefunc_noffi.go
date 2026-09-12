//go:build llgo_noffi

package reflect

func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	panic("reflect.MakeFunc is unavailable without libffi")
}
