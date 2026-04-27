package main

import (
	"runtime"

	dep "github.com/goplus/llgo/cl/_testgo/rewrite/dep"
)

var VarName = "main-default"
var VarPlain string

func printLine(label, value string) {
	println(label+":", value)
}

func main() {
	printLine("main.VarName", VarName)
	printLine("main.VarPlain", VarPlain)
	dep.PrintVar()
	printLine("runtime.GOROOT()", runtime.GOROOT())
	printLine("runtime.Version()", runtime.Version())
}
