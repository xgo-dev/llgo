package gotest

import "testing"

type recursiveNamedT1 struct {
	Next *recursiveNamedT2
}

type recursiveNamedT2 recursiveNamedT1

type recursiveNamedT3 struct {
	Next *recursiveNamedT4
}

type recursiveNamedT4 recursiveNamedT5
type recursiveNamedT5 recursiveNamedT6
type recursiveNamedT6 recursiveNamedT7
type recursiveNamedT7 recursiveNamedT8
type recursiveNamedT8 recursiveNamedT9
type recursiveNamedT9 recursiveNamedT3

type recursiveNamedT10 struct {
	x struct {
		y ***struct {
			z *struct {
				Next *recursiveNamedT11
			}
		}
	}
}

type recursiveNamedT11 recursiveNamedT10

type recursiveNamedT12 struct {
	F1 *recursiveNamedT15
	F2 *recursiveNamedT13
	F3 *recursiveNamedT16
}

type recursiveNamedT13 recursiveNamedT14
type recursiveNamedT14 recursiveNamedT15
type recursiveNamedT15 recursiveNamedT16
type recursiveNamedT16 recursiveNamedT17
type recursiveNamedT17 recursiveNamedT12

type recursiveNamedT18 *[10]recursiveNamedT19
type recursiveNamedT19 recursiveNamedT18

func TestRecursiveNamedTypeLiterals(t *testing.T) {
	_ = &recursiveNamedT1{&recursiveNamedT2{}}
	_ = &recursiveNamedT2{&recursiveNamedT2{}}
	_ = &recursiveNamedT3{&recursiveNamedT4{}}
	_ = &recursiveNamedT4{&recursiveNamedT4{}}
	_ = &recursiveNamedT5{&recursiveNamedT4{}}
	_ = &recursiveNamedT6{&recursiveNamedT4{}}
	_ = &recursiveNamedT7{&recursiveNamedT4{}}
	_ = &recursiveNamedT8{&recursiveNamedT4{}}
	_ = &recursiveNamedT9{&recursiveNamedT4{}}
	_ = &recursiveNamedT12{&recursiveNamedT15{}, &recursiveNamedT13{}, &recursiveNamedT16{}}

	var (
		tn struct{ Next *recursiveNamedT11 }
		tz struct {
			z *struct{ Next *recursiveNamedT11 }
		}
		tpz *struct {
			z *struct{ Next *recursiveNamedT11 }
		}
		tppz **struct {
			z *struct{ Next *recursiveNamedT11 }
		}
		tpppz ***struct {
			z *struct{ Next *recursiveNamedT11 }
		}
		ty struct {
			y ***struct {
				z *struct{ Next *recursiveNamedT11 }
			}
		}
	)
	tn.Next = &recursiveNamedT11{}
	tz.z = &tn
	tpz = &tz
	tppz = &tpz
	tpppz = &tppz
	ty.y = tpppz
	_ = &recursiveNamedT10{ty}

	t19s := &[10]recursiveNamedT19{}
	_ = recursiveNamedT18(t19s)
}
