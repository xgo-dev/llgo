package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/xgo-dev/llgo/internal/build/testdata/wasm-fmt-map/record"
)

type score uint16

type key struct {
	Name string
	IDs  [2]int
}

func main() {
	values := map[string]int{"世界": 42, "b": -7, "a": 0, "": 8}
	if got := fmt.Sprintf("%v", values); got != "map[:8 a:0 b:-7 世界:42]" {
		panic(got)
	}
	var empty map[string]int
	if got := fmt.Sprintf("%v", empty); got != "map[]" {
		panic(got)
	}
	nested := map[key][]score{
		{Name: "b"}:                    {4},
		{Name: "a", IDs: [2]int{1, 2}}: {2, 3},
		{Name: "a"}:                    nil,
	}
	if got := fmt.Sprintf("%v", nested); got != "map[{a [0 0]}:[] {a [1 2]}:[2 3] {b [0 0]}:[4]]" {
		panic(got)
	}
	aggregate := struct {
		Values []complex64
		Flags  [2]bool
	}{[]complex64{1 - 2i, -3 + 4i}, [2]bool{true, false}}
	if got := fmt.Sprintf("%v", &aggregate); got != "&{[(1-2i) (-3+4i)] [true false]}" {
		panic(got)
	}
	negativeZero := math.Copysign(0, -1)
	complexEdges := [4]complex128{
		complex(0, negativeZero),
		complex(negativeZero, 0),
		complex(math.NaN(), math.Inf(1)),
		complex(math.Inf(-1), math.NaN()),
	}
	if got := fmt.Sprintf("%v", complexEdges); got != "[(0-0i) (-0+0i) (NaN+Infi) (-Inf+NaNi)]" {
		panic(got)
	}
	nanMap := map[float64]string{math.NaN(): "nan", -1: "negative", 2: "positive"}
	if got := fmt.Sprintf("%v", nanMap); got != "map[NaN:nan -1:negative 2:positive]" {
		panic(got)
	}
	foreign := map[record.Key]bool{
		record.New("界", 4, 2): true,
		record.New("界", 1, 3): false,
	}
	if got := fmt.Sprintf("%v", foreign); got != "map[{界 {[1 3]}}:false {界 {[4 2]}}:true]" {
		panic(got)
	}
	type local struct {
		count int
		label string
	}
	if got := fmt.Sprintf("%v", local{4, "local"}); got != "{4 local}" {
		panic(got)
	}
	var emptyChan chan int
	if got := fmt.Sprintf("%v", emptyChan); got != "<nil>" {
		panic(got)
	}
	channels := map[chan int]string{emptyChan: "nil", make(chan int): "live"}
	got := fmt.Sprintf("%v", channels)
	if !strings.HasPrefix(got, "map[<nil>:nil 0x") || !strings.HasSuffix(got, ":live]") {
		panic(got)
	}
	println("wasi default value formatting ok")
}
