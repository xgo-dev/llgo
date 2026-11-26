// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import "sort"

type byLen []string

func (b byLen) Len() int           { return len(b) }
func (b byLen) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byLen) Less(i, j int) bool { return len(b[i]) < len(b[j]) }

func main() {
	a := []int{5, 2, 8, 1, 9, 3}
	sort.Ints(a)
	println("ints", a[0], a[2], a[5])
	println("sorted", sort.IsSorted(sort.IntSlice(a)))
	s := []string{"pear", "fig", "apple", "kiwi"}
	sort.Strings(s)
	println("strs", s[0], s[3])
	l := byLen{"ccc", "a", "bb"}
	sort.Sort(l)
	println("bylen", l[0], l[2])
	println("search", sort.SearchInts(a, 4), sort.SearchInts(a, 10))
	key := 8
	idx := sort.Search(len(a), func(i int) bool { return a[i] >= key })
	println("searchfn", idx)
	x := []int{3, 1, 2}
	sort.Slice(x, func(i, j int) bool { return x[i] > x[j] })
	println("slice", x[0], x[2])
}
