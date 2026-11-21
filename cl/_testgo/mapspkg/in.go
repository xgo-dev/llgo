// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import (
	"maps"
	"slices"
)

func main() {
	m := map[string]int{"b": 2, "a": 1, "c": 3}
	keys := slices.Sorted(maps.Keys(m))
	println("keys", len(keys), keys[0], keys[2])
	m2 := maps.Clone(m)
	m2["d"] = 4
	println("clone", len(m), len(m2))
	println("equal", maps.Equal(m, m), maps.Equal(m, m2))
	vals := slices.Sorted(maps.Values(m2))
	println("vals", vals[0], vals[3])
	maps.DeleteFunc(m2, func(k string, v int) bool { return v == 4 })
	println("delfn", len(m2))
	m3 := map[string]int{"b": 2, "a": 1, "c": 3}
	println("copyeq", maps.Equal(m, m3))
}
