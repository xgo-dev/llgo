package demo

//llgo:skip Drop

import "strings"

var Added = strings.ToUpper("added")

func Old() string   { return "new" }
func (T) M() string { return "new method" }
