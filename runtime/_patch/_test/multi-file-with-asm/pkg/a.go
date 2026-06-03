package demo

import "fmt"

var DropA = fmt.Sprint("drop-a")

func KeepA() string { return "keep-a" }

func OldA() string { return fmt.Sprint("old-a") }
