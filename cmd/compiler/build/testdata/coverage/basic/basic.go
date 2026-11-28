package basic

var Initial = Branch(true)

func Branch(yes bool) int {
	if yes {
		return 1
	}
	return 2
}

func Unused() int {
	return 3
}
