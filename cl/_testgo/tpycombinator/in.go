// LITTEST
package main

// CHECK-DAG: %"{{.*}}tpycombinator.internal[{{.*}},int,int;{{.*}},int,int]" = type { ptr, ptr }
// CHECK-DAG: %"{{.*}}tpycombinator.internal[{{.*}},string,string;{{.*}},string,string]" = type { ptr, ptr }

func Y[Endo ~func(RecFct) RecFct, RecFct ~func(T) R, T, R any](f Endo) RecFct {
	type internal[RecFct ~func(T) R, T, R any] func(internal[RecFct, T, R]) RecFct

	g := func(h internal[RecFct, T, R]) RecFct {
		return func(t T) R {
			return f(h(h))(t)
		}
	}
	return g(g)
}

func main() {
	factorial := Y(func(recur func(int) int) func(int) int {
		return func(n int) int {
			if n == 0 {
				return 1
			}
			return n * recur(n-1)
		}
	})
	repeat := Y(func(recur func(string) string) func(string) string {
		return func(s string) string {
			if len(s) == 3 {
				return s
			}
			return recur(s + "x")
		}
	})
	println(factorial(10))
	println(repeat(""))
}
