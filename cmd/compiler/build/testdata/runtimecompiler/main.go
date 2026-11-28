package main

import "github.com/xgo-dev/llgo/cmd/compiler/build/testdata/runtimecompiler/identity"

func main() {
	println(identity.Name, identity.Description, len(identity.NameBytes{}), identity.Branch())
}
