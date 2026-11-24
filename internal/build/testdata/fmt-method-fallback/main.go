package main

import "fmt"

type bad int

func (bad) String() string { panic("format failed") }

type member struct{ Text string }

func (m member) String() string { return "formatted " + m.Text }

type private struct{ field member }

func main() {
	println(fmt.Sprintf("%v", []bad{1}))
	println(fmt.Sprintf("%v", private{member{"plain"}}))
	var number int
	var word string
	if n, err := fmt.Sscan("17 scanner", &number, &word); n != 2 || err != nil || number != 17 || word != "scanner" {
		panic("fmt.Sscan lost its scanner state")
	}
}
