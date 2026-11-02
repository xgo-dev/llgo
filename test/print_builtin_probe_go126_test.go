//go:build go1.26

package test

import (
	"os"
	"strings"
	"unsafe"
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
	println(false, true)
	println(int64(0), int64(1), int64(-1), int64(-9223372036854775808), int64(9223372036854775807))
	println(uint64(0), uint64(1), uint64(9223372036854775808), uint64(18446744073709551615))
	println(unsafe.Pointer(nil))

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
		"false true",
		"0 1 -1 -9223372036854775808 9223372036854775807",
		"0 1 9223372036854775808 18446744073709551615",
		"0x0",
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
