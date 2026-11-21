// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "unicode"

func main() {
	println("upper", unicode.IsUpper('A'), unicode.IsUpper('a'))
	println("lower", unicode.IsLower('b'))
	println("digit", unicode.IsDigit('7'), unicode.IsDigit('x'))
	println("space", unicode.IsSpace(' '), unicode.IsSpace('\t'), unicode.IsSpace('a'))
	println("toupper", unicode.ToUpper('q'), unicode.ToLower('Q'))
	println("letter", unicode.IsLetter('e'), unicode.IsLetter('3'))
	println("print", unicode.IsPrint('a'))
	println("latin", unicode.Is(unicode.Latin, 'A'), unicode.Is(unicode.Latin, 'ñ'))
	println("utf8", string(unicode.ToUpper('ñ')), unicode.ToUpper('k'))
}
