package main

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

func main() {
	s := `{"name":"math","items":[{"name":"sqrt","sig":"(x, /)"},{"name":"pi"}]}`
	data := unsafe.Slice(unsafe.StringData(s), len(s))
	var v any
	json.Unmarshal(data, &v)
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
