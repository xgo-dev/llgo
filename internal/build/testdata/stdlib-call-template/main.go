package main

import (
	"encoding/binary"
	"encoding/json"
)

type namedInt int

func (namedInt) MarshalJSON() ([]byte, error) { return []byte(`"named"`), nil }

func show(data []byte, err error) {
	if err != nil {
		panic(err)
	}
	println(string(data))
}

func main() {
	show(json.Marshal(true))
	show(json.Marshal(int8(-128)))
	show(json.Marshal(^uint64(0)))
	show(json.Marshal("<>&\u2028\u2029\x80\n"))
	show(json.Marshal(namedInt(42)))
	var boxed any = json.Number("1e3")
	show(json.Marshal(boxed))

	println(binary.Size(uint16(1)), binary.Size([]uint32{1, 2}), binary.Size([2]uint16{}))

	var decoded string
	if err := json.Unmarshal([]byte(`"decoded"`), &decoded); err != nil {
		panic(err)
	}
	println(decoded)
}
