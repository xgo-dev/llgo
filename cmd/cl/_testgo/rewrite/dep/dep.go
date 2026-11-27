package dep

import "fmt"

var VarName = "dep-default"
var VarPlain string

func PrintVar() {
	fmt.Printf("dep.VarName: %s\n", VarName)
	fmt.Printf("dep.VarPlain: %s\n", VarPlain)
}
