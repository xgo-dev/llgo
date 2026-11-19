package main

import (
	"encoding/json"
	"strings"
)

type number uint64
type item struct {
	Label  string `json:"name"`
	Empty  []bool `json:"empty,omitempty"`
	Count  number `json:"count"`
	hidden int
}

type record struct {
	Items    []*item         `json:"items"`
	Map      map[string]int  `json:"map"`
	Array    [2]bool         `json:"pair"`
	NilMap   map[string]bool `json:"nilMap"`
	NoMore   []int           `json:"none"`
	Dropped  string          `json:"-"`
	Optional *int            `json:"opt,omitempty"`
	False    bool            `json:"false,omitempty"`
	Zero     int             `json:"zero,omitempty"`
	Blank    string          `json:"blank,omitempty"`
	Struct   struct{}        `json:"struct,omitempty"`
}

//go:noinline
func printRecord(v record) {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	const expected = `{"items":[{"name":"\u003c\u003e\u0026\u0001\u2028\u2029\ufffd�","count":18446744073709551615},null],"map":{"a":-3,"z":7},"pair":[true,false],"nilMap":null,"none":[],"struct":{}}`
	if string(b) != expected && string(b) != strings.Replace(expected, `\ufffd`, "�", 1) {
		panic(string(b))
	}
	println(string(b))
	println("typed JSON encoding ok")
}

func main() {
	printRecord(record{
		Items:   []*item{{Label: "<>&\x01\u2028\u2029\xff\ufffd", Count: ^number(0)}, nil},
		Map:     map[string]int{"z": 7, "a": -3},
		Array:   [2]bool{true},
		NoMore:  []int{},
		Dropped: "ignored",
	})
}
