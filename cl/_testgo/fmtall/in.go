// Behavior regression fixture: golden output must stay identical on hosted
// and on the RV32 baremetal XIP flash target. The image only fits the flash
// (XIP) layout; SRAM and the 256 KiB ESP32-C3 IRAM cannot hold the pulled-in
// fmt dependency.
package main

import "fmt"

type pt struct {
	X int
	Y string
}

func (p pt) String() string { return fmt.Sprintf("pt{%d %s}", p.X, p.Y) }

type bufWriter struct{ buf []byte }

func (b *bufWriter) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func main() {
	fmt.Println("println", 1, "two", 3.5, true)
	fmt.Printf("d=%d s=%s x=%#x o=%o b=%b q=%q\n", 42, "str", 255, 8, 5, "qq")
	fmt.Printf("f=%f g=%g e=%e T=%T\n", 1.25, 100000.0, 0.00125, pt{})
	fmt.Printf("v=%v +v=%+v\n", pt{3, "y"}, pt{-1, "z"})
	fmt.Println(fmt.Sprintf("spr%%f=%.2f pct=%d%%", 3.14159, 50))
	fmt.Println(fmt.Sprint(1, "-", "a"))
	var i int
	var s string
	n, _ := fmt.Sscan("99 x", &i, &s)
	fmt.Println("sscan", i, s, n)
	w := &bufWriter{}
	fmt.Fprintf(w, "fprintf %d\n", 7)
	fmt.Print(string(w.buf))
	fmt.Println(string(fmt.Appendf(nil, "app%d", 4)))
}
