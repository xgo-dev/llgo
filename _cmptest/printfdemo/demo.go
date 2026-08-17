package main

import (
	"fmt"

	"github.com/xgo-dev/llgo/xtool/nm"
)

func main() {
	sym := nm.Symbol{Name: "abc", Type: nm.Text}
	fmt.Printf("%016x %c %s\n", sym.Addr, sym.Type, sym.Name)
}
