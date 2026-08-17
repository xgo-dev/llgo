package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/math"
	"github.com/xgo-dev/llgo/_demo/c/cppmintf/foo"
)

type Bar struct {
	foo.Callback
	a c.Int
}

func NewBar(a c.Int) *Bar {
	return &Bar{
		Callback: foo.Callback{
			ICalc: foo.ICalc{
				Vptr: &foo.ICalcVtbl{
					Calc: c.Func((*Bar).sqrt),
				},
			},
			IVal: foo.IVal{
				Vptr: &foo.IValVtbl{
					Val: c.Func(bar_IVal_getA),
				},
			},
		},
		a: a,
	}
}

func (p *Bar) getA() c.Int {
	return p.a
}

func bar_IVal_getA(this c.Pointer) c.Int {
	const delta = -int(unsafe.Offsetof(foo.Callback{}.IVal))
	return (*Bar)(c.Advance(this, delta)).getA()
}

func (p *Bar) sqrt(v float64) float64 {
	return math.Sqrt(v)
}

func main() {
	bar := NewBar(1)
	foo.F(&bar.Callback)
}
