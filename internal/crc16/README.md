
# crc16
[![Build Status](https://travis-ci.org/sigurn/crc16.svg?branch=master)](https://travis-ci.org/sigurn/crc16)
[![Coverage Status](https://coveralls.io/repos/sigurn/crc16/badge.svg?branch=master&service=github)](https://coveralls.io/github/sigurn/crc16?branch=master)
[![Go Reference](https://pkg.go.dev/badge/github.com/sigurn/crc16.svg)](https://pkg.go.dev/github.com/sigurn/crc16)

Go implementation of CRC-16 calculation for majority of widely-used polynomials.</br>
It implements the golang hash.Hash interface.

## Usage
```go
package main

import (
	"fmt"

	"github.com/sigurn/crc16"
)

func main() {
	table := crc16.MakeTable(crc16.CRC16_MAXIM)

	crc := crc16.Checksum([]byte("Hello world!"), table)
	fmt.Printf("CRC-16 MAXIM: %X\n", crc)

	// using the standard library hash.Hash interface
	h := crc16.New(table)
	h.Write([]byte("Hello world!"))
	fmt.Printf("CRC-16 MAXIM: %X\n", h.Sum16())
}
```

## Documentation
For more documentation see [package documentation](https://godoc.org/github.com/sigurn/crc16)
