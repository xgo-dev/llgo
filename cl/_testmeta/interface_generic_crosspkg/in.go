package main

import (
	"github.com/xgo-dev/llgo/cl/_testmeta/interface_generic_crosspkg/api"
	"github.com/xgo-dev/llgo/cl/_testmeta/interface_generic_crosspkg/model"
)

var sink any

func main() {
	n := api.UseInt(model.NewIntBox(40))
	text := model.NewStringBox("go")
	sink = text
	println(n + model.UseStringBox(text))
}
