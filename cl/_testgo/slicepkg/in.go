// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "slices"

func main() {
	a := []int{3, 1, 2}
	slices.Sort(a)
	println("sorted", a[0], a[2])
	println("contains", slices.Contains(a, 2), slices.Contains(a, 9))
	println("index", slices.Index(a, 3), slices.Index(a, 7))
	b := slices.Clone(a)
	b[0] = 99
	println("clone", a[0], b[0])
	println("equal", slices.Equal(a, []int{1, 2, 3}), slices.Equal(a, b))
	println("maxmin", slices.Max(a), slices.Min(a))
	i, found := slices.BinarySearch(a, 2)
	println("bin", i, found)
	c := []int{1, 1, 2, 2, 3}
	c2 := slices.Compact(c)
	println("compact", len(c2), c2[1])
	s := []string{"a", "b"}
	s2 := slices.Insert(s, 1, "x")
	println("insert", len(s2), s2[1])
	s3 := slices.Delete(s2, 0, 1)
	println("delete", len(s3), s3[0])
	println("rev", slices.IsSorted(a), slices.Equal(s, []string{"a", "b"}))
}
