package main

import (
	"os"
	"strconv"

	"github.com/xgo-dev/llgo/internal/build/testdata/coverage/basic"
	"github.com/xgo-dev/llgo/internal/build/testdata/coverage/dep"
	"golang.org/x/mod/semver"
)

var initial = dep.Value()

func main() {
	defer os.Stdout.WriteString("deferred\n")
	os.Stdout.WriteString(strconv.Itoa(initial) + " " + strconv.Itoa(basic.Branch(len(os.Args) > 1)) + " " +
		strconv.Itoa(semver.Compare("v1.0.0", "v2.0.0")) + "\n")
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "exit":
			os.Exit(7)
		case "panic":
			panic("coverage crash fixture")
		}
	}
}
