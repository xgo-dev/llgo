package main

import "github.com/xgo-dev/llgo/benchmark/crossinline/value"

func Direct(v value.Mixed, delta int64) value.Mixed { return v.Shift(delta) }
func Summary(v value.Mixed) value.Summary           { return v.Summary() }
func Small(v value.ScalarPair, delta int64) value.ScalarPair {
	return v.Shift(delta)
}

//go:noinline
func Indirect(fn func(value.Mixed, int64) value.Mixed, v value.Mixed, delta int64) value.Mixed {
	return fn(v, delta)
}

var Result value.Mixed

func main() {
	Result = Indirect(value.Mixed.Shift, Direct(value.Mixed{}, 3), 4)
}
