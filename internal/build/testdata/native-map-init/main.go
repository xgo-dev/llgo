package main

import "unicode"

func main() {
	println(unicode.Is(unicode.Categories["L"], 'é'), unicode.Is(unicode.Scripts["Greek"], 'Ω'), unicode.IsDigit('٥'))
}
