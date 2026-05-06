package main

import (
	"bytes"
	"io"
)

func main() {
	var r io.Reader = bytes.NewBuffer(nil)
	_, _ = r.Read(nil)
}
