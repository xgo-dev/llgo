//go:build llgo_noffi

package reflect

func (v Value) call(op string, in []Value) (out []Value) {
	panic("reflect.Value.call is unavailable without libffi")
}
