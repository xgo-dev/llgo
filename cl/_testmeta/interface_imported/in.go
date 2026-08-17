package main

import "github.com/xgo-dev/llgo/cl/_testmeta/interface_imported/api"

func use(r api.Reader) int {
	n, _ := r.Read(nil)
	return n
}

func main() {
	_ = use(api.Source{})
}
