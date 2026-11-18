package record

type payload struct {
	value [2]int
}

type Key struct {
	Name string
	raw  payload
}

func New(name string, a, b int) Key {
	return Key{name, payload{[2]int{a, b}}}
}
