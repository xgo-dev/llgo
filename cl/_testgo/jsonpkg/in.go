// Behavior regression fixture: golden output must stay identical on hosted
// and on the RV32 baremetal XIP flash target. The image only fits the flash
// (XIP) layout; SRAM and the 256 KiB ESP32-C3 IRAM cannot hold the pulled-in
// fmt dependency.
package main

import (
	"encoding/json"
	"strings"
)

type msg struct {
	ID   int      `json:"id"`
	Name string   `json:"name"`
	Tags []string `json:"tags,omitempty"`
}

func main() {
	m := msg{ID: 7, Name: "box", Tags: []string{"a", "b"}}
	b, err := json.Marshal(m)
	println("marshal", err == nil, string(b))
	var out msg
	err = json.Unmarshal([]byte(`{"id":9,"name":"cup","tags":["x"]}`), &out)
	println("unmarshal", err == nil, out.ID, out.Name, len(out.Tags), out.Tags[0])
	d := json.NewDecoder(strings.NewReader(`{"id": 12}`))
	var raw map[string]any
	err = d.Decode(&raw)
	println("decode", err == nil, len(raw))
	s, _ := json.Marshal([]int{1, 2, 3})
	println("ints", string(s))
	var m2 map[string]int
	json.Unmarshal([]byte(`{"k":5}`), &m2)
	println("mapint", m2["k"])
}
