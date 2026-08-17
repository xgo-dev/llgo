package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/math"
	"github.com/xgo-dev/llgo/_demo/c/cppintf/foo"
)

type Bar struct {
	foo.Callback
	a c.Int
}

func NewBar(a c.Int) *Bar {
	return &Bar{
		Callback: foo.Callback{
			Vptr: &foo.CallbackVtbl{
				Val:  c.Func((*Bar).getA),
				Calc: c.Func((*Bar).sqrt),
			},
		},
		a: a,
	}
}

func (p *Bar) getA() c.Int {
	return p.a
}

func (p *Bar) sqrt(v float64) float64 {
	return math.Sqrt(v)
}

func main() {
	bar := NewBar(1)
	foo.F(&bar.Callback)
	foo.G(&bar.Callback)
}
