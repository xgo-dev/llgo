package main

import "reflect"

// Regression program for issue #2797. These ordinary Go structs end in a
// zero-sized field, so Go reserves a trailing padding byte and rounds the size
// up to alignment: Tail is size 8 (not 4) and Outer is size 16. Safe reflection
// must observe the same element stride and field offsets as typed access. The
// program imports neither unsafe nor any C type; it only exercises reflect.

// C: count at 0, end at 4, sizeof 4. Go: Count at 0, End at 4, sizeof 8.
type Tail struct {
	Count int32
	End   [0]int32
}

// Go: Head at 0, Next at 8, Guard at 12, sizeof 16. Tail's trailing padding
// shifts the following fields by 4 bytes.
type Outer struct {
	Head  Tail
	Next  int32
	Guard int32
}

func reflectIndexing() {
	// Four Tail elements start at 0, 8, 16, 24 and occupy 32 bytes.
	items := [4]Tail{{Count: 10}, {Count: 20}, {Count: 30}, {Count: 40}}
	v := reflect.ValueOf(&items).Elem()
	seen := v.Index(1).Field(0).Int()
	println("typed-second", items[1].Count)
	println("reflected-second", seen)
	v.Index(1).Field(0).SetInt(77)
	println("after-set", items[0].Count, items[1].Count, items[2].Count, items[3].Count)
}

func reflectCopy() {
	src := [8]Tail{{Count: 10}, {Count: 20}, {Count: 30}, {Count: 40}, {Count: 50}, {Count: 60}, {Count: 70}, {Count: 80}}
	dst := [8]Tail{{}, {}, {}, {}, {Count: 101}, {Count: 102}, {Count: 103}, {Count: 104}}
	n := reflect.Copy(reflect.ValueOf(dst[:4]), reflect.ValueOf(src[:4]))
	println("copied", n)
	println("values", dst[0].Count, dst[1].Count, dst[2].Count, dst[3].Count)
	println("guards", dst[4].Count, dst[5].Count, dst[6].Count, dst[7].Count)
}

func reflectNestedField() {
	o := Outer{Head: Tail{Count: 10}, Next: 20, Guard: 30}
	v := reflect.ValueOf(&o).Elem()
	// Field 1 is Next, which sits at offset 8 after Tail's trailing padding.
	println("reflected-next", v.Field(1).Int())
	v.Field(1).SetInt(77)
	println("after-set", o.Head.Count, o.Next, o.Guard)
}

func main() {
	reflectIndexing()
	reflectCopy()
	reflectNestedField()
}
