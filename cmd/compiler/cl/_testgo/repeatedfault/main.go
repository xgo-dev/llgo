package main

// Keep this program free of imports: runtime's richer fault handler must not
// hide a broken fallback in the core runtime's startup path.
var pointer *struct{ value int }

//go:noinline
func fault() {
	defer func() {
		if recover() == nil {
			panic("nil field access did not panic")
		}
	}()
	println(pointer.value)
}

func main() {
	fault()
	fault()
	println("ok")
}
