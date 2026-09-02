package main

import (
	"sync/atomic"

	"github.com/goplus/lib/c"
)

var bootCount uint32

func main() {
	count := atomic.AddUint32(&bootCount, 1)
	c.Printf(c.Str("Hello from ESP32-C6 via USB Serial JTAG! boot=%u\n"), count)
}
