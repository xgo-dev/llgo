// Behavior regression fixture: runs on hosted and on the RV32 baremetal
// scheduler; the golden output must stay identical on both.

package main

import (
	"math/rand"
	"strconv"
)

func main() {
	r := rand.New(rand.NewSource(42))
	println("intn", r.Intn(100), r.Intn(100))
	println("int63", strconv.FormatInt(r.Int63(), 16))
	println("float", strconv.FormatFloat(r.Float64(), 'f', 6, 64))
	println("int31", r.Int31())
	s := []int{1, 2, 3, 4, 5}
	r.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
	println("shuffle", s[0], s[4])
	perm := r.Perm(4)
	println("perm", perm[0], perm[1], perm[2], perm[3])
}
