package main

import "encoding/json"

type count int8
type flags []bool

type record struct {
	Tick   count
	Mask   uint16
	Name   string
	Ptr    *count
	Seen   flags
	Array  [2]count
	Values map[string]count
}

func main() {
	for _, input := range []string{
		`{"TICK":5,"Name":"x\ud83d\ude03","Seen":[false],"Array":[8,9,10],"Values":{"a":7},"ignore":{"key":[]}}`,
		"{\"Name\":\"\x80\",\"Mask\":123}",
		`{"Tick":null,"Ptr":null,"Array":[3],"Values":{"a":null},"Seen":[]}`,
		`{"Tick":999,"Array":[true],"Name":"updated","Seen":[null,false]}`,
		`{"Tick":7,"Tick":8,"Seen":[null,false],"Ptr":25}`,
		`{"Mask":-1,"Values":[3],"Ptr":{},"Name":{}}`,
		`{"Tick":7,"Seen":[`,
		`{"Tick":8} {"Tick":7}`,
		`null`, `42`, ` `,
	} {
		exercise(input)
	}
	var nilDst *record
	println(json.Unmarshal([]byte(`{}`), nilDst).Error())
}

//go:noinline
func exercise(input string) {
	n := count(2)
	seen := [3]bool{true, true, true}
	dst := record{Tick: 2, Ptr: &n, Seen: seen[:1], Array: [2]count{2, 2}, Values: map[string]count{"a": 2, "b": 3}}
	err := json.Unmarshal([]byte(input), &dst)
	if err != nil {
		println(err.Error())
		switch e := err.(type) {
		case *json.UnmarshalTypeError:
			println(e.Value, e.Offset, e.Struct, e.Field)
		case *json.SyntaxError:
			println(e.Error(), e.Offset)
		}
	}
	if dst.Ptr != nil {
		println(*dst.Ptr)
	}
	println(dst.Tick, dst.Mask, dst.Name, len(dst.Seen), dst.Array[0], dst.Array[1], dst.Values["a"], dst.Values["b"])
	println(seen[0], seen[1], seen[2])
}
