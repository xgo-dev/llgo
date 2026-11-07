// LITTEST
// Scope: common
// CHECK: define {{.*}} @main.publication(
package main

type Item struct{ X int }

var shared map[int]int
var observed, calls int

//go:noinline
func observe() { observed = len(shared) }

//go:noinline
func effect() int { calls++; return 7 }
func publication() {
	m := map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 6, 7: 7, 8: 8, 9: 9, 10: 10, 11: 11, 12: 12, 13: 13, 14: 14, 15: 15, 16: 16, 17: 17, 18: 18, 19: 19, 20: 20, 21: 21, 22: 22, 23: 23, 24: 24}
	shared = m
	observe()
	m[25] = 25
	if observed != 25 || len(m) != 26 {
		panic("early publication")
	}
}
func zeroSnapshot() {
	m := make(map[int]Item)
	var v Item
	m[0] = v
	v.X = 99
	m[1] = Item{1}
	m[2] = Item{2}
	m[3] = Item{3}
	m[4] = Item{4}
	m[5] = Item{5}
	m[6] = Item{6}
	m[7] = Item{7}
	m[8] = Item{8}
	m[9] = Item{9}
	m[10] = Item{10}
	m[11] = Item{11}
	m[12] = Item{12}
	m[13] = Item{13}
	m[14] = Item{14}
	m[15] = Item{15}
	m[16] = Item{16}
	m[17] = Item{17}
	m[18] = Item{18}
	m[19] = Item{19}
	m[20] = Item{20}
	m[21] = Item{21}
	m[22] = Item{22}
	m[23] = Item{23}
	m[24] = Item{24}
	m[25] = Item{25}
	if m[0].X != 0 {
		panic("zeroSnapshot")
	}
}
func initializedSnapshot() {
	m := make(map[int]Item)
	var v Item
	v.X = 1
	m[0] = v
	v.X = 99
	m[1] = Item{1}
	m[2] = Item{2}
	m[3] = Item{3}
	m[4] = Item{4}
	m[5] = Item{5}
	m[6] = Item{6}
	m[7] = Item{7}
	m[8] = Item{8}
	m[9] = Item{9}
	m[10] = Item{10}
	m[11] = Item{11}
	m[12] = Item{12}
	m[13] = Item{13}
	m[14] = Item{14}
	m[15] = Item{15}
	m[16] = Item{16}
	m[17] = Item{17}
	m[18] = Item{18}
	m[19] = Item{19}
	m[20] = Item{20}
	m[21] = Item{21}
	m[22] = Item{22}
	m[23] = Item{23}
	m[24] = Item{24}
	m[25] = Item{25}
	if m[0].X != 1 {
		panic("initializedSnapshot")
	}
}
func overwrittenEffect() {
	m := make(map[int]Item)
	var v Item
	v.X = effect()
	v.X = 99
	m[0] = v
	m[1] = Item{1}
	m[2] = Item{2}
	m[3] = Item{3}
	m[4] = Item{4}
	m[5] = Item{5}
	m[6] = Item{6}
	m[7] = Item{7}
	m[8] = Item{8}
	m[9] = Item{9}
	m[10] = Item{10}
	m[11] = Item{11}
	m[12] = Item{12}
	m[13] = Item{13}
	m[14] = Item{14}
	m[15] = Item{15}
	m[16] = Item{16}
	m[17] = Item{17}
	m[18] = Item{18}
	m[19] = Item{19}
	m[20] = Item{20}
	m[21] = Item{21}
	m[22] = Item{22}
	m[23] = Item{23}
	m[24] = Item{24}
	m[25] = Item{25}
	if m[0].X != 99 {
		panic("overwrittenEffect")
	}
}

type I interface {
	A() int
	Z() int
}
type J interface{ Z() int }
type N int

func (n N) A() int { return 11 }
func (n N) Z() int { return 22 }
func box(n N) I    { return n }

type AnyItem struct{ V any }
type InterfaceItem struct{ V J }

func conversions() {
	a := map[int]AnyItem{0: {box(0)}, 1: {box(1)}, 2: {box(2)}, 3: {box(3)}, 4: {box(4)}, 5: {box(5)}, 6: {box(6)}, 7: {box(7)}, 8: {box(8)}, 9: {box(9)}, 10: {box(10)}, 11: {box(11)}, 12: {box(12)}, 13: {box(13)}, 14: {box(14)}, 15: {box(15)}, 16: {box(16)}, 17: {box(17)}, 18: {box(18)}, 19: {box(19)}, 20: {box(20)}, 21: {box(21)}, 22: {box(22)}, 23: {box(23)}, 24: {box(24)}, 25: {box(25)}}
	n, ok := a[3].V.(N)
	if !ok || n != 3 {
		panic("empty interface conversion")
	}
	b := map[int]InterfaceItem{0: {box(0)}, 1: {box(1)}, 2: {box(2)}, 3: {box(3)}, 4: {box(4)}, 5: {box(5)}, 6: {box(6)}, 7: {box(7)}, 8: {box(8)}, 9: {box(9)}, 10: {box(10)}, 11: {box(11)}, 12: {box(12)}, 13: {box(13)}, 14: {box(14)}, 15: {box(15)}, 16: {box(16)}, 17: {box(17)}, 18: {box(18)}, 19: {box(19)}, 20: {box(20)}, 21: {box(21)}, 22: {box(22)}, 23: {box(23)}, 24: {box(24)}, 25: {box(25)}}
	if b[3].V.Z() != 22 {
		panic("interface method ordering")
	}
}
func main() {
	publication()
	zeroSnapshot()
	initializedSnapshot()
	overwrittenEffect()
	if calls != 1 {
		panic("side effect removed")
	}
	conversions()
	println("ok")
}
