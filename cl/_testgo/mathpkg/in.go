// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import (
	"math"
	"strconv"
)

func show(f float64) string { return strconv.FormatFloat(f, 'g', 6, 64) }

func main() {
	println("sqrt", show(math.Sqrt(2)), show(math.Sqrt(16)))
	println("abs", show(math.Abs(-3.5)), show(math.Abs(2)))
	println("floorceil", show(math.Floor(2.7)), show(math.Ceil(2.1)))
	println("trunc", show(math.Trunc(-2.7)))
	println("pow", show(math.Pow(2, 10)))
	println("mod", show(math.Mod(7, 3)))
	println("maxmin", show(math.Max(1.5, 2.5)), show(math.Min(-1, -2)))
	println("nan", math.IsNaN(math.NaN()), math.IsInf(math.Inf(1), 1))
	println("pi", show(math.Pi))
	println("trig", show(math.Sin(0)), show(math.Cos(0)))
}
