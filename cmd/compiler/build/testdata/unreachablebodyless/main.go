package main

func fail()

func init() {
	if false {
		fail()
	}
	if 0 == 1 {
		fail()
	}
}

func init() {
	const x = 0
	switch x {
	case 1:
		fail()
	}
	switch 1 {
	case x:
		fail()
	}
	switch {
	case false:
		fail()
	}
	const a = "a"
	switch a {
	case "b":
		fail()
	}
	const snowman = '☃'
	switch snowman {
	case '☀':
		fail()
	}
	const zero = float64(0)
	const one = float64(1)
	switch one {
	case -1:
		fail()
	case zero:
		fail()
	}
	switch 1i {
	case 1:
		fail()
	case -1i:
		fail()
	}
	const no = false
	switch no {
	case true:
		fail()
	}
	switch 5 {
	case 3, 4, 5, 6, 7:
	case 0, 1, 2:
		fail()
	default:
		fail()
	}
}

func main() {}
