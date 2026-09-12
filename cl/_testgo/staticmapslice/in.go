// LITTEST
package main

type Symbol struct {
	Name string
	Kind int8
	Ver  int8
	Sig  string
}

var table = map[string][]Symbol{
	"fmt": {
		{"Println", 1, 0, "func()"},
		{"Printf", 1, 0, "func()"},
	},
	"os": {
		{"Exit", 1, 0, "func(int)"},
	},
}

var empty = map[string][]int{
	"empty": {},
}

var dynamicValue = map[string][]int{
	"dynamic": make([]int, 1),
}

var dynamicKeyName = "dynamic-key"
var dynamicKey = map[string][]int{
	dynamicKeyName: {1},
}

var reassigned = map[string][]int{
	"first": {1},
}

func init() {
	reassigned = map[string][]int{"second": {2}}
}

// CHECK: @"main.table$m0" = {{.*}}global
// CHECK: @"main.table$m1" = {{.*}}global
// CHECK-LABEL: define void @main.init(){{.*}} {
// CHECK: call ptr @"{{.*}}MakeMap"(
// CHECK: call ptr @"{{.*}}MapAssignFastStr"(

func main() {
	println(table["fmt"][0].Name)
	println(table["fmt"][1].Name)
	println(table["os"][0].Name)
	table["fmt"][0].Name = "mut"
	println(table["fmt"][0].Name)
	println(len(empty["empty"]))
	println(dynamicValue["dynamic"][0])
	println(dynamicKey[dynamicKeyName][0])
	println(reassigned["second"][0])
}
