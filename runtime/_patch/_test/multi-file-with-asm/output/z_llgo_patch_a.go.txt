//line {{PATCH_ROOT}}/demo/a.go:1
package demo

//llgo_skip DropA

import "strings"

var AddedA = strings.ToUpper("added-a")

func OldA() string { return "new-a" }
