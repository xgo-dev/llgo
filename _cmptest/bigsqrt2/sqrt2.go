package main

import (
	"fmt"
	"math"
	"math/big"
)

func main() {
	// We'll do computations with 200 bits of precision in the mantissa.
	const prec = 200

	// Compute the square root of 2 using Newton's Method. We start with
	// an initial estimate for sqrt(2), and then iterate:
	//     x_{n+1} = 1/2 * ( x_n + (2.0 / x_n) )

	// Since Newton's Method doubles the number of correct digits at each
	// iteration, we need at least log_2(prec) steps.
	steps := int(math.Log2(prec))

	// Initialize values we need for the computation.
	two := new(big.Float).SetPrec(prec).SetInt64(2)
	half := new(big.Float).SetPrec(prec).SetFloat64(0.5)

	// Use 1 as the initial estimate.
	x := new(big.Float).SetPrec(prec).SetInt64(1)

	// We use t as a temporary variable. There's no need to set its precision
	// since big.Float values with unset (== 0) precision automatically assume
	// the largest precision of the arguments when used as the result (receiver)
	// of a big.Float operation.
	t := new(big.Float)

	// Iterate.
	for i := 0; i <= steps; i++ {
		t.Quo(two, x)  // t = 2.0 / x_n
		t.Add(x, t)    // t = x_n + (2.0 / x_n)
		x.Mul(half, t) // x_{n+1} = 0.5 * t
	}

	// We can use the usual fmt.Printf verbs since big.Float implements fmt.Formatter
	fmt.Printf("sqrt(2) = %.50f\n", x)

	// Print the error between 2 and x*x.
	t.Mul(x, x) // t = x*x
	fmt.Printf("error = %e\n", t.Sub(two, t))
}
