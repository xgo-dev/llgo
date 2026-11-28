[![Build Status](https://travis-ci.org/marcinbor85/gohex.svg?branch=master)](https://travis-ci.org/marcinbor85/gohex)
# gohex
A Go library for parsing Intel HEX files

## Documentation:
https://godoc.org/github.com/marcinbor85/gohex

## Features:
* robust intelhex parsing (full test coverage)
* support i32hex format
* two-way converting hex<->bin
* trivial but powerful api (only the most commonly used functions)
* interface-based IO functions

## Examples:

### Loading IntelHex file:
```go
package main

import (
	"fmt"
	"github.com/marcinbor85/gohex"
	"os"
)

func main() {
	file, err := os.Open("example.hex")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	
	mem := gohex.NewMemory()
	err = mem.ParseIntelHex(file)
	if err != nil {
		panic(err)
	}
	for _, segment := range mem.GetDataSegments() {
		fmt.Printf("%+v\n", segment)
	}
	bytes := mem.ToBinary(0xFFF0, 128, 0x00)
	fmt.Printf("%v\n", bytes)
}
```

### Dumping IntelHex file:
```go
package main

import (
	"github.com/marcinbor85/gohex"
	"os"
)

func main() {
	file, err := os.Create("output.hex")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	
	mem := gohex.NewMemory()
	mem.SetStartAddress(0x80008000)
	mem.AddBinary(0x10008000, []byte{0x01,0x02,0x03,0x04})
	mem.AddBinary(0x20000000, make([]byte, 256))
	
	mem.DumpIntelHex(file, 16)
}
```