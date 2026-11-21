// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both. The image only
// fits the 512 KiB S31 SRAM and the XIP flash layouts; the 256 KiB ESP32-C3
// IRAM target cannot hold it.
package main

import (
	"io"
	"strings"
)

func main() {
	r := strings.NewReader("abcdef")
	p := make([]byte, 4)
	n, err := r.Read(p)
	println("read", n, err == nil, string(p[:n]))
	n2, err2 := r.Read(p)
	println("read2", n2, err2 == nil)
	n3, err3 := r.Read(p)
	println("eof", n3, err3 == io.EOF)
	src := strings.NewReader("hello")
	dst := &sink{}
	cn, cerr := io.Copy(dst, src)
	println("copy", cn, cerr == nil, dst.s)
	println("readall", func() int {
		b, _ := io.ReadAll(strings.NewReader("xyzzy"))
		return len(b)
	}())
	r2 := io.MultiReader(strings.NewReader("ab"), strings.NewReader("cd"))
	all := make([]byte, 8)
	n4, _ := io.ReadFull(r2, all[:4])
	println("multi", n4, string(all[:4]))
}

type sink struct{ s string }

func (k *sink) Write(p []byte) (int, error) {
	k.s += string(p)
	return len(p), nil
}
