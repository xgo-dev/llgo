package main

//go:noinline
func helper(value NamedInt) NamedInt {
	intermediate := value + 1
	return intermediate // DWARF_LINE_MARKER
}
