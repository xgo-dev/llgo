package main

type value struct {
	n int
}

func (v *value) mutate() bool {
	v.n = 1
	return true
}

func result() (value, bool) {
	var v value
	return v, v.mutate()
}

func main() {
	v, ok := result()
	if !ok || v.n != 1 {
		panic("return value was loaded before mutation")
	}
	println("RETURN_ORDER_OK")
}
