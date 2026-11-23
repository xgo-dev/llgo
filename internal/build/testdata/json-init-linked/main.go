package main

import "encoding/json"

func main() {
	data, err := json.Marshal(struct{ N int }{N: 17})
	if err != nil {
		panic(err)
	}
	println(string(data), json.Number("3.5").String())
}
