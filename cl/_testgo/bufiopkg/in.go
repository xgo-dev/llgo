// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both. The image only
// fits the 512 KiB S31 SRAM and the XIP flash layouts; the 256 KiB ESP32-C3
// IRAM target cannot hold it.
package main

import (
	"bufio"
	"strings"
)

func main() {
	r := bufio.NewReader(strings.NewReader("line1\nline2\nend"))
	s1, err := r.ReadString('\n')
	println("r1", s1, err == nil)
	s2, _ := r.ReadString('\n')
	println("r2", s2)
	rest, _ := r.ReadString(0)
	println("r3", rest)
	var w strings.Builder
	bw := bufio.NewWriter(&w)
	bw.WriteString("wr-a,")
	bw.Flush()
	bw.WriteString("wr-b")
	bw.Flush()
	println("w", w.String())
}
