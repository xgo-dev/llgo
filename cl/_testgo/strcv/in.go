// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "strconv"

func main() {
	println("itoa", strconv.Itoa(-42))
	i, err := strconv.Atoi("12345")
	println("atoi", i, err == nil)
	bad, err := strconv.Atoi("12x")
	println("bad", bad, err != nil)
	v, err := strconv.ParseInt("-0x1f", 0, 64)
	println("parseint", v, err == nil)
	println("formatint", strconv.FormatInt(255, 16), strconv.FormatInt(8, 2))
	f, err := strconv.ParseFloat("3.25", 64)
	println("parsefloat", f == 3.25, err == nil)
	println("fmtfloat", strconv.FormatFloat(3.14159, 'f', 3, 64))
	println("fmtfloat2", strconv.FormatFloat(1e10, 'g', -1, 64))
	println("quote", strconv.Quote("a b"))
	uq, err := strconv.Unquote("\"hi\\tthere\"")
	println("unquote", uq, err == nil)
	println("bool", strconv.FormatBool(true), strconv.FormatBool(false))
}
