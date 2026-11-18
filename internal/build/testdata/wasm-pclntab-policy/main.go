package main

import (
	"os"
	"runtime"
	"strings"
	"time"
)

var sourceAtInit bool

func init() {
	println("pcln-policy-init")
	sourceAtInit = hasSource()
}

//go:noinline
func hasSource() bool {
//line pcln-policy.go:17
	_, file, line, _ := runtime.Caller(0)
	return strings.HasSuffix(file, "pcln-policy.go") && line == 17
}

func main() {
	switch os.Args[1] {
	case "required":
		if !sourceAtInit {
			panic("metadata missing during init")
		}
	case "none":
		if sourceAtInit || hasSource() {
			panic("metadata loaded in no-load policy")
		}
	case "background", "background-pending":
		if os.Args[1] == "background-pending" && sourceAtInit {
			panic("init waited for background metadata")
		}
		deadline := time.Now().Add(10 * time.Second)
		for !hasSource() {
			if time.Now().After(deadline) {
				panic("background metadata not published")
			}
			time.Sleep(5 * time.Millisecond)
		}
	default:
		panic("unexpected loading policy")
	}
	println("pcln-policy-ok")
}
