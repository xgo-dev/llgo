package main

type T struct{}

func sink(v any) {
	_ = v
}

func main() {
	sink(T{})
}
