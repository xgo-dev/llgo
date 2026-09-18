//go:build go1.26

package test

import (
	"os"
	"strings"
)

const builtinPrintChildArg = "-llgo.builtin-print-child"

func init() {
	child := os.Getenv("LLGO_PRINT_HELPER") != ""
	for _, arg := range os.Args[1:] {
		child = child || arg == builtinPrintChildArg
	}
	if !child {
		return
	}
	builtinPrintProbe()
	os.Exit(0)
}

func builtinPrintProbe() {
	zero := 0.0
	nan := zero / zero
	posInf := 1.0 / zero
	negInf := -1.0 / zero

	print(1e7, "\n")
	print(complex(1e7, -1e7), "\n")
	print(complex(1.5, -0.0), "\n")
	print(nan, "\n")
	print(posInf, "\n")
	print(negInf, "\n")
	print(complex(1, nan), "\n")
	print(complex(1, posInf), "\n")
	print(complex(1, negInf), "\n")
}

func builtinPrintWant() string {
	return strings.Join([]string{
		"1e+07",
		"(1e+07-1e+07i)",
		"(1.5+0i)",
		"NaN",
		"+Inf",
		"-Inf",
		"(1+NaNi)",
		"(1+Infi)",
		"(1-Infi)",
		"",
	}, "\n")
}
