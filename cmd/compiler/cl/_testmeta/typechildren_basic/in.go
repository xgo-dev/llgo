package main

type Inner struct {
	S string
}

type Outer struct {
	I Inner
	N int
}

func use(any) {}

func main() {
	use(Outer{})
}
