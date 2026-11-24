package main

import "encoding/json"

type custom int

func (custom) MarshalJSON() ([]byte, error) { return []byte(`"custom"`), nil }

//go:noinline
func check(v any, want string) {
	b, err := json.Marshal(v)
	if err != nil || string(b) != want {
		panic("generic Marshal changed behavior")
	}
}

func main() {
	b, err := json.Marshal(struct{ N int }{3})
	if err != nil || string(b) != `{"N":3}` {
		panic("ordinary Marshal changed behavior")
	}
	check(custom(1), `"custom"`)
	check(json.Number("9007199254740993"), "9007199254740993")
	println("generic JSON encoding ok")
}
