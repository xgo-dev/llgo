// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "time"

func main() {
	t0 := time.Now()
	println("nonzero", !t0.IsZero())
	d := time.Since(t0)
	println("since", d >= 0)
	u := time.Now().Unix()
	println("unix-nonneg", u >= 0)
	n := time.Now().Nanosecond()
	println("nanosec", n >= 0 && n < 1000000000)
	t2 := t0.Add(5 * time.Second)
	println("add", t2.After(t0), t0.Before(t2))
	println("sub", t2.Sub(t0) == 5*time.Second)
	println("date", t2.Year() >= 2025 || t2.Year() == 1970)
}
