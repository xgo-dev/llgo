//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var ghead_basic = `package main

import "unsafe"

const (
	LLGoFiles = "../wrap/basic.c"
)

type pointer = *int8

//go:linkname printf C.printf
func printf(format *byte, __llgo_va_list ...any) int32

func assert(info string, b bool) {
	if !b {
		printf(unsafe.StringData("Assertion failed: %s\n\000"), unsafe.StringData(info))
	}
}

func main() {
}
`

var gbasic = `
//go:linkname cbasic C.basic
func cbasic(a int) int

func basic(a int) int {
	return a
}

func init() {
	assert("cbasic\000", cbasic(100) == 100)
	assert("basic\000", basic(100) == 100)
}
`

var gbasic_pointer = `
//go:linkname cbasic_pointer C.basic_pointer
func cbasic_pointer(a pointer) pointer

func basic_pointer(a pointer) pointer {
	return a
}

func init() {
	var p int8
	assert("cbasic_pointer\000", cbasic_pointer(&p) == &p)
	assert("basic_pointer\000", basic_pointer(&p) == &p)
}
`

var garray = `
type array struct {
	x [N]int
}

//go:linkname cdemo C.demo
func cdemo(a array) array

func demo(a array) array {
	return a
}

func init() {
	a := array{x: [N]int{M}}
	assert("cdemo\000", cdemo(a) == a)
	assert("demo\000", demo(a) == a)
}
`

var gstruct = `
type point struct {
	x int
}

//go:linkname cdemo C.demo
func cdemo(a point) point

func demo(a point) point {
	return a
}

func init() {
	a := point{M}
	assert("cdemo\000", cdemo(a) == a)
	assert("demo\000", demo(a) == a)
}
`

var cbasic = `
int basic(int a) {
	return a;
}
`

var carray = `
struct array {
    int x[N];
};

struct array demo(struct array a) {
    return a;
}
`

var cstruct = `
struct point {
    int x;
};

struct point demo(struct point a) {
    return a;
}
`

func main() {
	// char
	// short
	// int
	// int*
	// float
	// double
	// 1 2 3 4 5 6 7 8

	types := []string{"char", "short", "int", "long long", "float", "double", "void*"}
	ids := []string{"int8", "int16", "int32", "int64", "float32", "float64", "pointer"}
	idTypes := make(map[string]string)
	for i, id := range ids {
		idTypes[id] = types[i]
	}

	// write  cbasic
	var cbuf bytes.Buffer
	for nid, id := range ids {
		data := strings.NewReplacer("int", types[nid], "basic", "basic_"+id).Replace(cbasic)
		cbuf.WriteString(data)
	}
	os.WriteFile("./wrap/basic.c", cbuf.Bytes(), 0666)

	// write go basic
	var buf bytes.Buffer
	buf.WriteString(ghead_basic)
	for _, id := range ids {
		data := strings.NewReplacer("int", id, "basic", "basic_"+id).Replace(gbasic)
		if id == "pointer" {
			data = gbasic_pointer
		}
		buf.WriteString(data)
	}
	os.WriteFile("./demo/basic.go", buf.Bytes(), 0666)

	// write c array
	for nid, id := range ids {
		f := filepath.Join("./wrap/array_" + id + ".c")
		var buf bytes.Buffer
		for i := 0; i < 20; i++ {
			N := strconv.Itoa(i + 1)
			data := strings.NewReplacer("int", types[nid], "N", N,
				"array", "array"+N, "demo", "demo"+N).Replace(carray)
			buf.WriteString(data)
		}
		os.WriteFile(f, buf.Bytes(), 0666)
	}

	// write c struct
	for nid, id := range ids {
		f := filepath.Join("./wrap/struct_" + id + ".c")
		var buf bytes.Buffer
		for i := 0; i < 20; i++ {
			N := strconv.Itoa(i + 1)
			ar := make([]string, i+1)
			for j := 0; j < i+1; j++ {
				ar[j] = types[nid] + " x" + strconv.Itoa(j) + ";"
			}
			data := strings.NewReplacer("int x;", strings.Join(ar, "\n\t"), "N", N,
				"point", "point"+N, "demo", "demo"+N).Replace(cstruct)
			buf.WriteString(data)
		}
		os.WriteFile(f, buf.Bytes(), 0666)
	}

	// write go array
	for _, id := range ids {
		f := filepath.Join("./demo/array_" + id + ".go")
		var buf bytes.Buffer
		buf.WriteString(strings.Replace(ghead_basic, "basic", "array_"+id, -1))
		for i := 0; i < 20; i++ {
			N := strconv.Itoa(i + 1)
			M := make([]string, i+1)
			if id == "pointer" {
				for j := 0; j < i+1; j++ {
					M[j] = "func() pointer { var a int8 = " + strconv.Itoa(j+1) + "; return &a}()"
				}
			} else {
				for j := 0; j < i+1; j++ {
					M[j] = strconv.Itoa(j + 1)
				}
			}
			data := strings.NewReplacer("int", id, "N", N, "M", strings.Join(M, ", "),
				"array", "array"+N, "demo", "demo"+N).Replace(garray)
			buf.WriteString(data)
		}
		os.WriteFile(f, buf.Bytes(), 0666)
	}

	// write go struct
	for _, id := range ids {
		f := filepath.Join("./demo/struct_" + id + ".go")
		var buf bytes.Buffer
		buf.WriteString(strings.Replace(ghead_basic, "basic", "struct_"+id, -1))
		for i := 0; i < 20; i++ {
			N := strconv.Itoa(i + 1)
			M := make([]string, i+1)
			if id == "pointer" {
				for j := 0; j < i+1; j++ {
					M[j] = "func() pointer { var a int8 = " + strconv.Itoa(j+1) + "; return &a}()"
				}
			} else {
				for j := 0; j < i+1; j++ {
					M[j] = strconv.Itoa(j + 1)
				}
			}
			ar := make([]string, i+1)
			for j := 0; j < i+1; j++ {
				ar[j] = "x" + strconv.Itoa(j) + " " + id
			}
			data := strings.NewReplacer("x int", strings.Join(ar, "\n\t"), "N", N, "M", strings.Join(M, ", "),
				"point", "point"+N, "demo", "demo"+N).Replace(gstruct)
			buf.WriteString(data)
		}
		os.WriteFile(f, buf.Bytes(), 0666)
	}
	var mixeds []string
	for _, id1 := range ids {
		for _, id2 := range ids {
			if id1 == id2 {
				continue
			}
			mixeds = append(mixeds, id1+","+id2)
		}
	}
	ids2 := []string{"int8", "int16", "int32", "float32"}
	for _, id1 := range ids2 {
		for _, id2 := range ids2 {
			if id1 == id2 {
				continue
			}
			for _, id3 := range ids2 {
				if id1 == id2 && id2 == id3 {
					continue
				}
				mixeds = append(mixeds, id1+","+id2+","+id3)
				for _, id4 := range ids2 {
					if id1 == id2 && id2 == id3 && id3 == id4 {
						continue
					}
					mixeds = append(mixeds, id1+","+id2+","+id3+","+id4)
					for _, id5 := range ids2 {
						if id1 == id2 && id2 == id3 && id3 == id4 && id4 == id5 {
							continue
						}
						mixeds = append(mixeds, id1+","+id2+","+id3+","+id4+","+id5)
					}
				}
			}
		}
	}
	// write c mixed
	cbuf.Reset()
	for i, mixed := range mixeds {
		N := strconv.Itoa(i + 1)
		ids := strings.Split(mixed, ",")
		ar := make([]string, len(ids))
		for j, id := range ids {
			ar[j] = idTypes[id] + " x" + strconv.Itoa(j) + ";"
		}
		data := strings.NewReplacer("int x;", strings.Join(ar, "\n\t"), "N", N,
			"point", "point"+N, "demo", "demo"+N).Replace(cstruct)
		cbuf.WriteString(data)
	}
	os.WriteFile("./wrap/struct_mixed.c", cbuf.Bytes(), 0666)
	// write go mixed
	buf.Reset()
	buf.WriteString(strings.Replace(ghead_basic, "basic", "struct_mixed", -1))
	for i, mixed := range mixeds {
		N := strconv.Itoa(i + 1)
		ids := strings.Split(mixed, ",")
		ar := make([]string, len(ids))
		for j, id := range ids {
			ar[j] = "x" + strconv.Itoa(j) + " " + id
		}
		M := make([]string, len(ids))
		for j := 0; j < len(ids); j++ {
			if ids[j] == "pointer" {
				M[j] = "func() pointer { var a int8 = " + strconv.Itoa(j+1) + "; return &a}()"
			} else {
				M[j] = strconv.Itoa(j + 1)
			}
		}
		data := strings.NewReplacer("x int", strings.Join(ar, "\n\t"), "N", N, "M", strings.Join(M, ", "),
			"point", "point"+N, "demo", "demo"+N).Replace(gstruct)
		buf.WriteString(data)
	}
	os.WriteFile("./demo/struct_mixed.go", buf.Bytes(), 0666)
}
