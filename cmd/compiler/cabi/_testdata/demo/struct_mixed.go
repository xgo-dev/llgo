package main

import "unsafe"

const (
	LLGoFiles = "../wrap/struct_mixed.c"
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

type point1 struct {
	x0 int8
	x1 int16
}

//go:linkname cdemo1 C.demo1
func cdemo1(a point1) point1

func demo1(a point1) point1 {
	return a
}

func init() {
	a := point1{1, 2}
	assert("cdemo1\000", cdemo1(a) == a)
	assert("demo1\000", demo1(a) == a)
}

type point2 struct {
	x0 int8
	x1 int32
}

//go:linkname cdemo2 C.demo2
func cdemo2(a point2) point2

func demo2(a point2) point2 {
	return a
}

func init() {
	a := point2{1, 2}
	assert("cdemo2\000", cdemo2(a) == a)
	assert("demo2\000", demo2(a) == a)
}

type point3 struct {
	x0 int8
	x1 int64
}

//go:linkname cdemo3 C.demo3
func cdemo3(a point3) point3

func demo3(a point3) point3 {
	return a
}

func init() {
	a := point3{1, 2}
	assert("cdemo3\000", cdemo3(a) == a)
	assert("demo3\000", demo3(a) == a)
}

type point4 struct {
	x0 int8
	x1 float32
}

//go:linkname cdemo4 C.demo4
func cdemo4(a point4) point4

func demo4(a point4) point4 {
	return a
}

func init() {
	a := point4{1, 2}
	assert("cdemo4\000", cdemo4(a) == a)
	assert("demo4\000", demo4(a) == a)
}

type point5 struct {
	x0 int8
	x1 float64
}

//go:linkname cdemo5 C.demo5
func cdemo5(a point5) point5

func demo5(a point5) point5 {
	return a
}

func init() {
	a := point5{1, 2}
	assert("cdemo5\000", cdemo5(a) == a)
	assert("demo5\000", demo5(a) == a)
}

type point6 struct {
	x0 int8
	x1 pointer
}

//go:linkname cdemo6 C.demo6
func cdemo6(a point6) point6

func demo6(a point6) point6 {
	return a
}

func init() {
	a := point6{1, func() pointer { var a int8 = 2; return &a }()}
	assert("cdemo6\000", cdemo6(a) == a)
	assert("demo6\000", demo6(a) == a)
}

type point7 struct {
	x0 int16
	x1 int8
}

//go:linkname cdemo7 C.demo7
func cdemo7(a point7) point7

func demo7(a point7) point7 {
	return a
}

func init() {
	a := point7{1, 2}
	assert("cdemo7\000", cdemo7(a) == a)
	assert("demo7\000", demo7(a) == a)
}

type point8 struct {
	x0 int16
	x1 int32
}

//go:linkname cdemo8 C.demo8
func cdemo8(a point8) point8

func demo8(a point8) point8 {
	return a
}

func init() {
	a := point8{1, 2}
	assert("cdemo8\000", cdemo8(a) == a)
	assert("demo8\000", demo8(a) == a)
}

type point9 struct {
	x0 int16
	x1 int64
}

//go:linkname cdemo9 C.demo9
func cdemo9(a point9) point9

func demo9(a point9) point9 {
	return a
}

func init() {
	a := point9{1, 2}
	assert("cdemo9\000", cdemo9(a) == a)
	assert("demo9\000", demo9(a) == a)
}

type point10 struct {
	x0 int16
	x1 float32
}

//go:linkname cdemo10 C.demo10
func cdemo10(a point10) point10

func demo10(a point10) point10 {
	return a
}

func init() {
	a := point10{1, 2}
	assert("cdemo10\000", cdemo10(a) == a)
	assert("demo10\000", demo10(a) == a)
}

type point11 struct {
	x0 int16
	x1 float64
}

//go:linkname cdemo11 C.demo11
func cdemo11(a point11) point11

func demo11(a point11) point11 {
	return a
}

func init() {
	a := point11{1, 2}
	assert("cdemo11\000", cdemo11(a) == a)
	assert("demo11\000", demo11(a) == a)
}

type point12 struct {
	x0 int16
	x1 pointer
}

//go:linkname cdemo12 C.demo12
func cdemo12(a point12) point12

func demo12(a point12) point12 {
	return a
}

func init() {
	a := point12{1, func() pointer { var a int8 = 2; return &a }()}
	assert("cdemo12\000", cdemo12(a) == a)
	assert("demo12\000", demo12(a) == a)
}

type point13 struct {
	x0 int32
	x1 int8
}

//go:linkname cdemo13 C.demo13
func cdemo13(a point13) point13

func demo13(a point13) point13 {
	return a
}

func init() {
	a := point13{1, 2}
	assert("cdemo13\000", cdemo13(a) == a)
	assert("demo13\000", demo13(a) == a)
}

type point14 struct {
	x0 int32
	x1 int16
}

//go:linkname cdemo14 C.demo14
func cdemo14(a point14) point14

func demo14(a point14) point14 {
	return a
}

func init() {
	a := point14{1, 2}
	assert("cdemo14\000", cdemo14(a) == a)
	assert("demo14\000", demo14(a) == a)
}

type point15 struct {
	x0 int32
	x1 int64
}

//go:linkname cdemo15 C.demo15
func cdemo15(a point15) point15

func demo15(a point15) point15 {
	return a
}

func init() {
	a := point15{1, 2}
	assert("cdemo15\000", cdemo15(a) == a)
	assert("demo15\000", demo15(a) == a)
}

type point16 struct {
	x0 int32
	x1 float32
}

//go:linkname cdemo16 C.demo16
func cdemo16(a point16) point16

func demo16(a point16) point16 {
	return a
}

func init() {
	a := point16{1, 2}
	assert("cdemo16\000", cdemo16(a) == a)
	assert("demo16\000", demo16(a) == a)
}

type point17 struct {
	x0 int32
	x1 float64
}

//go:linkname cdemo17 C.demo17
func cdemo17(a point17) point17

func demo17(a point17) point17 {
	return a
}

func init() {
	a := point17{1, 2}
	assert("cdemo17\000", cdemo17(a) == a)
	assert("demo17\000", demo17(a) == a)
}

type point18 struct {
	x0 int32
	x1 pointer
}

//go:linkname cdemo18 C.demo18
func cdemo18(a point18) point18

func demo18(a point18) point18 {
	return a
}

func init() {
	a := point18{1, func() pointer { var a int8 = 2; return &a }()}
	assert("cdemo18\000", cdemo18(a) == a)
	assert("demo18\000", demo18(a) == a)
}

type point19 struct {
	x0 int64
	x1 int8
}

//go:linkname cdemo19 C.demo19
func cdemo19(a point19) point19

func demo19(a point19) point19 {
	return a
}

func init() {
	a := point19{1, 2}
	assert("cdemo19\000", cdemo19(a) == a)
	assert("demo19\000", demo19(a) == a)
}

type point20 struct {
	x0 int64
	x1 int16
}

//go:linkname cdemo20 C.demo20
func cdemo20(a point20) point20

func demo20(a point20) point20 {
	return a
}

func init() {
	a := point20{1, 2}
	assert("cdemo20\000", cdemo20(a) == a)
	assert("demo20\000", demo20(a) == a)
}

type point21 struct {
	x0 int64
	x1 int32
}

//go:linkname cdemo21 C.demo21
func cdemo21(a point21) point21

func demo21(a point21) point21 {
	return a
}

func init() {
	a := point21{1, 2}
	assert("cdemo21\000", cdemo21(a) == a)
	assert("demo21\000", demo21(a) == a)
}

type point22 struct {
	x0 int64
	x1 float32
}

//go:linkname cdemo22 C.demo22
func cdemo22(a point22) point22

func demo22(a point22) point22 {
	return a
}

func init() {
	a := point22{1, 2}
	assert("cdemo22\000", cdemo22(a) == a)
	assert("demo22\000", demo22(a) == a)
}

type point23 struct {
	x0 int64
	x1 float64
}

//go:linkname cdemo23 C.demo23
func cdemo23(a point23) point23

func demo23(a point23) point23 {
	return a
}

func init() {
	a := point23{1, 2}
	assert("cdemo23\000", cdemo23(a) == a)
	assert("demo23\000", demo23(a) == a)
}

type point24 struct {
	x0 int64
	x1 pointer
}

//go:linkname cdemo24 C.demo24
func cdemo24(a point24) point24

func demo24(a point24) point24 {
	return a
}

func init() {
	a := point24{1, func() pointer { var a int8 = 2; return &a }()}
	assert("cdemo24\000", cdemo24(a) == a)
	assert("demo24\000", demo24(a) == a)
}

type point25 struct {
	x0 float32
	x1 int8
}

//go:linkname cdemo25 C.demo25
func cdemo25(a point25) point25

func demo25(a point25) point25 {
	return a
}

func init() {
	a := point25{1, 2}
	assert("cdemo25\000", cdemo25(a) == a)
	assert("demo25\000", demo25(a) == a)
}

type point26 struct {
	x0 float32
	x1 int16
}

//go:linkname cdemo26 C.demo26
func cdemo26(a point26) point26

func demo26(a point26) point26 {
	return a
}

func init() {
	a := point26{1, 2}
	assert("cdemo26\000", cdemo26(a) == a)
	assert("demo26\000", demo26(a) == a)
}

type point27 struct {
	x0 float32
	x1 int32
}

//go:linkname cdemo27 C.demo27
func cdemo27(a point27) point27

func demo27(a point27) point27 {
	return a
}

func init() {
	a := point27{1, 2}
	assert("cdemo27\000", cdemo27(a) == a)
	assert("demo27\000", demo27(a) == a)
}

type point28 struct {
	x0 float32
	x1 int64
}

//go:linkname cdemo28 C.demo28
func cdemo28(a point28) point28

func demo28(a point28) point28 {
	return a
}

func init() {
	a := point28{1, 2}
	assert("cdemo28\000", cdemo28(a) == a)
	assert("demo28\000", demo28(a) == a)
}

type point29 struct {
	x0 float32
	x1 float64
}

//go:linkname cdemo29 C.demo29
func cdemo29(a point29) point29

func demo29(a point29) point29 {
	return a
}

func init() {
	a := point29{1, 2}
	assert("cdemo29\000", cdemo29(a) == a)
	assert("demo29\000", demo29(a) == a)
}

type point30 struct {
	x0 float32
	x1 pointer
}

//go:linkname cdemo30 C.demo30
func cdemo30(a point30) point30

func demo30(a point30) point30 {
	return a
}

func init() {
	a := point30{1, func() pointer { var a int8 = 2; return &a }()}
	assert("cdemo30\000", cdemo30(a) == a)
	assert("demo30\000", demo30(a) == a)
}

type point31 struct {
	x0 float64
	x1 int8
}

//go:linkname cdemo31 C.demo31
func cdemo31(a point31) point31

func demo31(a point31) point31 {
	return a
}

func init() {
	a := point31{1, 2}
	assert("cdemo31\000", cdemo31(a) == a)
	assert("demo31\000", demo31(a) == a)
}

type point32 struct {
	x0 float64
	x1 int16
}

//go:linkname cdemo32 C.demo32
func cdemo32(a point32) point32

func demo32(a point32) point32 {
	return a
}

func init() {
	a := point32{1, 2}
	assert("cdemo32\000", cdemo32(a) == a)
	assert("demo32\000", demo32(a) == a)
}

type point33 struct {
	x0 float64
	x1 int32
}

//go:linkname cdemo33 C.demo33
func cdemo33(a point33) point33

func demo33(a point33) point33 {
	return a
}

func init() {
	a := point33{1, 2}
	assert("cdemo33\000", cdemo33(a) == a)
	assert("demo33\000", demo33(a) == a)
}

type point34 struct {
	x0 float64
	x1 int64
}

//go:linkname cdemo34 C.demo34
func cdemo34(a point34) point34

func demo34(a point34) point34 {
	return a
}

func init() {
	a := point34{1, 2}
	assert("cdemo34\000", cdemo34(a) == a)
	assert("demo34\000", demo34(a) == a)
}

type point35 struct {
	x0 float64
	x1 float32
}

//go:linkname cdemo35 C.demo35
func cdemo35(a point35) point35

func demo35(a point35) point35 {
	return a
}

func init() {
	a := point35{1, 2}
	assert("cdemo35\000", cdemo35(a) == a)
	assert("demo35\000", demo35(a) == a)
}

type point36 struct {
	x0 float64
	x1 pointer
}

//go:linkname cdemo36 C.demo36
func cdemo36(a point36) point36

func demo36(a point36) point36 {
	return a
}

func init() {
	a := point36{1, func() pointer { var a int8 = 2; return &a }()}
	assert("cdemo36\000", cdemo36(a) == a)
	assert("demo36\000", demo36(a) == a)
}

type point37 struct {
	x0 pointer
	x1 int8
}

//go:linkname cdemo37 C.demo37
func cdemo37(a point37) point37

func demo37(a point37) point37 {
	return a
}

func init() {
	a := point37{func() pointer { var a int8 = 1; return &a }(), 2}
	assert("cdemo37\000", cdemo37(a) == a)
	assert("demo37\000", demo37(a) == a)
}

type point38 struct {
	x0 pointer
	x1 int16
}

//go:linkname cdemo38 C.demo38
func cdemo38(a point38) point38

func demo38(a point38) point38 {
	return a
}

func init() {
	a := point38{func() pointer { var a int8 = 1; return &a }(), 2}
	assert("cdemo38\000", cdemo38(a) == a)
	assert("demo38\000", demo38(a) == a)
}

type point39 struct {
	x0 pointer
	x1 int32
}

//go:linkname cdemo39 C.demo39
func cdemo39(a point39) point39

func demo39(a point39) point39 {
	return a
}

func init() {
	a := point39{func() pointer { var a int8 = 1; return &a }(), 2}
	assert("cdemo39\000", cdemo39(a) == a)
	assert("demo39\000", demo39(a) == a)
}

type point40 struct {
	x0 pointer
	x1 int64
}

//go:linkname cdemo40 C.demo40
func cdemo40(a point40) point40

func demo40(a point40) point40 {
	return a
}

func init() {
	a := point40{func() pointer { var a int8 = 1; return &a }(), 2}
	assert("cdemo40\000", cdemo40(a) == a)
	assert("demo40\000", demo40(a) == a)
}

type point41 struct {
	x0 pointer
	x1 float32
}

//go:linkname cdemo41 C.demo41
func cdemo41(a point41) point41

func demo41(a point41) point41 {
	return a
}

func init() {
	a := point41{func() pointer { var a int8 = 1; return &a }(), 2}
	assert("cdemo41\000", cdemo41(a) == a)
	assert("demo41\000", demo41(a) == a)
}

type point42 struct {
	x0 pointer
	x1 float64
}

//go:linkname cdemo42 C.demo42
func cdemo42(a point42) point42

func demo42(a point42) point42 {
	return a
}

func init() {
	a := point42{func() pointer { var a int8 = 1; return &a }(), 2}
	assert("cdemo42\000", cdemo42(a) == a)
	assert("demo42\000", demo42(a) == a)
}

type point43 struct {
	x0 int8
	x1 int16
	x2 int8
}

//go:linkname cdemo43 C.demo43
func cdemo43(a point43) point43

func demo43(a point43) point43 {
	return a
}

func init() {
	a := point43{1, 2, 3}
	assert("cdemo43\000", cdemo43(a) == a)
	assert("demo43\000", demo43(a) == a)
}

type point44 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int8
}

//go:linkname cdemo44 C.demo44
func cdemo44(a point44) point44

func demo44(a point44) point44 {
	return a
}

func init() {
	a := point44{1, 2, 3, 4}
	assert("cdemo44\000", cdemo44(a) == a)
	assert("demo44\000", demo44(a) == a)
}

type point45 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo45 C.demo45
func cdemo45(a point45) point45

func demo45(a point45) point45 {
	return a
}

func init() {
	a := point45{1, 2, 3, 4, 5}
	assert("cdemo45\000", cdemo45(a) == a)
	assert("demo45\000", demo45(a) == a)
}

type point46 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo46 C.demo46
func cdemo46(a point46) point46

func demo46(a point46) point46 {
	return a
}

func init() {
	a := point46{1, 2, 3, 4, 5}
	assert("cdemo46\000", cdemo46(a) == a)
	assert("demo46\000", demo46(a) == a)
}

type point47 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo47 C.demo47
func cdemo47(a point47) point47

func demo47(a point47) point47 {
	return a
}

func init() {
	a := point47{1, 2, 3, 4, 5}
	assert("cdemo47\000", cdemo47(a) == a)
	assert("demo47\000", demo47(a) == a)
}

type point48 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo48 C.demo48
func cdemo48(a point48) point48

func demo48(a point48) point48 {
	return a
}

func init() {
	a := point48{1, 2, 3, 4, 5}
	assert("cdemo48\000", cdemo48(a) == a)
	assert("demo48\000", demo48(a) == a)
}

type point49 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int16
}

//go:linkname cdemo49 C.demo49
func cdemo49(a point49) point49

func demo49(a point49) point49 {
	return a
}

func init() {
	a := point49{1, 2, 3, 4}
	assert("cdemo49\000", cdemo49(a) == a)
	assert("demo49\000", demo49(a) == a)
}

type point50 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo50 C.demo50
func cdemo50(a point50) point50

func demo50(a point50) point50 {
	return a
}

func init() {
	a := point50{1, 2, 3, 4, 5}
	assert("cdemo50\000", cdemo50(a) == a)
	assert("demo50\000", demo50(a) == a)
}

type point51 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo51 C.demo51
func cdemo51(a point51) point51

func demo51(a point51) point51 {
	return a
}

func init() {
	a := point51{1, 2, 3, 4, 5}
	assert("cdemo51\000", cdemo51(a) == a)
	assert("demo51\000", demo51(a) == a)
}

type point52 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo52 C.demo52
func cdemo52(a point52) point52

func demo52(a point52) point52 {
	return a
}

func init() {
	a := point52{1, 2, 3, 4, 5}
	assert("cdemo52\000", cdemo52(a) == a)
	assert("demo52\000", demo52(a) == a)
}

type point53 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo53 C.demo53
func cdemo53(a point53) point53

func demo53(a point53) point53 {
	return a
}

func init() {
	a := point53{1, 2, 3, 4, 5}
	assert("cdemo53\000", cdemo53(a) == a)
	assert("demo53\000", demo53(a) == a)
}

type point54 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int32
}

//go:linkname cdemo54 C.demo54
func cdemo54(a point54) point54

func demo54(a point54) point54 {
	return a
}

func init() {
	a := point54{1, 2, 3, 4}
	assert("cdemo54\000", cdemo54(a) == a)
	assert("demo54\000", demo54(a) == a)
}

type point55 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo55 C.demo55
func cdemo55(a point55) point55

func demo55(a point55) point55 {
	return a
}

func init() {
	a := point55{1, 2, 3, 4, 5}
	assert("cdemo55\000", cdemo55(a) == a)
	assert("demo55\000", demo55(a) == a)
}

type point56 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo56 C.demo56
func cdemo56(a point56) point56

func demo56(a point56) point56 {
	return a
}

func init() {
	a := point56{1, 2, 3, 4, 5}
	assert("cdemo56\000", cdemo56(a) == a)
	assert("demo56\000", demo56(a) == a)
}

type point57 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo57 C.demo57
func cdemo57(a point57) point57

func demo57(a point57) point57 {
	return a
}

func init() {
	a := point57{1, 2, 3, 4, 5}
	assert("cdemo57\000", cdemo57(a) == a)
	assert("demo57\000", demo57(a) == a)
}

type point58 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo58 C.demo58
func cdemo58(a point58) point58

func demo58(a point58) point58 {
	return a
}

func init() {
	a := point58{1, 2, 3, 4, 5}
	assert("cdemo58\000", cdemo58(a) == a)
	assert("demo58\000", demo58(a) == a)
}

type point59 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 float32
}

//go:linkname cdemo59 C.demo59
func cdemo59(a point59) point59

func demo59(a point59) point59 {
	return a
}

func init() {
	a := point59{1, 2, 3, 4}
	assert("cdemo59\000", cdemo59(a) == a)
	assert("demo59\000", demo59(a) == a)
}

type point60 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo60 C.demo60
func cdemo60(a point60) point60

func demo60(a point60) point60 {
	return a
}

func init() {
	a := point60{1, 2, 3, 4, 5}
	assert("cdemo60\000", cdemo60(a) == a)
	assert("demo60\000", demo60(a) == a)
}

type point61 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo61 C.demo61
func cdemo61(a point61) point61

func demo61(a point61) point61 {
	return a
}

func init() {
	a := point61{1, 2, 3, 4, 5}
	assert("cdemo61\000", cdemo61(a) == a)
	assert("demo61\000", demo61(a) == a)
}

type point62 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo62 C.demo62
func cdemo62(a point62) point62

func demo62(a point62) point62 {
	return a
}

func init() {
	a := point62{1, 2, 3, 4, 5}
	assert("cdemo62\000", cdemo62(a) == a)
	assert("demo62\000", demo62(a) == a)
}

type point63 struct {
	x0 int8
	x1 int16
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo63 C.demo63
func cdemo63(a point63) point63

func demo63(a point63) point63 {
	return a
}

func init() {
	a := point63{1, 2, 3, 4, 5}
	assert("cdemo63\000", cdemo63(a) == a)
	assert("demo63\000", demo63(a) == a)
}

type point64 struct {
	x0 int8
	x1 int16
	x2 int16
}

//go:linkname cdemo64 C.demo64
func cdemo64(a point64) point64

func demo64(a point64) point64 {
	return a
}

func init() {
	a := point64{1, 2, 3}
	assert("cdemo64\000", cdemo64(a) == a)
	assert("demo64\000", demo64(a) == a)
}

type point65 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int8
}

//go:linkname cdemo65 C.demo65
func cdemo65(a point65) point65

func demo65(a point65) point65 {
	return a
}

func init() {
	a := point65{1, 2, 3, 4}
	assert("cdemo65\000", cdemo65(a) == a)
	assert("demo65\000", demo65(a) == a)
}

type point66 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo66 C.demo66
func cdemo66(a point66) point66

func demo66(a point66) point66 {
	return a
}

func init() {
	a := point66{1, 2, 3, 4, 5}
	assert("cdemo66\000", cdemo66(a) == a)
	assert("demo66\000", demo66(a) == a)
}

type point67 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo67 C.demo67
func cdemo67(a point67) point67

func demo67(a point67) point67 {
	return a
}

func init() {
	a := point67{1, 2, 3, 4, 5}
	assert("cdemo67\000", cdemo67(a) == a)
	assert("demo67\000", demo67(a) == a)
}

type point68 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo68 C.demo68
func cdemo68(a point68) point68

func demo68(a point68) point68 {
	return a
}

func init() {
	a := point68{1, 2, 3, 4, 5}
	assert("cdemo68\000", cdemo68(a) == a)
	assert("demo68\000", demo68(a) == a)
}

type point69 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo69 C.demo69
func cdemo69(a point69) point69

func demo69(a point69) point69 {
	return a
}

func init() {
	a := point69{1, 2, 3, 4, 5}
	assert("cdemo69\000", cdemo69(a) == a)
	assert("demo69\000", demo69(a) == a)
}

type point70 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int16
}

//go:linkname cdemo70 C.demo70
func cdemo70(a point70) point70

func demo70(a point70) point70 {
	return a
}

func init() {
	a := point70{1, 2, 3, 4}
	assert("cdemo70\000", cdemo70(a) == a)
	assert("demo70\000", demo70(a) == a)
}

type point71 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo71 C.demo71
func cdemo71(a point71) point71

func demo71(a point71) point71 {
	return a
}

func init() {
	a := point71{1, 2, 3, 4, 5}
	assert("cdemo71\000", cdemo71(a) == a)
	assert("demo71\000", demo71(a) == a)
}

type point72 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo72 C.demo72
func cdemo72(a point72) point72

func demo72(a point72) point72 {
	return a
}

func init() {
	a := point72{1, 2, 3, 4, 5}
	assert("cdemo72\000", cdemo72(a) == a)
	assert("demo72\000", demo72(a) == a)
}

type point73 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo73 C.demo73
func cdemo73(a point73) point73

func demo73(a point73) point73 {
	return a
}

func init() {
	a := point73{1, 2, 3, 4, 5}
	assert("cdemo73\000", cdemo73(a) == a)
	assert("demo73\000", demo73(a) == a)
}

type point74 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo74 C.demo74
func cdemo74(a point74) point74

func demo74(a point74) point74 {
	return a
}

func init() {
	a := point74{1, 2, 3, 4, 5}
	assert("cdemo74\000", cdemo74(a) == a)
	assert("demo74\000", demo74(a) == a)
}

type point75 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int32
}

//go:linkname cdemo75 C.demo75
func cdemo75(a point75) point75

func demo75(a point75) point75 {
	return a
}

func init() {
	a := point75{1, 2, 3, 4}
	assert("cdemo75\000", cdemo75(a) == a)
	assert("demo75\000", demo75(a) == a)
}

type point76 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo76 C.demo76
func cdemo76(a point76) point76

func demo76(a point76) point76 {
	return a
}

func init() {
	a := point76{1, 2, 3, 4, 5}
	assert("cdemo76\000", cdemo76(a) == a)
	assert("demo76\000", demo76(a) == a)
}

type point77 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo77 C.demo77
func cdemo77(a point77) point77

func demo77(a point77) point77 {
	return a
}

func init() {
	a := point77{1, 2, 3, 4, 5}
	assert("cdemo77\000", cdemo77(a) == a)
	assert("demo77\000", demo77(a) == a)
}

type point78 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo78 C.demo78
func cdemo78(a point78) point78

func demo78(a point78) point78 {
	return a
}

func init() {
	a := point78{1, 2, 3, 4, 5}
	assert("cdemo78\000", cdemo78(a) == a)
	assert("demo78\000", demo78(a) == a)
}

type point79 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo79 C.demo79
func cdemo79(a point79) point79

func demo79(a point79) point79 {
	return a
}

func init() {
	a := point79{1, 2, 3, 4, 5}
	assert("cdemo79\000", cdemo79(a) == a)
	assert("demo79\000", demo79(a) == a)
}

type point80 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 float32
}

//go:linkname cdemo80 C.demo80
func cdemo80(a point80) point80

func demo80(a point80) point80 {
	return a
}

func init() {
	a := point80{1, 2, 3, 4}
	assert("cdemo80\000", cdemo80(a) == a)
	assert("demo80\000", demo80(a) == a)
}

type point81 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo81 C.demo81
func cdemo81(a point81) point81

func demo81(a point81) point81 {
	return a
}

func init() {
	a := point81{1, 2, 3, 4, 5}
	assert("cdemo81\000", cdemo81(a) == a)
	assert("demo81\000", demo81(a) == a)
}

type point82 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo82 C.demo82
func cdemo82(a point82) point82

func demo82(a point82) point82 {
	return a
}

func init() {
	a := point82{1, 2, 3, 4, 5}
	assert("cdemo82\000", cdemo82(a) == a)
	assert("demo82\000", demo82(a) == a)
}

type point83 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo83 C.demo83
func cdemo83(a point83) point83

func demo83(a point83) point83 {
	return a
}

func init() {
	a := point83{1, 2, 3, 4, 5}
	assert("cdemo83\000", cdemo83(a) == a)
	assert("demo83\000", demo83(a) == a)
}

type point84 struct {
	x0 int8
	x1 int16
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo84 C.demo84
func cdemo84(a point84) point84

func demo84(a point84) point84 {
	return a
}

func init() {
	a := point84{1, 2, 3, 4, 5}
	assert("cdemo84\000", cdemo84(a) == a)
	assert("demo84\000", demo84(a) == a)
}

type point85 struct {
	x0 int8
	x1 int16
	x2 int32
}

//go:linkname cdemo85 C.demo85
func cdemo85(a point85) point85

func demo85(a point85) point85 {
	return a
}

func init() {
	a := point85{1, 2, 3}
	assert("cdemo85\000", cdemo85(a) == a)
	assert("demo85\000", demo85(a) == a)
}

type point86 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int8
}

//go:linkname cdemo86 C.demo86
func cdemo86(a point86) point86

func demo86(a point86) point86 {
	return a
}

func init() {
	a := point86{1, 2, 3, 4}
	assert("cdemo86\000", cdemo86(a) == a)
	assert("demo86\000", demo86(a) == a)
}

type point87 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo87 C.demo87
func cdemo87(a point87) point87

func demo87(a point87) point87 {
	return a
}

func init() {
	a := point87{1, 2, 3, 4, 5}
	assert("cdemo87\000", cdemo87(a) == a)
	assert("demo87\000", demo87(a) == a)
}

type point88 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo88 C.demo88
func cdemo88(a point88) point88

func demo88(a point88) point88 {
	return a
}

func init() {
	a := point88{1, 2, 3, 4, 5}
	assert("cdemo88\000", cdemo88(a) == a)
	assert("demo88\000", demo88(a) == a)
}

type point89 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo89 C.demo89
func cdemo89(a point89) point89

func demo89(a point89) point89 {
	return a
}

func init() {
	a := point89{1, 2, 3, 4, 5}
	assert("cdemo89\000", cdemo89(a) == a)
	assert("demo89\000", demo89(a) == a)
}

type point90 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo90 C.demo90
func cdemo90(a point90) point90

func demo90(a point90) point90 {
	return a
}

func init() {
	a := point90{1, 2, 3, 4, 5}
	assert("cdemo90\000", cdemo90(a) == a)
	assert("demo90\000", demo90(a) == a)
}

type point91 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int16
}

//go:linkname cdemo91 C.demo91
func cdemo91(a point91) point91

func demo91(a point91) point91 {
	return a
}

func init() {
	a := point91{1, 2, 3, 4}
	assert("cdemo91\000", cdemo91(a) == a)
	assert("demo91\000", demo91(a) == a)
}

type point92 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo92 C.demo92
func cdemo92(a point92) point92

func demo92(a point92) point92 {
	return a
}

func init() {
	a := point92{1, 2, 3, 4, 5}
	assert("cdemo92\000", cdemo92(a) == a)
	assert("demo92\000", demo92(a) == a)
}

type point93 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo93 C.demo93
func cdemo93(a point93) point93

func demo93(a point93) point93 {
	return a
}

func init() {
	a := point93{1, 2, 3, 4, 5}
	assert("cdemo93\000", cdemo93(a) == a)
	assert("demo93\000", demo93(a) == a)
}

type point94 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo94 C.demo94
func cdemo94(a point94) point94

func demo94(a point94) point94 {
	return a
}

func init() {
	a := point94{1, 2, 3, 4, 5}
	assert("cdemo94\000", cdemo94(a) == a)
	assert("demo94\000", demo94(a) == a)
}

type point95 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo95 C.demo95
func cdemo95(a point95) point95

func demo95(a point95) point95 {
	return a
}

func init() {
	a := point95{1, 2, 3, 4, 5}
	assert("cdemo95\000", cdemo95(a) == a)
	assert("demo95\000", demo95(a) == a)
}

type point96 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int32
}

//go:linkname cdemo96 C.demo96
func cdemo96(a point96) point96

func demo96(a point96) point96 {
	return a
}

func init() {
	a := point96{1, 2, 3, 4}
	assert("cdemo96\000", cdemo96(a) == a)
	assert("demo96\000", demo96(a) == a)
}

type point97 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo97 C.demo97
func cdemo97(a point97) point97

func demo97(a point97) point97 {
	return a
}

func init() {
	a := point97{1, 2, 3, 4, 5}
	assert("cdemo97\000", cdemo97(a) == a)
	assert("demo97\000", demo97(a) == a)
}

type point98 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo98 C.demo98
func cdemo98(a point98) point98

func demo98(a point98) point98 {
	return a
}

func init() {
	a := point98{1, 2, 3, 4, 5}
	assert("cdemo98\000", cdemo98(a) == a)
	assert("demo98\000", demo98(a) == a)
}

type point99 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo99 C.demo99
func cdemo99(a point99) point99

func demo99(a point99) point99 {
	return a
}

func init() {
	a := point99{1, 2, 3, 4, 5}
	assert("cdemo99\000", cdemo99(a) == a)
	assert("demo99\000", demo99(a) == a)
}

type point100 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo100 C.demo100
func cdemo100(a point100) point100

func demo100(a point100) point100 {
	return a
}

func init() {
	a := point100{1, 2, 3, 4, 5}
	assert("cdemo100\000", cdemo100(a) == a)
	assert("demo100\000", demo100(a) == a)
}

type point101 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 float32
}

//go:linkname cdemo101 C.demo101
func cdemo101(a point101) point101

func demo101(a point101) point101 {
	return a
}

func init() {
	a := point101{1, 2, 3, 4}
	assert("cdemo101\000", cdemo101(a) == a)
	assert("demo101\000", demo101(a) == a)
}

type point102 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo102 C.demo102
func cdemo102(a point102) point102

func demo102(a point102) point102 {
	return a
}

func init() {
	a := point102{1, 2, 3, 4, 5}
	assert("cdemo102\000", cdemo102(a) == a)
	assert("demo102\000", demo102(a) == a)
}

type point103 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo103 C.demo103
func cdemo103(a point103) point103

func demo103(a point103) point103 {
	return a
}

func init() {
	a := point103{1, 2, 3, 4, 5}
	assert("cdemo103\000", cdemo103(a) == a)
	assert("demo103\000", demo103(a) == a)
}

type point104 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo104 C.demo104
func cdemo104(a point104) point104

func demo104(a point104) point104 {
	return a
}

func init() {
	a := point104{1, 2, 3, 4, 5}
	assert("cdemo104\000", cdemo104(a) == a)
	assert("demo104\000", demo104(a) == a)
}

type point105 struct {
	x0 int8
	x1 int16
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo105 C.demo105
func cdemo105(a point105) point105

func demo105(a point105) point105 {
	return a
}

func init() {
	a := point105{1, 2, 3, 4, 5}
	assert("cdemo105\000", cdemo105(a) == a)
	assert("demo105\000", demo105(a) == a)
}

type point106 struct {
	x0 int8
	x1 int16
	x2 float32
}

//go:linkname cdemo106 C.demo106
func cdemo106(a point106) point106

func demo106(a point106) point106 {
	return a
}

func init() {
	a := point106{1, 2, 3}
	assert("cdemo106\000", cdemo106(a) == a)
	assert("demo106\000", demo106(a) == a)
}

type point107 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int8
}

//go:linkname cdemo107 C.demo107
func cdemo107(a point107) point107

func demo107(a point107) point107 {
	return a
}

func init() {
	a := point107{1, 2, 3, 4}
	assert("cdemo107\000", cdemo107(a) == a)
	assert("demo107\000", demo107(a) == a)
}

type point108 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo108 C.demo108
func cdemo108(a point108) point108

func demo108(a point108) point108 {
	return a
}

func init() {
	a := point108{1, 2, 3, 4, 5}
	assert("cdemo108\000", cdemo108(a) == a)
	assert("demo108\000", demo108(a) == a)
}

type point109 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo109 C.demo109
func cdemo109(a point109) point109

func demo109(a point109) point109 {
	return a
}

func init() {
	a := point109{1, 2, 3, 4, 5}
	assert("cdemo109\000", cdemo109(a) == a)
	assert("demo109\000", demo109(a) == a)
}

type point110 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo110 C.demo110
func cdemo110(a point110) point110

func demo110(a point110) point110 {
	return a
}

func init() {
	a := point110{1, 2, 3, 4, 5}
	assert("cdemo110\000", cdemo110(a) == a)
	assert("demo110\000", demo110(a) == a)
}

type point111 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo111 C.demo111
func cdemo111(a point111) point111

func demo111(a point111) point111 {
	return a
}

func init() {
	a := point111{1, 2, 3, 4, 5}
	assert("cdemo111\000", cdemo111(a) == a)
	assert("demo111\000", demo111(a) == a)
}

type point112 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int16
}

//go:linkname cdemo112 C.demo112
func cdemo112(a point112) point112

func demo112(a point112) point112 {
	return a
}

func init() {
	a := point112{1, 2, 3, 4}
	assert("cdemo112\000", cdemo112(a) == a)
	assert("demo112\000", demo112(a) == a)
}

type point113 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo113 C.demo113
func cdemo113(a point113) point113

func demo113(a point113) point113 {
	return a
}

func init() {
	a := point113{1, 2, 3, 4, 5}
	assert("cdemo113\000", cdemo113(a) == a)
	assert("demo113\000", demo113(a) == a)
}

type point114 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo114 C.demo114
func cdemo114(a point114) point114

func demo114(a point114) point114 {
	return a
}

func init() {
	a := point114{1, 2, 3, 4, 5}
	assert("cdemo114\000", cdemo114(a) == a)
	assert("demo114\000", demo114(a) == a)
}

type point115 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo115 C.demo115
func cdemo115(a point115) point115

func demo115(a point115) point115 {
	return a
}

func init() {
	a := point115{1, 2, 3, 4, 5}
	assert("cdemo115\000", cdemo115(a) == a)
	assert("demo115\000", demo115(a) == a)
}

type point116 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo116 C.demo116
func cdemo116(a point116) point116

func demo116(a point116) point116 {
	return a
}

func init() {
	a := point116{1, 2, 3, 4, 5}
	assert("cdemo116\000", cdemo116(a) == a)
	assert("demo116\000", demo116(a) == a)
}

type point117 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int32
}

//go:linkname cdemo117 C.demo117
func cdemo117(a point117) point117

func demo117(a point117) point117 {
	return a
}

func init() {
	a := point117{1, 2, 3, 4}
	assert("cdemo117\000", cdemo117(a) == a)
	assert("demo117\000", demo117(a) == a)
}

type point118 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo118 C.demo118
func cdemo118(a point118) point118

func demo118(a point118) point118 {
	return a
}

func init() {
	a := point118{1, 2, 3, 4, 5}
	assert("cdemo118\000", cdemo118(a) == a)
	assert("demo118\000", demo118(a) == a)
}

type point119 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo119 C.demo119
func cdemo119(a point119) point119

func demo119(a point119) point119 {
	return a
}

func init() {
	a := point119{1, 2, 3, 4, 5}
	assert("cdemo119\000", cdemo119(a) == a)
	assert("demo119\000", demo119(a) == a)
}

type point120 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo120 C.demo120
func cdemo120(a point120) point120

func demo120(a point120) point120 {
	return a
}

func init() {
	a := point120{1, 2, 3, 4, 5}
	assert("cdemo120\000", cdemo120(a) == a)
	assert("demo120\000", demo120(a) == a)
}

type point121 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo121 C.demo121
func cdemo121(a point121) point121

func demo121(a point121) point121 {
	return a
}

func init() {
	a := point121{1, 2, 3, 4, 5}
	assert("cdemo121\000", cdemo121(a) == a)
	assert("demo121\000", demo121(a) == a)
}

type point122 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 float32
}

//go:linkname cdemo122 C.demo122
func cdemo122(a point122) point122

func demo122(a point122) point122 {
	return a
}

func init() {
	a := point122{1, 2, 3, 4}
	assert("cdemo122\000", cdemo122(a) == a)
	assert("demo122\000", demo122(a) == a)
}

type point123 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo123 C.demo123
func cdemo123(a point123) point123

func demo123(a point123) point123 {
	return a
}

func init() {
	a := point123{1, 2, 3, 4, 5}
	assert("cdemo123\000", cdemo123(a) == a)
	assert("demo123\000", demo123(a) == a)
}

type point124 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo124 C.demo124
func cdemo124(a point124) point124

func demo124(a point124) point124 {
	return a
}

func init() {
	a := point124{1, 2, 3, 4, 5}
	assert("cdemo124\000", cdemo124(a) == a)
	assert("demo124\000", demo124(a) == a)
}

type point125 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo125 C.demo125
func cdemo125(a point125) point125

func demo125(a point125) point125 {
	return a
}

func init() {
	a := point125{1, 2, 3, 4, 5}
	assert("cdemo125\000", cdemo125(a) == a)
	assert("demo125\000", demo125(a) == a)
}

type point126 struct {
	x0 int8
	x1 int16
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo126 C.demo126
func cdemo126(a point126) point126

func demo126(a point126) point126 {
	return a
}

func init() {
	a := point126{1, 2, 3, 4, 5}
	assert("cdemo126\000", cdemo126(a) == a)
	assert("demo126\000", demo126(a) == a)
}

type point127 struct {
	x0 int8
	x1 int32
	x2 int8
}

//go:linkname cdemo127 C.demo127
func cdemo127(a point127) point127

func demo127(a point127) point127 {
	return a
}

func init() {
	a := point127{1, 2, 3}
	assert("cdemo127\000", cdemo127(a) == a)
	assert("demo127\000", demo127(a) == a)
}

type point128 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int8
}

//go:linkname cdemo128 C.demo128
func cdemo128(a point128) point128

func demo128(a point128) point128 {
	return a
}

func init() {
	a := point128{1, 2, 3, 4}
	assert("cdemo128\000", cdemo128(a) == a)
	assert("demo128\000", demo128(a) == a)
}

type point129 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo129 C.demo129
func cdemo129(a point129) point129

func demo129(a point129) point129 {
	return a
}

func init() {
	a := point129{1, 2, 3, 4, 5}
	assert("cdemo129\000", cdemo129(a) == a)
	assert("demo129\000", demo129(a) == a)
}

type point130 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo130 C.demo130
func cdemo130(a point130) point130

func demo130(a point130) point130 {
	return a
}

func init() {
	a := point130{1, 2, 3, 4, 5}
	assert("cdemo130\000", cdemo130(a) == a)
	assert("demo130\000", demo130(a) == a)
}

type point131 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo131 C.demo131
func cdemo131(a point131) point131

func demo131(a point131) point131 {
	return a
}

func init() {
	a := point131{1, 2, 3, 4, 5}
	assert("cdemo131\000", cdemo131(a) == a)
	assert("demo131\000", demo131(a) == a)
}

type point132 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo132 C.demo132
func cdemo132(a point132) point132

func demo132(a point132) point132 {
	return a
}

func init() {
	a := point132{1, 2, 3, 4, 5}
	assert("cdemo132\000", cdemo132(a) == a)
	assert("demo132\000", demo132(a) == a)
}

type point133 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int16
}

//go:linkname cdemo133 C.demo133
func cdemo133(a point133) point133

func demo133(a point133) point133 {
	return a
}

func init() {
	a := point133{1, 2, 3, 4}
	assert("cdemo133\000", cdemo133(a) == a)
	assert("demo133\000", demo133(a) == a)
}

type point134 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo134 C.demo134
func cdemo134(a point134) point134

func demo134(a point134) point134 {
	return a
}

func init() {
	a := point134{1, 2, 3, 4, 5}
	assert("cdemo134\000", cdemo134(a) == a)
	assert("demo134\000", demo134(a) == a)
}

type point135 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo135 C.demo135
func cdemo135(a point135) point135

func demo135(a point135) point135 {
	return a
}

func init() {
	a := point135{1, 2, 3, 4, 5}
	assert("cdemo135\000", cdemo135(a) == a)
	assert("demo135\000", demo135(a) == a)
}

type point136 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo136 C.demo136
func cdemo136(a point136) point136

func demo136(a point136) point136 {
	return a
}

func init() {
	a := point136{1, 2, 3, 4, 5}
	assert("cdemo136\000", cdemo136(a) == a)
	assert("demo136\000", demo136(a) == a)
}

type point137 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo137 C.demo137
func cdemo137(a point137) point137

func demo137(a point137) point137 {
	return a
}

func init() {
	a := point137{1, 2, 3, 4, 5}
	assert("cdemo137\000", cdemo137(a) == a)
	assert("demo137\000", demo137(a) == a)
}

type point138 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int32
}

//go:linkname cdemo138 C.demo138
func cdemo138(a point138) point138

func demo138(a point138) point138 {
	return a
}

func init() {
	a := point138{1, 2, 3, 4}
	assert("cdemo138\000", cdemo138(a) == a)
	assert("demo138\000", demo138(a) == a)
}

type point139 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo139 C.demo139
func cdemo139(a point139) point139

func demo139(a point139) point139 {
	return a
}

func init() {
	a := point139{1, 2, 3, 4, 5}
	assert("cdemo139\000", cdemo139(a) == a)
	assert("demo139\000", demo139(a) == a)
}

type point140 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo140 C.demo140
func cdemo140(a point140) point140

func demo140(a point140) point140 {
	return a
}

func init() {
	a := point140{1, 2, 3, 4, 5}
	assert("cdemo140\000", cdemo140(a) == a)
	assert("demo140\000", demo140(a) == a)
}

type point141 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo141 C.demo141
func cdemo141(a point141) point141

func demo141(a point141) point141 {
	return a
}

func init() {
	a := point141{1, 2, 3, 4, 5}
	assert("cdemo141\000", cdemo141(a) == a)
	assert("demo141\000", demo141(a) == a)
}

type point142 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo142 C.demo142
func cdemo142(a point142) point142

func demo142(a point142) point142 {
	return a
}

func init() {
	a := point142{1, 2, 3, 4, 5}
	assert("cdemo142\000", cdemo142(a) == a)
	assert("demo142\000", demo142(a) == a)
}

type point143 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 float32
}

//go:linkname cdemo143 C.demo143
func cdemo143(a point143) point143

func demo143(a point143) point143 {
	return a
}

func init() {
	a := point143{1, 2, 3, 4}
	assert("cdemo143\000", cdemo143(a) == a)
	assert("demo143\000", demo143(a) == a)
}

type point144 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo144 C.demo144
func cdemo144(a point144) point144

func demo144(a point144) point144 {
	return a
}

func init() {
	a := point144{1, 2, 3, 4, 5}
	assert("cdemo144\000", cdemo144(a) == a)
	assert("demo144\000", demo144(a) == a)
}

type point145 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo145 C.demo145
func cdemo145(a point145) point145

func demo145(a point145) point145 {
	return a
}

func init() {
	a := point145{1, 2, 3, 4, 5}
	assert("cdemo145\000", cdemo145(a) == a)
	assert("demo145\000", demo145(a) == a)
}

type point146 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo146 C.demo146
func cdemo146(a point146) point146

func demo146(a point146) point146 {
	return a
}

func init() {
	a := point146{1, 2, 3, 4, 5}
	assert("cdemo146\000", cdemo146(a) == a)
	assert("demo146\000", demo146(a) == a)
}

type point147 struct {
	x0 int8
	x1 int32
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo147 C.demo147
func cdemo147(a point147) point147

func demo147(a point147) point147 {
	return a
}

func init() {
	a := point147{1, 2, 3, 4, 5}
	assert("cdemo147\000", cdemo147(a) == a)
	assert("demo147\000", demo147(a) == a)
}

type point148 struct {
	x0 int8
	x1 int32
	x2 int16
}

//go:linkname cdemo148 C.demo148
func cdemo148(a point148) point148

func demo148(a point148) point148 {
	return a
}

func init() {
	a := point148{1, 2, 3}
	assert("cdemo148\000", cdemo148(a) == a)
	assert("demo148\000", demo148(a) == a)
}

type point149 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int8
}

//go:linkname cdemo149 C.demo149
func cdemo149(a point149) point149

func demo149(a point149) point149 {
	return a
}

func init() {
	a := point149{1, 2, 3, 4}
	assert("cdemo149\000", cdemo149(a) == a)
	assert("demo149\000", demo149(a) == a)
}

type point150 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo150 C.demo150
func cdemo150(a point150) point150

func demo150(a point150) point150 {
	return a
}

func init() {
	a := point150{1, 2, 3, 4, 5}
	assert("cdemo150\000", cdemo150(a) == a)
	assert("demo150\000", demo150(a) == a)
}

type point151 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo151 C.demo151
func cdemo151(a point151) point151

func demo151(a point151) point151 {
	return a
}

func init() {
	a := point151{1, 2, 3, 4, 5}
	assert("cdemo151\000", cdemo151(a) == a)
	assert("demo151\000", demo151(a) == a)
}

type point152 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo152 C.demo152
func cdemo152(a point152) point152

func demo152(a point152) point152 {
	return a
}

func init() {
	a := point152{1, 2, 3, 4, 5}
	assert("cdemo152\000", cdemo152(a) == a)
	assert("demo152\000", demo152(a) == a)
}

type point153 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo153 C.demo153
func cdemo153(a point153) point153

func demo153(a point153) point153 {
	return a
}

func init() {
	a := point153{1, 2, 3, 4, 5}
	assert("cdemo153\000", cdemo153(a) == a)
	assert("demo153\000", demo153(a) == a)
}

type point154 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int16
}

//go:linkname cdemo154 C.demo154
func cdemo154(a point154) point154

func demo154(a point154) point154 {
	return a
}

func init() {
	a := point154{1, 2, 3, 4}
	assert("cdemo154\000", cdemo154(a) == a)
	assert("demo154\000", demo154(a) == a)
}

type point155 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo155 C.demo155
func cdemo155(a point155) point155

func demo155(a point155) point155 {
	return a
}

func init() {
	a := point155{1, 2, 3, 4, 5}
	assert("cdemo155\000", cdemo155(a) == a)
	assert("demo155\000", demo155(a) == a)
}

type point156 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo156 C.demo156
func cdemo156(a point156) point156

func demo156(a point156) point156 {
	return a
}

func init() {
	a := point156{1, 2, 3, 4, 5}
	assert("cdemo156\000", cdemo156(a) == a)
	assert("demo156\000", demo156(a) == a)
}

type point157 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo157 C.demo157
func cdemo157(a point157) point157

func demo157(a point157) point157 {
	return a
}

func init() {
	a := point157{1, 2, 3, 4, 5}
	assert("cdemo157\000", cdemo157(a) == a)
	assert("demo157\000", demo157(a) == a)
}

type point158 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo158 C.demo158
func cdemo158(a point158) point158

func demo158(a point158) point158 {
	return a
}

func init() {
	a := point158{1, 2, 3, 4, 5}
	assert("cdemo158\000", cdemo158(a) == a)
	assert("demo158\000", demo158(a) == a)
}

type point159 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int32
}

//go:linkname cdemo159 C.demo159
func cdemo159(a point159) point159

func demo159(a point159) point159 {
	return a
}

func init() {
	a := point159{1, 2, 3, 4}
	assert("cdemo159\000", cdemo159(a) == a)
	assert("demo159\000", demo159(a) == a)
}

type point160 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo160 C.demo160
func cdemo160(a point160) point160

func demo160(a point160) point160 {
	return a
}

func init() {
	a := point160{1, 2, 3, 4, 5}
	assert("cdemo160\000", cdemo160(a) == a)
	assert("demo160\000", demo160(a) == a)
}

type point161 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo161 C.demo161
func cdemo161(a point161) point161

func demo161(a point161) point161 {
	return a
}

func init() {
	a := point161{1, 2, 3, 4, 5}
	assert("cdemo161\000", cdemo161(a) == a)
	assert("demo161\000", demo161(a) == a)
}

type point162 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo162 C.demo162
func cdemo162(a point162) point162

func demo162(a point162) point162 {
	return a
}

func init() {
	a := point162{1, 2, 3, 4, 5}
	assert("cdemo162\000", cdemo162(a) == a)
	assert("demo162\000", demo162(a) == a)
}

type point163 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo163 C.demo163
func cdemo163(a point163) point163

func demo163(a point163) point163 {
	return a
}

func init() {
	a := point163{1, 2, 3, 4, 5}
	assert("cdemo163\000", cdemo163(a) == a)
	assert("demo163\000", demo163(a) == a)
}

type point164 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 float32
}

//go:linkname cdemo164 C.demo164
func cdemo164(a point164) point164

func demo164(a point164) point164 {
	return a
}

func init() {
	a := point164{1, 2, 3, 4}
	assert("cdemo164\000", cdemo164(a) == a)
	assert("demo164\000", demo164(a) == a)
}

type point165 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo165 C.demo165
func cdemo165(a point165) point165

func demo165(a point165) point165 {
	return a
}

func init() {
	a := point165{1, 2, 3, 4, 5}
	assert("cdemo165\000", cdemo165(a) == a)
	assert("demo165\000", demo165(a) == a)
}

type point166 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo166 C.demo166
func cdemo166(a point166) point166

func demo166(a point166) point166 {
	return a
}

func init() {
	a := point166{1, 2, 3, 4, 5}
	assert("cdemo166\000", cdemo166(a) == a)
	assert("demo166\000", demo166(a) == a)
}

type point167 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo167 C.demo167
func cdemo167(a point167) point167

func demo167(a point167) point167 {
	return a
}

func init() {
	a := point167{1, 2, 3, 4, 5}
	assert("cdemo167\000", cdemo167(a) == a)
	assert("demo167\000", demo167(a) == a)
}

type point168 struct {
	x0 int8
	x1 int32
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo168 C.demo168
func cdemo168(a point168) point168

func demo168(a point168) point168 {
	return a
}

func init() {
	a := point168{1, 2, 3, 4, 5}
	assert("cdemo168\000", cdemo168(a) == a)
	assert("demo168\000", demo168(a) == a)
}

type point169 struct {
	x0 int8
	x1 int32
	x2 int32
}

//go:linkname cdemo169 C.demo169
func cdemo169(a point169) point169

func demo169(a point169) point169 {
	return a
}

func init() {
	a := point169{1, 2, 3}
	assert("cdemo169\000", cdemo169(a) == a)
	assert("demo169\000", demo169(a) == a)
}

type point170 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int8
}

//go:linkname cdemo170 C.demo170
func cdemo170(a point170) point170

func demo170(a point170) point170 {
	return a
}

func init() {
	a := point170{1, 2, 3, 4}
	assert("cdemo170\000", cdemo170(a) == a)
	assert("demo170\000", demo170(a) == a)
}

type point171 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo171 C.demo171
func cdemo171(a point171) point171

func demo171(a point171) point171 {
	return a
}

func init() {
	a := point171{1, 2, 3, 4, 5}
	assert("cdemo171\000", cdemo171(a) == a)
	assert("demo171\000", demo171(a) == a)
}

type point172 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo172 C.demo172
func cdemo172(a point172) point172

func demo172(a point172) point172 {
	return a
}

func init() {
	a := point172{1, 2, 3, 4, 5}
	assert("cdemo172\000", cdemo172(a) == a)
	assert("demo172\000", demo172(a) == a)
}

type point173 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo173 C.demo173
func cdemo173(a point173) point173

func demo173(a point173) point173 {
	return a
}

func init() {
	a := point173{1, 2, 3, 4, 5}
	assert("cdemo173\000", cdemo173(a) == a)
	assert("demo173\000", demo173(a) == a)
}

type point174 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo174 C.demo174
func cdemo174(a point174) point174

func demo174(a point174) point174 {
	return a
}

func init() {
	a := point174{1, 2, 3, 4, 5}
	assert("cdemo174\000", cdemo174(a) == a)
	assert("demo174\000", demo174(a) == a)
}

type point175 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int16
}

//go:linkname cdemo175 C.demo175
func cdemo175(a point175) point175

func demo175(a point175) point175 {
	return a
}

func init() {
	a := point175{1, 2, 3, 4}
	assert("cdemo175\000", cdemo175(a) == a)
	assert("demo175\000", demo175(a) == a)
}

type point176 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo176 C.demo176
func cdemo176(a point176) point176

func demo176(a point176) point176 {
	return a
}

func init() {
	a := point176{1, 2, 3, 4, 5}
	assert("cdemo176\000", cdemo176(a) == a)
	assert("demo176\000", demo176(a) == a)
}

type point177 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo177 C.demo177
func cdemo177(a point177) point177

func demo177(a point177) point177 {
	return a
}

func init() {
	a := point177{1, 2, 3, 4, 5}
	assert("cdemo177\000", cdemo177(a) == a)
	assert("demo177\000", demo177(a) == a)
}

type point178 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo178 C.demo178
func cdemo178(a point178) point178

func demo178(a point178) point178 {
	return a
}

func init() {
	a := point178{1, 2, 3, 4, 5}
	assert("cdemo178\000", cdemo178(a) == a)
	assert("demo178\000", demo178(a) == a)
}

type point179 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo179 C.demo179
func cdemo179(a point179) point179

func demo179(a point179) point179 {
	return a
}

func init() {
	a := point179{1, 2, 3, 4, 5}
	assert("cdemo179\000", cdemo179(a) == a)
	assert("demo179\000", demo179(a) == a)
}

type point180 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int32
}

//go:linkname cdemo180 C.demo180
func cdemo180(a point180) point180

func demo180(a point180) point180 {
	return a
}

func init() {
	a := point180{1, 2, 3, 4}
	assert("cdemo180\000", cdemo180(a) == a)
	assert("demo180\000", demo180(a) == a)
}

type point181 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo181 C.demo181
func cdemo181(a point181) point181

func demo181(a point181) point181 {
	return a
}

func init() {
	a := point181{1, 2, 3, 4, 5}
	assert("cdemo181\000", cdemo181(a) == a)
	assert("demo181\000", demo181(a) == a)
}

type point182 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo182 C.demo182
func cdemo182(a point182) point182

func demo182(a point182) point182 {
	return a
}

func init() {
	a := point182{1, 2, 3, 4, 5}
	assert("cdemo182\000", cdemo182(a) == a)
	assert("demo182\000", demo182(a) == a)
}

type point183 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo183 C.demo183
func cdemo183(a point183) point183

func demo183(a point183) point183 {
	return a
}

func init() {
	a := point183{1, 2, 3, 4, 5}
	assert("cdemo183\000", cdemo183(a) == a)
	assert("demo183\000", demo183(a) == a)
}

type point184 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo184 C.demo184
func cdemo184(a point184) point184

func demo184(a point184) point184 {
	return a
}

func init() {
	a := point184{1, 2, 3, 4, 5}
	assert("cdemo184\000", cdemo184(a) == a)
	assert("demo184\000", demo184(a) == a)
}

type point185 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 float32
}

//go:linkname cdemo185 C.demo185
func cdemo185(a point185) point185

func demo185(a point185) point185 {
	return a
}

func init() {
	a := point185{1, 2, 3, 4}
	assert("cdemo185\000", cdemo185(a) == a)
	assert("demo185\000", demo185(a) == a)
}

type point186 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo186 C.demo186
func cdemo186(a point186) point186

func demo186(a point186) point186 {
	return a
}

func init() {
	a := point186{1, 2, 3, 4, 5}
	assert("cdemo186\000", cdemo186(a) == a)
	assert("demo186\000", demo186(a) == a)
}

type point187 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo187 C.demo187
func cdemo187(a point187) point187

func demo187(a point187) point187 {
	return a
}

func init() {
	a := point187{1, 2, 3, 4, 5}
	assert("cdemo187\000", cdemo187(a) == a)
	assert("demo187\000", demo187(a) == a)
}

type point188 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo188 C.demo188
func cdemo188(a point188) point188

func demo188(a point188) point188 {
	return a
}

func init() {
	a := point188{1, 2, 3, 4, 5}
	assert("cdemo188\000", cdemo188(a) == a)
	assert("demo188\000", demo188(a) == a)
}

type point189 struct {
	x0 int8
	x1 int32
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo189 C.demo189
func cdemo189(a point189) point189

func demo189(a point189) point189 {
	return a
}

func init() {
	a := point189{1, 2, 3, 4, 5}
	assert("cdemo189\000", cdemo189(a) == a)
	assert("demo189\000", demo189(a) == a)
}

type point190 struct {
	x0 int8
	x1 int32
	x2 float32
}

//go:linkname cdemo190 C.demo190
func cdemo190(a point190) point190

func demo190(a point190) point190 {
	return a
}

func init() {
	a := point190{1, 2, 3}
	assert("cdemo190\000", cdemo190(a) == a)
	assert("demo190\000", demo190(a) == a)
}

type point191 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int8
}

//go:linkname cdemo191 C.demo191
func cdemo191(a point191) point191

func demo191(a point191) point191 {
	return a
}

func init() {
	a := point191{1, 2, 3, 4}
	assert("cdemo191\000", cdemo191(a) == a)
	assert("demo191\000", demo191(a) == a)
}

type point192 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo192 C.demo192
func cdemo192(a point192) point192

func demo192(a point192) point192 {
	return a
}

func init() {
	a := point192{1, 2, 3, 4, 5}
	assert("cdemo192\000", cdemo192(a) == a)
	assert("demo192\000", demo192(a) == a)
}

type point193 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo193 C.demo193
func cdemo193(a point193) point193

func demo193(a point193) point193 {
	return a
}

func init() {
	a := point193{1, 2, 3, 4, 5}
	assert("cdemo193\000", cdemo193(a) == a)
	assert("demo193\000", demo193(a) == a)
}

type point194 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo194 C.demo194
func cdemo194(a point194) point194

func demo194(a point194) point194 {
	return a
}

func init() {
	a := point194{1, 2, 3, 4, 5}
	assert("cdemo194\000", cdemo194(a) == a)
	assert("demo194\000", demo194(a) == a)
}

type point195 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo195 C.demo195
func cdemo195(a point195) point195

func demo195(a point195) point195 {
	return a
}

func init() {
	a := point195{1, 2, 3, 4, 5}
	assert("cdemo195\000", cdemo195(a) == a)
	assert("demo195\000", demo195(a) == a)
}

type point196 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int16
}

//go:linkname cdemo196 C.demo196
func cdemo196(a point196) point196

func demo196(a point196) point196 {
	return a
}

func init() {
	a := point196{1, 2, 3, 4}
	assert("cdemo196\000", cdemo196(a) == a)
	assert("demo196\000", demo196(a) == a)
}

type point197 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo197 C.demo197
func cdemo197(a point197) point197

func demo197(a point197) point197 {
	return a
}

func init() {
	a := point197{1, 2, 3, 4, 5}
	assert("cdemo197\000", cdemo197(a) == a)
	assert("demo197\000", demo197(a) == a)
}

type point198 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo198 C.demo198
func cdemo198(a point198) point198

func demo198(a point198) point198 {
	return a
}

func init() {
	a := point198{1, 2, 3, 4, 5}
	assert("cdemo198\000", cdemo198(a) == a)
	assert("demo198\000", demo198(a) == a)
}

type point199 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo199 C.demo199
func cdemo199(a point199) point199

func demo199(a point199) point199 {
	return a
}

func init() {
	a := point199{1, 2, 3, 4, 5}
	assert("cdemo199\000", cdemo199(a) == a)
	assert("demo199\000", demo199(a) == a)
}

type point200 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo200 C.demo200
func cdemo200(a point200) point200

func demo200(a point200) point200 {
	return a
}

func init() {
	a := point200{1, 2, 3, 4, 5}
	assert("cdemo200\000", cdemo200(a) == a)
	assert("demo200\000", demo200(a) == a)
}

type point201 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int32
}

//go:linkname cdemo201 C.demo201
func cdemo201(a point201) point201

func demo201(a point201) point201 {
	return a
}

func init() {
	a := point201{1, 2, 3, 4}
	assert("cdemo201\000", cdemo201(a) == a)
	assert("demo201\000", demo201(a) == a)
}

type point202 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo202 C.demo202
func cdemo202(a point202) point202

func demo202(a point202) point202 {
	return a
}

func init() {
	a := point202{1, 2, 3, 4, 5}
	assert("cdemo202\000", cdemo202(a) == a)
	assert("demo202\000", demo202(a) == a)
}

type point203 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo203 C.demo203
func cdemo203(a point203) point203

func demo203(a point203) point203 {
	return a
}

func init() {
	a := point203{1, 2, 3, 4, 5}
	assert("cdemo203\000", cdemo203(a) == a)
	assert("demo203\000", demo203(a) == a)
}

type point204 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo204 C.demo204
func cdemo204(a point204) point204

func demo204(a point204) point204 {
	return a
}

func init() {
	a := point204{1, 2, 3, 4, 5}
	assert("cdemo204\000", cdemo204(a) == a)
	assert("demo204\000", demo204(a) == a)
}

type point205 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo205 C.demo205
func cdemo205(a point205) point205

func demo205(a point205) point205 {
	return a
}

func init() {
	a := point205{1, 2, 3, 4, 5}
	assert("cdemo205\000", cdemo205(a) == a)
	assert("demo205\000", demo205(a) == a)
}

type point206 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 float32
}

//go:linkname cdemo206 C.demo206
func cdemo206(a point206) point206

func demo206(a point206) point206 {
	return a
}

func init() {
	a := point206{1, 2, 3, 4}
	assert("cdemo206\000", cdemo206(a) == a)
	assert("demo206\000", demo206(a) == a)
}

type point207 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo207 C.demo207
func cdemo207(a point207) point207

func demo207(a point207) point207 {
	return a
}

func init() {
	a := point207{1, 2, 3, 4, 5}
	assert("cdemo207\000", cdemo207(a) == a)
	assert("demo207\000", demo207(a) == a)
}

type point208 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo208 C.demo208
func cdemo208(a point208) point208

func demo208(a point208) point208 {
	return a
}

func init() {
	a := point208{1, 2, 3, 4, 5}
	assert("cdemo208\000", cdemo208(a) == a)
	assert("demo208\000", demo208(a) == a)
}

type point209 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo209 C.demo209
func cdemo209(a point209) point209

func demo209(a point209) point209 {
	return a
}

func init() {
	a := point209{1, 2, 3, 4, 5}
	assert("cdemo209\000", cdemo209(a) == a)
	assert("demo209\000", demo209(a) == a)
}

type point210 struct {
	x0 int8
	x1 int32
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo210 C.demo210
func cdemo210(a point210) point210

func demo210(a point210) point210 {
	return a
}

func init() {
	a := point210{1, 2, 3, 4, 5}
	assert("cdemo210\000", cdemo210(a) == a)
	assert("demo210\000", demo210(a) == a)
}

type point211 struct {
	x0 int8
	x1 float32
	x2 int8
}

//go:linkname cdemo211 C.demo211
func cdemo211(a point211) point211

func demo211(a point211) point211 {
	return a
}

func init() {
	a := point211{1, 2, 3}
	assert("cdemo211\000", cdemo211(a) == a)
	assert("demo211\000", demo211(a) == a)
}

type point212 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int8
}

//go:linkname cdemo212 C.demo212
func cdemo212(a point212) point212

func demo212(a point212) point212 {
	return a
}

func init() {
	a := point212{1, 2, 3, 4}
	assert("cdemo212\000", cdemo212(a) == a)
	assert("demo212\000", demo212(a) == a)
}

type point213 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo213 C.demo213
func cdemo213(a point213) point213

func demo213(a point213) point213 {
	return a
}

func init() {
	a := point213{1, 2, 3, 4, 5}
	assert("cdemo213\000", cdemo213(a) == a)
	assert("demo213\000", demo213(a) == a)
}

type point214 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo214 C.demo214
func cdemo214(a point214) point214

func demo214(a point214) point214 {
	return a
}

func init() {
	a := point214{1, 2, 3, 4, 5}
	assert("cdemo214\000", cdemo214(a) == a)
	assert("demo214\000", demo214(a) == a)
}

type point215 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo215 C.demo215
func cdemo215(a point215) point215

func demo215(a point215) point215 {
	return a
}

func init() {
	a := point215{1, 2, 3, 4, 5}
	assert("cdemo215\000", cdemo215(a) == a)
	assert("demo215\000", demo215(a) == a)
}

type point216 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo216 C.demo216
func cdemo216(a point216) point216

func demo216(a point216) point216 {
	return a
}

func init() {
	a := point216{1, 2, 3, 4, 5}
	assert("cdemo216\000", cdemo216(a) == a)
	assert("demo216\000", demo216(a) == a)
}

type point217 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int16
}

//go:linkname cdemo217 C.demo217
func cdemo217(a point217) point217

func demo217(a point217) point217 {
	return a
}

func init() {
	a := point217{1, 2, 3, 4}
	assert("cdemo217\000", cdemo217(a) == a)
	assert("demo217\000", demo217(a) == a)
}

type point218 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo218 C.demo218
func cdemo218(a point218) point218

func demo218(a point218) point218 {
	return a
}

func init() {
	a := point218{1, 2, 3, 4, 5}
	assert("cdemo218\000", cdemo218(a) == a)
	assert("demo218\000", demo218(a) == a)
}

type point219 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo219 C.demo219
func cdemo219(a point219) point219

func demo219(a point219) point219 {
	return a
}

func init() {
	a := point219{1, 2, 3, 4, 5}
	assert("cdemo219\000", cdemo219(a) == a)
	assert("demo219\000", demo219(a) == a)
}

type point220 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo220 C.demo220
func cdemo220(a point220) point220

func demo220(a point220) point220 {
	return a
}

func init() {
	a := point220{1, 2, 3, 4, 5}
	assert("cdemo220\000", cdemo220(a) == a)
	assert("demo220\000", demo220(a) == a)
}

type point221 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo221 C.demo221
func cdemo221(a point221) point221

func demo221(a point221) point221 {
	return a
}

func init() {
	a := point221{1, 2, 3, 4, 5}
	assert("cdemo221\000", cdemo221(a) == a)
	assert("demo221\000", demo221(a) == a)
}

type point222 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int32
}

//go:linkname cdemo222 C.demo222
func cdemo222(a point222) point222

func demo222(a point222) point222 {
	return a
}

func init() {
	a := point222{1, 2, 3, 4}
	assert("cdemo222\000", cdemo222(a) == a)
	assert("demo222\000", demo222(a) == a)
}

type point223 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo223 C.demo223
func cdemo223(a point223) point223

func demo223(a point223) point223 {
	return a
}

func init() {
	a := point223{1, 2, 3, 4, 5}
	assert("cdemo223\000", cdemo223(a) == a)
	assert("demo223\000", demo223(a) == a)
}

type point224 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo224 C.demo224
func cdemo224(a point224) point224

func demo224(a point224) point224 {
	return a
}

func init() {
	a := point224{1, 2, 3, 4, 5}
	assert("cdemo224\000", cdemo224(a) == a)
	assert("demo224\000", demo224(a) == a)
}

type point225 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo225 C.demo225
func cdemo225(a point225) point225

func demo225(a point225) point225 {
	return a
}

func init() {
	a := point225{1, 2, 3, 4, 5}
	assert("cdemo225\000", cdemo225(a) == a)
	assert("demo225\000", demo225(a) == a)
}

type point226 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo226 C.demo226
func cdemo226(a point226) point226

func demo226(a point226) point226 {
	return a
}

func init() {
	a := point226{1, 2, 3, 4, 5}
	assert("cdemo226\000", cdemo226(a) == a)
	assert("demo226\000", demo226(a) == a)
}

type point227 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 float32
}

//go:linkname cdemo227 C.demo227
func cdemo227(a point227) point227

func demo227(a point227) point227 {
	return a
}

func init() {
	a := point227{1, 2, 3, 4}
	assert("cdemo227\000", cdemo227(a) == a)
	assert("demo227\000", demo227(a) == a)
}

type point228 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo228 C.demo228
func cdemo228(a point228) point228

func demo228(a point228) point228 {
	return a
}

func init() {
	a := point228{1, 2, 3, 4, 5}
	assert("cdemo228\000", cdemo228(a) == a)
	assert("demo228\000", demo228(a) == a)
}

type point229 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo229 C.demo229
func cdemo229(a point229) point229

func demo229(a point229) point229 {
	return a
}

func init() {
	a := point229{1, 2, 3, 4, 5}
	assert("cdemo229\000", cdemo229(a) == a)
	assert("demo229\000", demo229(a) == a)
}

type point230 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo230 C.demo230
func cdemo230(a point230) point230

func demo230(a point230) point230 {
	return a
}

func init() {
	a := point230{1, 2, 3, 4, 5}
	assert("cdemo230\000", cdemo230(a) == a)
	assert("demo230\000", demo230(a) == a)
}

type point231 struct {
	x0 int8
	x1 float32
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo231 C.demo231
func cdemo231(a point231) point231

func demo231(a point231) point231 {
	return a
}

func init() {
	a := point231{1, 2, 3, 4, 5}
	assert("cdemo231\000", cdemo231(a) == a)
	assert("demo231\000", demo231(a) == a)
}

type point232 struct {
	x0 int8
	x1 float32
	x2 int16
}

//go:linkname cdemo232 C.demo232
func cdemo232(a point232) point232

func demo232(a point232) point232 {
	return a
}

func init() {
	a := point232{1, 2, 3}
	assert("cdemo232\000", cdemo232(a) == a)
	assert("demo232\000", demo232(a) == a)
}

type point233 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int8
}

//go:linkname cdemo233 C.demo233
func cdemo233(a point233) point233

func demo233(a point233) point233 {
	return a
}

func init() {
	a := point233{1, 2, 3, 4}
	assert("cdemo233\000", cdemo233(a) == a)
	assert("demo233\000", demo233(a) == a)
}

type point234 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo234 C.demo234
func cdemo234(a point234) point234

func demo234(a point234) point234 {
	return a
}

func init() {
	a := point234{1, 2, 3, 4, 5}
	assert("cdemo234\000", cdemo234(a) == a)
	assert("demo234\000", demo234(a) == a)
}

type point235 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo235 C.demo235
func cdemo235(a point235) point235

func demo235(a point235) point235 {
	return a
}

func init() {
	a := point235{1, 2, 3, 4, 5}
	assert("cdemo235\000", cdemo235(a) == a)
	assert("demo235\000", demo235(a) == a)
}

type point236 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo236 C.demo236
func cdemo236(a point236) point236

func demo236(a point236) point236 {
	return a
}

func init() {
	a := point236{1, 2, 3, 4, 5}
	assert("cdemo236\000", cdemo236(a) == a)
	assert("demo236\000", demo236(a) == a)
}

type point237 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo237 C.demo237
func cdemo237(a point237) point237

func demo237(a point237) point237 {
	return a
}

func init() {
	a := point237{1, 2, 3, 4, 5}
	assert("cdemo237\000", cdemo237(a) == a)
	assert("demo237\000", demo237(a) == a)
}

type point238 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int16
}

//go:linkname cdemo238 C.demo238
func cdemo238(a point238) point238

func demo238(a point238) point238 {
	return a
}

func init() {
	a := point238{1, 2, 3, 4}
	assert("cdemo238\000", cdemo238(a) == a)
	assert("demo238\000", demo238(a) == a)
}

type point239 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo239 C.demo239
func cdemo239(a point239) point239

func demo239(a point239) point239 {
	return a
}

func init() {
	a := point239{1, 2, 3, 4, 5}
	assert("cdemo239\000", cdemo239(a) == a)
	assert("demo239\000", demo239(a) == a)
}

type point240 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo240 C.demo240
func cdemo240(a point240) point240

func demo240(a point240) point240 {
	return a
}

func init() {
	a := point240{1, 2, 3, 4, 5}
	assert("cdemo240\000", cdemo240(a) == a)
	assert("demo240\000", demo240(a) == a)
}

type point241 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo241 C.demo241
func cdemo241(a point241) point241

func demo241(a point241) point241 {
	return a
}

func init() {
	a := point241{1, 2, 3, 4, 5}
	assert("cdemo241\000", cdemo241(a) == a)
	assert("demo241\000", demo241(a) == a)
}

type point242 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo242 C.demo242
func cdemo242(a point242) point242

func demo242(a point242) point242 {
	return a
}

func init() {
	a := point242{1, 2, 3, 4, 5}
	assert("cdemo242\000", cdemo242(a) == a)
	assert("demo242\000", demo242(a) == a)
}

type point243 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int32
}

//go:linkname cdemo243 C.demo243
func cdemo243(a point243) point243

func demo243(a point243) point243 {
	return a
}

func init() {
	a := point243{1, 2, 3, 4}
	assert("cdemo243\000", cdemo243(a) == a)
	assert("demo243\000", demo243(a) == a)
}

type point244 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo244 C.demo244
func cdemo244(a point244) point244

func demo244(a point244) point244 {
	return a
}

func init() {
	a := point244{1, 2, 3, 4, 5}
	assert("cdemo244\000", cdemo244(a) == a)
	assert("demo244\000", demo244(a) == a)
}

type point245 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo245 C.demo245
func cdemo245(a point245) point245

func demo245(a point245) point245 {
	return a
}

func init() {
	a := point245{1, 2, 3, 4, 5}
	assert("cdemo245\000", cdemo245(a) == a)
	assert("demo245\000", demo245(a) == a)
}

type point246 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo246 C.demo246
func cdemo246(a point246) point246

func demo246(a point246) point246 {
	return a
}

func init() {
	a := point246{1, 2, 3, 4, 5}
	assert("cdemo246\000", cdemo246(a) == a)
	assert("demo246\000", demo246(a) == a)
}

type point247 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo247 C.demo247
func cdemo247(a point247) point247

func demo247(a point247) point247 {
	return a
}

func init() {
	a := point247{1, 2, 3, 4, 5}
	assert("cdemo247\000", cdemo247(a) == a)
	assert("demo247\000", demo247(a) == a)
}

type point248 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 float32
}

//go:linkname cdemo248 C.demo248
func cdemo248(a point248) point248

func demo248(a point248) point248 {
	return a
}

func init() {
	a := point248{1, 2, 3, 4}
	assert("cdemo248\000", cdemo248(a) == a)
	assert("demo248\000", demo248(a) == a)
}

type point249 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo249 C.demo249
func cdemo249(a point249) point249

func demo249(a point249) point249 {
	return a
}

func init() {
	a := point249{1, 2, 3, 4, 5}
	assert("cdemo249\000", cdemo249(a) == a)
	assert("demo249\000", demo249(a) == a)
}

type point250 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo250 C.demo250
func cdemo250(a point250) point250

func demo250(a point250) point250 {
	return a
}

func init() {
	a := point250{1, 2, 3, 4, 5}
	assert("cdemo250\000", cdemo250(a) == a)
	assert("demo250\000", demo250(a) == a)
}

type point251 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo251 C.demo251
func cdemo251(a point251) point251

func demo251(a point251) point251 {
	return a
}

func init() {
	a := point251{1, 2, 3, 4, 5}
	assert("cdemo251\000", cdemo251(a) == a)
	assert("demo251\000", demo251(a) == a)
}

type point252 struct {
	x0 int8
	x1 float32
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo252 C.demo252
func cdemo252(a point252) point252

func demo252(a point252) point252 {
	return a
}

func init() {
	a := point252{1, 2, 3, 4, 5}
	assert("cdemo252\000", cdemo252(a) == a)
	assert("demo252\000", demo252(a) == a)
}

type point253 struct {
	x0 int8
	x1 float32
	x2 int32
}

//go:linkname cdemo253 C.demo253
func cdemo253(a point253) point253

func demo253(a point253) point253 {
	return a
}

func init() {
	a := point253{1, 2, 3}
	assert("cdemo253\000", cdemo253(a) == a)
	assert("demo253\000", demo253(a) == a)
}

type point254 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int8
}

//go:linkname cdemo254 C.demo254
func cdemo254(a point254) point254

func demo254(a point254) point254 {
	return a
}

func init() {
	a := point254{1, 2, 3, 4}
	assert("cdemo254\000", cdemo254(a) == a)
	assert("demo254\000", demo254(a) == a)
}

type point255 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo255 C.demo255
func cdemo255(a point255) point255

func demo255(a point255) point255 {
	return a
}

func init() {
	a := point255{1, 2, 3, 4, 5}
	assert("cdemo255\000", cdemo255(a) == a)
	assert("demo255\000", demo255(a) == a)
}

type point256 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo256 C.demo256
func cdemo256(a point256) point256

func demo256(a point256) point256 {
	return a
}

func init() {
	a := point256{1, 2, 3, 4, 5}
	assert("cdemo256\000", cdemo256(a) == a)
	assert("demo256\000", demo256(a) == a)
}

type point257 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo257 C.demo257
func cdemo257(a point257) point257

func demo257(a point257) point257 {
	return a
}

func init() {
	a := point257{1, 2, 3, 4, 5}
	assert("cdemo257\000", cdemo257(a) == a)
	assert("demo257\000", demo257(a) == a)
}

type point258 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo258 C.demo258
func cdemo258(a point258) point258

func demo258(a point258) point258 {
	return a
}

func init() {
	a := point258{1, 2, 3, 4, 5}
	assert("cdemo258\000", cdemo258(a) == a)
	assert("demo258\000", demo258(a) == a)
}

type point259 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int16
}

//go:linkname cdemo259 C.demo259
func cdemo259(a point259) point259

func demo259(a point259) point259 {
	return a
}

func init() {
	a := point259{1, 2, 3, 4}
	assert("cdemo259\000", cdemo259(a) == a)
	assert("demo259\000", demo259(a) == a)
}

type point260 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo260 C.demo260
func cdemo260(a point260) point260

func demo260(a point260) point260 {
	return a
}

func init() {
	a := point260{1, 2, 3, 4, 5}
	assert("cdemo260\000", cdemo260(a) == a)
	assert("demo260\000", demo260(a) == a)
}

type point261 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo261 C.demo261
func cdemo261(a point261) point261

func demo261(a point261) point261 {
	return a
}

func init() {
	a := point261{1, 2, 3, 4, 5}
	assert("cdemo261\000", cdemo261(a) == a)
	assert("demo261\000", demo261(a) == a)
}

type point262 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo262 C.demo262
func cdemo262(a point262) point262

func demo262(a point262) point262 {
	return a
}

func init() {
	a := point262{1, 2, 3, 4, 5}
	assert("cdemo262\000", cdemo262(a) == a)
	assert("demo262\000", demo262(a) == a)
}

type point263 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo263 C.demo263
func cdemo263(a point263) point263

func demo263(a point263) point263 {
	return a
}

func init() {
	a := point263{1, 2, 3, 4, 5}
	assert("cdemo263\000", cdemo263(a) == a)
	assert("demo263\000", demo263(a) == a)
}

type point264 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int32
}

//go:linkname cdemo264 C.demo264
func cdemo264(a point264) point264

func demo264(a point264) point264 {
	return a
}

func init() {
	a := point264{1, 2, 3, 4}
	assert("cdemo264\000", cdemo264(a) == a)
	assert("demo264\000", demo264(a) == a)
}

type point265 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo265 C.demo265
func cdemo265(a point265) point265

func demo265(a point265) point265 {
	return a
}

func init() {
	a := point265{1, 2, 3, 4, 5}
	assert("cdemo265\000", cdemo265(a) == a)
	assert("demo265\000", demo265(a) == a)
}

type point266 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo266 C.demo266
func cdemo266(a point266) point266

func demo266(a point266) point266 {
	return a
}

func init() {
	a := point266{1, 2, 3, 4, 5}
	assert("cdemo266\000", cdemo266(a) == a)
	assert("demo266\000", demo266(a) == a)
}

type point267 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo267 C.demo267
func cdemo267(a point267) point267

func demo267(a point267) point267 {
	return a
}

func init() {
	a := point267{1, 2, 3, 4, 5}
	assert("cdemo267\000", cdemo267(a) == a)
	assert("demo267\000", demo267(a) == a)
}

type point268 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo268 C.demo268
func cdemo268(a point268) point268

func demo268(a point268) point268 {
	return a
}

func init() {
	a := point268{1, 2, 3, 4, 5}
	assert("cdemo268\000", cdemo268(a) == a)
	assert("demo268\000", demo268(a) == a)
}

type point269 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 float32
}

//go:linkname cdemo269 C.demo269
func cdemo269(a point269) point269

func demo269(a point269) point269 {
	return a
}

func init() {
	a := point269{1, 2, 3, 4}
	assert("cdemo269\000", cdemo269(a) == a)
	assert("demo269\000", demo269(a) == a)
}

type point270 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo270 C.demo270
func cdemo270(a point270) point270

func demo270(a point270) point270 {
	return a
}

func init() {
	a := point270{1, 2, 3, 4, 5}
	assert("cdemo270\000", cdemo270(a) == a)
	assert("demo270\000", demo270(a) == a)
}

type point271 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo271 C.demo271
func cdemo271(a point271) point271

func demo271(a point271) point271 {
	return a
}

func init() {
	a := point271{1, 2, 3, 4, 5}
	assert("cdemo271\000", cdemo271(a) == a)
	assert("demo271\000", demo271(a) == a)
}

type point272 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo272 C.demo272
func cdemo272(a point272) point272

func demo272(a point272) point272 {
	return a
}

func init() {
	a := point272{1, 2, 3, 4, 5}
	assert("cdemo272\000", cdemo272(a) == a)
	assert("demo272\000", demo272(a) == a)
}

type point273 struct {
	x0 int8
	x1 float32
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo273 C.demo273
func cdemo273(a point273) point273

func demo273(a point273) point273 {
	return a
}

func init() {
	a := point273{1, 2, 3, 4, 5}
	assert("cdemo273\000", cdemo273(a) == a)
	assert("demo273\000", demo273(a) == a)
}

type point274 struct {
	x0 int8
	x1 float32
	x2 float32
}

//go:linkname cdemo274 C.demo274
func cdemo274(a point274) point274

func demo274(a point274) point274 {
	return a
}

func init() {
	a := point274{1, 2, 3}
	assert("cdemo274\000", cdemo274(a) == a)
	assert("demo274\000", demo274(a) == a)
}

type point275 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int8
}

//go:linkname cdemo275 C.demo275
func cdemo275(a point275) point275

func demo275(a point275) point275 {
	return a
}

func init() {
	a := point275{1, 2, 3, 4}
	assert("cdemo275\000", cdemo275(a) == a)
	assert("demo275\000", demo275(a) == a)
}

type point276 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo276 C.demo276
func cdemo276(a point276) point276

func demo276(a point276) point276 {
	return a
}

func init() {
	a := point276{1, 2, 3, 4, 5}
	assert("cdemo276\000", cdemo276(a) == a)
	assert("demo276\000", demo276(a) == a)
}

type point277 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo277 C.demo277
func cdemo277(a point277) point277

func demo277(a point277) point277 {
	return a
}

func init() {
	a := point277{1, 2, 3, 4, 5}
	assert("cdemo277\000", cdemo277(a) == a)
	assert("demo277\000", demo277(a) == a)
}

type point278 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo278 C.demo278
func cdemo278(a point278) point278

func demo278(a point278) point278 {
	return a
}

func init() {
	a := point278{1, 2, 3, 4, 5}
	assert("cdemo278\000", cdemo278(a) == a)
	assert("demo278\000", demo278(a) == a)
}

type point279 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo279 C.demo279
func cdemo279(a point279) point279

func demo279(a point279) point279 {
	return a
}

func init() {
	a := point279{1, 2, 3, 4, 5}
	assert("cdemo279\000", cdemo279(a) == a)
	assert("demo279\000", demo279(a) == a)
}

type point280 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int16
}

//go:linkname cdemo280 C.demo280
func cdemo280(a point280) point280

func demo280(a point280) point280 {
	return a
}

func init() {
	a := point280{1, 2, 3, 4}
	assert("cdemo280\000", cdemo280(a) == a)
	assert("demo280\000", demo280(a) == a)
}

type point281 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo281 C.demo281
func cdemo281(a point281) point281

func demo281(a point281) point281 {
	return a
}

func init() {
	a := point281{1, 2, 3, 4, 5}
	assert("cdemo281\000", cdemo281(a) == a)
	assert("demo281\000", demo281(a) == a)
}

type point282 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo282 C.demo282
func cdemo282(a point282) point282

func demo282(a point282) point282 {
	return a
}

func init() {
	a := point282{1, 2, 3, 4, 5}
	assert("cdemo282\000", cdemo282(a) == a)
	assert("demo282\000", demo282(a) == a)
}

type point283 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo283 C.demo283
func cdemo283(a point283) point283

func demo283(a point283) point283 {
	return a
}

func init() {
	a := point283{1, 2, 3, 4, 5}
	assert("cdemo283\000", cdemo283(a) == a)
	assert("demo283\000", demo283(a) == a)
}

type point284 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo284 C.demo284
func cdemo284(a point284) point284

func demo284(a point284) point284 {
	return a
}

func init() {
	a := point284{1, 2, 3, 4, 5}
	assert("cdemo284\000", cdemo284(a) == a)
	assert("demo284\000", demo284(a) == a)
}

type point285 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int32
}

//go:linkname cdemo285 C.demo285
func cdemo285(a point285) point285

func demo285(a point285) point285 {
	return a
}

func init() {
	a := point285{1, 2, 3, 4}
	assert("cdemo285\000", cdemo285(a) == a)
	assert("demo285\000", demo285(a) == a)
}

type point286 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo286 C.demo286
func cdemo286(a point286) point286

func demo286(a point286) point286 {
	return a
}

func init() {
	a := point286{1, 2, 3, 4, 5}
	assert("cdemo286\000", cdemo286(a) == a)
	assert("demo286\000", demo286(a) == a)
}

type point287 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo287 C.demo287
func cdemo287(a point287) point287

func demo287(a point287) point287 {
	return a
}

func init() {
	a := point287{1, 2, 3, 4, 5}
	assert("cdemo287\000", cdemo287(a) == a)
	assert("demo287\000", demo287(a) == a)
}

type point288 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo288 C.demo288
func cdemo288(a point288) point288

func demo288(a point288) point288 {
	return a
}

func init() {
	a := point288{1, 2, 3, 4, 5}
	assert("cdemo288\000", cdemo288(a) == a)
	assert("demo288\000", demo288(a) == a)
}

type point289 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo289 C.demo289
func cdemo289(a point289) point289

func demo289(a point289) point289 {
	return a
}

func init() {
	a := point289{1, 2, 3, 4, 5}
	assert("cdemo289\000", cdemo289(a) == a)
	assert("demo289\000", demo289(a) == a)
}

type point290 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 float32
}

//go:linkname cdemo290 C.demo290
func cdemo290(a point290) point290

func demo290(a point290) point290 {
	return a
}

func init() {
	a := point290{1, 2, 3, 4}
	assert("cdemo290\000", cdemo290(a) == a)
	assert("demo290\000", demo290(a) == a)
}

type point291 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo291 C.demo291
func cdemo291(a point291) point291

func demo291(a point291) point291 {
	return a
}

func init() {
	a := point291{1, 2, 3, 4, 5}
	assert("cdemo291\000", cdemo291(a) == a)
	assert("demo291\000", demo291(a) == a)
}

type point292 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo292 C.demo292
func cdemo292(a point292) point292

func demo292(a point292) point292 {
	return a
}

func init() {
	a := point292{1, 2, 3, 4, 5}
	assert("cdemo292\000", cdemo292(a) == a)
	assert("demo292\000", demo292(a) == a)
}

type point293 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo293 C.demo293
func cdemo293(a point293) point293

func demo293(a point293) point293 {
	return a
}

func init() {
	a := point293{1, 2, 3, 4, 5}
	assert("cdemo293\000", cdemo293(a) == a)
	assert("demo293\000", demo293(a) == a)
}

type point294 struct {
	x0 int8
	x1 float32
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo294 C.demo294
func cdemo294(a point294) point294

func demo294(a point294) point294 {
	return a
}

func init() {
	a := point294{1, 2, 3, 4, 5}
	assert("cdemo294\000", cdemo294(a) == a)
	assert("demo294\000", demo294(a) == a)
}

type point295 struct {
	x0 int16
	x1 int8
	x2 int8
}

//go:linkname cdemo295 C.demo295
func cdemo295(a point295) point295

func demo295(a point295) point295 {
	return a
}

func init() {
	a := point295{1, 2, 3}
	assert("cdemo295\000", cdemo295(a) == a)
	assert("demo295\000", demo295(a) == a)
}

type point296 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int8
}

//go:linkname cdemo296 C.demo296
func cdemo296(a point296) point296

func demo296(a point296) point296 {
	return a
}

func init() {
	a := point296{1, 2, 3, 4}
	assert("cdemo296\000", cdemo296(a) == a)
	assert("demo296\000", demo296(a) == a)
}

type point297 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo297 C.demo297
func cdemo297(a point297) point297

func demo297(a point297) point297 {
	return a
}

func init() {
	a := point297{1, 2, 3, 4, 5}
	assert("cdemo297\000", cdemo297(a) == a)
	assert("demo297\000", demo297(a) == a)
}

type point298 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo298 C.demo298
func cdemo298(a point298) point298

func demo298(a point298) point298 {
	return a
}

func init() {
	a := point298{1, 2, 3, 4, 5}
	assert("cdemo298\000", cdemo298(a) == a)
	assert("demo298\000", demo298(a) == a)
}

type point299 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo299 C.demo299
func cdemo299(a point299) point299

func demo299(a point299) point299 {
	return a
}

func init() {
	a := point299{1, 2, 3, 4, 5}
	assert("cdemo299\000", cdemo299(a) == a)
	assert("demo299\000", demo299(a) == a)
}

type point300 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo300 C.demo300
func cdemo300(a point300) point300

func demo300(a point300) point300 {
	return a
}

func init() {
	a := point300{1, 2, 3, 4, 5}
	assert("cdemo300\000", cdemo300(a) == a)
	assert("demo300\000", demo300(a) == a)
}

type point301 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int16
}

//go:linkname cdemo301 C.demo301
func cdemo301(a point301) point301

func demo301(a point301) point301 {
	return a
}

func init() {
	a := point301{1, 2, 3, 4}
	assert("cdemo301\000", cdemo301(a) == a)
	assert("demo301\000", demo301(a) == a)
}

type point302 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo302 C.demo302
func cdemo302(a point302) point302

func demo302(a point302) point302 {
	return a
}

func init() {
	a := point302{1, 2, 3, 4, 5}
	assert("cdemo302\000", cdemo302(a) == a)
	assert("demo302\000", demo302(a) == a)
}

type point303 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo303 C.demo303
func cdemo303(a point303) point303

func demo303(a point303) point303 {
	return a
}

func init() {
	a := point303{1, 2, 3, 4, 5}
	assert("cdemo303\000", cdemo303(a) == a)
	assert("demo303\000", demo303(a) == a)
}

type point304 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo304 C.demo304
func cdemo304(a point304) point304

func demo304(a point304) point304 {
	return a
}

func init() {
	a := point304{1, 2, 3, 4, 5}
	assert("cdemo304\000", cdemo304(a) == a)
	assert("demo304\000", demo304(a) == a)
}

type point305 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo305 C.demo305
func cdemo305(a point305) point305

func demo305(a point305) point305 {
	return a
}

func init() {
	a := point305{1, 2, 3, 4, 5}
	assert("cdemo305\000", cdemo305(a) == a)
	assert("demo305\000", demo305(a) == a)
}

type point306 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int32
}

//go:linkname cdemo306 C.demo306
func cdemo306(a point306) point306

func demo306(a point306) point306 {
	return a
}

func init() {
	a := point306{1, 2, 3, 4}
	assert("cdemo306\000", cdemo306(a) == a)
	assert("demo306\000", demo306(a) == a)
}

type point307 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo307 C.demo307
func cdemo307(a point307) point307

func demo307(a point307) point307 {
	return a
}

func init() {
	a := point307{1, 2, 3, 4, 5}
	assert("cdemo307\000", cdemo307(a) == a)
	assert("demo307\000", demo307(a) == a)
}

type point308 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo308 C.demo308
func cdemo308(a point308) point308

func demo308(a point308) point308 {
	return a
}

func init() {
	a := point308{1, 2, 3, 4, 5}
	assert("cdemo308\000", cdemo308(a) == a)
	assert("demo308\000", demo308(a) == a)
}

type point309 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo309 C.demo309
func cdemo309(a point309) point309

func demo309(a point309) point309 {
	return a
}

func init() {
	a := point309{1, 2, 3, 4, 5}
	assert("cdemo309\000", cdemo309(a) == a)
	assert("demo309\000", demo309(a) == a)
}

type point310 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo310 C.demo310
func cdemo310(a point310) point310

func demo310(a point310) point310 {
	return a
}

func init() {
	a := point310{1, 2, 3, 4, 5}
	assert("cdemo310\000", cdemo310(a) == a)
	assert("demo310\000", demo310(a) == a)
}

type point311 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 float32
}

//go:linkname cdemo311 C.demo311
func cdemo311(a point311) point311

func demo311(a point311) point311 {
	return a
}

func init() {
	a := point311{1, 2, 3, 4}
	assert("cdemo311\000", cdemo311(a) == a)
	assert("demo311\000", demo311(a) == a)
}

type point312 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo312 C.demo312
func cdemo312(a point312) point312

func demo312(a point312) point312 {
	return a
}

func init() {
	a := point312{1, 2, 3, 4, 5}
	assert("cdemo312\000", cdemo312(a) == a)
	assert("demo312\000", demo312(a) == a)
}

type point313 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo313 C.demo313
func cdemo313(a point313) point313

func demo313(a point313) point313 {
	return a
}

func init() {
	a := point313{1, 2, 3, 4, 5}
	assert("cdemo313\000", cdemo313(a) == a)
	assert("demo313\000", demo313(a) == a)
}

type point314 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo314 C.demo314
func cdemo314(a point314) point314

func demo314(a point314) point314 {
	return a
}

func init() {
	a := point314{1, 2, 3, 4, 5}
	assert("cdemo314\000", cdemo314(a) == a)
	assert("demo314\000", demo314(a) == a)
}

type point315 struct {
	x0 int16
	x1 int8
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo315 C.demo315
func cdemo315(a point315) point315

func demo315(a point315) point315 {
	return a
}

func init() {
	a := point315{1, 2, 3, 4, 5}
	assert("cdemo315\000", cdemo315(a) == a)
	assert("demo315\000", demo315(a) == a)
}

type point316 struct {
	x0 int16
	x1 int8
	x2 int16
}

//go:linkname cdemo316 C.demo316
func cdemo316(a point316) point316

func demo316(a point316) point316 {
	return a
}

func init() {
	a := point316{1, 2, 3}
	assert("cdemo316\000", cdemo316(a) == a)
	assert("demo316\000", demo316(a) == a)
}

type point317 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int8
}

//go:linkname cdemo317 C.demo317
func cdemo317(a point317) point317

func demo317(a point317) point317 {
	return a
}

func init() {
	a := point317{1, 2, 3, 4}
	assert("cdemo317\000", cdemo317(a) == a)
	assert("demo317\000", demo317(a) == a)
}

type point318 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo318 C.demo318
func cdemo318(a point318) point318

func demo318(a point318) point318 {
	return a
}

func init() {
	a := point318{1, 2, 3, 4, 5}
	assert("cdemo318\000", cdemo318(a) == a)
	assert("demo318\000", demo318(a) == a)
}

type point319 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo319 C.demo319
func cdemo319(a point319) point319

func demo319(a point319) point319 {
	return a
}

func init() {
	a := point319{1, 2, 3, 4, 5}
	assert("cdemo319\000", cdemo319(a) == a)
	assert("demo319\000", demo319(a) == a)
}

type point320 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo320 C.demo320
func cdemo320(a point320) point320

func demo320(a point320) point320 {
	return a
}

func init() {
	a := point320{1, 2, 3, 4, 5}
	assert("cdemo320\000", cdemo320(a) == a)
	assert("demo320\000", demo320(a) == a)
}

type point321 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo321 C.demo321
func cdemo321(a point321) point321

func demo321(a point321) point321 {
	return a
}

func init() {
	a := point321{1, 2, 3, 4, 5}
	assert("cdemo321\000", cdemo321(a) == a)
	assert("demo321\000", demo321(a) == a)
}

type point322 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int16
}

//go:linkname cdemo322 C.demo322
func cdemo322(a point322) point322

func demo322(a point322) point322 {
	return a
}

func init() {
	a := point322{1, 2, 3, 4}
	assert("cdemo322\000", cdemo322(a) == a)
	assert("demo322\000", demo322(a) == a)
}

type point323 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo323 C.demo323
func cdemo323(a point323) point323

func demo323(a point323) point323 {
	return a
}

func init() {
	a := point323{1, 2, 3, 4, 5}
	assert("cdemo323\000", cdemo323(a) == a)
	assert("demo323\000", demo323(a) == a)
}

type point324 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo324 C.demo324
func cdemo324(a point324) point324

func demo324(a point324) point324 {
	return a
}

func init() {
	a := point324{1, 2, 3, 4, 5}
	assert("cdemo324\000", cdemo324(a) == a)
	assert("demo324\000", demo324(a) == a)
}

type point325 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo325 C.demo325
func cdemo325(a point325) point325

func demo325(a point325) point325 {
	return a
}

func init() {
	a := point325{1, 2, 3, 4, 5}
	assert("cdemo325\000", cdemo325(a) == a)
	assert("demo325\000", demo325(a) == a)
}

type point326 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo326 C.demo326
func cdemo326(a point326) point326

func demo326(a point326) point326 {
	return a
}

func init() {
	a := point326{1, 2, 3, 4, 5}
	assert("cdemo326\000", cdemo326(a) == a)
	assert("demo326\000", demo326(a) == a)
}

type point327 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int32
}

//go:linkname cdemo327 C.demo327
func cdemo327(a point327) point327

func demo327(a point327) point327 {
	return a
}

func init() {
	a := point327{1, 2, 3, 4}
	assert("cdemo327\000", cdemo327(a) == a)
	assert("demo327\000", demo327(a) == a)
}

type point328 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo328 C.demo328
func cdemo328(a point328) point328

func demo328(a point328) point328 {
	return a
}

func init() {
	a := point328{1, 2, 3, 4, 5}
	assert("cdemo328\000", cdemo328(a) == a)
	assert("demo328\000", demo328(a) == a)
}

type point329 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo329 C.demo329
func cdemo329(a point329) point329

func demo329(a point329) point329 {
	return a
}

func init() {
	a := point329{1, 2, 3, 4, 5}
	assert("cdemo329\000", cdemo329(a) == a)
	assert("demo329\000", demo329(a) == a)
}

type point330 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo330 C.demo330
func cdemo330(a point330) point330

func demo330(a point330) point330 {
	return a
}

func init() {
	a := point330{1, 2, 3, 4, 5}
	assert("cdemo330\000", cdemo330(a) == a)
	assert("demo330\000", demo330(a) == a)
}

type point331 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo331 C.demo331
func cdemo331(a point331) point331

func demo331(a point331) point331 {
	return a
}

func init() {
	a := point331{1, 2, 3, 4, 5}
	assert("cdemo331\000", cdemo331(a) == a)
	assert("demo331\000", demo331(a) == a)
}

type point332 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 float32
}

//go:linkname cdemo332 C.demo332
func cdemo332(a point332) point332

func demo332(a point332) point332 {
	return a
}

func init() {
	a := point332{1, 2, 3, 4}
	assert("cdemo332\000", cdemo332(a) == a)
	assert("demo332\000", demo332(a) == a)
}

type point333 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo333 C.demo333
func cdemo333(a point333) point333

func demo333(a point333) point333 {
	return a
}

func init() {
	a := point333{1, 2, 3, 4, 5}
	assert("cdemo333\000", cdemo333(a) == a)
	assert("demo333\000", demo333(a) == a)
}

type point334 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo334 C.demo334
func cdemo334(a point334) point334

func demo334(a point334) point334 {
	return a
}

func init() {
	a := point334{1, 2, 3, 4, 5}
	assert("cdemo334\000", cdemo334(a) == a)
	assert("demo334\000", demo334(a) == a)
}

type point335 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo335 C.demo335
func cdemo335(a point335) point335

func demo335(a point335) point335 {
	return a
}

func init() {
	a := point335{1, 2, 3, 4, 5}
	assert("cdemo335\000", cdemo335(a) == a)
	assert("demo335\000", demo335(a) == a)
}

type point336 struct {
	x0 int16
	x1 int8
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo336 C.demo336
func cdemo336(a point336) point336

func demo336(a point336) point336 {
	return a
}

func init() {
	a := point336{1, 2, 3, 4, 5}
	assert("cdemo336\000", cdemo336(a) == a)
	assert("demo336\000", demo336(a) == a)
}

type point337 struct {
	x0 int16
	x1 int8
	x2 int32
}

//go:linkname cdemo337 C.demo337
func cdemo337(a point337) point337

func demo337(a point337) point337 {
	return a
}

func init() {
	a := point337{1, 2, 3}
	assert("cdemo337\000", cdemo337(a) == a)
	assert("demo337\000", demo337(a) == a)
}

type point338 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int8
}

//go:linkname cdemo338 C.demo338
func cdemo338(a point338) point338

func demo338(a point338) point338 {
	return a
}

func init() {
	a := point338{1, 2, 3, 4}
	assert("cdemo338\000", cdemo338(a) == a)
	assert("demo338\000", demo338(a) == a)
}

type point339 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo339 C.demo339
func cdemo339(a point339) point339

func demo339(a point339) point339 {
	return a
}

func init() {
	a := point339{1, 2, 3, 4, 5}
	assert("cdemo339\000", cdemo339(a) == a)
	assert("demo339\000", demo339(a) == a)
}

type point340 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo340 C.demo340
func cdemo340(a point340) point340

func demo340(a point340) point340 {
	return a
}

func init() {
	a := point340{1, 2, 3, 4, 5}
	assert("cdemo340\000", cdemo340(a) == a)
	assert("demo340\000", demo340(a) == a)
}

type point341 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo341 C.demo341
func cdemo341(a point341) point341

func demo341(a point341) point341 {
	return a
}

func init() {
	a := point341{1, 2, 3, 4, 5}
	assert("cdemo341\000", cdemo341(a) == a)
	assert("demo341\000", demo341(a) == a)
}

type point342 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo342 C.demo342
func cdemo342(a point342) point342

func demo342(a point342) point342 {
	return a
}

func init() {
	a := point342{1, 2, 3, 4, 5}
	assert("cdemo342\000", cdemo342(a) == a)
	assert("demo342\000", demo342(a) == a)
}

type point343 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int16
}

//go:linkname cdemo343 C.demo343
func cdemo343(a point343) point343

func demo343(a point343) point343 {
	return a
}

func init() {
	a := point343{1, 2, 3, 4}
	assert("cdemo343\000", cdemo343(a) == a)
	assert("demo343\000", demo343(a) == a)
}

type point344 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo344 C.demo344
func cdemo344(a point344) point344

func demo344(a point344) point344 {
	return a
}

func init() {
	a := point344{1, 2, 3, 4, 5}
	assert("cdemo344\000", cdemo344(a) == a)
	assert("demo344\000", demo344(a) == a)
}

type point345 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo345 C.demo345
func cdemo345(a point345) point345

func demo345(a point345) point345 {
	return a
}

func init() {
	a := point345{1, 2, 3, 4, 5}
	assert("cdemo345\000", cdemo345(a) == a)
	assert("demo345\000", demo345(a) == a)
}

type point346 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo346 C.demo346
func cdemo346(a point346) point346

func demo346(a point346) point346 {
	return a
}

func init() {
	a := point346{1, 2, 3, 4, 5}
	assert("cdemo346\000", cdemo346(a) == a)
	assert("demo346\000", demo346(a) == a)
}

type point347 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo347 C.demo347
func cdemo347(a point347) point347

func demo347(a point347) point347 {
	return a
}

func init() {
	a := point347{1, 2, 3, 4, 5}
	assert("cdemo347\000", cdemo347(a) == a)
	assert("demo347\000", demo347(a) == a)
}

type point348 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int32
}

//go:linkname cdemo348 C.demo348
func cdemo348(a point348) point348

func demo348(a point348) point348 {
	return a
}

func init() {
	a := point348{1, 2, 3, 4}
	assert("cdemo348\000", cdemo348(a) == a)
	assert("demo348\000", demo348(a) == a)
}

type point349 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo349 C.demo349
func cdemo349(a point349) point349

func demo349(a point349) point349 {
	return a
}

func init() {
	a := point349{1, 2, 3, 4, 5}
	assert("cdemo349\000", cdemo349(a) == a)
	assert("demo349\000", demo349(a) == a)
}

type point350 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo350 C.demo350
func cdemo350(a point350) point350

func demo350(a point350) point350 {
	return a
}

func init() {
	a := point350{1, 2, 3, 4, 5}
	assert("cdemo350\000", cdemo350(a) == a)
	assert("demo350\000", demo350(a) == a)
}

type point351 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo351 C.demo351
func cdemo351(a point351) point351

func demo351(a point351) point351 {
	return a
}

func init() {
	a := point351{1, 2, 3, 4, 5}
	assert("cdemo351\000", cdemo351(a) == a)
	assert("demo351\000", demo351(a) == a)
}

type point352 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo352 C.demo352
func cdemo352(a point352) point352

func demo352(a point352) point352 {
	return a
}

func init() {
	a := point352{1, 2, 3, 4, 5}
	assert("cdemo352\000", cdemo352(a) == a)
	assert("demo352\000", demo352(a) == a)
}

type point353 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 float32
}

//go:linkname cdemo353 C.demo353
func cdemo353(a point353) point353

func demo353(a point353) point353 {
	return a
}

func init() {
	a := point353{1, 2, 3, 4}
	assert("cdemo353\000", cdemo353(a) == a)
	assert("demo353\000", demo353(a) == a)
}

type point354 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo354 C.demo354
func cdemo354(a point354) point354

func demo354(a point354) point354 {
	return a
}

func init() {
	a := point354{1, 2, 3, 4, 5}
	assert("cdemo354\000", cdemo354(a) == a)
	assert("demo354\000", demo354(a) == a)
}

type point355 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo355 C.demo355
func cdemo355(a point355) point355

func demo355(a point355) point355 {
	return a
}

func init() {
	a := point355{1, 2, 3, 4, 5}
	assert("cdemo355\000", cdemo355(a) == a)
	assert("demo355\000", demo355(a) == a)
}

type point356 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo356 C.demo356
func cdemo356(a point356) point356

func demo356(a point356) point356 {
	return a
}

func init() {
	a := point356{1, 2, 3, 4, 5}
	assert("cdemo356\000", cdemo356(a) == a)
	assert("demo356\000", demo356(a) == a)
}

type point357 struct {
	x0 int16
	x1 int8
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo357 C.demo357
func cdemo357(a point357) point357

func demo357(a point357) point357 {
	return a
}

func init() {
	a := point357{1, 2, 3, 4, 5}
	assert("cdemo357\000", cdemo357(a) == a)
	assert("demo357\000", demo357(a) == a)
}

type point358 struct {
	x0 int16
	x1 int8
	x2 float32
}

//go:linkname cdemo358 C.demo358
func cdemo358(a point358) point358

func demo358(a point358) point358 {
	return a
}

func init() {
	a := point358{1, 2, 3}
	assert("cdemo358\000", cdemo358(a) == a)
	assert("demo358\000", demo358(a) == a)
}

type point359 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int8
}

//go:linkname cdemo359 C.demo359
func cdemo359(a point359) point359

func demo359(a point359) point359 {
	return a
}

func init() {
	a := point359{1, 2, 3, 4}
	assert("cdemo359\000", cdemo359(a) == a)
	assert("demo359\000", demo359(a) == a)
}

type point360 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo360 C.demo360
func cdemo360(a point360) point360

func demo360(a point360) point360 {
	return a
}

func init() {
	a := point360{1, 2, 3, 4, 5}
	assert("cdemo360\000", cdemo360(a) == a)
	assert("demo360\000", demo360(a) == a)
}

type point361 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo361 C.demo361
func cdemo361(a point361) point361

func demo361(a point361) point361 {
	return a
}

func init() {
	a := point361{1, 2, 3, 4, 5}
	assert("cdemo361\000", cdemo361(a) == a)
	assert("demo361\000", demo361(a) == a)
}

type point362 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo362 C.demo362
func cdemo362(a point362) point362

func demo362(a point362) point362 {
	return a
}

func init() {
	a := point362{1, 2, 3, 4, 5}
	assert("cdemo362\000", cdemo362(a) == a)
	assert("demo362\000", demo362(a) == a)
}

type point363 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo363 C.demo363
func cdemo363(a point363) point363

func demo363(a point363) point363 {
	return a
}

func init() {
	a := point363{1, 2, 3, 4, 5}
	assert("cdemo363\000", cdemo363(a) == a)
	assert("demo363\000", demo363(a) == a)
}

type point364 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int16
}

//go:linkname cdemo364 C.demo364
func cdemo364(a point364) point364

func demo364(a point364) point364 {
	return a
}

func init() {
	a := point364{1, 2, 3, 4}
	assert("cdemo364\000", cdemo364(a) == a)
	assert("demo364\000", demo364(a) == a)
}

type point365 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo365 C.demo365
func cdemo365(a point365) point365

func demo365(a point365) point365 {
	return a
}

func init() {
	a := point365{1, 2, 3, 4, 5}
	assert("cdemo365\000", cdemo365(a) == a)
	assert("demo365\000", demo365(a) == a)
}

type point366 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo366 C.demo366
func cdemo366(a point366) point366

func demo366(a point366) point366 {
	return a
}

func init() {
	a := point366{1, 2, 3, 4, 5}
	assert("cdemo366\000", cdemo366(a) == a)
	assert("demo366\000", demo366(a) == a)
}

type point367 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo367 C.demo367
func cdemo367(a point367) point367

func demo367(a point367) point367 {
	return a
}

func init() {
	a := point367{1, 2, 3, 4, 5}
	assert("cdemo367\000", cdemo367(a) == a)
	assert("demo367\000", demo367(a) == a)
}

type point368 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo368 C.demo368
func cdemo368(a point368) point368

func demo368(a point368) point368 {
	return a
}

func init() {
	a := point368{1, 2, 3, 4, 5}
	assert("cdemo368\000", cdemo368(a) == a)
	assert("demo368\000", demo368(a) == a)
}

type point369 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int32
}

//go:linkname cdemo369 C.demo369
func cdemo369(a point369) point369

func demo369(a point369) point369 {
	return a
}

func init() {
	a := point369{1, 2, 3, 4}
	assert("cdemo369\000", cdemo369(a) == a)
	assert("demo369\000", demo369(a) == a)
}

type point370 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo370 C.demo370
func cdemo370(a point370) point370

func demo370(a point370) point370 {
	return a
}

func init() {
	a := point370{1, 2, 3, 4, 5}
	assert("cdemo370\000", cdemo370(a) == a)
	assert("demo370\000", demo370(a) == a)
}

type point371 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo371 C.demo371
func cdemo371(a point371) point371

func demo371(a point371) point371 {
	return a
}

func init() {
	a := point371{1, 2, 3, 4, 5}
	assert("cdemo371\000", cdemo371(a) == a)
	assert("demo371\000", demo371(a) == a)
}

type point372 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo372 C.demo372
func cdemo372(a point372) point372

func demo372(a point372) point372 {
	return a
}

func init() {
	a := point372{1, 2, 3, 4, 5}
	assert("cdemo372\000", cdemo372(a) == a)
	assert("demo372\000", demo372(a) == a)
}

type point373 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo373 C.demo373
func cdemo373(a point373) point373

func demo373(a point373) point373 {
	return a
}

func init() {
	a := point373{1, 2, 3, 4, 5}
	assert("cdemo373\000", cdemo373(a) == a)
	assert("demo373\000", demo373(a) == a)
}

type point374 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 float32
}

//go:linkname cdemo374 C.demo374
func cdemo374(a point374) point374

func demo374(a point374) point374 {
	return a
}

func init() {
	a := point374{1, 2, 3, 4}
	assert("cdemo374\000", cdemo374(a) == a)
	assert("demo374\000", demo374(a) == a)
}

type point375 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo375 C.demo375
func cdemo375(a point375) point375

func demo375(a point375) point375 {
	return a
}

func init() {
	a := point375{1, 2, 3, 4, 5}
	assert("cdemo375\000", cdemo375(a) == a)
	assert("demo375\000", demo375(a) == a)
}

type point376 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo376 C.demo376
func cdemo376(a point376) point376

func demo376(a point376) point376 {
	return a
}

func init() {
	a := point376{1, 2, 3, 4, 5}
	assert("cdemo376\000", cdemo376(a) == a)
	assert("demo376\000", demo376(a) == a)
}

type point377 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo377 C.demo377
func cdemo377(a point377) point377

func demo377(a point377) point377 {
	return a
}

func init() {
	a := point377{1, 2, 3, 4, 5}
	assert("cdemo377\000", cdemo377(a) == a)
	assert("demo377\000", demo377(a) == a)
}

type point378 struct {
	x0 int16
	x1 int8
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo378 C.demo378
func cdemo378(a point378) point378

func demo378(a point378) point378 {
	return a
}

func init() {
	a := point378{1, 2, 3, 4, 5}
	assert("cdemo378\000", cdemo378(a) == a)
	assert("demo378\000", demo378(a) == a)
}

type point379 struct {
	x0 int16
	x1 int32
	x2 int8
}

//go:linkname cdemo379 C.demo379
func cdemo379(a point379) point379

func demo379(a point379) point379 {
	return a
}

func init() {
	a := point379{1, 2, 3}
	assert("cdemo379\000", cdemo379(a) == a)
	assert("demo379\000", demo379(a) == a)
}

type point380 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int8
}

//go:linkname cdemo380 C.demo380
func cdemo380(a point380) point380

func demo380(a point380) point380 {
	return a
}

func init() {
	a := point380{1, 2, 3, 4}
	assert("cdemo380\000", cdemo380(a) == a)
	assert("demo380\000", demo380(a) == a)
}

type point381 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo381 C.demo381
func cdemo381(a point381) point381

func demo381(a point381) point381 {
	return a
}

func init() {
	a := point381{1, 2, 3, 4, 5}
	assert("cdemo381\000", cdemo381(a) == a)
	assert("demo381\000", demo381(a) == a)
}

type point382 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo382 C.demo382
func cdemo382(a point382) point382

func demo382(a point382) point382 {
	return a
}

func init() {
	a := point382{1, 2, 3, 4, 5}
	assert("cdemo382\000", cdemo382(a) == a)
	assert("demo382\000", demo382(a) == a)
}

type point383 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo383 C.demo383
func cdemo383(a point383) point383

func demo383(a point383) point383 {
	return a
}

func init() {
	a := point383{1, 2, 3, 4, 5}
	assert("cdemo383\000", cdemo383(a) == a)
	assert("demo383\000", demo383(a) == a)
}

type point384 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo384 C.demo384
func cdemo384(a point384) point384

func demo384(a point384) point384 {
	return a
}

func init() {
	a := point384{1, 2, 3, 4, 5}
	assert("cdemo384\000", cdemo384(a) == a)
	assert("demo384\000", demo384(a) == a)
}

type point385 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int16
}

//go:linkname cdemo385 C.demo385
func cdemo385(a point385) point385

func demo385(a point385) point385 {
	return a
}

func init() {
	a := point385{1, 2, 3, 4}
	assert("cdemo385\000", cdemo385(a) == a)
	assert("demo385\000", demo385(a) == a)
}

type point386 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo386 C.demo386
func cdemo386(a point386) point386

func demo386(a point386) point386 {
	return a
}

func init() {
	a := point386{1, 2, 3, 4, 5}
	assert("cdemo386\000", cdemo386(a) == a)
	assert("demo386\000", demo386(a) == a)
}

type point387 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo387 C.demo387
func cdemo387(a point387) point387

func demo387(a point387) point387 {
	return a
}

func init() {
	a := point387{1, 2, 3, 4, 5}
	assert("cdemo387\000", cdemo387(a) == a)
	assert("demo387\000", demo387(a) == a)
}

type point388 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo388 C.demo388
func cdemo388(a point388) point388

func demo388(a point388) point388 {
	return a
}

func init() {
	a := point388{1, 2, 3, 4, 5}
	assert("cdemo388\000", cdemo388(a) == a)
	assert("demo388\000", demo388(a) == a)
}

type point389 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo389 C.demo389
func cdemo389(a point389) point389

func demo389(a point389) point389 {
	return a
}

func init() {
	a := point389{1, 2, 3, 4, 5}
	assert("cdemo389\000", cdemo389(a) == a)
	assert("demo389\000", demo389(a) == a)
}

type point390 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int32
}

//go:linkname cdemo390 C.demo390
func cdemo390(a point390) point390

func demo390(a point390) point390 {
	return a
}

func init() {
	a := point390{1, 2, 3, 4}
	assert("cdemo390\000", cdemo390(a) == a)
	assert("demo390\000", demo390(a) == a)
}

type point391 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo391 C.demo391
func cdemo391(a point391) point391

func demo391(a point391) point391 {
	return a
}

func init() {
	a := point391{1, 2, 3, 4, 5}
	assert("cdemo391\000", cdemo391(a) == a)
	assert("demo391\000", demo391(a) == a)
}

type point392 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo392 C.demo392
func cdemo392(a point392) point392

func demo392(a point392) point392 {
	return a
}

func init() {
	a := point392{1, 2, 3, 4, 5}
	assert("cdemo392\000", cdemo392(a) == a)
	assert("demo392\000", demo392(a) == a)
}

type point393 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo393 C.demo393
func cdemo393(a point393) point393

func demo393(a point393) point393 {
	return a
}

func init() {
	a := point393{1, 2, 3, 4, 5}
	assert("cdemo393\000", cdemo393(a) == a)
	assert("demo393\000", demo393(a) == a)
}

type point394 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo394 C.demo394
func cdemo394(a point394) point394

func demo394(a point394) point394 {
	return a
}

func init() {
	a := point394{1, 2, 3, 4, 5}
	assert("cdemo394\000", cdemo394(a) == a)
	assert("demo394\000", demo394(a) == a)
}

type point395 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 float32
}

//go:linkname cdemo395 C.demo395
func cdemo395(a point395) point395

func demo395(a point395) point395 {
	return a
}

func init() {
	a := point395{1, 2, 3, 4}
	assert("cdemo395\000", cdemo395(a) == a)
	assert("demo395\000", demo395(a) == a)
}

type point396 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo396 C.demo396
func cdemo396(a point396) point396

func demo396(a point396) point396 {
	return a
}

func init() {
	a := point396{1, 2, 3, 4, 5}
	assert("cdemo396\000", cdemo396(a) == a)
	assert("demo396\000", demo396(a) == a)
}

type point397 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo397 C.demo397
func cdemo397(a point397) point397

func demo397(a point397) point397 {
	return a
}

func init() {
	a := point397{1, 2, 3, 4, 5}
	assert("cdemo397\000", cdemo397(a) == a)
	assert("demo397\000", demo397(a) == a)
}

type point398 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo398 C.demo398
func cdemo398(a point398) point398

func demo398(a point398) point398 {
	return a
}

func init() {
	a := point398{1, 2, 3, 4, 5}
	assert("cdemo398\000", cdemo398(a) == a)
	assert("demo398\000", demo398(a) == a)
}

type point399 struct {
	x0 int16
	x1 int32
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo399 C.demo399
func cdemo399(a point399) point399

func demo399(a point399) point399 {
	return a
}

func init() {
	a := point399{1, 2, 3, 4, 5}
	assert("cdemo399\000", cdemo399(a) == a)
	assert("demo399\000", demo399(a) == a)
}

type point400 struct {
	x0 int16
	x1 int32
	x2 int16
}

//go:linkname cdemo400 C.demo400
func cdemo400(a point400) point400

func demo400(a point400) point400 {
	return a
}

func init() {
	a := point400{1, 2, 3}
	assert("cdemo400\000", cdemo400(a) == a)
	assert("demo400\000", demo400(a) == a)
}

type point401 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int8
}

//go:linkname cdemo401 C.demo401
func cdemo401(a point401) point401

func demo401(a point401) point401 {
	return a
}

func init() {
	a := point401{1, 2, 3, 4}
	assert("cdemo401\000", cdemo401(a) == a)
	assert("demo401\000", demo401(a) == a)
}

type point402 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo402 C.demo402
func cdemo402(a point402) point402

func demo402(a point402) point402 {
	return a
}

func init() {
	a := point402{1, 2, 3, 4, 5}
	assert("cdemo402\000", cdemo402(a) == a)
	assert("demo402\000", demo402(a) == a)
}

type point403 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo403 C.demo403
func cdemo403(a point403) point403

func demo403(a point403) point403 {
	return a
}

func init() {
	a := point403{1, 2, 3, 4, 5}
	assert("cdemo403\000", cdemo403(a) == a)
	assert("demo403\000", demo403(a) == a)
}

type point404 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo404 C.demo404
func cdemo404(a point404) point404

func demo404(a point404) point404 {
	return a
}

func init() {
	a := point404{1, 2, 3, 4, 5}
	assert("cdemo404\000", cdemo404(a) == a)
	assert("demo404\000", demo404(a) == a)
}

type point405 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo405 C.demo405
func cdemo405(a point405) point405

func demo405(a point405) point405 {
	return a
}

func init() {
	a := point405{1, 2, 3, 4, 5}
	assert("cdemo405\000", cdemo405(a) == a)
	assert("demo405\000", demo405(a) == a)
}

type point406 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int16
}

//go:linkname cdemo406 C.demo406
func cdemo406(a point406) point406

func demo406(a point406) point406 {
	return a
}

func init() {
	a := point406{1, 2, 3, 4}
	assert("cdemo406\000", cdemo406(a) == a)
	assert("demo406\000", demo406(a) == a)
}

type point407 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo407 C.demo407
func cdemo407(a point407) point407

func demo407(a point407) point407 {
	return a
}

func init() {
	a := point407{1, 2, 3, 4, 5}
	assert("cdemo407\000", cdemo407(a) == a)
	assert("demo407\000", demo407(a) == a)
}

type point408 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo408 C.demo408
func cdemo408(a point408) point408

func demo408(a point408) point408 {
	return a
}

func init() {
	a := point408{1, 2, 3, 4, 5}
	assert("cdemo408\000", cdemo408(a) == a)
	assert("demo408\000", demo408(a) == a)
}

type point409 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo409 C.demo409
func cdemo409(a point409) point409

func demo409(a point409) point409 {
	return a
}

func init() {
	a := point409{1, 2, 3, 4, 5}
	assert("cdemo409\000", cdemo409(a) == a)
	assert("demo409\000", demo409(a) == a)
}

type point410 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo410 C.demo410
func cdemo410(a point410) point410

func demo410(a point410) point410 {
	return a
}

func init() {
	a := point410{1, 2, 3, 4, 5}
	assert("cdemo410\000", cdemo410(a) == a)
	assert("demo410\000", demo410(a) == a)
}

type point411 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int32
}

//go:linkname cdemo411 C.demo411
func cdemo411(a point411) point411

func demo411(a point411) point411 {
	return a
}

func init() {
	a := point411{1, 2, 3, 4}
	assert("cdemo411\000", cdemo411(a) == a)
	assert("demo411\000", demo411(a) == a)
}

type point412 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo412 C.demo412
func cdemo412(a point412) point412

func demo412(a point412) point412 {
	return a
}

func init() {
	a := point412{1, 2, 3, 4, 5}
	assert("cdemo412\000", cdemo412(a) == a)
	assert("demo412\000", demo412(a) == a)
}

type point413 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo413 C.demo413
func cdemo413(a point413) point413

func demo413(a point413) point413 {
	return a
}

func init() {
	a := point413{1, 2, 3, 4, 5}
	assert("cdemo413\000", cdemo413(a) == a)
	assert("demo413\000", demo413(a) == a)
}

type point414 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo414 C.demo414
func cdemo414(a point414) point414

func demo414(a point414) point414 {
	return a
}

func init() {
	a := point414{1, 2, 3, 4, 5}
	assert("cdemo414\000", cdemo414(a) == a)
	assert("demo414\000", demo414(a) == a)
}

type point415 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo415 C.demo415
func cdemo415(a point415) point415

func demo415(a point415) point415 {
	return a
}

func init() {
	a := point415{1, 2, 3, 4, 5}
	assert("cdemo415\000", cdemo415(a) == a)
	assert("demo415\000", demo415(a) == a)
}

type point416 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 float32
}

//go:linkname cdemo416 C.demo416
func cdemo416(a point416) point416

func demo416(a point416) point416 {
	return a
}

func init() {
	a := point416{1, 2, 3, 4}
	assert("cdemo416\000", cdemo416(a) == a)
	assert("demo416\000", demo416(a) == a)
}

type point417 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo417 C.demo417
func cdemo417(a point417) point417

func demo417(a point417) point417 {
	return a
}

func init() {
	a := point417{1, 2, 3, 4, 5}
	assert("cdemo417\000", cdemo417(a) == a)
	assert("demo417\000", demo417(a) == a)
}

type point418 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo418 C.demo418
func cdemo418(a point418) point418

func demo418(a point418) point418 {
	return a
}

func init() {
	a := point418{1, 2, 3, 4, 5}
	assert("cdemo418\000", cdemo418(a) == a)
	assert("demo418\000", demo418(a) == a)
}

type point419 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo419 C.demo419
func cdemo419(a point419) point419

func demo419(a point419) point419 {
	return a
}

func init() {
	a := point419{1, 2, 3, 4, 5}
	assert("cdemo419\000", cdemo419(a) == a)
	assert("demo419\000", demo419(a) == a)
}

type point420 struct {
	x0 int16
	x1 int32
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo420 C.demo420
func cdemo420(a point420) point420

func demo420(a point420) point420 {
	return a
}

func init() {
	a := point420{1, 2, 3, 4, 5}
	assert("cdemo420\000", cdemo420(a) == a)
	assert("demo420\000", demo420(a) == a)
}

type point421 struct {
	x0 int16
	x1 int32
	x2 int32
}

//go:linkname cdemo421 C.demo421
func cdemo421(a point421) point421

func demo421(a point421) point421 {
	return a
}

func init() {
	a := point421{1, 2, 3}
	assert("cdemo421\000", cdemo421(a) == a)
	assert("demo421\000", demo421(a) == a)
}

type point422 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int8
}

//go:linkname cdemo422 C.demo422
func cdemo422(a point422) point422

func demo422(a point422) point422 {
	return a
}

func init() {
	a := point422{1, 2, 3, 4}
	assert("cdemo422\000", cdemo422(a) == a)
	assert("demo422\000", demo422(a) == a)
}

type point423 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo423 C.demo423
func cdemo423(a point423) point423

func demo423(a point423) point423 {
	return a
}

func init() {
	a := point423{1, 2, 3, 4, 5}
	assert("cdemo423\000", cdemo423(a) == a)
	assert("demo423\000", demo423(a) == a)
}

type point424 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo424 C.demo424
func cdemo424(a point424) point424

func demo424(a point424) point424 {
	return a
}

func init() {
	a := point424{1, 2, 3, 4, 5}
	assert("cdemo424\000", cdemo424(a) == a)
	assert("demo424\000", demo424(a) == a)
}

type point425 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo425 C.demo425
func cdemo425(a point425) point425

func demo425(a point425) point425 {
	return a
}

func init() {
	a := point425{1, 2, 3, 4, 5}
	assert("cdemo425\000", cdemo425(a) == a)
	assert("demo425\000", demo425(a) == a)
}

type point426 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo426 C.demo426
func cdemo426(a point426) point426

func demo426(a point426) point426 {
	return a
}

func init() {
	a := point426{1, 2, 3, 4, 5}
	assert("cdemo426\000", cdemo426(a) == a)
	assert("demo426\000", demo426(a) == a)
}

type point427 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int16
}

//go:linkname cdemo427 C.demo427
func cdemo427(a point427) point427

func demo427(a point427) point427 {
	return a
}

func init() {
	a := point427{1, 2, 3, 4}
	assert("cdemo427\000", cdemo427(a) == a)
	assert("demo427\000", demo427(a) == a)
}

type point428 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo428 C.demo428
func cdemo428(a point428) point428

func demo428(a point428) point428 {
	return a
}

func init() {
	a := point428{1, 2, 3, 4, 5}
	assert("cdemo428\000", cdemo428(a) == a)
	assert("demo428\000", demo428(a) == a)
}

type point429 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo429 C.demo429
func cdemo429(a point429) point429

func demo429(a point429) point429 {
	return a
}

func init() {
	a := point429{1, 2, 3, 4, 5}
	assert("cdemo429\000", cdemo429(a) == a)
	assert("demo429\000", demo429(a) == a)
}

type point430 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo430 C.demo430
func cdemo430(a point430) point430

func demo430(a point430) point430 {
	return a
}

func init() {
	a := point430{1, 2, 3, 4, 5}
	assert("cdemo430\000", cdemo430(a) == a)
	assert("demo430\000", demo430(a) == a)
}

type point431 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo431 C.demo431
func cdemo431(a point431) point431

func demo431(a point431) point431 {
	return a
}

func init() {
	a := point431{1, 2, 3, 4, 5}
	assert("cdemo431\000", cdemo431(a) == a)
	assert("demo431\000", demo431(a) == a)
}

type point432 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int32
}

//go:linkname cdemo432 C.demo432
func cdemo432(a point432) point432

func demo432(a point432) point432 {
	return a
}

func init() {
	a := point432{1, 2, 3, 4}
	assert("cdemo432\000", cdemo432(a) == a)
	assert("demo432\000", demo432(a) == a)
}

type point433 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo433 C.demo433
func cdemo433(a point433) point433

func demo433(a point433) point433 {
	return a
}

func init() {
	a := point433{1, 2, 3, 4, 5}
	assert("cdemo433\000", cdemo433(a) == a)
	assert("demo433\000", demo433(a) == a)
}

type point434 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo434 C.demo434
func cdemo434(a point434) point434

func demo434(a point434) point434 {
	return a
}

func init() {
	a := point434{1, 2, 3, 4, 5}
	assert("cdemo434\000", cdemo434(a) == a)
	assert("demo434\000", demo434(a) == a)
}

type point435 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo435 C.demo435
func cdemo435(a point435) point435

func demo435(a point435) point435 {
	return a
}

func init() {
	a := point435{1, 2, 3, 4, 5}
	assert("cdemo435\000", cdemo435(a) == a)
	assert("demo435\000", demo435(a) == a)
}

type point436 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo436 C.demo436
func cdemo436(a point436) point436

func demo436(a point436) point436 {
	return a
}

func init() {
	a := point436{1, 2, 3, 4, 5}
	assert("cdemo436\000", cdemo436(a) == a)
	assert("demo436\000", demo436(a) == a)
}

type point437 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 float32
}

//go:linkname cdemo437 C.demo437
func cdemo437(a point437) point437

func demo437(a point437) point437 {
	return a
}

func init() {
	a := point437{1, 2, 3, 4}
	assert("cdemo437\000", cdemo437(a) == a)
	assert("demo437\000", demo437(a) == a)
}

type point438 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo438 C.demo438
func cdemo438(a point438) point438

func demo438(a point438) point438 {
	return a
}

func init() {
	a := point438{1, 2, 3, 4, 5}
	assert("cdemo438\000", cdemo438(a) == a)
	assert("demo438\000", demo438(a) == a)
}

type point439 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo439 C.demo439
func cdemo439(a point439) point439

func demo439(a point439) point439 {
	return a
}

func init() {
	a := point439{1, 2, 3, 4, 5}
	assert("cdemo439\000", cdemo439(a) == a)
	assert("demo439\000", demo439(a) == a)
}

type point440 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo440 C.demo440
func cdemo440(a point440) point440

func demo440(a point440) point440 {
	return a
}

func init() {
	a := point440{1, 2, 3, 4, 5}
	assert("cdemo440\000", cdemo440(a) == a)
	assert("demo440\000", demo440(a) == a)
}

type point441 struct {
	x0 int16
	x1 int32
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo441 C.demo441
func cdemo441(a point441) point441

func demo441(a point441) point441 {
	return a
}

func init() {
	a := point441{1, 2, 3, 4, 5}
	assert("cdemo441\000", cdemo441(a) == a)
	assert("demo441\000", demo441(a) == a)
}

type point442 struct {
	x0 int16
	x1 int32
	x2 float32
}

//go:linkname cdemo442 C.demo442
func cdemo442(a point442) point442

func demo442(a point442) point442 {
	return a
}

func init() {
	a := point442{1, 2, 3}
	assert("cdemo442\000", cdemo442(a) == a)
	assert("demo442\000", demo442(a) == a)
}

type point443 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int8
}

//go:linkname cdemo443 C.demo443
func cdemo443(a point443) point443

func demo443(a point443) point443 {
	return a
}

func init() {
	a := point443{1, 2, 3, 4}
	assert("cdemo443\000", cdemo443(a) == a)
	assert("demo443\000", demo443(a) == a)
}

type point444 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo444 C.demo444
func cdemo444(a point444) point444

func demo444(a point444) point444 {
	return a
}

func init() {
	a := point444{1, 2, 3, 4, 5}
	assert("cdemo444\000", cdemo444(a) == a)
	assert("demo444\000", demo444(a) == a)
}

type point445 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo445 C.demo445
func cdemo445(a point445) point445

func demo445(a point445) point445 {
	return a
}

func init() {
	a := point445{1, 2, 3, 4, 5}
	assert("cdemo445\000", cdemo445(a) == a)
	assert("demo445\000", demo445(a) == a)
}

type point446 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo446 C.demo446
func cdemo446(a point446) point446

func demo446(a point446) point446 {
	return a
}

func init() {
	a := point446{1, 2, 3, 4, 5}
	assert("cdemo446\000", cdemo446(a) == a)
	assert("demo446\000", demo446(a) == a)
}

type point447 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo447 C.demo447
func cdemo447(a point447) point447

func demo447(a point447) point447 {
	return a
}

func init() {
	a := point447{1, 2, 3, 4, 5}
	assert("cdemo447\000", cdemo447(a) == a)
	assert("demo447\000", demo447(a) == a)
}

type point448 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int16
}

//go:linkname cdemo448 C.demo448
func cdemo448(a point448) point448

func demo448(a point448) point448 {
	return a
}

func init() {
	a := point448{1, 2, 3, 4}
	assert("cdemo448\000", cdemo448(a) == a)
	assert("demo448\000", demo448(a) == a)
}

type point449 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo449 C.demo449
func cdemo449(a point449) point449

func demo449(a point449) point449 {
	return a
}

func init() {
	a := point449{1, 2, 3, 4, 5}
	assert("cdemo449\000", cdemo449(a) == a)
	assert("demo449\000", demo449(a) == a)
}

type point450 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo450 C.demo450
func cdemo450(a point450) point450

func demo450(a point450) point450 {
	return a
}

func init() {
	a := point450{1, 2, 3, 4, 5}
	assert("cdemo450\000", cdemo450(a) == a)
	assert("demo450\000", demo450(a) == a)
}

type point451 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo451 C.demo451
func cdemo451(a point451) point451

func demo451(a point451) point451 {
	return a
}

func init() {
	a := point451{1, 2, 3, 4, 5}
	assert("cdemo451\000", cdemo451(a) == a)
	assert("demo451\000", demo451(a) == a)
}

type point452 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo452 C.demo452
func cdemo452(a point452) point452

func demo452(a point452) point452 {
	return a
}

func init() {
	a := point452{1, 2, 3, 4, 5}
	assert("cdemo452\000", cdemo452(a) == a)
	assert("demo452\000", demo452(a) == a)
}

type point453 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int32
}

//go:linkname cdemo453 C.demo453
func cdemo453(a point453) point453

func demo453(a point453) point453 {
	return a
}

func init() {
	a := point453{1, 2, 3, 4}
	assert("cdemo453\000", cdemo453(a) == a)
	assert("demo453\000", demo453(a) == a)
}

type point454 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo454 C.demo454
func cdemo454(a point454) point454

func demo454(a point454) point454 {
	return a
}

func init() {
	a := point454{1, 2, 3, 4, 5}
	assert("cdemo454\000", cdemo454(a) == a)
	assert("demo454\000", demo454(a) == a)
}

type point455 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo455 C.demo455
func cdemo455(a point455) point455

func demo455(a point455) point455 {
	return a
}

func init() {
	a := point455{1, 2, 3, 4, 5}
	assert("cdemo455\000", cdemo455(a) == a)
	assert("demo455\000", demo455(a) == a)
}

type point456 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo456 C.demo456
func cdemo456(a point456) point456

func demo456(a point456) point456 {
	return a
}

func init() {
	a := point456{1, 2, 3, 4, 5}
	assert("cdemo456\000", cdemo456(a) == a)
	assert("demo456\000", demo456(a) == a)
}

type point457 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo457 C.demo457
func cdemo457(a point457) point457

func demo457(a point457) point457 {
	return a
}

func init() {
	a := point457{1, 2, 3, 4, 5}
	assert("cdemo457\000", cdemo457(a) == a)
	assert("demo457\000", demo457(a) == a)
}

type point458 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 float32
}

//go:linkname cdemo458 C.demo458
func cdemo458(a point458) point458

func demo458(a point458) point458 {
	return a
}

func init() {
	a := point458{1, 2, 3, 4}
	assert("cdemo458\000", cdemo458(a) == a)
	assert("demo458\000", demo458(a) == a)
}

type point459 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo459 C.demo459
func cdemo459(a point459) point459

func demo459(a point459) point459 {
	return a
}

func init() {
	a := point459{1, 2, 3, 4, 5}
	assert("cdemo459\000", cdemo459(a) == a)
	assert("demo459\000", demo459(a) == a)
}

type point460 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo460 C.demo460
func cdemo460(a point460) point460

func demo460(a point460) point460 {
	return a
}

func init() {
	a := point460{1, 2, 3, 4, 5}
	assert("cdemo460\000", cdemo460(a) == a)
	assert("demo460\000", demo460(a) == a)
}

type point461 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo461 C.demo461
func cdemo461(a point461) point461

func demo461(a point461) point461 {
	return a
}

func init() {
	a := point461{1, 2, 3, 4, 5}
	assert("cdemo461\000", cdemo461(a) == a)
	assert("demo461\000", demo461(a) == a)
}

type point462 struct {
	x0 int16
	x1 int32
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo462 C.demo462
func cdemo462(a point462) point462

func demo462(a point462) point462 {
	return a
}

func init() {
	a := point462{1, 2, 3, 4, 5}
	assert("cdemo462\000", cdemo462(a) == a)
	assert("demo462\000", demo462(a) == a)
}

type point463 struct {
	x0 int16
	x1 float32
	x2 int8
}

//go:linkname cdemo463 C.demo463
func cdemo463(a point463) point463

func demo463(a point463) point463 {
	return a
}

func init() {
	a := point463{1, 2, 3}
	assert("cdemo463\000", cdemo463(a) == a)
	assert("demo463\000", demo463(a) == a)
}

type point464 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int8
}

//go:linkname cdemo464 C.demo464
func cdemo464(a point464) point464

func demo464(a point464) point464 {
	return a
}

func init() {
	a := point464{1, 2, 3, 4}
	assert("cdemo464\000", cdemo464(a) == a)
	assert("demo464\000", demo464(a) == a)
}

type point465 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo465 C.demo465
func cdemo465(a point465) point465

func demo465(a point465) point465 {
	return a
}

func init() {
	a := point465{1, 2, 3, 4, 5}
	assert("cdemo465\000", cdemo465(a) == a)
	assert("demo465\000", demo465(a) == a)
}

type point466 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo466 C.demo466
func cdemo466(a point466) point466

func demo466(a point466) point466 {
	return a
}

func init() {
	a := point466{1, 2, 3, 4, 5}
	assert("cdemo466\000", cdemo466(a) == a)
	assert("demo466\000", demo466(a) == a)
}

type point467 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo467 C.demo467
func cdemo467(a point467) point467

func demo467(a point467) point467 {
	return a
}

func init() {
	a := point467{1, 2, 3, 4, 5}
	assert("cdemo467\000", cdemo467(a) == a)
	assert("demo467\000", demo467(a) == a)
}

type point468 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo468 C.demo468
func cdemo468(a point468) point468

func demo468(a point468) point468 {
	return a
}

func init() {
	a := point468{1, 2, 3, 4, 5}
	assert("cdemo468\000", cdemo468(a) == a)
	assert("demo468\000", demo468(a) == a)
}

type point469 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int16
}

//go:linkname cdemo469 C.demo469
func cdemo469(a point469) point469

func demo469(a point469) point469 {
	return a
}

func init() {
	a := point469{1, 2, 3, 4}
	assert("cdemo469\000", cdemo469(a) == a)
	assert("demo469\000", demo469(a) == a)
}

type point470 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo470 C.demo470
func cdemo470(a point470) point470

func demo470(a point470) point470 {
	return a
}

func init() {
	a := point470{1, 2, 3, 4, 5}
	assert("cdemo470\000", cdemo470(a) == a)
	assert("demo470\000", demo470(a) == a)
}

type point471 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo471 C.demo471
func cdemo471(a point471) point471

func demo471(a point471) point471 {
	return a
}

func init() {
	a := point471{1, 2, 3, 4, 5}
	assert("cdemo471\000", cdemo471(a) == a)
	assert("demo471\000", demo471(a) == a)
}

type point472 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo472 C.demo472
func cdemo472(a point472) point472

func demo472(a point472) point472 {
	return a
}

func init() {
	a := point472{1, 2, 3, 4, 5}
	assert("cdemo472\000", cdemo472(a) == a)
	assert("demo472\000", demo472(a) == a)
}

type point473 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo473 C.demo473
func cdemo473(a point473) point473

func demo473(a point473) point473 {
	return a
}

func init() {
	a := point473{1, 2, 3, 4, 5}
	assert("cdemo473\000", cdemo473(a) == a)
	assert("demo473\000", demo473(a) == a)
}

type point474 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int32
}

//go:linkname cdemo474 C.demo474
func cdemo474(a point474) point474

func demo474(a point474) point474 {
	return a
}

func init() {
	a := point474{1, 2, 3, 4}
	assert("cdemo474\000", cdemo474(a) == a)
	assert("demo474\000", demo474(a) == a)
}

type point475 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo475 C.demo475
func cdemo475(a point475) point475

func demo475(a point475) point475 {
	return a
}

func init() {
	a := point475{1, 2, 3, 4, 5}
	assert("cdemo475\000", cdemo475(a) == a)
	assert("demo475\000", demo475(a) == a)
}

type point476 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo476 C.demo476
func cdemo476(a point476) point476

func demo476(a point476) point476 {
	return a
}

func init() {
	a := point476{1, 2, 3, 4, 5}
	assert("cdemo476\000", cdemo476(a) == a)
	assert("demo476\000", demo476(a) == a)
}

type point477 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo477 C.demo477
func cdemo477(a point477) point477

func demo477(a point477) point477 {
	return a
}

func init() {
	a := point477{1, 2, 3, 4, 5}
	assert("cdemo477\000", cdemo477(a) == a)
	assert("demo477\000", demo477(a) == a)
}

type point478 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo478 C.demo478
func cdemo478(a point478) point478

func demo478(a point478) point478 {
	return a
}

func init() {
	a := point478{1, 2, 3, 4, 5}
	assert("cdemo478\000", cdemo478(a) == a)
	assert("demo478\000", demo478(a) == a)
}

type point479 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 float32
}

//go:linkname cdemo479 C.demo479
func cdemo479(a point479) point479

func demo479(a point479) point479 {
	return a
}

func init() {
	a := point479{1, 2, 3, 4}
	assert("cdemo479\000", cdemo479(a) == a)
	assert("demo479\000", demo479(a) == a)
}

type point480 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo480 C.demo480
func cdemo480(a point480) point480

func demo480(a point480) point480 {
	return a
}

func init() {
	a := point480{1, 2, 3, 4, 5}
	assert("cdemo480\000", cdemo480(a) == a)
	assert("demo480\000", demo480(a) == a)
}

type point481 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo481 C.demo481
func cdemo481(a point481) point481

func demo481(a point481) point481 {
	return a
}

func init() {
	a := point481{1, 2, 3, 4, 5}
	assert("cdemo481\000", cdemo481(a) == a)
	assert("demo481\000", demo481(a) == a)
}

type point482 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo482 C.demo482
func cdemo482(a point482) point482

func demo482(a point482) point482 {
	return a
}

func init() {
	a := point482{1, 2, 3, 4, 5}
	assert("cdemo482\000", cdemo482(a) == a)
	assert("demo482\000", demo482(a) == a)
}

type point483 struct {
	x0 int16
	x1 float32
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo483 C.demo483
func cdemo483(a point483) point483

func demo483(a point483) point483 {
	return a
}

func init() {
	a := point483{1, 2, 3, 4, 5}
	assert("cdemo483\000", cdemo483(a) == a)
	assert("demo483\000", demo483(a) == a)
}

type point484 struct {
	x0 int16
	x1 float32
	x2 int16
}

//go:linkname cdemo484 C.demo484
func cdemo484(a point484) point484

func demo484(a point484) point484 {
	return a
}

func init() {
	a := point484{1, 2, 3}
	assert("cdemo484\000", cdemo484(a) == a)
	assert("demo484\000", demo484(a) == a)
}

type point485 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int8
}

//go:linkname cdemo485 C.demo485
func cdemo485(a point485) point485

func demo485(a point485) point485 {
	return a
}

func init() {
	a := point485{1, 2, 3, 4}
	assert("cdemo485\000", cdemo485(a) == a)
	assert("demo485\000", demo485(a) == a)
}

type point486 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo486 C.demo486
func cdemo486(a point486) point486

func demo486(a point486) point486 {
	return a
}

func init() {
	a := point486{1, 2, 3, 4, 5}
	assert("cdemo486\000", cdemo486(a) == a)
	assert("demo486\000", demo486(a) == a)
}

type point487 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo487 C.demo487
func cdemo487(a point487) point487

func demo487(a point487) point487 {
	return a
}

func init() {
	a := point487{1, 2, 3, 4, 5}
	assert("cdemo487\000", cdemo487(a) == a)
	assert("demo487\000", demo487(a) == a)
}

type point488 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo488 C.demo488
func cdemo488(a point488) point488

func demo488(a point488) point488 {
	return a
}

func init() {
	a := point488{1, 2, 3, 4, 5}
	assert("cdemo488\000", cdemo488(a) == a)
	assert("demo488\000", demo488(a) == a)
}

type point489 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo489 C.demo489
func cdemo489(a point489) point489

func demo489(a point489) point489 {
	return a
}

func init() {
	a := point489{1, 2, 3, 4, 5}
	assert("cdemo489\000", cdemo489(a) == a)
	assert("demo489\000", demo489(a) == a)
}

type point490 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int16
}

//go:linkname cdemo490 C.demo490
func cdemo490(a point490) point490

func demo490(a point490) point490 {
	return a
}

func init() {
	a := point490{1, 2, 3, 4}
	assert("cdemo490\000", cdemo490(a) == a)
	assert("demo490\000", demo490(a) == a)
}

type point491 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo491 C.demo491
func cdemo491(a point491) point491

func demo491(a point491) point491 {
	return a
}

func init() {
	a := point491{1, 2, 3, 4, 5}
	assert("cdemo491\000", cdemo491(a) == a)
	assert("demo491\000", demo491(a) == a)
}

type point492 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo492 C.demo492
func cdemo492(a point492) point492

func demo492(a point492) point492 {
	return a
}

func init() {
	a := point492{1, 2, 3, 4, 5}
	assert("cdemo492\000", cdemo492(a) == a)
	assert("demo492\000", demo492(a) == a)
}

type point493 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo493 C.demo493
func cdemo493(a point493) point493

func demo493(a point493) point493 {
	return a
}

func init() {
	a := point493{1, 2, 3, 4, 5}
	assert("cdemo493\000", cdemo493(a) == a)
	assert("demo493\000", demo493(a) == a)
}

type point494 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo494 C.demo494
func cdemo494(a point494) point494

func demo494(a point494) point494 {
	return a
}

func init() {
	a := point494{1, 2, 3, 4, 5}
	assert("cdemo494\000", cdemo494(a) == a)
	assert("demo494\000", demo494(a) == a)
}

type point495 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int32
}

//go:linkname cdemo495 C.demo495
func cdemo495(a point495) point495

func demo495(a point495) point495 {
	return a
}

func init() {
	a := point495{1, 2, 3, 4}
	assert("cdemo495\000", cdemo495(a) == a)
	assert("demo495\000", demo495(a) == a)
}

type point496 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo496 C.demo496
func cdemo496(a point496) point496

func demo496(a point496) point496 {
	return a
}

func init() {
	a := point496{1, 2, 3, 4, 5}
	assert("cdemo496\000", cdemo496(a) == a)
	assert("demo496\000", demo496(a) == a)
}

type point497 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo497 C.demo497
func cdemo497(a point497) point497

func demo497(a point497) point497 {
	return a
}

func init() {
	a := point497{1, 2, 3, 4, 5}
	assert("cdemo497\000", cdemo497(a) == a)
	assert("demo497\000", demo497(a) == a)
}

type point498 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo498 C.demo498
func cdemo498(a point498) point498

func demo498(a point498) point498 {
	return a
}

func init() {
	a := point498{1, 2, 3, 4, 5}
	assert("cdemo498\000", cdemo498(a) == a)
	assert("demo498\000", demo498(a) == a)
}

type point499 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo499 C.demo499
func cdemo499(a point499) point499

func demo499(a point499) point499 {
	return a
}

func init() {
	a := point499{1, 2, 3, 4, 5}
	assert("cdemo499\000", cdemo499(a) == a)
	assert("demo499\000", demo499(a) == a)
}

type point500 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 float32
}

//go:linkname cdemo500 C.demo500
func cdemo500(a point500) point500

func demo500(a point500) point500 {
	return a
}

func init() {
	a := point500{1, 2, 3, 4}
	assert("cdemo500\000", cdemo500(a) == a)
	assert("demo500\000", demo500(a) == a)
}

type point501 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo501 C.demo501
func cdemo501(a point501) point501

func demo501(a point501) point501 {
	return a
}

func init() {
	a := point501{1, 2, 3, 4, 5}
	assert("cdemo501\000", cdemo501(a) == a)
	assert("demo501\000", demo501(a) == a)
}

type point502 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo502 C.demo502
func cdemo502(a point502) point502

func demo502(a point502) point502 {
	return a
}

func init() {
	a := point502{1, 2, 3, 4, 5}
	assert("cdemo502\000", cdemo502(a) == a)
	assert("demo502\000", demo502(a) == a)
}

type point503 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo503 C.demo503
func cdemo503(a point503) point503

func demo503(a point503) point503 {
	return a
}

func init() {
	a := point503{1, 2, 3, 4, 5}
	assert("cdemo503\000", cdemo503(a) == a)
	assert("demo503\000", demo503(a) == a)
}

type point504 struct {
	x0 int16
	x1 float32
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo504 C.demo504
func cdemo504(a point504) point504

func demo504(a point504) point504 {
	return a
}

func init() {
	a := point504{1, 2, 3, 4, 5}
	assert("cdemo504\000", cdemo504(a) == a)
	assert("demo504\000", demo504(a) == a)
}

type point505 struct {
	x0 int16
	x1 float32
	x2 int32
}

//go:linkname cdemo505 C.demo505
func cdemo505(a point505) point505

func demo505(a point505) point505 {
	return a
}

func init() {
	a := point505{1, 2, 3}
	assert("cdemo505\000", cdemo505(a) == a)
	assert("demo505\000", demo505(a) == a)
}

type point506 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int8
}

//go:linkname cdemo506 C.demo506
func cdemo506(a point506) point506

func demo506(a point506) point506 {
	return a
}

func init() {
	a := point506{1, 2, 3, 4}
	assert("cdemo506\000", cdemo506(a) == a)
	assert("demo506\000", demo506(a) == a)
}

type point507 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo507 C.demo507
func cdemo507(a point507) point507

func demo507(a point507) point507 {
	return a
}

func init() {
	a := point507{1, 2, 3, 4, 5}
	assert("cdemo507\000", cdemo507(a) == a)
	assert("demo507\000", demo507(a) == a)
}

type point508 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo508 C.demo508
func cdemo508(a point508) point508

func demo508(a point508) point508 {
	return a
}

func init() {
	a := point508{1, 2, 3, 4, 5}
	assert("cdemo508\000", cdemo508(a) == a)
	assert("demo508\000", demo508(a) == a)
}

type point509 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo509 C.demo509
func cdemo509(a point509) point509

func demo509(a point509) point509 {
	return a
}

func init() {
	a := point509{1, 2, 3, 4, 5}
	assert("cdemo509\000", cdemo509(a) == a)
	assert("demo509\000", demo509(a) == a)
}

type point510 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo510 C.demo510
func cdemo510(a point510) point510

func demo510(a point510) point510 {
	return a
}

func init() {
	a := point510{1, 2, 3, 4, 5}
	assert("cdemo510\000", cdemo510(a) == a)
	assert("demo510\000", demo510(a) == a)
}

type point511 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int16
}

//go:linkname cdemo511 C.demo511
func cdemo511(a point511) point511

func demo511(a point511) point511 {
	return a
}

func init() {
	a := point511{1, 2, 3, 4}
	assert("cdemo511\000", cdemo511(a) == a)
	assert("demo511\000", demo511(a) == a)
}

type point512 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo512 C.demo512
func cdemo512(a point512) point512

func demo512(a point512) point512 {
	return a
}

func init() {
	a := point512{1, 2, 3, 4, 5}
	assert("cdemo512\000", cdemo512(a) == a)
	assert("demo512\000", demo512(a) == a)
}

type point513 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo513 C.demo513
func cdemo513(a point513) point513

func demo513(a point513) point513 {
	return a
}

func init() {
	a := point513{1, 2, 3, 4, 5}
	assert("cdemo513\000", cdemo513(a) == a)
	assert("demo513\000", demo513(a) == a)
}

type point514 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo514 C.demo514
func cdemo514(a point514) point514

func demo514(a point514) point514 {
	return a
}

func init() {
	a := point514{1, 2, 3, 4, 5}
	assert("cdemo514\000", cdemo514(a) == a)
	assert("demo514\000", demo514(a) == a)
}

type point515 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo515 C.demo515
func cdemo515(a point515) point515

func demo515(a point515) point515 {
	return a
}

func init() {
	a := point515{1, 2, 3, 4, 5}
	assert("cdemo515\000", cdemo515(a) == a)
	assert("demo515\000", demo515(a) == a)
}

type point516 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int32
}

//go:linkname cdemo516 C.demo516
func cdemo516(a point516) point516

func demo516(a point516) point516 {
	return a
}

func init() {
	a := point516{1, 2, 3, 4}
	assert("cdemo516\000", cdemo516(a) == a)
	assert("demo516\000", demo516(a) == a)
}

type point517 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo517 C.demo517
func cdemo517(a point517) point517

func demo517(a point517) point517 {
	return a
}

func init() {
	a := point517{1, 2, 3, 4, 5}
	assert("cdemo517\000", cdemo517(a) == a)
	assert("demo517\000", demo517(a) == a)
}

type point518 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo518 C.demo518
func cdemo518(a point518) point518

func demo518(a point518) point518 {
	return a
}

func init() {
	a := point518{1, 2, 3, 4, 5}
	assert("cdemo518\000", cdemo518(a) == a)
	assert("demo518\000", demo518(a) == a)
}

type point519 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo519 C.demo519
func cdemo519(a point519) point519

func demo519(a point519) point519 {
	return a
}

func init() {
	a := point519{1, 2, 3, 4, 5}
	assert("cdemo519\000", cdemo519(a) == a)
	assert("demo519\000", demo519(a) == a)
}

type point520 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo520 C.demo520
func cdemo520(a point520) point520

func demo520(a point520) point520 {
	return a
}

func init() {
	a := point520{1, 2, 3, 4, 5}
	assert("cdemo520\000", cdemo520(a) == a)
	assert("demo520\000", demo520(a) == a)
}

type point521 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 float32
}

//go:linkname cdemo521 C.demo521
func cdemo521(a point521) point521

func demo521(a point521) point521 {
	return a
}

func init() {
	a := point521{1, 2, 3, 4}
	assert("cdemo521\000", cdemo521(a) == a)
	assert("demo521\000", demo521(a) == a)
}

type point522 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo522 C.demo522
func cdemo522(a point522) point522

func demo522(a point522) point522 {
	return a
}

func init() {
	a := point522{1, 2, 3, 4, 5}
	assert("cdemo522\000", cdemo522(a) == a)
	assert("demo522\000", demo522(a) == a)
}

type point523 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo523 C.demo523
func cdemo523(a point523) point523

func demo523(a point523) point523 {
	return a
}

func init() {
	a := point523{1, 2, 3, 4, 5}
	assert("cdemo523\000", cdemo523(a) == a)
	assert("demo523\000", demo523(a) == a)
}

type point524 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo524 C.demo524
func cdemo524(a point524) point524

func demo524(a point524) point524 {
	return a
}

func init() {
	a := point524{1, 2, 3, 4, 5}
	assert("cdemo524\000", cdemo524(a) == a)
	assert("demo524\000", demo524(a) == a)
}

type point525 struct {
	x0 int16
	x1 float32
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo525 C.demo525
func cdemo525(a point525) point525

func demo525(a point525) point525 {
	return a
}

func init() {
	a := point525{1, 2, 3, 4, 5}
	assert("cdemo525\000", cdemo525(a) == a)
	assert("demo525\000", demo525(a) == a)
}

type point526 struct {
	x0 int16
	x1 float32
	x2 float32
}

//go:linkname cdemo526 C.demo526
func cdemo526(a point526) point526

func demo526(a point526) point526 {
	return a
}

func init() {
	a := point526{1, 2, 3}
	assert("cdemo526\000", cdemo526(a) == a)
	assert("demo526\000", demo526(a) == a)
}

type point527 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int8
}

//go:linkname cdemo527 C.demo527
func cdemo527(a point527) point527

func demo527(a point527) point527 {
	return a
}

func init() {
	a := point527{1, 2, 3, 4}
	assert("cdemo527\000", cdemo527(a) == a)
	assert("demo527\000", demo527(a) == a)
}

type point528 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo528 C.demo528
func cdemo528(a point528) point528

func demo528(a point528) point528 {
	return a
}

func init() {
	a := point528{1, 2, 3, 4, 5}
	assert("cdemo528\000", cdemo528(a) == a)
	assert("demo528\000", demo528(a) == a)
}

type point529 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo529 C.demo529
func cdemo529(a point529) point529

func demo529(a point529) point529 {
	return a
}

func init() {
	a := point529{1, 2, 3, 4, 5}
	assert("cdemo529\000", cdemo529(a) == a)
	assert("demo529\000", demo529(a) == a)
}

type point530 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo530 C.demo530
func cdemo530(a point530) point530

func demo530(a point530) point530 {
	return a
}

func init() {
	a := point530{1, 2, 3, 4, 5}
	assert("cdemo530\000", cdemo530(a) == a)
	assert("demo530\000", demo530(a) == a)
}

type point531 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo531 C.demo531
func cdemo531(a point531) point531

func demo531(a point531) point531 {
	return a
}

func init() {
	a := point531{1, 2, 3, 4, 5}
	assert("cdemo531\000", cdemo531(a) == a)
	assert("demo531\000", demo531(a) == a)
}

type point532 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int16
}

//go:linkname cdemo532 C.demo532
func cdemo532(a point532) point532

func demo532(a point532) point532 {
	return a
}

func init() {
	a := point532{1, 2, 3, 4}
	assert("cdemo532\000", cdemo532(a) == a)
	assert("demo532\000", demo532(a) == a)
}

type point533 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo533 C.demo533
func cdemo533(a point533) point533

func demo533(a point533) point533 {
	return a
}

func init() {
	a := point533{1, 2, 3, 4, 5}
	assert("cdemo533\000", cdemo533(a) == a)
	assert("demo533\000", demo533(a) == a)
}

type point534 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo534 C.demo534
func cdemo534(a point534) point534

func demo534(a point534) point534 {
	return a
}

func init() {
	a := point534{1, 2, 3, 4, 5}
	assert("cdemo534\000", cdemo534(a) == a)
	assert("demo534\000", demo534(a) == a)
}

type point535 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo535 C.demo535
func cdemo535(a point535) point535

func demo535(a point535) point535 {
	return a
}

func init() {
	a := point535{1, 2, 3, 4, 5}
	assert("cdemo535\000", cdemo535(a) == a)
	assert("demo535\000", demo535(a) == a)
}

type point536 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo536 C.demo536
func cdemo536(a point536) point536

func demo536(a point536) point536 {
	return a
}

func init() {
	a := point536{1, 2, 3, 4, 5}
	assert("cdemo536\000", cdemo536(a) == a)
	assert("demo536\000", demo536(a) == a)
}

type point537 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int32
}

//go:linkname cdemo537 C.demo537
func cdemo537(a point537) point537

func demo537(a point537) point537 {
	return a
}

func init() {
	a := point537{1, 2, 3, 4}
	assert("cdemo537\000", cdemo537(a) == a)
	assert("demo537\000", demo537(a) == a)
}

type point538 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo538 C.demo538
func cdemo538(a point538) point538

func demo538(a point538) point538 {
	return a
}

func init() {
	a := point538{1, 2, 3, 4, 5}
	assert("cdemo538\000", cdemo538(a) == a)
	assert("demo538\000", demo538(a) == a)
}

type point539 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo539 C.demo539
func cdemo539(a point539) point539

func demo539(a point539) point539 {
	return a
}

func init() {
	a := point539{1, 2, 3, 4, 5}
	assert("cdemo539\000", cdemo539(a) == a)
	assert("demo539\000", demo539(a) == a)
}

type point540 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo540 C.demo540
func cdemo540(a point540) point540

func demo540(a point540) point540 {
	return a
}

func init() {
	a := point540{1, 2, 3, 4, 5}
	assert("cdemo540\000", cdemo540(a) == a)
	assert("demo540\000", demo540(a) == a)
}

type point541 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo541 C.demo541
func cdemo541(a point541) point541

func demo541(a point541) point541 {
	return a
}

func init() {
	a := point541{1, 2, 3, 4, 5}
	assert("cdemo541\000", cdemo541(a) == a)
	assert("demo541\000", demo541(a) == a)
}

type point542 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 float32
}

//go:linkname cdemo542 C.demo542
func cdemo542(a point542) point542

func demo542(a point542) point542 {
	return a
}

func init() {
	a := point542{1, 2, 3, 4}
	assert("cdemo542\000", cdemo542(a) == a)
	assert("demo542\000", demo542(a) == a)
}

type point543 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo543 C.demo543
func cdemo543(a point543) point543

func demo543(a point543) point543 {
	return a
}

func init() {
	a := point543{1, 2, 3, 4, 5}
	assert("cdemo543\000", cdemo543(a) == a)
	assert("demo543\000", demo543(a) == a)
}

type point544 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo544 C.demo544
func cdemo544(a point544) point544

func demo544(a point544) point544 {
	return a
}

func init() {
	a := point544{1, 2, 3, 4, 5}
	assert("cdemo544\000", cdemo544(a) == a)
	assert("demo544\000", demo544(a) == a)
}

type point545 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo545 C.demo545
func cdemo545(a point545) point545

func demo545(a point545) point545 {
	return a
}

func init() {
	a := point545{1, 2, 3, 4, 5}
	assert("cdemo545\000", cdemo545(a) == a)
	assert("demo545\000", demo545(a) == a)
}

type point546 struct {
	x0 int16
	x1 float32
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo546 C.demo546
func cdemo546(a point546) point546

func demo546(a point546) point546 {
	return a
}

func init() {
	a := point546{1, 2, 3, 4, 5}
	assert("cdemo546\000", cdemo546(a) == a)
	assert("demo546\000", demo546(a) == a)
}

type point547 struct {
	x0 int32
	x1 int8
	x2 int8
}

//go:linkname cdemo547 C.demo547
func cdemo547(a point547) point547

func demo547(a point547) point547 {
	return a
}

func init() {
	a := point547{1, 2, 3}
	assert("cdemo547\000", cdemo547(a) == a)
	assert("demo547\000", demo547(a) == a)
}

type point548 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int8
}

//go:linkname cdemo548 C.demo548
func cdemo548(a point548) point548

func demo548(a point548) point548 {
	return a
}

func init() {
	a := point548{1, 2, 3, 4}
	assert("cdemo548\000", cdemo548(a) == a)
	assert("demo548\000", demo548(a) == a)
}

type point549 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo549 C.demo549
func cdemo549(a point549) point549

func demo549(a point549) point549 {
	return a
}

func init() {
	a := point549{1, 2, 3, 4, 5}
	assert("cdemo549\000", cdemo549(a) == a)
	assert("demo549\000", demo549(a) == a)
}

type point550 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo550 C.demo550
func cdemo550(a point550) point550

func demo550(a point550) point550 {
	return a
}

func init() {
	a := point550{1, 2, 3, 4, 5}
	assert("cdemo550\000", cdemo550(a) == a)
	assert("demo550\000", demo550(a) == a)
}

type point551 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo551 C.demo551
func cdemo551(a point551) point551

func demo551(a point551) point551 {
	return a
}

func init() {
	a := point551{1, 2, 3, 4, 5}
	assert("cdemo551\000", cdemo551(a) == a)
	assert("demo551\000", demo551(a) == a)
}

type point552 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo552 C.demo552
func cdemo552(a point552) point552

func demo552(a point552) point552 {
	return a
}

func init() {
	a := point552{1, 2, 3, 4, 5}
	assert("cdemo552\000", cdemo552(a) == a)
	assert("demo552\000", demo552(a) == a)
}

type point553 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int16
}

//go:linkname cdemo553 C.demo553
func cdemo553(a point553) point553

func demo553(a point553) point553 {
	return a
}

func init() {
	a := point553{1, 2, 3, 4}
	assert("cdemo553\000", cdemo553(a) == a)
	assert("demo553\000", demo553(a) == a)
}

type point554 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo554 C.demo554
func cdemo554(a point554) point554

func demo554(a point554) point554 {
	return a
}

func init() {
	a := point554{1, 2, 3, 4, 5}
	assert("cdemo554\000", cdemo554(a) == a)
	assert("demo554\000", demo554(a) == a)
}

type point555 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo555 C.demo555
func cdemo555(a point555) point555

func demo555(a point555) point555 {
	return a
}

func init() {
	a := point555{1, 2, 3, 4, 5}
	assert("cdemo555\000", cdemo555(a) == a)
	assert("demo555\000", demo555(a) == a)
}

type point556 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo556 C.demo556
func cdemo556(a point556) point556

func demo556(a point556) point556 {
	return a
}

func init() {
	a := point556{1, 2, 3, 4, 5}
	assert("cdemo556\000", cdemo556(a) == a)
	assert("demo556\000", demo556(a) == a)
}

type point557 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo557 C.demo557
func cdemo557(a point557) point557

func demo557(a point557) point557 {
	return a
}

func init() {
	a := point557{1, 2, 3, 4, 5}
	assert("cdemo557\000", cdemo557(a) == a)
	assert("demo557\000", demo557(a) == a)
}

type point558 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int32
}

//go:linkname cdemo558 C.demo558
func cdemo558(a point558) point558

func demo558(a point558) point558 {
	return a
}

func init() {
	a := point558{1, 2, 3, 4}
	assert("cdemo558\000", cdemo558(a) == a)
	assert("demo558\000", demo558(a) == a)
}

type point559 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo559 C.demo559
func cdemo559(a point559) point559

func demo559(a point559) point559 {
	return a
}

func init() {
	a := point559{1, 2, 3, 4, 5}
	assert("cdemo559\000", cdemo559(a) == a)
	assert("demo559\000", demo559(a) == a)
}

type point560 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo560 C.demo560
func cdemo560(a point560) point560

func demo560(a point560) point560 {
	return a
}

func init() {
	a := point560{1, 2, 3, 4, 5}
	assert("cdemo560\000", cdemo560(a) == a)
	assert("demo560\000", demo560(a) == a)
}

type point561 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo561 C.demo561
func cdemo561(a point561) point561

func demo561(a point561) point561 {
	return a
}

func init() {
	a := point561{1, 2, 3, 4, 5}
	assert("cdemo561\000", cdemo561(a) == a)
	assert("demo561\000", demo561(a) == a)
}

type point562 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo562 C.demo562
func cdemo562(a point562) point562

func demo562(a point562) point562 {
	return a
}

func init() {
	a := point562{1, 2, 3, 4, 5}
	assert("cdemo562\000", cdemo562(a) == a)
	assert("demo562\000", demo562(a) == a)
}

type point563 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 float32
}

//go:linkname cdemo563 C.demo563
func cdemo563(a point563) point563

func demo563(a point563) point563 {
	return a
}

func init() {
	a := point563{1, 2, 3, 4}
	assert("cdemo563\000", cdemo563(a) == a)
	assert("demo563\000", demo563(a) == a)
}

type point564 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo564 C.demo564
func cdemo564(a point564) point564

func demo564(a point564) point564 {
	return a
}

func init() {
	a := point564{1, 2, 3, 4, 5}
	assert("cdemo564\000", cdemo564(a) == a)
	assert("demo564\000", demo564(a) == a)
}

type point565 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo565 C.demo565
func cdemo565(a point565) point565

func demo565(a point565) point565 {
	return a
}

func init() {
	a := point565{1, 2, 3, 4, 5}
	assert("cdemo565\000", cdemo565(a) == a)
	assert("demo565\000", demo565(a) == a)
}

type point566 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo566 C.demo566
func cdemo566(a point566) point566

func demo566(a point566) point566 {
	return a
}

func init() {
	a := point566{1, 2, 3, 4, 5}
	assert("cdemo566\000", cdemo566(a) == a)
	assert("demo566\000", demo566(a) == a)
}

type point567 struct {
	x0 int32
	x1 int8
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo567 C.demo567
func cdemo567(a point567) point567

func demo567(a point567) point567 {
	return a
}

func init() {
	a := point567{1, 2, 3, 4, 5}
	assert("cdemo567\000", cdemo567(a) == a)
	assert("demo567\000", demo567(a) == a)
}

type point568 struct {
	x0 int32
	x1 int8
	x2 int16
}

//go:linkname cdemo568 C.demo568
func cdemo568(a point568) point568

func demo568(a point568) point568 {
	return a
}

func init() {
	a := point568{1, 2, 3}
	assert("cdemo568\000", cdemo568(a) == a)
	assert("demo568\000", demo568(a) == a)
}

type point569 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int8
}

//go:linkname cdemo569 C.demo569
func cdemo569(a point569) point569

func demo569(a point569) point569 {
	return a
}

func init() {
	a := point569{1, 2, 3, 4}
	assert("cdemo569\000", cdemo569(a) == a)
	assert("demo569\000", demo569(a) == a)
}

type point570 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo570 C.demo570
func cdemo570(a point570) point570

func demo570(a point570) point570 {
	return a
}

func init() {
	a := point570{1, 2, 3, 4, 5}
	assert("cdemo570\000", cdemo570(a) == a)
	assert("demo570\000", demo570(a) == a)
}

type point571 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo571 C.demo571
func cdemo571(a point571) point571

func demo571(a point571) point571 {
	return a
}

func init() {
	a := point571{1, 2, 3, 4, 5}
	assert("cdemo571\000", cdemo571(a) == a)
	assert("demo571\000", demo571(a) == a)
}

type point572 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo572 C.demo572
func cdemo572(a point572) point572

func demo572(a point572) point572 {
	return a
}

func init() {
	a := point572{1, 2, 3, 4, 5}
	assert("cdemo572\000", cdemo572(a) == a)
	assert("demo572\000", demo572(a) == a)
}

type point573 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo573 C.demo573
func cdemo573(a point573) point573

func demo573(a point573) point573 {
	return a
}

func init() {
	a := point573{1, 2, 3, 4, 5}
	assert("cdemo573\000", cdemo573(a) == a)
	assert("demo573\000", demo573(a) == a)
}

type point574 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int16
}

//go:linkname cdemo574 C.demo574
func cdemo574(a point574) point574

func demo574(a point574) point574 {
	return a
}

func init() {
	a := point574{1, 2, 3, 4}
	assert("cdemo574\000", cdemo574(a) == a)
	assert("demo574\000", demo574(a) == a)
}

type point575 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo575 C.demo575
func cdemo575(a point575) point575

func demo575(a point575) point575 {
	return a
}

func init() {
	a := point575{1, 2, 3, 4, 5}
	assert("cdemo575\000", cdemo575(a) == a)
	assert("demo575\000", demo575(a) == a)
}

type point576 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo576 C.demo576
func cdemo576(a point576) point576

func demo576(a point576) point576 {
	return a
}

func init() {
	a := point576{1, 2, 3, 4, 5}
	assert("cdemo576\000", cdemo576(a) == a)
	assert("demo576\000", demo576(a) == a)
}

type point577 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo577 C.demo577
func cdemo577(a point577) point577

func demo577(a point577) point577 {
	return a
}

func init() {
	a := point577{1, 2, 3, 4, 5}
	assert("cdemo577\000", cdemo577(a) == a)
	assert("demo577\000", demo577(a) == a)
}

type point578 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo578 C.demo578
func cdemo578(a point578) point578

func demo578(a point578) point578 {
	return a
}

func init() {
	a := point578{1, 2, 3, 4, 5}
	assert("cdemo578\000", cdemo578(a) == a)
	assert("demo578\000", demo578(a) == a)
}

type point579 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int32
}

//go:linkname cdemo579 C.demo579
func cdemo579(a point579) point579

func demo579(a point579) point579 {
	return a
}

func init() {
	a := point579{1, 2, 3, 4}
	assert("cdemo579\000", cdemo579(a) == a)
	assert("demo579\000", demo579(a) == a)
}

type point580 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo580 C.demo580
func cdemo580(a point580) point580

func demo580(a point580) point580 {
	return a
}

func init() {
	a := point580{1, 2, 3, 4, 5}
	assert("cdemo580\000", cdemo580(a) == a)
	assert("demo580\000", demo580(a) == a)
}

type point581 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo581 C.demo581
func cdemo581(a point581) point581

func demo581(a point581) point581 {
	return a
}

func init() {
	a := point581{1, 2, 3, 4, 5}
	assert("cdemo581\000", cdemo581(a) == a)
	assert("demo581\000", demo581(a) == a)
}

type point582 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo582 C.demo582
func cdemo582(a point582) point582

func demo582(a point582) point582 {
	return a
}

func init() {
	a := point582{1, 2, 3, 4, 5}
	assert("cdemo582\000", cdemo582(a) == a)
	assert("demo582\000", demo582(a) == a)
}

type point583 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo583 C.demo583
func cdemo583(a point583) point583

func demo583(a point583) point583 {
	return a
}

func init() {
	a := point583{1, 2, 3, 4, 5}
	assert("cdemo583\000", cdemo583(a) == a)
	assert("demo583\000", demo583(a) == a)
}

type point584 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 float32
}

//go:linkname cdemo584 C.demo584
func cdemo584(a point584) point584

func demo584(a point584) point584 {
	return a
}

func init() {
	a := point584{1, 2, 3, 4}
	assert("cdemo584\000", cdemo584(a) == a)
	assert("demo584\000", demo584(a) == a)
}

type point585 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo585 C.demo585
func cdemo585(a point585) point585

func demo585(a point585) point585 {
	return a
}

func init() {
	a := point585{1, 2, 3, 4, 5}
	assert("cdemo585\000", cdemo585(a) == a)
	assert("demo585\000", demo585(a) == a)
}

type point586 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo586 C.demo586
func cdemo586(a point586) point586

func demo586(a point586) point586 {
	return a
}

func init() {
	a := point586{1, 2, 3, 4, 5}
	assert("cdemo586\000", cdemo586(a) == a)
	assert("demo586\000", demo586(a) == a)
}

type point587 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo587 C.demo587
func cdemo587(a point587) point587

func demo587(a point587) point587 {
	return a
}

func init() {
	a := point587{1, 2, 3, 4, 5}
	assert("cdemo587\000", cdemo587(a) == a)
	assert("demo587\000", demo587(a) == a)
}

type point588 struct {
	x0 int32
	x1 int8
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo588 C.demo588
func cdemo588(a point588) point588

func demo588(a point588) point588 {
	return a
}

func init() {
	a := point588{1, 2, 3, 4, 5}
	assert("cdemo588\000", cdemo588(a) == a)
	assert("demo588\000", demo588(a) == a)
}

type point589 struct {
	x0 int32
	x1 int8
	x2 int32
}

//go:linkname cdemo589 C.demo589
func cdemo589(a point589) point589

func demo589(a point589) point589 {
	return a
}

func init() {
	a := point589{1, 2, 3}
	assert("cdemo589\000", cdemo589(a) == a)
	assert("demo589\000", demo589(a) == a)
}

type point590 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int8
}

//go:linkname cdemo590 C.demo590
func cdemo590(a point590) point590

func demo590(a point590) point590 {
	return a
}

func init() {
	a := point590{1, 2, 3, 4}
	assert("cdemo590\000", cdemo590(a) == a)
	assert("demo590\000", demo590(a) == a)
}

type point591 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo591 C.demo591
func cdemo591(a point591) point591

func demo591(a point591) point591 {
	return a
}

func init() {
	a := point591{1, 2, 3, 4, 5}
	assert("cdemo591\000", cdemo591(a) == a)
	assert("demo591\000", demo591(a) == a)
}

type point592 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo592 C.demo592
func cdemo592(a point592) point592

func demo592(a point592) point592 {
	return a
}

func init() {
	a := point592{1, 2, 3, 4, 5}
	assert("cdemo592\000", cdemo592(a) == a)
	assert("demo592\000", demo592(a) == a)
}

type point593 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo593 C.demo593
func cdemo593(a point593) point593

func demo593(a point593) point593 {
	return a
}

func init() {
	a := point593{1, 2, 3, 4, 5}
	assert("cdemo593\000", cdemo593(a) == a)
	assert("demo593\000", demo593(a) == a)
}

type point594 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo594 C.demo594
func cdemo594(a point594) point594

func demo594(a point594) point594 {
	return a
}

func init() {
	a := point594{1, 2, 3, 4, 5}
	assert("cdemo594\000", cdemo594(a) == a)
	assert("demo594\000", demo594(a) == a)
}

type point595 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int16
}

//go:linkname cdemo595 C.demo595
func cdemo595(a point595) point595

func demo595(a point595) point595 {
	return a
}

func init() {
	a := point595{1, 2, 3, 4}
	assert("cdemo595\000", cdemo595(a) == a)
	assert("demo595\000", demo595(a) == a)
}

type point596 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo596 C.demo596
func cdemo596(a point596) point596

func demo596(a point596) point596 {
	return a
}

func init() {
	a := point596{1, 2, 3, 4, 5}
	assert("cdemo596\000", cdemo596(a) == a)
	assert("demo596\000", demo596(a) == a)
}

type point597 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo597 C.demo597
func cdemo597(a point597) point597

func demo597(a point597) point597 {
	return a
}

func init() {
	a := point597{1, 2, 3, 4, 5}
	assert("cdemo597\000", cdemo597(a) == a)
	assert("demo597\000", demo597(a) == a)
}

type point598 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo598 C.demo598
func cdemo598(a point598) point598

func demo598(a point598) point598 {
	return a
}

func init() {
	a := point598{1, 2, 3, 4, 5}
	assert("cdemo598\000", cdemo598(a) == a)
	assert("demo598\000", demo598(a) == a)
}

type point599 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo599 C.demo599
func cdemo599(a point599) point599

func demo599(a point599) point599 {
	return a
}

func init() {
	a := point599{1, 2, 3, 4, 5}
	assert("cdemo599\000", cdemo599(a) == a)
	assert("demo599\000", demo599(a) == a)
}

type point600 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int32
}

//go:linkname cdemo600 C.demo600
func cdemo600(a point600) point600

func demo600(a point600) point600 {
	return a
}

func init() {
	a := point600{1, 2, 3, 4}
	assert("cdemo600\000", cdemo600(a) == a)
	assert("demo600\000", demo600(a) == a)
}

type point601 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo601 C.demo601
func cdemo601(a point601) point601

func demo601(a point601) point601 {
	return a
}

func init() {
	a := point601{1, 2, 3, 4, 5}
	assert("cdemo601\000", cdemo601(a) == a)
	assert("demo601\000", demo601(a) == a)
}

type point602 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo602 C.demo602
func cdemo602(a point602) point602

func demo602(a point602) point602 {
	return a
}

func init() {
	a := point602{1, 2, 3, 4, 5}
	assert("cdemo602\000", cdemo602(a) == a)
	assert("demo602\000", demo602(a) == a)
}

type point603 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo603 C.demo603
func cdemo603(a point603) point603

func demo603(a point603) point603 {
	return a
}

func init() {
	a := point603{1, 2, 3, 4, 5}
	assert("cdemo603\000", cdemo603(a) == a)
	assert("demo603\000", demo603(a) == a)
}

type point604 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo604 C.demo604
func cdemo604(a point604) point604

func demo604(a point604) point604 {
	return a
}

func init() {
	a := point604{1, 2, 3, 4, 5}
	assert("cdemo604\000", cdemo604(a) == a)
	assert("demo604\000", demo604(a) == a)
}

type point605 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 float32
}

//go:linkname cdemo605 C.demo605
func cdemo605(a point605) point605

func demo605(a point605) point605 {
	return a
}

func init() {
	a := point605{1, 2, 3, 4}
	assert("cdemo605\000", cdemo605(a) == a)
	assert("demo605\000", demo605(a) == a)
}

type point606 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo606 C.demo606
func cdemo606(a point606) point606

func demo606(a point606) point606 {
	return a
}

func init() {
	a := point606{1, 2, 3, 4, 5}
	assert("cdemo606\000", cdemo606(a) == a)
	assert("demo606\000", demo606(a) == a)
}

type point607 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo607 C.demo607
func cdemo607(a point607) point607

func demo607(a point607) point607 {
	return a
}

func init() {
	a := point607{1, 2, 3, 4, 5}
	assert("cdemo607\000", cdemo607(a) == a)
	assert("demo607\000", demo607(a) == a)
}

type point608 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo608 C.demo608
func cdemo608(a point608) point608

func demo608(a point608) point608 {
	return a
}

func init() {
	a := point608{1, 2, 3, 4, 5}
	assert("cdemo608\000", cdemo608(a) == a)
	assert("demo608\000", demo608(a) == a)
}

type point609 struct {
	x0 int32
	x1 int8
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo609 C.demo609
func cdemo609(a point609) point609

func demo609(a point609) point609 {
	return a
}

func init() {
	a := point609{1, 2, 3, 4, 5}
	assert("cdemo609\000", cdemo609(a) == a)
	assert("demo609\000", demo609(a) == a)
}

type point610 struct {
	x0 int32
	x1 int8
	x2 float32
}

//go:linkname cdemo610 C.demo610
func cdemo610(a point610) point610

func demo610(a point610) point610 {
	return a
}

func init() {
	a := point610{1, 2, 3}
	assert("cdemo610\000", cdemo610(a) == a)
	assert("demo610\000", demo610(a) == a)
}

type point611 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int8
}

//go:linkname cdemo611 C.demo611
func cdemo611(a point611) point611

func demo611(a point611) point611 {
	return a
}

func init() {
	a := point611{1, 2, 3, 4}
	assert("cdemo611\000", cdemo611(a) == a)
	assert("demo611\000", demo611(a) == a)
}

type point612 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo612 C.demo612
func cdemo612(a point612) point612

func demo612(a point612) point612 {
	return a
}

func init() {
	a := point612{1, 2, 3, 4, 5}
	assert("cdemo612\000", cdemo612(a) == a)
	assert("demo612\000", demo612(a) == a)
}

type point613 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo613 C.demo613
func cdemo613(a point613) point613

func demo613(a point613) point613 {
	return a
}

func init() {
	a := point613{1, 2, 3, 4, 5}
	assert("cdemo613\000", cdemo613(a) == a)
	assert("demo613\000", demo613(a) == a)
}

type point614 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo614 C.demo614
func cdemo614(a point614) point614

func demo614(a point614) point614 {
	return a
}

func init() {
	a := point614{1, 2, 3, 4, 5}
	assert("cdemo614\000", cdemo614(a) == a)
	assert("demo614\000", demo614(a) == a)
}

type point615 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo615 C.demo615
func cdemo615(a point615) point615

func demo615(a point615) point615 {
	return a
}

func init() {
	a := point615{1, 2, 3, 4, 5}
	assert("cdemo615\000", cdemo615(a) == a)
	assert("demo615\000", demo615(a) == a)
}

type point616 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int16
}

//go:linkname cdemo616 C.demo616
func cdemo616(a point616) point616

func demo616(a point616) point616 {
	return a
}

func init() {
	a := point616{1, 2, 3, 4}
	assert("cdemo616\000", cdemo616(a) == a)
	assert("demo616\000", demo616(a) == a)
}

type point617 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo617 C.demo617
func cdemo617(a point617) point617

func demo617(a point617) point617 {
	return a
}

func init() {
	a := point617{1, 2, 3, 4, 5}
	assert("cdemo617\000", cdemo617(a) == a)
	assert("demo617\000", demo617(a) == a)
}

type point618 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo618 C.demo618
func cdemo618(a point618) point618

func demo618(a point618) point618 {
	return a
}

func init() {
	a := point618{1, 2, 3, 4, 5}
	assert("cdemo618\000", cdemo618(a) == a)
	assert("demo618\000", demo618(a) == a)
}

type point619 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo619 C.demo619
func cdemo619(a point619) point619

func demo619(a point619) point619 {
	return a
}

func init() {
	a := point619{1, 2, 3, 4, 5}
	assert("cdemo619\000", cdemo619(a) == a)
	assert("demo619\000", demo619(a) == a)
}

type point620 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo620 C.demo620
func cdemo620(a point620) point620

func demo620(a point620) point620 {
	return a
}

func init() {
	a := point620{1, 2, 3, 4, 5}
	assert("cdemo620\000", cdemo620(a) == a)
	assert("demo620\000", demo620(a) == a)
}

type point621 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int32
}

//go:linkname cdemo621 C.demo621
func cdemo621(a point621) point621

func demo621(a point621) point621 {
	return a
}

func init() {
	a := point621{1, 2, 3, 4}
	assert("cdemo621\000", cdemo621(a) == a)
	assert("demo621\000", demo621(a) == a)
}

type point622 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo622 C.demo622
func cdemo622(a point622) point622

func demo622(a point622) point622 {
	return a
}

func init() {
	a := point622{1, 2, 3, 4, 5}
	assert("cdemo622\000", cdemo622(a) == a)
	assert("demo622\000", demo622(a) == a)
}

type point623 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo623 C.demo623
func cdemo623(a point623) point623

func demo623(a point623) point623 {
	return a
}

func init() {
	a := point623{1, 2, 3, 4, 5}
	assert("cdemo623\000", cdemo623(a) == a)
	assert("demo623\000", demo623(a) == a)
}

type point624 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo624 C.demo624
func cdemo624(a point624) point624

func demo624(a point624) point624 {
	return a
}

func init() {
	a := point624{1, 2, 3, 4, 5}
	assert("cdemo624\000", cdemo624(a) == a)
	assert("demo624\000", demo624(a) == a)
}

type point625 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo625 C.demo625
func cdemo625(a point625) point625

func demo625(a point625) point625 {
	return a
}

func init() {
	a := point625{1, 2, 3, 4, 5}
	assert("cdemo625\000", cdemo625(a) == a)
	assert("demo625\000", demo625(a) == a)
}

type point626 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 float32
}

//go:linkname cdemo626 C.demo626
func cdemo626(a point626) point626

func demo626(a point626) point626 {
	return a
}

func init() {
	a := point626{1, 2, 3, 4}
	assert("cdemo626\000", cdemo626(a) == a)
	assert("demo626\000", demo626(a) == a)
}

type point627 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo627 C.demo627
func cdemo627(a point627) point627

func demo627(a point627) point627 {
	return a
}

func init() {
	a := point627{1, 2, 3, 4, 5}
	assert("cdemo627\000", cdemo627(a) == a)
	assert("demo627\000", demo627(a) == a)
}

type point628 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo628 C.demo628
func cdemo628(a point628) point628

func demo628(a point628) point628 {
	return a
}

func init() {
	a := point628{1, 2, 3, 4, 5}
	assert("cdemo628\000", cdemo628(a) == a)
	assert("demo628\000", demo628(a) == a)
}

type point629 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo629 C.demo629
func cdemo629(a point629) point629

func demo629(a point629) point629 {
	return a
}

func init() {
	a := point629{1, 2, 3, 4, 5}
	assert("cdemo629\000", cdemo629(a) == a)
	assert("demo629\000", demo629(a) == a)
}

type point630 struct {
	x0 int32
	x1 int8
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo630 C.demo630
func cdemo630(a point630) point630

func demo630(a point630) point630 {
	return a
}

func init() {
	a := point630{1, 2, 3, 4, 5}
	assert("cdemo630\000", cdemo630(a) == a)
	assert("demo630\000", demo630(a) == a)
}

type point631 struct {
	x0 int32
	x1 int16
	x2 int8
}

//go:linkname cdemo631 C.demo631
func cdemo631(a point631) point631

func demo631(a point631) point631 {
	return a
}

func init() {
	a := point631{1, 2, 3}
	assert("cdemo631\000", cdemo631(a) == a)
	assert("demo631\000", demo631(a) == a)
}

type point632 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int8
}

//go:linkname cdemo632 C.demo632
func cdemo632(a point632) point632

func demo632(a point632) point632 {
	return a
}

func init() {
	a := point632{1, 2, 3, 4}
	assert("cdemo632\000", cdemo632(a) == a)
	assert("demo632\000", demo632(a) == a)
}

type point633 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo633 C.demo633
func cdemo633(a point633) point633

func demo633(a point633) point633 {
	return a
}

func init() {
	a := point633{1, 2, 3, 4, 5}
	assert("cdemo633\000", cdemo633(a) == a)
	assert("demo633\000", demo633(a) == a)
}

type point634 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo634 C.demo634
func cdemo634(a point634) point634

func demo634(a point634) point634 {
	return a
}

func init() {
	a := point634{1, 2, 3, 4, 5}
	assert("cdemo634\000", cdemo634(a) == a)
	assert("demo634\000", demo634(a) == a)
}

type point635 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo635 C.demo635
func cdemo635(a point635) point635

func demo635(a point635) point635 {
	return a
}

func init() {
	a := point635{1, 2, 3, 4, 5}
	assert("cdemo635\000", cdemo635(a) == a)
	assert("demo635\000", demo635(a) == a)
}

type point636 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo636 C.demo636
func cdemo636(a point636) point636

func demo636(a point636) point636 {
	return a
}

func init() {
	a := point636{1, 2, 3, 4, 5}
	assert("cdemo636\000", cdemo636(a) == a)
	assert("demo636\000", demo636(a) == a)
}

type point637 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int16
}

//go:linkname cdemo637 C.demo637
func cdemo637(a point637) point637

func demo637(a point637) point637 {
	return a
}

func init() {
	a := point637{1, 2, 3, 4}
	assert("cdemo637\000", cdemo637(a) == a)
	assert("demo637\000", demo637(a) == a)
}

type point638 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo638 C.demo638
func cdemo638(a point638) point638

func demo638(a point638) point638 {
	return a
}

func init() {
	a := point638{1, 2, 3, 4, 5}
	assert("cdemo638\000", cdemo638(a) == a)
	assert("demo638\000", demo638(a) == a)
}

type point639 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo639 C.demo639
func cdemo639(a point639) point639

func demo639(a point639) point639 {
	return a
}

func init() {
	a := point639{1, 2, 3, 4, 5}
	assert("cdemo639\000", cdemo639(a) == a)
	assert("demo639\000", demo639(a) == a)
}

type point640 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo640 C.demo640
func cdemo640(a point640) point640

func demo640(a point640) point640 {
	return a
}

func init() {
	a := point640{1, 2, 3, 4, 5}
	assert("cdemo640\000", cdemo640(a) == a)
	assert("demo640\000", demo640(a) == a)
}

type point641 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo641 C.demo641
func cdemo641(a point641) point641

func demo641(a point641) point641 {
	return a
}

func init() {
	a := point641{1, 2, 3, 4, 5}
	assert("cdemo641\000", cdemo641(a) == a)
	assert("demo641\000", demo641(a) == a)
}

type point642 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int32
}

//go:linkname cdemo642 C.demo642
func cdemo642(a point642) point642

func demo642(a point642) point642 {
	return a
}

func init() {
	a := point642{1, 2, 3, 4}
	assert("cdemo642\000", cdemo642(a) == a)
	assert("demo642\000", demo642(a) == a)
}

type point643 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo643 C.demo643
func cdemo643(a point643) point643

func demo643(a point643) point643 {
	return a
}

func init() {
	a := point643{1, 2, 3, 4, 5}
	assert("cdemo643\000", cdemo643(a) == a)
	assert("demo643\000", demo643(a) == a)
}

type point644 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo644 C.demo644
func cdemo644(a point644) point644

func demo644(a point644) point644 {
	return a
}

func init() {
	a := point644{1, 2, 3, 4, 5}
	assert("cdemo644\000", cdemo644(a) == a)
	assert("demo644\000", demo644(a) == a)
}

type point645 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo645 C.demo645
func cdemo645(a point645) point645

func demo645(a point645) point645 {
	return a
}

func init() {
	a := point645{1, 2, 3, 4, 5}
	assert("cdemo645\000", cdemo645(a) == a)
	assert("demo645\000", demo645(a) == a)
}

type point646 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo646 C.demo646
func cdemo646(a point646) point646

func demo646(a point646) point646 {
	return a
}

func init() {
	a := point646{1, 2, 3, 4, 5}
	assert("cdemo646\000", cdemo646(a) == a)
	assert("demo646\000", demo646(a) == a)
}

type point647 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 float32
}

//go:linkname cdemo647 C.demo647
func cdemo647(a point647) point647

func demo647(a point647) point647 {
	return a
}

func init() {
	a := point647{1, 2, 3, 4}
	assert("cdemo647\000", cdemo647(a) == a)
	assert("demo647\000", demo647(a) == a)
}

type point648 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo648 C.demo648
func cdemo648(a point648) point648

func demo648(a point648) point648 {
	return a
}

func init() {
	a := point648{1, 2, 3, 4, 5}
	assert("cdemo648\000", cdemo648(a) == a)
	assert("demo648\000", demo648(a) == a)
}

type point649 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo649 C.demo649
func cdemo649(a point649) point649

func demo649(a point649) point649 {
	return a
}

func init() {
	a := point649{1, 2, 3, 4, 5}
	assert("cdemo649\000", cdemo649(a) == a)
	assert("demo649\000", demo649(a) == a)
}

type point650 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo650 C.demo650
func cdemo650(a point650) point650

func demo650(a point650) point650 {
	return a
}

func init() {
	a := point650{1, 2, 3, 4, 5}
	assert("cdemo650\000", cdemo650(a) == a)
	assert("demo650\000", demo650(a) == a)
}

type point651 struct {
	x0 int32
	x1 int16
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo651 C.demo651
func cdemo651(a point651) point651

func demo651(a point651) point651 {
	return a
}

func init() {
	a := point651{1, 2, 3, 4, 5}
	assert("cdemo651\000", cdemo651(a) == a)
	assert("demo651\000", demo651(a) == a)
}

type point652 struct {
	x0 int32
	x1 int16
	x2 int16
}

//go:linkname cdemo652 C.demo652
func cdemo652(a point652) point652

func demo652(a point652) point652 {
	return a
}

func init() {
	a := point652{1, 2, 3}
	assert("cdemo652\000", cdemo652(a) == a)
	assert("demo652\000", demo652(a) == a)
}

type point653 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int8
}

//go:linkname cdemo653 C.demo653
func cdemo653(a point653) point653

func demo653(a point653) point653 {
	return a
}

func init() {
	a := point653{1, 2, 3, 4}
	assert("cdemo653\000", cdemo653(a) == a)
	assert("demo653\000", demo653(a) == a)
}

type point654 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo654 C.demo654
func cdemo654(a point654) point654

func demo654(a point654) point654 {
	return a
}

func init() {
	a := point654{1, 2, 3, 4, 5}
	assert("cdemo654\000", cdemo654(a) == a)
	assert("demo654\000", demo654(a) == a)
}

type point655 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo655 C.demo655
func cdemo655(a point655) point655

func demo655(a point655) point655 {
	return a
}

func init() {
	a := point655{1, 2, 3, 4, 5}
	assert("cdemo655\000", cdemo655(a) == a)
	assert("demo655\000", demo655(a) == a)
}

type point656 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo656 C.demo656
func cdemo656(a point656) point656

func demo656(a point656) point656 {
	return a
}

func init() {
	a := point656{1, 2, 3, 4, 5}
	assert("cdemo656\000", cdemo656(a) == a)
	assert("demo656\000", demo656(a) == a)
}

type point657 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo657 C.demo657
func cdemo657(a point657) point657

func demo657(a point657) point657 {
	return a
}

func init() {
	a := point657{1, 2, 3, 4, 5}
	assert("cdemo657\000", cdemo657(a) == a)
	assert("demo657\000", demo657(a) == a)
}

type point658 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int16
}

//go:linkname cdemo658 C.demo658
func cdemo658(a point658) point658

func demo658(a point658) point658 {
	return a
}

func init() {
	a := point658{1, 2, 3, 4}
	assert("cdemo658\000", cdemo658(a) == a)
	assert("demo658\000", demo658(a) == a)
}

type point659 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo659 C.demo659
func cdemo659(a point659) point659

func demo659(a point659) point659 {
	return a
}

func init() {
	a := point659{1, 2, 3, 4, 5}
	assert("cdemo659\000", cdemo659(a) == a)
	assert("demo659\000", demo659(a) == a)
}

type point660 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo660 C.demo660
func cdemo660(a point660) point660

func demo660(a point660) point660 {
	return a
}

func init() {
	a := point660{1, 2, 3, 4, 5}
	assert("cdemo660\000", cdemo660(a) == a)
	assert("demo660\000", demo660(a) == a)
}

type point661 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo661 C.demo661
func cdemo661(a point661) point661

func demo661(a point661) point661 {
	return a
}

func init() {
	a := point661{1, 2, 3, 4, 5}
	assert("cdemo661\000", cdemo661(a) == a)
	assert("demo661\000", demo661(a) == a)
}

type point662 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo662 C.demo662
func cdemo662(a point662) point662

func demo662(a point662) point662 {
	return a
}

func init() {
	a := point662{1, 2, 3, 4, 5}
	assert("cdemo662\000", cdemo662(a) == a)
	assert("demo662\000", demo662(a) == a)
}

type point663 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int32
}

//go:linkname cdemo663 C.demo663
func cdemo663(a point663) point663

func demo663(a point663) point663 {
	return a
}

func init() {
	a := point663{1, 2, 3, 4}
	assert("cdemo663\000", cdemo663(a) == a)
	assert("demo663\000", demo663(a) == a)
}

type point664 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo664 C.demo664
func cdemo664(a point664) point664

func demo664(a point664) point664 {
	return a
}

func init() {
	a := point664{1, 2, 3, 4, 5}
	assert("cdemo664\000", cdemo664(a) == a)
	assert("demo664\000", demo664(a) == a)
}

type point665 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo665 C.demo665
func cdemo665(a point665) point665

func demo665(a point665) point665 {
	return a
}

func init() {
	a := point665{1, 2, 3, 4, 5}
	assert("cdemo665\000", cdemo665(a) == a)
	assert("demo665\000", demo665(a) == a)
}

type point666 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo666 C.demo666
func cdemo666(a point666) point666

func demo666(a point666) point666 {
	return a
}

func init() {
	a := point666{1, 2, 3, 4, 5}
	assert("cdemo666\000", cdemo666(a) == a)
	assert("demo666\000", demo666(a) == a)
}

type point667 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo667 C.demo667
func cdemo667(a point667) point667

func demo667(a point667) point667 {
	return a
}

func init() {
	a := point667{1, 2, 3, 4, 5}
	assert("cdemo667\000", cdemo667(a) == a)
	assert("demo667\000", demo667(a) == a)
}

type point668 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 float32
}

//go:linkname cdemo668 C.demo668
func cdemo668(a point668) point668

func demo668(a point668) point668 {
	return a
}

func init() {
	a := point668{1, 2, 3, 4}
	assert("cdemo668\000", cdemo668(a) == a)
	assert("demo668\000", demo668(a) == a)
}

type point669 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo669 C.demo669
func cdemo669(a point669) point669

func demo669(a point669) point669 {
	return a
}

func init() {
	a := point669{1, 2, 3, 4, 5}
	assert("cdemo669\000", cdemo669(a) == a)
	assert("demo669\000", demo669(a) == a)
}

type point670 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo670 C.demo670
func cdemo670(a point670) point670

func demo670(a point670) point670 {
	return a
}

func init() {
	a := point670{1, 2, 3, 4, 5}
	assert("cdemo670\000", cdemo670(a) == a)
	assert("demo670\000", demo670(a) == a)
}

type point671 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo671 C.demo671
func cdemo671(a point671) point671

func demo671(a point671) point671 {
	return a
}

func init() {
	a := point671{1, 2, 3, 4, 5}
	assert("cdemo671\000", cdemo671(a) == a)
	assert("demo671\000", demo671(a) == a)
}

type point672 struct {
	x0 int32
	x1 int16
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo672 C.demo672
func cdemo672(a point672) point672

func demo672(a point672) point672 {
	return a
}

func init() {
	a := point672{1, 2, 3, 4, 5}
	assert("cdemo672\000", cdemo672(a) == a)
	assert("demo672\000", demo672(a) == a)
}

type point673 struct {
	x0 int32
	x1 int16
	x2 int32
}

//go:linkname cdemo673 C.demo673
func cdemo673(a point673) point673

func demo673(a point673) point673 {
	return a
}

func init() {
	a := point673{1, 2, 3}
	assert("cdemo673\000", cdemo673(a) == a)
	assert("demo673\000", demo673(a) == a)
}

type point674 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int8
}

//go:linkname cdemo674 C.demo674
func cdemo674(a point674) point674

func demo674(a point674) point674 {
	return a
}

func init() {
	a := point674{1, 2, 3, 4}
	assert("cdemo674\000", cdemo674(a) == a)
	assert("demo674\000", demo674(a) == a)
}

type point675 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo675 C.demo675
func cdemo675(a point675) point675

func demo675(a point675) point675 {
	return a
}

func init() {
	a := point675{1, 2, 3, 4, 5}
	assert("cdemo675\000", cdemo675(a) == a)
	assert("demo675\000", demo675(a) == a)
}

type point676 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo676 C.demo676
func cdemo676(a point676) point676

func demo676(a point676) point676 {
	return a
}

func init() {
	a := point676{1, 2, 3, 4, 5}
	assert("cdemo676\000", cdemo676(a) == a)
	assert("demo676\000", demo676(a) == a)
}

type point677 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo677 C.demo677
func cdemo677(a point677) point677

func demo677(a point677) point677 {
	return a
}

func init() {
	a := point677{1, 2, 3, 4, 5}
	assert("cdemo677\000", cdemo677(a) == a)
	assert("demo677\000", demo677(a) == a)
}

type point678 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo678 C.demo678
func cdemo678(a point678) point678

func demo678(a point678) point678 {
	return a
}

func init() {
	a := point678{1, 2, 3, 4, 5}
	assert("cdemo678\000", cdemo678(a) == a)
	assert("demo678\000", demo678(a) == a)
}

type point679 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int16
}

//go:linkname cdemo679 C.demo679
func cdemo679(a point679) point679

func demo679(a point679) point679 {
	return a
}

func init() {
	a := point679{1, 2, 3, 4}
	assert("cdemo679\000", cdemo679(a) == a)
	assert("demo679\000", demo679(a) == a)
}

type point680 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo680 C.demo680
func cdemo680(a point680) point680

func demo680(a point680) point680 {
	return a
}

func init() {
	a := point680{1, 2, 3, 4, 5}
	assert("cdemo680\000", cdemo680(a) == a)
	assert("demo680\000", demo680(a) == a)
}

type point681 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo681 C.demo681
func cdemo681(a point681) point681

func demo681(a point681) point681 {
	return a
}

func init() {
	a := point681{1, 2, 3, 4, 5}
	assert("cdemo681\000", cdemo681(a) == a)
	assert("demo681\000", demo681(a) == a)
}

type point682 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo682 C.demo682
func cdemo682(a point682) point682

func demo682(a point682) point682 {
	return a
}

func init() {
	a := point682{1, 2, 3, 4, 5}
	assert("cdemo682\000", cdemo682(a) == a)
	assert("demo682\000", demo682(a) == a)
}

type point683 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo683 C.demo683
func cdemo683(a point683) point683

func demo683(a point683) point683 {
	return a
}

func init() {
	a := point683{1, 2, 3, 4, 5}
	assert("cdemo683\000", cdemo683(a) == a)
	assert("demo683\000", demo683(a) == a)
}

type point684 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int32
}

//go:linkname cdemo684 C.demo684
func cdemo684(a point684) point684

func demo684(a point684) point684 {
	return a
}

func init() {
	a := point684{1, 2, 3, 4}
	assert("cdemo684\000", cdemo684(a) == a)
	assert("demo684\000", demo684(a) == a)
}

type point685 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo685 C.demo685
func cdemo685(a point685) point685

func demo685(a point685) point685 {
	return a
}

func init() {
	a := point685{1, 2, 3, 4, 5}
	assert("cdemo685\000", cdemo685(a) == a)
	assert("demo685\000", demo685(a) == a)
}

type point686 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo686 C.demo686
func cdemo686(a point686) point686

func demo686(a point686) point686 {
	return a
}

func init() {
	a := point686{1, 2, 3, 4, 5}
	assert("cdemo686\000", cdemo686(a) == a)
	assert("demo686\000", demo686(a) == a)
}

type point687 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo687 C.demo687
func cdemo687(a point687) point687

func demo687(a point687) point687 {
	return a
}

func init() {
	a := point687{1, 2, 3, 4, 5}
	assert("cdemo687\000", cdemo687(a) == a)
	assert("demo687\000", demo687(a) == a)
}

type point688 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo688 C.demo688
func cdemo688(a point688) point688

func demo688(a point688) point688 {
	return a
}

func init() {
	a := point688{1, 2, 3, 4, 5}
	assert("cdemo688\000", cdemo688(a) == a)
	assert("demo688\000", demo688(a) == a)
}

type point689 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 float32
}

//go:linkname cdemo689 C.demo689
func cdemo689(a point689) point689

func demo689(a point689) point689 {
	return a
}

func init() {
	a := point689{1, 2, 3, 4}
	assert("cdemo689\000", cdemo689(a) == a)
	assert("demo689\000", demo689(a) == a)
}

type point690 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo690 C.demo690
func cdemo690(a point690) point690

func demo690(a point690) point690 {
	return a
}

func init() {
	a := point690{1, 2, 3, 4, 5}
	assert("cdemo690\000", cdemo690(a) == a)
	assert("demo690\000", demo690(a) == a)
}

type point691 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo691 C.demo691
func cdemo691(a point691) point691

func demo691(a point691) point691 {
	return a
}

func init() {
	a := point691{1, 2, 3, 4, 5}
	assert("cdemo691\000", cdemo691(a) == a)
	assert("demo691\000", demo691(a) == a)
}

type point692 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo692 C.demo692
func cdemo692(a point692) point692

func demo692(a point692) point692 {
	return a
}

func init() {
	a := point692{1, 2, 3, 4, 5}
	assert("cdemo692\000", cdemo692(a) == a)
	assert("demo692\000", demo692(a) == a)
}

type point693 struct {
	x0 int32
	x1 int16
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo693 C.demo693
func cdemo693(a point693) point693

func demo693(a point693) point693 {
	return a
}

func init() {
	a := point693{1, 2, 3, 4, 5}
	assert("cdemo693\000", cdemo693(a) == a)
	assert("demo693\000", demo693(a) == a)
}

type point694 struct {
	x0 int32
	x1 int16
	x2 float32
}

//go:linkname cdemo694 C.demo694
func cdemo694(a point694) point694

func demo694(a point694) point694 {
	return a
}

func init() {
	a := point694{1, 2, 3}
	assert("cdemo694\000", cdemo694(a) == a)
	assert("demo694\000", demo694(a) == a)
}

type point695 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int8
}

//go:linkname cdemo695 C.demo695
func cdemo695(a point695) point695

func demo695(a point695) point695 {
	return a
}

func init() {
	a := point695{1, 2, 3, 4}
	assert("cdemo695\000", cdemo695(a) == a)
	assert("demo695\000", demo695(a) == a)
}

type point696 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo696 C.demo696
func cdemo696(a point696) point696

func demo696(a point696) point696 {
	return a
}

func init() {
	a := point696{1, 2, 3, 4, 5}
	assert("cdemo696\000", cdemo696(a) == a)
	assert("demo696\000", demo696(a) == a)
}

type point697 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo697 C.demo697
func cdemo697(a point697) point697

func demo697(a point697) point697 {
	return a
}

func init() {
	a := point697{1, 2, 3, 4, 5}
	assert("cdemo697\000", cdemo697(a) == a)
	assert("demo697\000", demo697(a) == a)
}

type point698 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo698 C.demo698
func cdemo698(a point698) point698

func demo698(a point698) point698 {
	return a
}

func init() {
	a := point698{1, 2, 3, 4, 5}
	assert("cdemo698\000", cdemo698(a) == a)
	assert("demo698\000", demo698(a) == a)
}

type point699 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo699 C.demo699
func cdemo699(a point699) point699

func demo699(a point699) point699 {
	return a
}

func init() {
	a := point699{1, 2, 3, 4, 5}
	assert("cdemo699\000", cdemo699(a) == a)
	assert("demo699\000", demo699(a) == a)
}

type point700 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int16
}

//go:linkname cdemo700 C.demo700
func cdemo700(a point700) point700

func demo700(a point700) point700 {
	return a
}

func init() {
	a := point700{1, 2, 3, 4}
	assert("cdemo700\000", cdemo700(a) == a)
	assert("demo700\000", demo700(a) == a)
}

type point701 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo701 C.demo701
func cdemo701(a point701) point701

func demo701(a point701) point701 {
	return a
}

func init() {
	a := point701{1, 2, 3, 4, 5}
	assert("cdemo701\000", cdemo701(a) == a)
	assert("demo701\000", demo701(a) == a)
}

type point702 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo702 C.demo702
func cdemo702(a point702) point702

func demo702(a point702) point702 {
	return a
}

func init() {
	a := point702{1, 2, 3, 4, 5}
	assert("cdemo702\000", cdemo702(a) == a)
	assert("demo702\000", demo702(a) == a)
}

type point703 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo703 C.demo703
func cdemo703(a point703) point703

func demo703(a point703) point703 {
	return a
}

func init() {
	a := point703{1, 2, 3, 4, 5}
	assert("cdemo703\000", cdemo703(a) == a)
	assert("demo703\000", demo703(a) == a)
}

type point704 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo704 C.demo704
func cdemo704(a point704) point704

func demo704(a point704) point704 {
	return a
}

func init() {
	a := point704{1, 2, 3, 4, 5}
	assert("cdemo704\000", cdemo704(a) == a)
	assert("demo704\000", demo704(a) == a)
}

type point705 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int32
}

//go:linkname cdemo705 C.demo705
func cdemo705(a point705) point705

func demo705(a point705) point705 {
	return a
}

func init() {
	a := point705{1, 2, 3, 4}
	assert("cdemo705\000", cdemo705(a) == a)
	assert("demo705\000", demo705(a) == a)
}

type point706 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo706 C.demo706
func cdemo706(a point706) point706

func demo706(a point706) point706 {
	return a
}

func init() {
	a := point706{1, 2, 3, 4, 5}
	assert("cdemo706\000", cdemo706(a) == a)
	assert("demo706\000", demo706(a) == a)
}

type point707 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo707 C.demo707
func cdemo707(a point707) point707

func demo707(a point707) point707 {
	return a
}

func init() {
	a := point707{1, 2, 3, 4, 5}
	assert("cdemo707\000", cdemo707(a) == a)
	assert("demo707\000", demo707(a) == a)
}

type point708 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo708 C.demo708
func cdemo708(a point708) point708

func demo708(a point708) point708 {
	return a
}

func init() {
	a := point708{1, 2, 3, 4, 5}
	assert("cdemo708\000", cdemo708(a) == a)
	assert("demo708\000", demo708(a) == a)
}

type point709 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo709 C.demo709
func cdemo709(a point709) point709

func demo709(a point709) point709 {
	return a
}

func init() {
	a := point709{1, 2, 3, 4, 5}
	assert("cdemo709\000", cdemo709(a) == a)
	assert("demo709\000", demo709(a) == a)
}

type point710 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 float32
}

//go:linkname cdemo710 C.demo710
func cdemo710(a point710) point710

func demo710(a point710) point710 {
	return a
}

func init() {
	a := point710{1, 2, 3, 4}
	assert("cdemo710\000", cdemo710(a) == a)
	assert("demo710\000", demo710(a) == a)
}

type point711 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo711 C.demo711
func cdemo711(a point711) point711

func demo711(a point711) point711 {
	return a
}

func init() {
	a := point711{1, 2, 3, 4, 5}
	assert("cdemo711\000", cdemo711(a) == a)
	assert("demo711\000", demo711(a) == a)
}

type point712 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo712 C.demo712
func cdemo712(a point712) point712

func demo712(a point712) point712 {
	return a
}

func init() {
	a := point712{1, 2, 3, 4, 5}
	assert("cdemo712\000", cdemo712(a) == a)
	assert("demo712\000", demo712(a) == a)
}

type point713 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo713 C.demo713
func cdemo713(a point713) point713

func demo713(a point713) point713 {
	return a
}

func init() {
	a := point713{1, 2, 3, 4, 5}
	assert("cdemo713\000", cdemo713(a) == a)
	assert("demo713\000", demo713(a) == a)
}

type point714 struct {
	x0 int32
	x1 int16
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo714 C.demo714
func cdemo714(a point714) point714

func demo714(a point714) point714 {
	return a
}

func init() {
	a := point714{1, 2, 3, 4, 5}
	assert("cdemo714\000", cdemo714(a) == a)
	assert("demo714\000", demo714(a) == a)
}

type point715 struct {
	x0 int32
	x1 float32
	x2 int8
}

//go:linkname cdemo715 C.demo715
func cdemo715(a point715) point715

func demo715(a point715) point715 {
	return a
}

func init() {
	a := point715{1, 2, 3}
	assert("cdemo715\000", cdemo715(a) == a)
	assert("demo715\000", demo715(a) == a)
}

type point716 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int8
}

//go:linkname cdemo716 C.demo716
func cdemo716(a point716) point716

func demo716(a point716) point716 {
	return a
}

func init() {
	a := point716{1, 2, 3, 4}
	assert("cdemo716\000", cdemo716(a) == a)
	assert("demo716\000", demo716(a) == a)
}

type point717 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo717 C.demo717
func cdemo717(a point717) point717

func demo717(a point717) point717 {
	return a
}

func init() {
	a := point717{1, 2, 3, 4, 5}
	assert("cdemo717\000", cdemo717(a) == a)
	assert("demo717\000", demo717(a) == a)
}

type point718 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo718 C.demo718
func cdemo718(a point718) point718

func demo718(a point718) point718 {
	return a
}

func init() {
	a := point718{1, 2, 3, 4, 5}
	assert("cdemo718\000", cdemo718(a) == a)
	assert("demo718\000", demo718(a) == a)
}

type point719 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo719 C.demo719
func cdemo719(a point719) point719

func demo719(a point719) point719 {
	return a
}

func init() {
	a := point719{1, 2, 3, 4, 5}
	assert("cdemo719\000", cdemo719(a) == a)
	assert("demo719\000", demo719(a) == a)
}

type point720 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo720 C.demo720
func cdemo720(a point720) point720

func demo720(a point720) point720 {
	return a
}

func init() {
	a := point720{1, 2, 3, 4, 5}
	assert("cdemo720\000", cdemo720(a) == a)
	assert("demo720\000", demo720(a) == a)
}

type point721 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int16
}

//go:linkname cdemo721 C.demo721
func cdemo721(a point721) point721

func demo721(a point721) point721 {
	return a
}

func init() {
	a := point721{1, 2, 3, 4}
	assert("cdemo721\000", cdemo721(a) == a)
	assert("demo721\000", demo721(a) == a)
}

type point722 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo722 C.demo722
func cdemo722(a point722) point722

func demo722(a point722) point722 {
	return a
}

func init() {
	a := point722{1, 2, 3, 4, 5}
	assert("cdemo722\000", cdemo722(a) == a)
	assert("demo722\000", demo722(a) == a)
}

type point723 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo723 C.demo723
func cdemo723(a point723) point723

func demo723(a point723) point723 {
	return a
}

func init() {
	a := point723{1, 2, 3, 4, 5}
	assert("cdemo723\000", cdemo723(a) == a)
	assert("demo723\000", demo723(a) == a)
}

type point724 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo724 C.demo724
func cdemo724(a point724) point724

func demo724(a point724) point724 {
	return a
}

func init() {
	a := point724{1, 2, 3, 4, 5}
	assert("cdemo724\000", cdemo724(a) == a)
	assert("demo724\000", demo724(a) == a)
}

type point725 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo725 C.demo725
func cdemo725(a point725) point725

func demo725(a point725) point725 {
	return a
}

func init() {
	a := point725{1, 2, 3, 4, 5}
	assert("cdemo725\000", cdemo725(a) == a)
	assert("demo725\000", demo725(a) == a)
}

type point726 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int32
}

//go:linkname cdemo726 C.demo726
func cdemo726(a point726) point726

func demo726(a point726) point726 {
	return a
}

func init() {
	a := point726{1, 2, 3, 4}
	assert("cdemo726\000", cdemo726(a) == a)
	assert("demo726\000", demo726(a) == a)
}

type point727 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo727 C.demo727
func cdemo727(a point727) point727

func demo727(a point727) point727 {
	return a
}

func init() {
	a := point727{1, 2, 3, 4, 5}
	assert("cdemo727\000", cdemo727(a) == a)
	assert("demo727\000", demo727(a) == a)
}

type point728 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo728 C.demo728
func cdemo728(a point728) point728

func demo728(a point728) point728 {
	return a
}

func init() {
	a := point728{1, 2, 3, 4, 5}
	assert("cdemo728\000", cdemo728(a) == a)
	assert("demo728\000", demo728(a) == a)
}

type point729 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo729 C.demo729
func cdemo729(a point729) point729

func demo729(a point729) point729 {
	return a
}

func init() {
	a := point729{1, 2, 3, 4, 5}
	assert("cdemo729\000", cdemo729(a) == a)
	assert("demo729\000", demo729(a) == a)
}

type point730 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo730 C.demo730
func cdemo730(a point730) point730

func demo730(a point730) point730 {
	return a
}

func init() {
	a := point730{1, 2, 3, 4, 5}
	assert("cdemo730\000", cdemo730(a) == a)
	assert("demo730\000", demo730(a) == a)
}

type point731 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 float32
}

//go:linkname cdemo731 C.demo731
func cdemo731(a point731) point731

func demo731(a point731) point731 {
	return a
}

func init() {
	a := point731{1, 2, 3, 4}
	assert("cdemo731\000", cdemo731(a) == a)
	assert("demo731\000", demo731(a) == a)
}

type point732 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo732 C.demo732
func cdemo732(a point732) point732

func demo732(a point732) point732 {
	return a
}

func init() {
	a := point732{1, 2, 3, 4, 5}
	assert("cdemo732\000", cdemo732(a) == a)
	assert("demo732\000", demo732(a) == a)
}

type point733 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo733 C.demo733
func cdemo733(a point733) point733

func demo733(a point733) point733 {
	return a
}

func init() {
	a := point733{1, 2, 3, 4, 5}
	assert("cdemo733\000", cdemo733(a) == a)
	assert("demo733\000", demo733(a) == a)
}

type point734 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo734 C.demo734
func cdemo734(a point734) point734

func demo734(a point734) point734 {
	return a
}

func init() {
	a := point734{1, 2, 3, 4, 5}
	assert("cdemo734\000", cdemo734(a) == a)
	assert("demo734\000", demo734(a) == a)
}

type point735 struct {
	x0 int32
	x1 float32
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo735 C.demo735
func cdemo735(a point735) point735

func demo735(a point735) point735 {
	return a
}

func init() {
	a := point735{1, 2, 3, 4, 5}
	assert("cdemo735\000", cdemo735(a) == a)
	assert("demo735\000", demo735(a) == a)
}

type point736 struct {
	x0 int32
	x1 float32
	x2 int16
}

//go:linkname cdemo736 C.demo736
func cdemo736(a point736) point736

func demo736(a point736) point736 {
	return a
}

func init() {
	a := point736{1, 2, 3}
	assert("cdemo736\000", cdemo736(a) == a)
	assert("demo736\000", demo736(a) == a)
}

type point737 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int8
}

//go:linkname cdemo737 C.demo737
func cdemo737(a point737) point737

func demo737(a point737) point737 {
	return a
}

func init() {
	a := point737{1, 2, 3, 4}
	assert("cdemo737\000", cdemo737(a) == a)
	assert("demo737\000", demo737(a) == a)
}

type point738 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo738 C.demo738
func cdemo738(a point738) point738

func demo738(a point738) point738 {
	return a
}

func init() {
	a := point738{1, 2, 3, 4, 5}
	assert("cdemo738\000", cdemo738(a) == a)
	assert("demo738\000", demo738(a) == a)
}

type point739 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo739 C.demo739
func cdemo739(a point739) point739

func demo739(a point739) point739 {
	return a
}

func init() {
	a := point739{1, 2, 3, 4, 5}
	assert("cdemo739\000", cdemo739(a) == a)
	assert("demo739\000", demo739(a) == a)
}

type point740 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo740 C.demo740
func cdemo740(a point740) point740

func demo740(a point740) point740 {
	return a
}

func init() {
	a := point740{1, 2, 3, 4, 5}
	assert("cdemo740\000", cdemo740(a) == a)
	assert("demo740\000", demo740(a) == a)
}

type point741 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo741 C.demo741
func cdemo741(a point741) point741

func demo741(a point741) point741 {
	return a
}

func init() {
	a := point741{1, 2, 3, 4, 5}
	assert("cdemo741\000", cdemo741(a) == a)
	assert("demo741\000", demo741(a) == a)
}

type point742 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int16
}

//go:linkname cdemo742 C.demo742
func cdemo742(a point742) point742

func demo742(a point742) point742 {
	return a
}

func init() {
	a := point742{1, 2, 3, 4}
	assert("cdemo742\000", cdemo742(a) == a)
	assert("demo742\000", demo742(a) == a)
}

type point743 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo743 C.demo743
func cdemo743(a point743) point743

func demo743(a point743) point743 {
	return a
}

func init() {
	a := point743{1, 2, 3, 4, 5}
	assert("cdemo743\000", cdemo743(a) == a)
	assert("demo743\000", demo743(a) == a)
}

type point744 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo744 C.demo744
func cdemo744(a point744) point744

func demo744(a point744) point744 {
	return a
}

func init() {
	a := point744{1, 2, 3, 4, 5}
	assert("cdemo744\000", cdemo744(a) == a)
	assert("demo744\000", demo744(a) == a)
}

type point745 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo745 C.demo745
func cdemo745(a point745) point745

func demo745(a point745) point745 {
	return a
}

func init() {
	a := point745{1, 2, 3, 4, 5}
	assert("cdemo745\000", cdemo745(a) == a)
	assert("demo745\000", demo745(a) == a)
}

type point746 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo746 C.demo746
func cdemo746(a point746) point746

func demo746(a point746) point746 {
	return a
}

func init() {
	a := point746{1, 2, 3, 4, 5}
	assert("cdemo746\000", cdemo746(a) == a)
	assert("demo746\000", demo746(a) == a)
}

type point747 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int32
}

//go:linkname cdemo747 C.demo747
func cdemo747(a point747) point747

func demo747(a point747) point747 {
	return a
}

func init() {
	a := point747{1, 2, 3, 4}
	assert("cdemo747\000", cdemo747(a) == a)
	assert("demo747\000", demo747(a) == a)
}

type point748 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo748 C.demo748
func cdemo748(a point748) point748

func demo748(a point748) point748 {
	return a
}

func init() {
	a := point748{1, 2, 3, 4, 5}
	assert("cdemo748\000", cdemo748(a) == a)
	assert("demo748\000", demo748(a) == a)
}

type point749 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo749 C.demo749
func cdemo749(a point749) point749

func demo749(a point749) point749 {
	return a
}

func init() {
	a := point749{1, 2, 3, 4, 5}
	assert("cdemo749\000", cdemo749(a) == a)
	assert("demo749\000", demo749(a) == a)
}

type point750 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo750 C.demo750
func cdemo750(a point750) point750

func demo750(a point750) point750 {
	return a
}

func init() {
	a := point750{1, 2, 3, 4, 5}
	assert("cdemo750\000", cdemo750(a) == a)
	assert("demo750\000", demo750(a) == a)
}

type point751 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo751 C.demo751
func cdemo751(a point751) point751

func demo751(a point751) point751 {
	return a
}

func init() {
	a := point751{1, 2, 3, 4, 5}
	assert("cdemo751\000", cdemo751(a) == a)
	assert("demo751\000", demo751(a) == a)
}

type point752 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 float32
}

//go:linkname cdemo752 C.demo752
func cdemo752(a point752) point752

func demo752(a point752) point752 {
	return a
}

func init() {
	a := point752{1, 2, 3, 4}
	assert("cdemo752\000", cdemo752(a) == a)
	assert("demo752\000", demo752(a) == a)
}

type point753 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo753 C.demo753
func cdemo753(a point753) point753

func demo753(a point753) point753 {
	return a
}

func init() {
	a := point753{1, 2, 3, 4, 5}
	assert("cdemo753\000", cdemo753(a) == a)
	assert("demo753\000", demo753(a) == a)
}

type point754 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo754 C.demo754
func cdemo754(a point754) point754

func demo754(a point754) point754 {
	return a
}

func init() {
	a := point754{1, 2, 3, 4, 5}
	assert("cdemo754\000", cdemo754(a) == a)
	assert("demo754\000", demo754(a) == a)
}

type point755 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo755 C.demo755
func cdemo755(a point755) point755

func demo755(a point755) point755 {
	return a
}

func init() {
	a := point755{1, 2, 3, 4, 5}
	assert("cdemo755\000", cdemo755(a) == a)
	assert("demo755\000", demo755(a) == a)
}

type point756 struct {
	x0 int32
	x1 float32
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo756 C.demo756
func cdemo756(a point756) point756

func demo756(a point756) point756 {
	return a
}

func init() {
	a := point756{1, 2, 3, 4, 5}
	assert("cdemo756\000", cdemo756(a) == a)
	assert("demo756\000", demo756(a) == a)
}

type point757 struct {
	x0 int32
	x1 float32
	x2 int32
}

//go:linkname cdemo757 C.demo757
func cdemo757(a point757) point757

func demo757(a point757) point757 {
	return a
}

func init() {
	a := point757{1, 2, 3}
	assert("cdemo757\000", cdemo757(a) == a)
	assert("demo757\000", demo757(a) == a)
}

type point758 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int8
}

//go:linkname cdemo758 C.demo758
func cdemo758(a point758) point758

func demo758(a point758) point758 {
	return a
}

func init() {
	a := point758{1, 2, 3, 4}
	assert("cdemo758\000", cdemo758(a) == a)
	assert("demo758\000", demo758(a) == a)
}

type point759 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo759 C.demo759
func cdemo759(a point759) point759

func demo759(a point759) point759 {
	return a
}

func init() {
	a := point759{1, 2, 3, 4, 5}
	assert("cdemo759\000", cdemo759(a) == a)
	assert("demo759\000", demo759(a) == a)
}

type point760 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo760 C.demo760
func cdemo760(a point760) point760

func demo760(a point760) point760 {
	return a
}

func init() {
	a := point760{1, 2, 3, 4, 5}
	assert("cdemo760\000", cdemo760(a) == a)
	assert("demo760\000", demo760(a) == a)
}

type point761 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo761 C.demo761
func cdemo761(a point761) point761

func demo761(a point761) point761 {
	return a
}

func init() {
	a := point761{1, 2, 3, 4, 5}
	assert("cdemo761\000", cdemo761(a) == a)
	assert("demo761\000", demo761(a) == a)
}

type point762 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo762 C.demo762
func cdemo762(a point762) point762

func demo762(a point762) point762 {
	return a
}

func init() {
	a := point762{1, 2, 3, 4, 5}
	assert("cdemo762\000", cdemo762(a) == a)
	assert("demo762\000", demo762(a) == a)
}

type point763 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int16
}

//go:linkname cdemo763 C.demo763
func cdemo763(a point763) point763

func demo763(a point763) point763 {
	return a
}

func init() {
	a := point763{1, 2, 3, 4}
	assert("cdemo763\000", cdemo763(a) == a)
	assert("demo763\000", demo763(a) == a)
}

type point764 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo764 C.demo764
func cdemo764(a point764) point764

func demo764(a point764) point764 {
	return a
}

func init() {
	a := point764{1, 2, 3, 4, 5}
	assert("cdemo764\000", cdemo764(a) == a)
	assert("demo764\000", demo764(a) == a)
}

type point765 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo765 C.demo765
func cdemo765(a point765) point765

func demo765(a point765) point765 {
	return a
}

func init() {
	a := point765{1, 2, 3, 4, 5}
	assert("cdemo765\000", cdemo765(a) == a)
	assert("demo765\000", demo765(a) == a)
}

type point766 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo766 C.demo766
func cdemo766(a point766) point766

func demo766(a point766) point766 {
	return a
}

func init() {
	a := point766{1, 2, 3, 4, 5}
	assert("cdemo766\000", cdemo766(a) == a)
	assert("demo766\000", demo766(a) == a)
}

type point767 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo767 C.demo767
func cdemo767(a point767) point767

func demo767(a point767) point767 {
	return a
}

func init() {
	a := point767{1, 2, 3, 4, 5}
	assert("cdemo767\000", cdemo767(a) == a)
	assert("demo767\000", demo767(a) == a)
}

type point768 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int32
}

//go:linkname cdemo768 C.demo768
func cdemo768(a point768) point768

func demo768(a point768) point768 {
	return a
}

func init() {
	a := point768{1, 2, 3, 4}
	assert("cdemo768\000", cdemo768(a) == a)
	assert("demo768\000", demo768(a) == a)
}

type point769 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo769 C.demo769
func cdemo769(a point769) point769

func demo769(a point769) point769 {
	return a
}

func init() {
	a := point769{1, 2, 3, 4, 5}
	assert("cdemo769\000", cdemo769(a) == a)
	assert("demo769\000", demo769(a) == a)
}

type point770 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo770 C.demo770
func cdemo770(a point770) point770

func demo770(a point770) point770 {
	return a
}

func init() {
	a := point770{1, 2, 3, 4, 5}
	assert("cdemo770\000", cdemo770(a) == a)
	assert("demo770\000", demo770(a) == a)
}

type point771 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo771 C.demo771
func cdemo771(a point771) point771

func demo771(a point771) point771 {
	return a
}

func init() {
	a := point771{1, 2, 3, 4, 5}
	assert("cdemo771\000", cdemo771(a) == a)
	assert("demo771\000", demo771(a) == a)
}

type point772 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo772 C.demo772
func cdemo772(a point772) point772

func demo772(a point772) point772 {
	return a
}

func init() {
	a := point772{1, 2, 3, 4, 5}
	assert("cdemo772\000", cdemo772(a) == a)
	assert("demo772\000", demo772(a) == a)
}

type point773 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 float32
}

//go:linkname cdemo773 C.demo773
func cdemo773(a point773) point773

func demo773(a point773) point773 {
	return a
}

func init() {
	a := point773{1, 2, 3, 4}
	assert("cdemo773\000", cdemo773(a) == a)
	assert("demo773\000", demo773(a) == a)
}

type point774 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo774 C.demo774
func cdemo774(a point774) point774

func demo774(a point774) point774 {
	return a
}

func init() {
	a := point774{1, 2, 3, 4, 5}
	assert("cdemo774\000", cdemo774(a) == a)
	assert("demo774\000", demo774(a) == a)
}

type point775 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo775 C.demo775
func cdemo775(a point775) point775

func demo775(a point775) point775 {
	return a
}

func init() {
	a := point775{1, 2, 3, 4, 5}
	assert("cdemo775\000", cdemo775(a) == a)
	assert("demo775\000", demo775(a) == a)
}

type point776 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo776 C.demo776
func cdemo776(a point776) point776

func demo776(a point776) point776 {
	return a
}

func init() {
	a := point776{1, 2, 3, 4, 5}
	assert("cdemo776\000", cdemo776(a) == a)
	assert("demo776\000", demo776(a) == a)
}

type point777 struct {
	x0 int32
	x1 float32
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo777 C.demo777
func cdemo777(a point777) point777

func demo777(a point777) point777 {
	return a
}

func init() {
	a := point777{1, 2, 3, 4, 5}
	assert("cdemo777\000", cdemo777(a) == a)
	assert("demo777\000", demo777(a) == a)
}

type point778 struct {
	x0 int32
	x1 float32
	x2 float32
}

//go:linkname cdemo778 C.demo778
func cdemo778(a point778) point778

func demo778(a point778) point778 {
	return a
}

func init() {
	a := point778{1, 2, 3}
	assert("cdemo778\000", cdemo778(a) == a)
	assert("demo778\000", demo778(a) == a)
}

type point779 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int8
}

//go:linkname cdemo779 C.demo779
func cdemo779(a point779) point779

func demo779(a point779) point779 {
	return a
}

func init() {
	a := point779{1, 2, 3, 4}
	assert("cdemo779\000", cdemo779(a) == a)
	assert("demo779\000", demo779(a) == a)
}

type point780 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo780 C.demo780
func cdemo780(a point780) point780

func demo780(a point780) point780 {
	return a
}

func init() {
	a := point780{1, 2, 3, 4, 5}
	assert("cdemo780\000", cdemo780(a) == a)
	assert("demo780\000", demo780(a) == a)
}

type point781 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo781 C.demo781
func cdemo781(a point781) point781

func demo781(a point781) point781 {
	return a
}

func init() {
	a := point781{1, 2, 3, 4, 5}
	assert("cdemo781\000", cdemo781(a) == a)
	assert("demo781\000", demo781(a) == a)
}

type point782 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo782 C.demo782
func cdemo782(a point782) point782

func demo782(a point782) point782 {
	return a
}

func init() {
	a := point782{1, 2, 3, 4, 5}
	assert("cdemo782\000", cdemo782(a) == a)
	assert("demo782\000", demo782(a) == a)
}

type point783 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo783 C.demo783
func cdemo783(a point783) point783

func demo783(a point783) point783 {
	return a
}

func init() {
	a := point783{1, 2, 3, 4, 5}
	assert("cdemo783\000", cdemo783(a) == a)
	assert("demo783\000", demo783(a) == a)
}

type point784 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int16
}

//go:linkname cdemo784 C.demo784
func cdemo784(a point784) point784

func demo784(a point784) point784 {
	return a
}

func init() {
	a := point784{1, 2, 3, 4}
	assert("cdemo784\000", cdemo784(a) == a)
	assert("demo784\000", demo784(a) == a)
}

type point785 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo785 C.demo785
func cdemo785(a point785) point785

func demo785(a point785) point785 {
	return a
}

func init() {
	a := point785{1, 2, 3, 4, 5}
	assert("cdemo785\000", cdemo785(a) == a)
	assert("demo785\000", demo785(a) == a)
}

type point786 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo786 C.demo786
func cdemo786(a point786) point786

func demo786(a point786) point786 {
	return a
}

func init() {
	a := point786{1, 2, 3, 4, 5}
	assert("cdemo786\000", cdemo786(a) == a)
	assert("demo786\000", demo786(a) == a)
}

type point787 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo787 C.demo787
func cdemo787(a point787) point787

func demo787(a point787) point787 {
	return a
}

func init() {
	a := point787{1, 2, 3, 4, 5}
	assert("cdemo787\000", cdemo787(a) == a)
	assert("demo787\000", demo787(a) == a)
}

type point788 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo788 C.demo788
func cdemo788(a point788) point788

func demo788(a point788) point788 {
	return a
}

func init() {
	a := point788{1, 2, 3, 4, 5}
	assert("cdemo788\000", cdemo788(a) == a)
	assert("demo788\000", demo788(a) == a)
}

type point789 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int32
}

//go:linkname cdemo789 C.demo789
func cdemo789(a point789) point789

func demo789(a point789) point789 {
	return a
}

func init() {
	a := point789{1, 2, 3, 4}
	assert("cdemo789\000", cdemo789(a) == a)
	assert("demo789\000", demo789(a) == a)
}

type point790 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo790 C.demo790
func cdemo790(a point790) point790

func demo790(a point790) point790 {
	return a
}

func init() {
	a := point790{1, 2, 3, 4, 5}
	assert("cdemo790\000", cdemo790(a) == a)
	assert("demo790\000", demo790(a) == a)
}

type point791 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo791 C.demo791
func cdemo791(a point791) point791

func demo791(a point791) point791 {
	return a
}

func init() {
	a := point791{1, 2, 3, 4, 5}
	assert("cdemo791\000", cdemo791(a) == a)
	assert("demo791\000", demo791(a) == a)
}

type point792 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo792 C.demo792
func cdemo792(a point792) point792

func demo792(a point792) point792 {
	return a
}

func init() {
	a := point792{1, 2, 3, 4, 5}
	assert("cdemo792\000", cdemo792(a) == a)
	assert("demo792\000", demo792(a) == a)
}

type point793 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo793 C.demo793
func cdemo793(a point793) point793

func demo793(a point793) point793 {
	return a
}

func init() {
	a := point793{1, 2, 3, 4, 5}
	assert("cdemo793\000", cdemo793(a) == a)
	assert("demo793\000", demo793(a) == a)
}

type point794 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 float32
}

//go:linkname cdemo794 C.demo794
func cdemo794(a point794) point794

func demo794(a point794) point794 {
	return a
}

func init() {
	a := point794{1, 2, 3, 4}
	assert("cdemo794\000", cdemo794(a) == a)
	assert("demo794\000", demo794(a) == a)
}

type point795 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo795 C.demo795
func cdemo795(a point795) point795

func demo795(a point795) point795 {
	return a
}

func init() {
	a := point795{1, 2, 3, 4, 5}
	assert("cdemo795\000", cdemo795(a) == a)
	assert("demo795\000", demo795(a) == a)
}

type point796 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo796 C.demo796
func cdemo796(a point796) point796

func demo796(a point796) point796 {
	return a
}

func init() {
	a := point796{1, 2, 3, 4, 5}
	assert("cdemo796\000", cdemo796(a) == a)
	assert("demo796\000", demo796(a) == a)
}

type point797 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo797 C.demo797
func cdemo797(a point797) point797

func demo797(a point797) point797 {
	return a
}

func init() {
	a := point797{1, 2, 3, 4, 5}
	assert("cdemo797\000", cdemo797(a) == a)
	assert("demo797\000", demo797(a) == a)
}

type point798 struct {
	x0 int32
	x1 float32
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo798 C.demo798
func cdemo798(a point798) point798

func demo798(a point798) point798 {
	return a
}

func init() {
	a := point798{1, 2, 3, 4, 5}
	assert("cdemo798\000", cdemo798(a) == a)
	assert("demo798\000", demo798(a) == a)
}

type point799 struct {
	x0 float32
	x1 int8
	x2 int8
}

//go:linkname cdemo799 C.demo799
func cdemo799(a point799) point799

func demo799(a point799) point799 {
	return a
}

func init() {
	a := point799{1, 2, 3}
	assert("cdemo799\000", cdemo799(a) == a)
	assert("demo799\000", demo799(a) == a)
}

type point800 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int8
}

//go:linkname cdemo800 C.demo800
func cdemo800(a point800) point800

func demo800(a point800) point800 {
	return a
}

func init() {
	a := point800{1, 2, 3, 4}
	assert("cdemo800\000", cdemo800(a) == a)
	assert("demo800\000", demo800(a) == a)
}

type point801 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo801 C.demo801
func cdemo801(a point801) point801

func demo801(a point801) point801 {
	return a
}

func init() {
	a := point801{1, 2, 3, 4, 5}
	assert("cdemo801\000", cdemo801(a) == a)
	assert("demo801\000", demo801(a) == a)
}

type point802 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo802 C.demo802
func cdemo802(a point802) point802

func demo802(a point802) point802 {
	return a
}

func init() {
	a := point802{1, 2, 3, 4, 5}
	assert("cdemo802\000", cdemo802(a) == a)
	assert("demo802\000", demo802(a) == a)
}

type point803 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo803 C.demo803
func cdemo803(a point803) point803

func demo803(a point803) point803 {
	return a
}

func init() {
	a := point803{1, 2, 3, 4, 5}
	assert("cdemo803\000", cdemo803(a) == a)
	assert("demo803\000", demo803(a) == a)
}

type point804 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo804 C.demo804
func cdemo804(a point804) point804

func demo804(a point804) point804 {
	return a
}

func init() {
	a := point804{1, 2, 3, 4, 5}
	assert("cdemo804\000", cdemo804(a) == a)
	assert("demo804\000", demo804(a) == a)
}

type point805 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int16
}

//go:linkname cdemo805 C.demo805
func cdemo805(a point805) point805

func demo805(a point805) point805 {
	return a
}

func init() {
	a := point805{1, 2, 3, 4}
	assert("cdemo805\000", cdemo805(a) == a)
	assert("demo805\000", demo805(a) == a)
}

type point806 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo806 C.demo806
func cdemo806(a point806) point806

func demo806(a point806) point806 {
	return a
}

func init() {
	a := point806{1, 2, 3, 4, 5}
	assert("cdemo806\000", cdemo806(a) == a)
	assert("demo806\000", demo806(a) == a)
}

type point807 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo807 C.demo807
func cdemo807(a point807) point807

func demo807(a point807) point807 {
	return a
}

func init() {
	a := point807{1, 2, 3, 4, 5}
	assert("cdemo807\000", cdemo807(a) == a)
	assert("demo807\000", demo807(a) == a)
}

type point808 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo808 C.demo808
func cdemo808(a point808) point808

func demo808(a point808) point808 {
	return a
}

func init() {
	a := point808{1, 2, 3, 4, 5}
	assert("cdemo808\000", cdemo808(a) == a)
	assert("demo808\000", demo808(a) == a)
}

type point809 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo809 C.demo809
func cdemo809(a point809) point809

func demo809(a point809) point809 {
	return a
}

func init() {
	a := point809{1, 2, 3, 4, 5}
	assert("cdemo809\000", cdemo809(a) == a)
	assert("demo809\000", demo809(a) == a)
}

type point810 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int32
}

//go:linkname cdemo810 C.demo810
func cdemo810(a point810) point810

func demo810(a point810) point810 {
	return a
}

func init() {
	a := point810{1, 2, 3, 4}
	assert("cdemo810\000", cdemo810(a) == a)
	assert("demo810\000", demo810(a) == a)
}

type point811 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo811 C.demo811
func cdemo811(a point811) point811

func demo811(a point811) point811 {
	return a
}

func init() {
	a := point811{1, 2, 3, 4, 5}
	assert("cdemo811\000", cdemo811(a) == a)
	assert("demo811\000", demo811(a) == a)
}

type point812 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo812 C.demo812
func cdemo812(a point812) point812

func demo812(a point812) point812 {
	return a
}

func init() {
	a := point812{1, 2, 3, 4, 5}
	assert("cdemo812\000", cdemo812(a) == a)
	assert("demo812\000", demo812(a) == a)
}

type point813 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo813 C.demo813
func cdemo813(a point813) point813

func demo813(a point813) point813 {
	return a
}

func init() {
	a := point813{1, 2, 3, 4, 5}
	assert("cdemo813\000", cdemo813(a) == a)
	assert("demo813\000", demo813(a) == a)
}

type point814 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo814 C.demo814
func cdemo814(a point814) point814

func demo814(a point814) point814 {
	return a
}

func init() {
	a := point814{1, 2, 3, 4, 5}
	assert("cdemo814\000", cdemo814(a) == a)
	assert("demo814\000", demo814(a) == a)
}

type point815 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 float32
}

//go:linkname cdemo815 C.demo815
func cdemo815(a point815) point815

func demo815(a point815) point815 {
	return a
}

func init() {
	a := point815{1, 2, 3, 4}
	assert("cdemo815\000", cdemo815(a) == a)
	assert("demo815\000", demo815(a) == a)
}

type point816 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo816 C.demo816
func cdemo816(a point816) point816

func demo816(a point816) point816 {
	return a
}

func init() {
	a := point816{1, 2, 3, 4, 5}
	assert("cdemo816\000", cdemo816(a) == a)
	assert("demo816\000", demo816(a) == a)
}

type point817 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo817 C.demo817
func cdemo817(a point817) point817

func demo817(a point817) point817 {
	return a
}

func init() {
	a := point817{1, 2, 3, 4, 5}
	assert("cdemo817\000", cdemo817(a) == a)
	assert("demo817\000", demo817(a) == a)
}

type point818 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo818 C.demo818
func cdemo818(a point818) point818

func demo818(a point818) point818 {
	return a
}

func init() {
	a := point818{1, 2, 3, 4, 5}
	assert("cdemo818\000", cdemo818(a) == a)
	assert("demo818\000", demo818(a) == a)
}

type point819 struct {
	x0 float32
	x1 int8
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo819 C.demo819
func cdemo819(a point819) point819

func demo819(a point819) point819 {
	return a
}

func init() {
	a := point819{1, 2, 3, 4, 5}
	assert("cdemo819\000", cdemo819(a) == a)
	assert("demo819\000", demo819(a) == a)
}

type point820 struct {
	x0 float32
	x1 int8
	x2 int16
}

//go:linkname cdemo820 C.demo820
func cdemo820(a point820) point820

func demo820(a point820) point820 {
	return a
}

func init() {
	a := point820{1, 2, 3}
	assert("cdemo820\000", cdemo820(a) == a)
	assert("demo820\000", demo820(a) == a)
}

type point821 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int8
}

//go:linkname cdemo821 C.demo821
func cdemo821(a point821) point821

func demo821(a point821) point821 {
	return a
}

func init() {
	a := point821{1, 2, 3, 4}
	assert("cdemo821\000", cdemo821(a) == a)
	assert("demo821\000", demo821(a) == a)
}

type point822 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo822 C.demo822
func cdemo822(a point822) point822

func demo822(a point822) point822 {
	return a
}

func init() {
	a := point822{1, 2, 3, 4, 5}
	assert("cdemo822\000", cdemo822(a) == a)
	assert("demo822\000", demo822(a) == a)
}

type point823 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo823 C.demo823
func cdemo823(a point823) point823

func demo823(a point823) point823 {
	return a
}

func init() {
	a := point823{1, 2, 3, 4, 5}
	assert("cdemo823\000", cdemo823(a) == a)
	assert("demo823\000", demo823(a) == a)
}

type point824 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo824 C.demo824
func cdemo824(a point824) point824

func demo824(a point824) point824 {
	return a
}

func init() {
	a := point824{1, 2, 3, 4, 5}
	assert("cdemo824\000", cdemo824(a) == a)
	assert("demo824\000", demo824(a) == a)
}

type point825 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo825 C.demo825
func cdemo825(a point825) point825

func demo825(a point825) point825 {
	return a
}

func init() {
	a := point825{1, 2, 3, 4, 5}
	assert("cdemo825\000", cdemo825(a) == a)
	assert("demo825\000", demo825(a) == a)
}

type point826 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int16
}

//go:linkname cdemo826 C.demo826
func cdemo826(a point826) point826

func demo826(a point826) point826 {
	return a
}

func init() {
	a := point826{1, 2, 3, 4}
	assert("cdemo826\000", cdemo826(a) == a)
	assert("demo826\000", demo826(a) == a)
}

type point827 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo827 C.demo827
func cdemo827(a point827) point827

func demo827(a point827) point827 {
	return a
}

func init() {
	a := point827{1, 2, 3, 4, 5}
	assert("cdemo827\000", cdemo827(a) == a)
	assert("demo827\000", demo827(a) == a)
}

type point828 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo828 C.demo828
func cdemo828(a point828) point828

func demo828(a point828) point828 {
	return a
}

func init() {
	a := point828{1, 2, 3, 4, 5}
	assert("cdemo828\000", cdemo828(a) == a)
	assert("demo828\000", demo828(a) == a)
}

type point829 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo829 C.demo829
func cdemo829(a point829) point829

func demo829(a point829) point829 {
	return a
}

func init() {
	a := point829{1, 2, 3, 4, 5}
	assert("cdemo829\000", cdemo829(a) == a)
	assert("demo829\000", demo829(a) == a)
}

type point830 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo830 C.demo830
func cdemo830(a point830) point830

func demo830(a point830) point830 {
	return a
}

func init() {
	a := point830{1, 2, 3, 4, 5}
	assert("cdemo830\000", cdemo830(a) == a)
	assert("demo830\000", demo830(a) == a)
}

type point831 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int32
}

//go:linkname cdemo831 C.demo831
func cdemo831(a point831) point831

func demo831(a point831) point831 {
	return a
}

func init() {
	a := point831{1, 2, 3, 4}
	assert("cdemo831\000", cdemo831(a) == a)
	assert("demo831\000", demo831(a) == a)
}

type point832 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo832 C.demo832
func cdemo832(a point832) point832

func demo832(a point832) point832 {
	return a
}

func init() {
	a := point832{1, 2, 3, 4, 5}
	assert("cdemo832\000", cdemo832(a) == a)
	assert("demo832\000", demo832(a) == a)
}

type point833 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo833 C.demo833
func cdemo833(a point833) point833

func demo833(a point833) point833 {
	return a
}

func init() {
	a := point833{1, 2, 3, 4, 5}
	assert("cdemo833\000", cdemo833(a) == a)
	assert("demo833\000", demo833(a) == a)
}

type point834 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo834 C.demo834
func cdemo834(a point834) point834

func demo834(a point834) point834 {
	return a
}

func init() {
	a := point834{1, 2, 3, 4, 5}
	assert("cdemo834\000", cdemo834(a) == a)
	assert("demo834\000", demo834(a) == a)
}

type point835 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo835 C.demo835
func cdemo835(a point835) point835

func demo835(a point835) point835 {
	return a
}

func init() {
	a := point835{1, 2, 3, 4, 5}
	assert("cdemo835\000", cdemo835(a) == a)
	assert("demo835\000", demo835(a) == a)
}

type point836 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 float32
}

//go:linkname cdemo836 C.demo836
func cdemo836(a point836) point836

func demo836(a point836) point836 {
	return a
}

func init() {
	a := point836{1, 2, 3, 4}
	assert("cdemo836\000", cdemo836(a) == a)
	assert("demo836\000", demo836(a) == a)
}

type point837 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo837 C.demo837
func cdemo837(a point837) point837

func demo837(a point837) point837 {
	return a
}

func init() {
	a := point837{1, 2, 3, 4, 5}
	assert("cdemo837\000", cdemo837(a) == a)
	assert("demo837\000", demo837(a) == a)
}

type point838 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo838 C.demo838
func cdemo838(a point838) point838

func demo838(a point838) point838 {
	return a
}

func init() {
	a := point838{1, 2, 3, 4, 5}
	assert("cdemo838\000", cdemo838(a) == a)
	assert("demo838\000", demo838(a) == a)
}

type point839 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo839 C.demo839
func cdemo839(a point839) point839

func demo839(a point839) point839 {
	return a
}

func init() {
	a := point839{1, 2, 3, 4, 5}
	assert("cdemo839\000", cdemo839(a) == a)
	assert("demo839\000", demo839(a) == a)
}

type point840 struct {
	x0 float32
	x1 int8
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo840 C.demo840
func cdemo840(a point840) point840

func demo840(a point840) point840 {
	return a
}

func init() {
	a := point840{1, 2, 3, 4, 5}
	assert("cdemo840\000", cdemo840(a) == a)
	assert("demo840\000", demo840(a) == a)
}

type point841 struct {
	x0 float32
	x1 int8
	x2 int32
}

//go:linkname cdemo841 C.demo841
func cdemo841(a point841) point841

func demo841(a point841) point841 {
	return a
}

func init() {
	a := point841{1, 2, 3}
	assert("cdemo841\000", cdemo841(a) == a)
	assert("demo841\000", demo841(a) == a)
}

type point842 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int8
}

//go:linkname cdemo842 C.demo842
func cdemo842(a point842) point842

func demo842(a point842) point842 {
	return a
}

func init() {
	a := point842{1, 2, 3, 4}
	assert("cdemo842\000", cdemo842(a) == a)
	assert("demo842\000", demo842(a) == a)
}

type point843 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo843 C.demo843
func cdemo843(a point843) point843

func demo843(a point843) point843 {
	return a
}

func init() {
	a := point843{1, 2, 3, 4, 5}
	assert("cdemo843\000", cdemo843(a) == a)
	assert("demo843\000", demo843(a) == a)
}

type point844 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo844 C.demo844
func cdemo844(a point844) point844

func demo844(a point844) point844 {
	return a
}

func init() {
	a := point844{1, 2, 3, 4, 5}
	assert("cdemo844\000", cdemo844(a) == a)
	assert("demo844\000", demo844(a) == a)
}

type point845 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo845 C.demo845
func cdemo845(a point845) point845

func demo845(a point845) point845 {
	return a
}

func init() {
	a := point845{1, 2, 3, 4, 5}
	assert("cdemo845\000", cdemo845(a) == a)
	assert("demo845\000", demo845(a) == a)
}

type point846 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo846 C.demo846
func cdemo846(a point846) point846

func demo846(a point846) point846 {
	return a
}

func init() {
	a := point846{1, 2, 3, 4, 5}
	assert("cdemo846\000", cdemo846(a) == a)
	assert("demo846\000", demo846(a) == a)
}

type point847 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int16
}

//go:linkname cdemo847 C.demo847
func cdemo847(a point847) point847

func demo847(a point847) point847 {
	return a
}

func init() {
	a := point847{1, 2, 3, 4}
	assert("cdemo847\000", cdemo847(a) == a)
	assert("demo847\000", demo847(a) == a)
}

type point848 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo848 C.demo848
func cdemo848(a point848) point848

func demo848(a point848) point848 {
	return a
}

func init() {
	a := point848{1, 2, 3, 4, 5}
	assert("cdemo848\000", cdemo848(a) == a)
	assert("demo848\000", demo848(a) == a)
}

type point849 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo849 C.demo849
func cdemo849(a point849) point849

func demo849(a point849) point849 {
	return a
}

func init() {
	a := point849{1, 2, 3, 4, 5}
	assert("cdemo849\000", cdemo849(a) == a)
	assert("demo849\000", demo849(a) == a)
}

type point850 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo850 C.demo850
func cdemo850(a point850) point850

func demo850(a point850) point850 {
	return a
}

func init() {
	a := point850{1, 2, 3, 4, 5}
	assert("cdemo850\000", cdemo850(a) == a)
	assert("demo850\000", demo850(a) == a)
}

type point851 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo851 C.demo851
func cdemo851(a point851) point851

func demo851(a point851) point851 {
	return a
}

func init() {
	a := point851{1, 2, 3, 4, 5}
	assert("cdemo851\000", cdemo851(a) == a)
	assert("demo851\000", demo851(a) == a)
}

type point852 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int32
}

//go:linkname cdemo852 C.demo852
func cdemo852(a point852) point852

func demo852(a point852) point852 {
	return a
}

func init() {
	a := point852{1, 2, 3, 4}
	assert("cdemo852\000", cdemo852(a) == a)
	assert("demo852\000", demo852(a) == a)
}

type point853 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo853 C.demo853
func cdemo853(a point853) point853

func demo853(a point853) point853 {
	return a
}

func init() {
	a := point853{1, 2, 3, 4, 5}
	assert("cdemo853\000", cdemo853(a) == a)
	assert("demo853\000", demo853(a) == a)
}

type point854 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo854 C.demo854
func cdemo854(a point854) point854

func demo854(a point854) point854 {
	return a
}

func init() {
	a := point854{1, 2, 3, 4, 5}
	assert("cdemo854\000", cdemo854(a) == a)
	assert("demo854\000", demo854(a) == a)
}

type point855 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo855 C.demo855
func cdemo855(a point855) point855

func demo855(a point855) point855 {
	return a
}

func init() {
	a := point855{1, 2, 3, 4, 5}
	assert("cdemo855\000", cdemo855(a) == a)
	assert("demo855\000", demo855(a) == a)
}

type point856 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo856 C.demo856
func cdemo856(a point856) point856

func demo856(a point856) point856 {
	return a
}

func init() {
	a := point856{1, 2, 3, 4, 5}
	assert("cdemo856\000", cdemo856(a) == a)
	assert("demo856\000", demo856(a) == a)
}

type point857 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 float32
}

//go:linkname cdemo857 C.demo857
func cdemo857(a point857) point857

func demo857(a point857) point857 {
	return a
}

func init() {
	a := point857{1, 2, 3, 4}
	assert("cdemo857\000", cdemo857(a) == a)
	assert("demo857\000", demo857(a) == a)
}

type point858 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo858 C.demo858
func cdemo858(a point858) point858

func demo858(a point858) point858 {
	return a
}

func init() {
	a := point858{1, 2, 3, 4, 5}
	assert("cdemo858\000", cdemo858(a) == a)
	assert("demo858\000", demo858(a) == a)
}

type point859 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo859 C.demo859
func cdemo859(a point859) point859

func demo859(a point859) point859 {
	return a
}

func init() {
	a := point859{1, 2, 3, 4, 5}
	assert("cdemo859\000", cdemo859(a) == a)
	assert("demo859\000", demo859(a) == a)
}

type point860 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo860 C.demo860
func cdemo860(a point860) point860

func demo860(a point860) point860 {
	return a
}

func init() {
	a := point860{1, 2, 3, 4, 5}
	assert("cdemo860\000", cdemo860(a) == a)
	assert("demo860\000", demo860(a) == a)
}

type point861 struct {
	x0 float32
	x1 int8
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo861 C.demo861
func cdemo861(a point861) point861

func demo861(a point861) point861 {
	return a
}

func init() {
	a := point861{1, 2, 3, 4, 5}
	assert("cdemo861\000", cdemo861(a) == a)
	assert("demo861\000", demo861(a) == a)
}

type point862 struct {
	x0 float32
	x1 int8
	x2 float32
}

//go:linkname cdemo862 C.demo862
func cdemo862(a point862) point862

func demo862(a point862) point862 {
	return a
}

func init() {
	a := point862{1, 2, 3}
	assert("cdemo862\000", cdemo862(a) == a)
	assert("demo862\000", demo862(a) == a)
}

type point863 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int8
}

//go:linkname cdemo863 C.demo863
func cdemo863(a point863) point863

func demo863(a point863) point863 {
	return a
}

func init() {
	a := point863{1, 2, 3, 4}
	assert("cdemo863\000", cdemo863(a) == a)
	assert("demo863\000", demo863(a) == a)
}

type point864 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo864 C.demo864
func cdemo864(a point864) point864

func demo864(a point864) point864 {
	return a
}

func init() {
	a := point864{1, 2, 3, 4, 5}
	assert("cdemo864\000", cdemo864(a) == a)
	assert("demo864\000", demo864(a) == a)
}

type point865 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo865 C.demo865
func cdemo865(a point865) point865

func demo865(a point865) point865 {
	return a
}

func init() {
	a := point865{1, 2, 3, 4, 5}
	assert("cdemo865\000", cdemo865(a) == a)
	assert("demo865\000", demo865(a) == a)
}

type point866 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo866 C.demo866
func cdemo866(a point866) point866

func demo866(a point866) point866 {
	return a
}

func init() {
	a := point866{1, 2, 3, 4, 5}
	assert("cdemo866\000", cdemo866(a) == a)
	assert("demo866\000", demo866(a) == a)
}

type point867 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo867 C.demo867
func cdemo867(a point867) point867

func demo867(a point867) point867 {
	return a
}

func init() {
	a := point867{1, 2, 3, 4, 5}
	assert("cdemo867\000", cdemo867(a) == a)
	assert("demo867\000", demo867(a) == a)
}

type point868 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int16
}

//go:linkname cdemo868 C.demo868
func cdemo868(a point868) point868

func demo868(a point868) point868 {
	return a
}

func init() {
	a := point868{1, 2, 3, 4}
	assert("cdemo868\000", cdemo868(a) == a)
	assert("demo868\000", demo868(a) == a)
}

type point869 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo869 C.demo869
func cdemo869(a point869) point869

func demo869(a point869) point869 {
	return a
}

func init() {
	a := point869{1, 2, 3, 4, 5}
	assert("cdemo869\000", cdemo869(a) == a)
	assert("demo869\000", demo869(a) == a)
}

type point870 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo870 C.demo870
func cdemo870(a point870) point870

func demo870(a point870) point870 {
	return a
}

func init() {
	a := point870{1, 2, 3, 4, 5}
	assert("cdemo870\000", cdemo870(a) == a)
	assert("demo870\000", demo870(a) == a)
}

type point871 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo871 C.demo871
func cdemo871(a point871) point871

func demo871(a point871) point871 {
	return a
}

func init() {
	a := point871{1, 2, 3, 4, 5}
	assert("cdemo871\000", cdemo871(a) == a)
	assert("demo871\000", demo871(a) == a)
}

type point872 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo872 C.demo872
func cdemo872(a point872) point872

func demo872(a point872) point872 {
	return a
}

func init() {
	a := point872{1, 2, 3, 4, 5}
	assert("cdemo872\000", cdemo872(a) == a)
	assert("demo872\000", demo872(a) == a)
}

type point873 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int32
}

//go:linkname cdemo873 C.demo873
func cdemo873(a point873) point873

func demo873(a point873) point873 {
	return a
}

func init() {
	a := point873{1, 2, 3, 4}
	assert("cdemo873\000", cdemo873(a) == a)
	assert("demo873\000", demo873(a) == a)
}

type point874 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo874 C.demo874
func cdemo874(a point874) point874

func demo874(a point874) point874 {
	return a
}

func init() {
	a := point874{1, 2, 3, 4, 5}
	assert("cdemo874\000", cdemo874(a) == a)
	assert("demo874\000", demo874(a) == a)
}

type point875 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo875 C.demo875
func cdemo875(a point875) point875

func demo875(a point875) point875 {
	return a
}

func init() {
	a := point875{1, 2, 3, 4, 5}
	assert("cdemo875\000", cdemo875(a) == a)
	assert("demo875\000", demo875(a) == a)
}

type point876 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo876 C.demo876
func cdemo876(a point876) point876

func demo876(a point876) point876 {
	return a
}

func init() {
	a := point876{1, 2, 3, 4, 5}
	assert("cdemo876\000", cdemo876(a) == a)
	assert("demo876\000", demo876(a) == a)
}

type point877 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo877 C.demo877
func cdemo877(a point877) point877

func demo877(a point877) point877 {
	return a
}

func init() {
	a := point877{1, 2, 3, 4, 5}
	assert("cdemo877\000", cdemo877(a) == a)
	assert("demo877\000", demo877(a) == a)
}

type point878 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 float32
}

//go:linkname cdemo878 C.demo878
func cdemo878(a point878) point878

func demo878(a point878) point878 {
	return a
}

func init() {
	a := point878{1, 2, 3, 4}
	assert("cdemo878\000", cdemo878(a) == a)
	assert("demo878\000", demo878(a) == a)
}

type point879 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo879 C.demo879
func cdemo879(a point879) point879

func demo879(a point879) point879 {
	return a
}

func init() {
	a := point879{1, 2, 3, 4, 5}
	assert("cdemo879\000", cdemo879(a) == a)
	assert("demo879\000", demo879(a) == a)
}

type point880 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo880 C.demo880
func cdemo880(a point880) point880

func demo880(a point880) point880 {
	return a
}

func init() {
	a := point880{1, 2, 3, 4, 5}
	assert("cdemo880\000", cdemo880(a) == a)
	assert("demo880\000", demo880(a) == a)
}

type point881 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo881 C.demo881
func cdemo881(a point881) point881

func demo881(a point881) point881 {
	return a
}

func init() {
	a := point881{1, 2, 3, 4, 5}
	assert("cdemo881\000", cdemo881(a) == a)
	assert("demo881\000", demo881(a) == a)
}

type point882 struct {
	x0 float32
	x1 int8
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo882 C.demo882
func cdemo882(a point882) point882

func demo882(a point882) point882 {
	return a
}

func init() {
	a := point882{1, 2, 3, 4, 5}
	assert("cdemo882\000", cdemo882(a) == a)
	assert("demo882\000", demo882(a) == a)
}

type point883 struct {
	x0 float32
	x1 int16
	x2 int8
}

//go:linkname cdemo883 C.demo883
func cdemo883(a point883) point883

func demo883(a point883) point883 {
	return a
}

func init() {
	a := point883{1, 2, 3}
	assert("cdemo883\000", cdemo883(a) == a)
	assert("demo883\000", demo883(a) == a)
}

type point884 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int8
}

//go:linkname cdemo884 C.demo884
func cdemo884(a point884) point884

func demo884(a point884) point884 {
	return a
}

func init() {
	a := point884{1, 2, 3, 4}
	assert("cdemo884\000", cdemo884(a) == a)
	assert("demo884\000", demo884(a) == a)
}

type point885 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo885 C.demo885
func cdemo885(a point885) point885

func demo885(a point885) point885 {
	return a
}

func init() {
	a := point885{1, 2, 3, 4, 5}
	assert("cdemo885\000", cdemo885(a) == a)
	assert("demo885\000", demo885(a) == a)
}

type point886 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo886 C.demo886
func cdemo886(a point886) point886

func demo886(a point886) point886 {
	return a
}

func init() {
	a := point886{1, 2, 3, 4, 5}
	assert("cdemo886\000", cdemo886(a) == a)
	assert("demo886\000", demo886(a) == a)
}

type point887 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo887 C.demo887
func cdemo887(a point887) point887

func demo887(a point887) point887 {
	return a
}

func init() {
	a := point887{1, 2, 3, 4, 5}
	assert("cdemo887\000", cdemo887(a) == a)
	assert("demo887\000", demo887(a) == a)
}

type point888 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo888 C.demo888
func cdemo888(a point888) point888

func demo888(a point888) point888 {
	return a
}

func init() {
	a := point888{1, 2, 3, 4, 5}
	assert("cdemo888\000", cdemo888(a) == a)
	assert("demo888\000", demo888(a) == a)
}

type point889 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int16
}

//go:linkname cdemo889 C.demo889
func cdemo889(a point889) point889

func demo889(a point889) point889 {
	return a
}

func init() {
	a := point889{1, 2, 3, 4}
	assert("cdemo889\000", cdemo889(a) == a)
	assert("demo889\000", demo889(a) == a)
}

type point890 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo890 C.demo890
func cdemo890(a point890) point890

func demo890(a point890) point890 {
	return a
}

func init() {
	a := point890{1, 2, 3, 4, 5}
	assert("cdemo890\000", cdemo890(a) == a)
	assert("demo890\000", demo890(a) == a)
}

type point891 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo891 C.demo891
func cdemo891(a point891) point891

func demo891(a point891) point891 {
	return a
}

func init() {
	a := point891{1, 2, 3, 4, 5}
	assert("cdemo891\000", cdemo891(a) == a)
	assert("demo891\000", demo891(a) == a)
}

type point892 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo892 C.demo892
func cdemo892(a point892) point892

func demo892(a point892) point892 {
	return a
}

func init() {
	a := point892{1, 2, 3, 4, 5}
	assert("cdemo892\000", cdemo892(a) == a)
	assert("demo892\000", demo892(a) == a)
}

type point893 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo893 C.demo893
func cdemo893(a point893) point893

func demo893(a point893) point893 {
	return a
}

func init() {
	a := point893{1, 2, 3, 4, 5}
	assert("cdemo893\000", cdemo893(a) == a)
	assert("demo893\000", demo893(a) == a)
}

type point894 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int32
}

//go:linkname cdemo894 C.demo894
func cdemo894(a point894) point894

func demo894(a point894) point894 {
	return a
}

func init() {
	a := point894{1, 2, 3, 4}
	assert("cdemo894\000", cdemo894(a) == a)
	assert("demo894\000", demo894(a) == a)
}

type point895 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo895 C.demo895
func cdemo895(a point895) point895

func demo895(a point895) point895 {
	return a
}

func init() {
	a := point895{1, 2, 3, 4, 5}
	assert("cdemo895\000", cdemo895(a) == a)
	assert("demo895\000", demo895(a) == a)
}

type point896 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo896 C.demo896
func cdemo896(a point896) point896

func demo896(a point896) point896 {
	return a
}

func init() {
	a := point896{1, 2, 3, 4, 5}
	assert("cdemo896\000", cdemo896(a) == a)
	assert("demo896\000", demo896(a) == a)
}

type point897 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo897 C.demo897
func cdemo897(a point897) point897

func demo897(a point897) point897 {
	return a
}

func init() {
	a := point897{1, 2, 3, 4, 5}
	assert("cdemo897\000", cdemo897(a) == a)
	assert("demo897\000", demo897(a) == a)
}

type point898 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo898 C.demo898
func cdemo898(a point898) point898

func demo898(a point898) point898 {
	return a
}

func init() {
	a := point898{1, 2, 3, 4, 5}
	assert("cdemo898\000", cdemo898(a) == a)
	assert("demo898\000", demo898(a) == a)
}

type point899 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 float32
}

//go:linkname cdemo899 C.demo899
func cdemo899(a point899) point899

func demo899(a point899) point899 {
	return a
}

func init() {
	a := point899{1, 2, 3, 4}
	assert("cdemo899\000", cdemo899(a) == a)
	assert("demo899\000", demo899(a) == a)
}

type point900 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo900 C.demo900
func cdemo900(a point900) point900

func demo900(a point900) point900 {
	return a
}

func init() {
	a := point900{1, 2, 3, 4, 5}
	assert("cdemo900\000", cdemo900(a) == a)
	assert("demo900\000", demo900(a) == a)
}

type point901 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo901 C.demo901
func cdemo901(a point901) point901

func demo901(a point901) point901 {
	return a
}

func init() {
	a := point901{1, 2, 3, 4, 5}
	assert("cdemo901\000", cdemo901(a) == a)
	assert("demo901\000", demo901(a) == a)
}

type point902 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo902 C.demo902
func cdemo902(a point902) point902

func demo902(a point902) point902 {
	return a
}

func init() {
	a := point902{1, 2, 3, 4, 5}
	assert("cdemo902\000", cdemo902(a) == a)
	assert("demo902\000", demo902(a) == a)
}

type point903 struct {
	x0 float32
	x1 int16
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo903 C.demo903
func cdemo903(a point903) point903

func demo903(a point903) point903 {
	return a
}

func init() {
	a := point903{1, 2, 3, 4, 5}
	assert("cdemo903\000", cdemo903(a) == a)
	assert("demo903\000", demo903(a) == a)
}

type point904 struct {
	x0 float32
	x1 int16
	x2 int16
}

//go:linkname cdemo904 C.demo904
func cdemo904(a point904) point904

func demo904(a point904) point904 {
	return a
}

func init() {
	a := point904{1, 2, 3}
	assert("cdemo904\000", cdemo904(a) == a)
	assert("demo904\000", demo904(a) == a)
}

type point905 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int8
}

//go:linkname cdemo905 C.demo905
func cdemo905(a point905) point905

func demo905(a point905) point905 {
	return a
}

func init() {
	a := point905{1, 2, 3, 4}
	assert("cdemo905\000", cdemo905(a) == a)
	assert("demo905\000", demo905(a) == a)
}

type point906 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo906 C.demo906
func cdemo906(a point906) point906

func demo906(a point906) point906 {
	return a
}

func init() {
	a := point906{1, 2, 3, 4, 5}
	assert("cdemo906\000", cdemo906(a) == a)
	assert("demo906\000", demo906(a) == a)
}

type point907 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo907 C.demo907
func cdemo907(a point907) point907

func demo907(a point907) point907 {
	return a
}

func init() {
	a := point907{1, 2, 3, 4, 5}
	assert("cdemo907\000", cdemo907(a) == a)
	assert("demo907\000", demo907(a) == a)
}

type point908 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo908 C.demo908
func cdemo908(a point908) point908

func demo908(a point908) point908 {
	return a
}

func init() {
	a := point908{1, 2, 3, 4, 5}
	assert("cdemo908\000", cdemo908(a) == a)
	assert("demo908\000", demo908(a) == a)
}

type point909 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo909 C.demo909
func cdemo909(a point909) point909

func demo909(a point909) point909 {
	return a
}

func init() {
	a := point909{1, 2, 3, 4, 5}
	assert("cdemo909\000", cdemo909(a) == a)
	assert("demo909\000", demo909(a) == a)
}

type point910 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int16
}

//go:linkname cdemo910 C.demo910
func cdemo910(a point910) point910

func demo910(a point910) point910 {
	return a
}

func init() {
	a := point910{1, 2, 3, 4}
	assert("cdemo910\000", cdemo910(a) == a)
	assert("demo910\000", demo910(a) == a)
}

type point911 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo911 C.demo911
func cdemo911(a point911) point911

func demo911(a point911) point911 {
	return a
}

func init() {
	a := point911{1, 2, 3, 4, 5}
	assert("cdemo911\000", cdemo911(a) == a)
	assert("demo911\000", demo911(a) == a)
}

type point912 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo912 C.demo912
func cdemo912(a point912) point912

func demo912(a point912) point912 {
	return a
}

func init() {
	a := point912{1, 2, 3, 4, 5}
	assert("cdemo912\000", cdemo912(a) == a)
	assert("demo912\000", demo912(a) == a)
}

type point913 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo913 C.demo913
func cdemo913(a point913) point913

func demo913(a point913) point913 {
	return a
}

func init() {
	a := point913{1, 2, 3, 4, 5}
	assert("cdemo913\000", cdemo913(a) == a)
	assert("demo913\000", demo913(a) == a)
}

type point914 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo914 C.demo914
func cdemo914(a point914) point914

func demo914(a point914) point914 {
	return a
}

func init() {
	a := point914{1, 2, 3, 4, 5}
	assert("cdemo914\000", cdemo914(a) == a)
	assert("demo914\000", demo914(a) == a)
}

type point915 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int32
}

//go:linkname cdemo915 C.demo915
func cdemo915(a point915) point915

func demo915(a point915) point915 {
	return a
}

func init() {
	a := point915{1, 2, 3, 4}
	assert("cdemo915\000", cdemo915(a) == a)
	assert("demo915\000", demo915(a) == a)
}

type point916 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo916 C.demo916
func cdemo916(a point916) point916

func demo916(a point916) point916 {
	return a
}

func init() {
	a := point916{1, 2, 3, 4, 5}
	assert("cdemo916\000", cdemo916(a) == a)
	assert("demo916\000", demo916(a) == a)
}

type point917 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo917 C.demo917
func cdemo917(a point917) point917

func demo917(a point917) point917 {
	return a
}

func init() {
	a := point917{1, 2, 3, 4, 5}
	assert("cdemo917\000", cdemo917(a) == a)
	assert("demo917\000", demo917(a) == a)
}

type point918 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo918 C.demo918
func cdemo918(a point918) point918

func demo918(a point918) point918 {
	return a
}

func init() {
	a := point918{1, 2, 3, 4, 5}
	assert("cdemo918\000", cdemo918(a) == a)
	assert("demo918\000", demo918(a) == a)
}

type point919 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo919 C.demo919
func cdemo919(a point919) point919

func demo919(a point919) point919 {
	return a
}

func init() {
	a := point919{1, 2, 3, 4, 5}
	assert("cdemo919\000", cdemo919(a) == a)
	assert("demo919\000", demo919(a) == a)
}

type point920 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 float32
}

//go:linkname cdemo920 C.demo920
func cdemo920(a point920) point920

func demo920(a point920) point920 {
	return a
}

func init() {
	a := point920{1, 2, 3, 4}
	assert("cdemo920\000", cdemo920(a) == a)
	assert("demo920\000", demo920(a) == a)
}

type point921 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo921 C.demo921
func cdemo921(a point921) point921

func demo921(a point921) point921 {
	return a
}

func init() {
	a := point921{1, 2, 3, 4, 5}
	assert("cdemo921\000", cdemo921(a) == a)
	assert("demo921\000", demo921(a) == a)
}

type point922 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo922 C.demo922
func cdemo922(a point922) point922

func demo922(a point922) point922 {
	return a
}

func init() {
	a := point922{1, 2, 3, 4, 5}
	assert("cdemo922\000", cdemo922(a) == a)
	assert("demo922\000", demo922(a) == a)
}

type point923 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo923 C.demo923
func cdemo923(a point923) point923

func demo923(a point923) point923 {
	return a
}

func init() {
	a := point923{1, 2, 3, 4, 5}
	assert("cdemo923\000", cdemo923(a) == a)
	assert("demo923\000", demo923(a) == a)
}

type point924 struct {
	x0 float32
	x1 int16
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo924 C.demo924
func cdemo924(a point924) point924

func demo924(a point924) point924 {
	return a
}

func init() {
	a := point924{1, 2, 3, 4, 5}
	assert("cdemo924\000", cdemo924(a) == a)
	assert("demo924\000", demo924(a) == a)
}

type point925 struct {
	x0 float32
	x1 int16
	x2 int32
}

//go:linkname cdemo925 C.demo925
func cdemo925(a point925) point925

func demo925(a point925) point925 {
	return a
}

func init() {
	a := point925{1, 2, 3}
	assert("cdemo925\000", cdemo925(a) == a)
	assert("demo925\000", demo925(a) == a)
}

type point926 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int8
}

//go:linkname cdemo926 C.demo926
func cdemo926(a point926) point926

func demo926(a point926) point926 {
	return a
}

func init() {
	a := point926{1, 2, 3, 4}
	assert("cdemo926\000", cdemo926(a) == a)
	assert("demo926\000", demo926(a) == a)
}

type point927 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo927 C.demo927
func cdemo927(a point927) point927

func demo927(a point927) point927 {
	return a
}

func init() {
	a := point927{1, 2, 3, 4, 5}
	assert("cdemo927\000", cdemo927(a) == a)
	assert("demo927\000", demo927(a) == a)
}

type point928 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo928 C.demo928
func cdemo928(a point928) point928

func demo928(a point928) point928 {
	return a
}

func init() {
	a := point928{1, 2, 3, 4, 5}
	assert("cdemo928\000", cdemo928(a) == a)
	assert("demo928\000", demo928(a) == a)
}

type point929 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo929 C.demo929
func cdemo929(a point929) point929

func demo929(a point929) point929 {
	return a
}

func init() {
	a := point929{1, 2, 3, 4, 5}
	assert("cdemo929\000", cdemo929(a) == a)
	assert("demo929\000", demo929(a) == a)
}

type point930 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo930 C.demo930
func cdemo930(a point930) point930

func demo930(a point930) point930 {
	return a
}

func init() {
	a := point930{1, 2, 3, 4, 5}
	assert("cdemo930\000", cdemo930(a) == a)
	assert("demo930\000", demo930(a) == a)
}

type point931 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int16
}

//go:linkname cdemo931 C.demo931
func cdemo931(a point931) point931

func demo931(a point931) point931 {
	return a
}

func init() {
	a := point931{1, 2, 3, 4}
	assert("cdemo931\000", cdemo931(a) == a)
	assert("demo931\000", demo931(a) == a)
}

type point932 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo932 C.demo932
func cdemo932(a point932) point932

func demo932(a point932) point932 {
	return a
}

func init() {
	a := point932{1, 2, 3, 4, 5}
	assert("cdemo932\000", cdemo932(a) == a)
	assert("demo932\000", demo932(a) == a)
}

type point933 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo933 C.demo933
func cdemo933(a point933) point933

func demo933(a point933) point933 {
	return a
}

func init() {
	a := point933{1, 2, 3, 4, 5}
	assert("cdemo933\000", cdemo933(a) == a)
	assert("demo933\000", demo933(a) == a)
}

type point934 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo934 C.demo934
func cdemo934(a point934) point934

func demo934(a point934) point934 {
	return a
}

func init() {
	a := point934{1, 2, 3, 4, 5}
	assert("cdemo934\000", cdemo934(a) == a)
	assert("demo934\000", demo934(a) == a)
}

type point935 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo935 C.demo935
func cdemo935(a point935) point935

func demo935(a point935) point935 {
	return a
}

func init() {
	a := point935{1, 2, 3, 4, 5}
	assert("cdemo935\000", cdemo935(a) == a)
	assert("demo935\000", demo935(a) == a)
}

type point936 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int32
}

//go:linkname cdemo936 C.demo936
func cdemo936(a point936) point936

func demo936(a point936) point936 {
	return a
}

func init() {
	a := point936{1, 2, 3, 4}
	assert("cdemo936\000", cdemo936(a) == a)
	assert("demo936\000", demo936(a) == a)
}

type point937 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo937 C.demo937
func cdemo937(a point937) point937

func demo937(a point937) point937 {
	return a
}

func init() {
	a := point937{1, 2, 3, 4, 5}
	assert("cdemo937\000", cdemo937(a) == a)
	assert("demo937\000", demo937(a) == a)
}

type point938 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo938 C.demo938
func cdemo938(a point938) point938

func demo938(a point938) point938 {
	return a
}

func init() {
	a := point938{1, 2, 3, 4, 5}
	assert("cdemo938\000", cdemo938(a) == a)
	assert("demo938\000", demo938(a) == a)
}

type point939 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo939 C.demo939
func cdemo939(a point939) point939

func demo939(a point939) point939 {
	return a
}

func init() {
	a := point939{1, 2, 3, 4, 5}
	assert("cdemo939\000", cdemo939(a) == a)
	assert("demo939\000", demo939(a) == a)
}

type point940 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo940 C.demo940
func cdemo940(a point940) point940

func demo940(a point940) point940 {
	return a
}

func init() {
	a := point940{1, 2, 3, 4, 5}
	assert("cdemo940\000", cdemo940(a) == a)
	assert("demo940\000", demo940(a) == a)
}

type point941 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 float32
}

//go:linkname cdemo941 C.demo941
func cdemo941(a point941) point941

func demo941(a point941) point941 {
	return a
}

func init() {
	a := point941{1, 2, 3, 4}
	assert("cdemo941\000", cdemo941(a) == a)
	assert("demo941\000", demo941(a) == a)
}

type point942 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo942 C.demo942
func cdemo942(a point942) point942

func demo942(a point942) point942 {
	return a
}

func init() {
	a := point942{1, 2, 3, 4, 5}
	assert("cdemo942\000", cdemo942(a) == a)
	assert("demo942\000", demo942(a) == a)
}

type point943 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo943 C.demo943
func cdemo943(a point943) point943

func demo943(a point943) point943 {
	return a
}

func init() {
	a := point943{1, 2, 3, 4, 5}
	assert("cdemo943\000", cdemo943(a) == a)
	assert("demo943\000", demo943(a) == a)
}

type point944 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo944 C.demo944
func cdemo944(a point944) point944

func demo944(a point944) point944 {
	return a
}

func init() {
	a := point944{1, 2, 3, 4, 5}
	assert("cdemo944\000", cdemo944(a) == a)
	assert("demo944\000", demo944(a) == a)
}

type point945 struct {
	x0 float32
	x1 int16
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo945 C.demo945
func cdemo945(a point945) point945

func demo945(a point945) point945 {
	return a
}

func init() {
	a := point945{1, 2, 3, 4, 5}
	assert("cdemo945\000", cdemo945(a) == a)
	assert("demo945\000", demo945(a) == a)
}

type point946 struct {
	x0 float32
	x1 int16
	x2 float32
}

//go:linkname cdemo946 C.demo946
func cdemo946(a point946) point946

func demo946(a point946) point946 {
	return a
}

func init() {
	a := point946{1, 2, 3}
	assert("cdemo946\000", cdemo946(a) == a)
	assert("demo946\000", demo946(a) == a)
}

type point947 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int8
}

//go:linkname cdemo947 C.demo947
func cdemo947(a point947) point947

func demo947(a point947) point947 {
	return a
}

func init() {
	a := point947{1, 2, 3, 4}
	assert("cdemo947\000", cdemo947(a) == a)
	assert("demo947\000", demo947(a) == a)
}

type point948 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo948 C.demo948
func cdemo948(a point948) point948

func demo948(a point948) point948 {
	return a
}

func init() {
	a := point948{1, 2, 3, 4, 5}
	assert("cdemo948\000", cdemo948(a) == a)
	assert("demo948\000", demo948(a) == a)
}

type point949 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo949 C.demo949
func cdemo949(a point949) point949

func demo949(a point949) point949 {
	return a
}

func init() {
	a := point949{1, 2, 3, 4, 5}
	assert("cdemo949\000", cdemo949(a) == a)
	assert("demo949\000", demo949(a) == a)
}

type point950 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo950 C.demo950
func cdemo950(a point950) point950

func demo950(a point950) point950 {
	return a
}

func init() {
	a := point950{1, 2, 3, 4, 5}
	assert("cdemo950\000", cdemo950(a) == a)
	assert("demo950\000", demo950(a) == a)
}

type point951 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo951 C.demo951
func cdemo951(a point951) point951

func demo951(a point951) point951 {
	return a
}

func init() {
	a := point951{1, 2, 3, 4, 5}
	assert("cdemo951\000", cdemo951(a) == a)
	assert("demo951\000", demo951(a) == a)
}

type point952 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int16
}

//go:linkname cdemo952 C.demo952
func cdemo952(a point952) point952

func demo952(a point952) point952 {
	return a
}

func init() {
	a := point952{1, 2, 3, 4}
	assert("cdemo952\000", cdemo952(a) == a)
	assert("demo952\000", demo952(a) == a)
}

type point953 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo953 C.demo953
func cdemo953(a point953) point953

func demo953(a point953) point953 {
	return a
}

func init() {
	a := point953{1, 2, 3, 4, 5}
	assert("cdemo953\000", cdemo953(a) == a)
	assert("demo953\000", demo953(a) == a)
}

type point954 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo954 C.demo954
func cdemo954(a point954) point954

func demo954(a point954) point954 {
	return a
}

func init() {
	a := point954{1, 2, 3, 4, 5}
	assert("cdemo954\000", cdemo954(a) == a)
	assert("demo954\000", demo954(a) == a)
}

type point955 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo955 C.demo955
func cdemo955(a point955) point955

func demo955(a point955) point955 {
	return a
}

func init() {
	a := point955{1, 2, 3, 4, 5}
	assert("cdemo955\000", cdemo955(a) == a)
	assert("demo955\000", demo955(a) == a)
}

type point956 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo956 C.demo956
func cdemo956(a point956) point956

func demo956(a point956) point956 {
	return a
}

func init() {
	a := point956{1, 2, 3, 4, 5}
	assert("cdemo956\000", cdemo956(a) == a)
	assert("demo956\000", demo956(a) == a)
}

type point957 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int32
}

//go:linkname cdemo957 C.demo957
func cdemo957(a point957) point957

func demo957(a point957) point957 {
	return a
}

func init() {
	a := point957{1, 2, 3, 4}
	assert("cdemo957\000", cdemo957(a) == a)
	assert("demo957\000", demo957(a) == a)
}

type point958 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo958 C.demo958
func cdemo958(a point958) point958

func demo958(a point958) point958 {
	return a
}

func init() {
	a := point958{1, 2, 3, 4, 5}
	assert("cdemo958\000", cdemo958(a) == a)
	assert("demo958\000", demo958(a) == a)
}

type point959 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo959 C.demo959
func cdemo959(a point959) point959

func demo959(a point959) point959 {
	return a
}

func init() {
	a := point959{1, 2, 3, 4, 5}
	assert("cdemo959\000", cdemo959(a) == a)
	assert("demo959\000", demo959(a) == a)
}

type point960 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo960 C.demo960
func cdemo960(a point960) point960

func demo960(a point960) point960 {
	return a
}

func init() {
	a := point960{1, 2, 3, 4, 5}
	assert("cdemo960\000", cdemo960(a) == a)
	assert("demo960\000", demo960(a) == a)
}

type point961 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo961 C.demo961
func cdemo961(a point961) point961

func demo961(a point961) point961 {
	return a
}

func init() {
	a := point961{1, 2, 3, 4, 5}
	assert("cdemo961\000", cdemo961(a) == a)
	assert("demo961\000", demo961(a) == a)
}

type point962 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 float32
}

//go:linkname cdemo962 C.demo962
func cdemo962(a point962) point962

func demo962(a point962) point962 {
	return a
}

func init() {
	a := point962{1, 2, 3, 4}
	assert("cdemo962\000", cdemo962(a) == a)
	assert("demo962\000", demo962(a) == a)
}

type point963 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo963 C.demo963
func cdemo963(a point963) point963

func demo963(a point963) point963 {
	return a
}

func init() {
	a := point963{1, 2, 3, 4, 5}
	assert("cdemo963\000", cdemo963(a) == a)
	assert("demo963\000", demo963(a) == a)
}

type point964 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo964 C.demo964
func cdemo964(a point964) point964

func demo964(a point964) point964 {
	return a
}

func init() {
	a := point964{1, 2, 3, 4, 5}
	assert("cdemo964\000", cdemo964(a) == a)
	assert("demo964\000", demo964(a) == a)
}

type point965 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo965 C.demo965
func cdemo965(a point965) point965

func demo965(a point965) point965 {
	return a
}

func init() {
	a := point965{1, 2, 3, 4, 5}
	assert("cdemo965\000", cdemo965(a) == a)
	assert("demo965\000", demo965(a) == a)
}

type point966 struct {
	x0 float32
	x1 int16
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo966 C.demo966
func cdemo966(a point966) point966

func demo966(a point966) point966 {
	return a
}

func init() {
	a := point966{1, 2, 3, 4, 5}
	assert("cdemo966\000", cdemo966(a) == a)
	assert("demo966\000", demo966(a) == a)
}

type point967 struct {
	x0 float32
	x1 int32
	x2 int8
}

//go:linkname cdemo967 C.demo967
func cdemo967(a point967) point967

func demo967(a point967) point967 {
	return a
}

func init() {
	a := point967{1, 2, 3}
	assert("cdemo967\000", cdemo967(a) == a)
	assert("demo967\000", demo967(a) == a)
}

type point968 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int8
}

//go:linkname cdemo968 C.demo968
func cdemo968(a point968) point968

func demo968(a point968) point968 {
	return a
}

func init() {
	a := point968{1, 2, 3, 4}
	assert("cdemo968\000", cdemo968(a) == a)
	assert("demo968\000", demo968(a) == a)
}

type point969 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int8
	x4 int8
}

//go:linkname cdemo969 C.demo969
func cdemo969(a point969) point969

func demo969(a point969) point969 {
	return a
}

func init() {
	a := point969{1, 2, 3, 4, 5}
	assert("cdemo969\000", cdemo969(a) == a)
	assert("demo969\000", demo969(a) == a)
}

type point970 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int8
	x4 int16
}

//go:linkname cdemo970 C.demo970
func cdemo970(a point970) point970

func demo970(a point970) point970 {
	return a
}

func init() {
	a := point970{1, 2, 3, 4, 5}
	assert("cdemo970\000", cdemo970(a) == a)
	assert("demo970\000", demo970(a) == a)
}

type point971 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int8
	x4 int32
}

//go:linkname cdemo971 C.demo971
func cdemo971(a point971) point971

func demo971(a point971) point971 {
	return a
}

func init() {
	a := point971{1, 2, 3, 4, 5}
	assert("cdemo971\000", cdemo971(a) == a)
	assert("demo971\000", demo971(a) == a)
}

type point972 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int8
	x4 float32
}

//go:linkname cdemo972 C.demo972
func cdemo972(a point972) point972

func demo972(a point972) point972 {
	return a
}

func init() {
	a := point972{1, 2, 3, 4, 5}
	assert("cdemo972\000", cdemo972(a) == a)
	assert("demo972\000", demo972(a) == a)
}

type point973 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int16
}

//go:linkname cdemo973 C.demo973
func cdemo973(a point973) point973

func demo973(a point973) point973 {
	return a
}

func init() {
	a := point973{1, 2, 3, 4}
	assert("cdemo973\000", cdemo973(a) == a)
	assert("demo973\000", demo973(a) == a)
}

type point974 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int16
	x4 int8
}

//go:linkname cdemo974 C.demo974
func cdemo974(a point974) point974

func demo974(a point974) point974 {
	return a
}

func init() {
	a := point974{1, 2, 3, 4, 5}
	assert("cdemo974\000", cdemo974(a) == a)
	assert("demo974\000", demo974(a) == a)
}

type point975 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int16
	x4 int16
}

//go:linkname cdemo975 C.demo975
func cdemo975(a point975) point975

func demo975(a point975) point975 {
	return a
}

func init() {
	a := point975{1, 2, 3, 4, 5}
	assert("cdemo975\000", cdemo975(a) == a)
	assert("demo975\000", demo975(a) == a)
}

type point976 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int16
	x4 int32
}

//go:linkname cdemo976 C.demo976
func cdemo976(a point976) point976

func demo976(a point976) point976 {
	return a
}

func init() {
	a := point976{1, 2, 3, 4, 5}
	assert("cdemo976\000", cdemo976(a) == a)
	assert("demo976\000", demo976(a) == a)
}

type point977 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int16
	x4 float32
}

//go:linkname cdemo977 C.demo977
func cdemo977(a point977) point977

func demo977(a point977) point977 {
	return a
}

func init() {
	a := point977{1, 2, 3, 4, 5}
	assert("cdemo977\000", cdemo977(a) == a)
	assert("demo977\000", demo977(a) == a)
}

type point978 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int32
}

//go:linkname cdemo978 C.demo978
func cdemo978(a point978) point978

func demo978(a point978) point978 {
	return a
}

func init() {
	a := point978{1, 2, 3, 4}
	assert("cdemo978\000", cdemo978(a) == a)
	assert("demo978\000", demo978(a) == a)
}

type point979 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int32
	x4 int8
}

//go:linkname cdemo979 C.demo979
func cdemo979(a point979) point979

func demo979(a point979) point979 {
	return a
}

func init() {
	a := point979{1, 2, 3, 4, 5}
	assert("cdemo979\000", cdemo979(a) == a)
	assert("demo979\000", demo979(a) == a)
}

type point980 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int32
	x4 int16
}

//go:linkname cdemo980 C.demo980
func cdemo980(a point980) point980

func demo980(a point980) point980 {
	return a
}

func init() {
	a := point980{1, 2, 3, 4, 5}
	assert("cdemo980\000", cdemo980(a) == a)
	assert("demo980\000", demo980(a) == a)
}

type point981 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int32
	x4 int32
}

//go:linkname cdemo981 C.demo981
func cdemo981(a point981) point981

func demo981(a point981) point981 {
	return a
}

func init() {
	a := point981{1, 2, 3, 4, 5}
	assert("cdemo981\000", cdemo981(a) == a)
	assert("demo981\000", demo981(a) == a)
}

type point982 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 int32
	x4 float32
}

//go:linkname cdemo982 C.demo982
func cdemo982(a point982) point982

func demo982(a point982) point982 {
	return a
}

func init() {
	a := point982{1, 2, 3, 4, 5}
	assert("cdemo982\000", cdemo982(a) == a)
	assert("demo982\000", demo982(a) == a)
}

type point983 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 float32
}

//go:linkname cdemo983 C.demo983
func cdemo983(a point983) point983

func demo983(a point983) point983 {
	return a
}

func init() {
	a := point983{1, 2, 3, 4}
	assert("cdemo983\000", cdemo983(a) == a)
	assert("demo983\000", demo983(a) == a)
}

type point984 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 float32
	x4 int8
}

//go:linkname cdemo984 C.demo984
func cdemo984(a point984) point984

func demo984(a point984) point984 {
	return a
}

func init() {
	a := point984{1, 2, 3, 4, 5}
	assert("cdemo984\000", cdemo984(a) == a)
	assert("demo984\000", demo984(a) == a)
}

type point985 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 float32
	x4 int16
}

//go:linkname cdemo985 C.demo985
func cdemo985(a point985) point985

func demo985(a point985) point985 {
	return a
}

func init() {
	a := point985{1, 2, 3, 4, 5}
	assert("cdemo985\000", cdemo985(a) == a)
	assert("demo985\000", demo985(a) == a)
}

type point986 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 float32
	x4 int32
}

//go:linkname cdemo986 C.demo986
func cdemo986(a point986) point986

func demo986(a point986) point986 {
	return a
}

func init() {
	a := point986{1, 2, 3, 4, 5}
	assert("cdemo986\000", cdemo986(a) == a)
	assert("demo986\000", demo986(a) == a)
}

type point987 struct {
	x0 float32
	x1 int32
	x2 int8
	x3 float32
	x4 float32
}

//go:linkname cdemo987 C.demo987
func cdemo987(a point987) point987

func demo987(a point987) point987 {
	return a
}

func init() {
	a := point987{1, 2, 3, 4, 5}
	assert("cdemo987\000", cdemo987(a) == a)
	assert("demo987\000", demo987(a) == a)
}

type point988 struct {
	x0 float32
	x1 int32
	x2 int16
}

//go:linkname cdemo988 C.demo988
func cdemo988(a point988) point988

func demo988(a point988) point988 {
	return a
}

func init() {
	a := point988{1, 2, 3}
	assert("cdemo988\000", cdemo988(a) == a)
	assert("demo988\000", demo988(a) == a)
}

type point989 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int8
}

//go:linkname cdemo989 C.demo989
func cdemo989(a point989) point989

func demo989(a point989) point989 {
	return a
}

func init() {
	a := point989{1, 2, 3, 4}
	assert("cdemo989\000", cdemo989(a) == a)
	assert("demo989\000", demo989(a) == a)
}

type point990 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int8
	x4 int8
}

//go:linkname cdemo990 C.demo990
func cdemo990(a point990) point990

func demo990(a point990) point990 {
	return a
}

func init() {
	a := point990{1, 2, 3, 4, 5}
	assert("cdemo990\000", cdemo990(a) == a)
	assert("demo990\000", demo990(a) == a)
}

type point991 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int8
	x4 int16
}

//go:linkname cdemo991 C.demo991
func cdemo991(a point991) point991

func demo991(a point991) point991 {
	return a
}

func init() {
	a := point991{1, 2, 3, 4, 5}
	assert("cdemo991\000", cdemo991(a) == a)
	assert("demo991\000", demo991(a) == a)
}

type point992 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int8
	x4 int32
}

//go:linkname cdemo992 C.demo992
func cdemo992(a point992) point992

func demo992(a point992) point992 {
	return a
}

func init() {
	a := point992{1, 2, 3, 4, 5}
	assert("cdemo992\000", cdemo992(a) == a)
	assert("demo992\000", demo992(a) == a)
}

type point993 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int8
	x4 float32
}

//go:linkname cdemo993 C.demo993
func cdemo993(a point993) point993

func demo993(a point993) point993 {
	return a
}

func init() {
	a := point993{1, 2, 3, 4, 5}
	assert("cdemo993\000", cdemo993(a) == a)
	assert("demo993\000", demo993(a) == a)
}

type point994 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int16
}

//go:linkname cdemo994 C.demo994
func cdemo994(a point994) point994

func demo994(a point994) point994 {
	return a
}

func init() {
	a := point994{1, 2, 3, 4}
	assert("cdemo994\000", cdemo994(a) == a)
	assert("demo994\000", demo994(a) == a)
}

type point995 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int16
	x4 int8
}

//go:linkname cdemo995 C.demo995
func cdemo995(a point995) point995

func demo995(a point995) point995 {
	return a
}

func init() {
	a := point995{1, 2, 3, 4, 5}
	assert("cdemo995\000", cdemo995(a) == a)
	assert("demo995\000", demo995(a) == a)
}

type point996 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int16
	x4 int16
}

//go:linkname cdemo996 C.demo996
func cdemo996(a point996) point996

func demo996(a point996) point996 {
	return a
}

func init() {
	a := point996{1, 2, 3, 4, 5}
	assert("cdemo996\000", cdemo996(a) == a)
	assert("demo996\000", demo996(a) == a)
}

type point997 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int16
	x4 int32
}

//go:linkname cdemo997 C.demo997
func cdemo997(a point997) point997

func demo997(a point997) point997 {
	return a
}

func init() {
	a := point997{1, 2, 3, 4, 5}
	assert("cdemo997\000", cdemo997(a) == a)
	assert("demo997\000", demo997(a) == a)
}

type point998 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int16
	x4 float32
}

//go:linkname cdemo998 C.demo998
func cdemo998(a point998) point998

func demo998(a point998) point998 {
	return a
}

func init() {
	a := point998{1, 2, 3, 4, 5}
	assert("cdemo998\000", cdemo998(a) == a)
	assert("demo998\000", demo998(a) == a)
}

type point999 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int32
}

//go:linkname cdemo999 C.demo999
func cdemo999(a point999) point999

func demo999(a point999) point999 {
	return a
}

func init() {
	a := point999{1, 2, 3, 4}
	assert("cdemo999\000", cdemo999(a) == a)
	assert("demo999\000", demo999(a) == a)
}

type point1000 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int32
	x4 int8
}

//go:linkname cdemo1000 C.demo1000
func cdemo1000(a point1000) point1000

func demo1000(a point1000) point1000 {
	return a
}

func init() {
	a := point1000{1, 2, 3, 4, 5}
	assert("cdemo1000\000", cdemo1000(a) == a)
	assert("demo1000\000", demo1000(a) == a)
}

type point1001 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int32
	x4 int16
}

//go:linkname cdemo1001 C.demo1001
func cdemo1001(a point1001) point1001

func demo1001(a point1001) point1001 {
	return a
}

func init() {
	a := point1001{1, 2, 3, 4, 5}
	assert("cdemo1001\000", cdemo1001(a) == a)
	assert("demo1001\000", demo1001(a) == a)
}

type point1002 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int32
	x4 int32
}

//go:linkname cdemo1002 C.demo1002
func cdemo1002(a point1002) point1002

func demo1002(a point1002) point1002 {
	return a
}

func init() {
	a := point1002{1, 2, 3, 4, 5}
	assert("cdemo1002\000", cdemo1002(a) == a)
	assert("demo1002\000", demo1002(a) == a)
}

type point1003 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 int32
	x4 float32
}

//go:linkname cdemo1003 C.demo1003
func cdemo1003(a point1003) point1003

func demo1003(a point1003) point1003 {
	return a
}

func init() {
	a := point1003{1, 2, 3, 4, 5}
	assert("cdemo1003\000", cdemo1003(a) == a)
	assert("demo1003\000", demo1003(a) == a)
}

type point1004 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 float32
}

//go:linkname cdemo1004 C.demo1004
func cdemo1004(a point1004) point1004

func demo1004(a point1004) point1004 {
	return a
}

func init() {
	a := point1004{1, 2, 3, 4}
	assert("cdemo1004\000", cdemo1004(a) == a)
	assert("demo1004\000", demo1004(a) == a)
}

type point1005 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 float32
	x4 int8
}

//go:linkname cdemo1005 C.demo1005
func cdemo1005(a point1005) point1005

func demo1005(a point1005) point1005 {
	return a
}

func init() {
	a := point1005{1, 2, 3, 4, 5}
	assert("cdemo1005\000", cdemo1005(a) == a)
	assert("demo1005\000", demo1005(a) == a)
}

type point1006 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 float32
	x4 int16
}

//go:linkname cdemo1006 C.demo1006
func cdemo1006(a point1006) point1006

func demo1006(a point1006) point1006 {
	return a
}

func init() {
	a := point1006{1, 2, 3, 4, 5}
	assert("cdemo1006\000", cdemo1006(a) == a)
	assert("demo1006\000", demo1006(a) == a)
}

type point1007 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 float32
	x4 int32
}

//go:linkname cdemo1007 C.demo1007
func cdemo1007(a point1007) point1007

func demo1007(a point1007) point1007 {
	return a
}

func init() {
	a := point1007{1, 2, 3, 4, 5}
	assert("cdemo1007\000", cdemo1007(a) == a)
	assert("demo1007\000", demo1007(a) == a)
}

type point1008 struct {
	x0 float32
	x1 int32
	x2 int16
	x3 float32
	x4 float32
}

//go:linkname cdemo1008 C.demo1008
func cdemo1008(a point1008) point1008

func demo1008(a point1008) point1008 {
	return a
}

func init() {
	a := point1008{1, 2, 3, 4, 5}
	assert("cdemo1008\000", cdemo1008(a) == a)
	assert("demo1008\000", demo1008(a) == a)
}

type point1009 struct {
	x0 float32
	x1 int32
	x2 int32
}

//go:linkname cdemo1009 C.demo1009
func cdemo1009(a point1009) point1009

func demo1009(a point1009) point1009 {
	return a
}

func init() {
	a := point1009{1, 2, 3}
	assert("cdemo1009\000", cdemo1009(a) == a)
	assert("demo1009\000", demo1009(a) == a)
}

type point1010 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int8
}

//go:linkname cdemo1010 C.demo1010
func cdemo1010(a point1010) point1010

func demo1010(a point1010) point1010 {
	return a
}

func init() {
	a := point1010{1, 2, 3, 4}
	assert("cdemo1010\000", cdemo1010(a) == a)
	assert("demo1010\000", demo1010(a) == a)
}

type point1011 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int8
	x4 int8
}

//go:linkname cdemo1011 C.demo1011
func cdemo1011(a point1011) point1011

func demo1011(a point1011) point1011 {
	return a
}

func init() {
	a := point1011{1, 2, 3, 4, 5}
	assert("cdemo1011\000", cdemo1011(a) == a)
	assert("demo1011\000", demo1011(a) == a)
}

type point1012 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int8
	x4 int16
}

//go:linkname cdemo1012 C.demo1012
func cdemo1012(a point1012) point1012

func demo1012(a point1012) point1012 {
	return a
}

func init() {
	a := point1012{1, 2, 3, 4, 5}
	assert("cdemo1012\000", cdemo1012(a) == a)
	assert("demo1012\000", demo1012(a) == a)
}

type point1013 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int8
	x4 int32
}

//go:linkname cdemo1013 C.demo1013
func cdemo1013(a point1013) point1013

func demo1013(a point1013) point1013 {
	return a
}

func init() {
	a := point1013{1, 2, 3, 4, 5}
	assert("cdemo1013\000", cdemo1013(a) == a)
	assert("demo1013\000", demo1013(a) == a)
}

type point1014 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int8
	x4 float32
}

//go:linkname cdemo1014 C.demo1014
func cdemo1014(a point1014) point1014

func demo1014(a point1014) point1014 {
	return a
}

func init() {
	a := point1014{1, 2, 3, 4, 5}
	assert("cdemo1014\000", cdemo1014(a) == a)
	assert("demo1014\000", demo1014(a) == a)
}

type point1015 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int16
}

//go:linkname cdemo1015 C.demo1015
func cdemo1015(a point1015) point1015

func demo1015(a point1015) point1015 {
	return a
}

func init() {
	a := point1015{1, 2, 3, 4}
	assert("cdemo1015\000", cdemo1015(a) == a)
	assert("demo1015\000", demo1015(a) == a)
}

type point1016 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int16
	x4 int8
}

//go:linkname cdemo1016 C.demo1016
func cdemo1016(a point1016) point1016

func demo1016(a point1016) point1016 {
	return a
}

func init() {
	a := point1016{1, 2, 3, 4, 5}
	assert("cdemo1016\000", cdemo1016(a) == a)
	assert("demo1016\000", demo1016(a) == a)
}

type point1017 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int16
	x4 int16
}

//go:linkname cdemo1017 C.demo1017
func cdemo1017(a point1017) point1017

func demo1017(a point1017) point1017 {
	return a
}

func init() {
	a := point1017{1, 2, 3, 4, 5}
	assert("cdemo1017\000", cdemo1017(a) == a)
	assert("demo1017\000", demo1017(a) == a)
}

type point1018 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int16
	x4 int32
}

//go:linkname cdemo1018 C.demo1018
func cdemo1018(a point1018) point1018

func demo1018(a point1018) point1018 {
	return a
}

func init() {
	a := point1018{1, 2, 3, 4, 5}
	assert("cdemo1018\000", cdemo1018(a) == a)
	assert("demo1018\000", demo1018(a) == a)
}

type point1019 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int16
	x4 float32
}

//go:linkname cdemo1019 C.demo1019
func cdemo1019(a point1019) point1019

func demo1019(a point1019) point1019 {
	return a
}

func init() {
	a := point1019{1, 2, 3, 4, 5}
	assert("cdemo1019\000", cdemo1019(a) == a)
	assert("demo1019\000", demo1019(a) == a)
}

type point1020 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int32
}

//go:linkname cdemo1020 C.demo1020
func cdemo1020(a point1020) point1020

func demo1020(a point1020) point1020 {
	return a
}

func init() {
	a := point1020{1, 2, 3, 4}
	assert("cdemo1020\000", cdemo1020(a) == a)
	assert("demo1020\000", demo1020(a) == a)
}

type point1021 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int32
	x4 int8
}

//go:linkname cdemo1021 C.demo1021
func cdemo1021(a point1021) point1021

func demo1021(a point1021) point1021 {
	return a
}

func init() {
	a := point1021{1, 2, 3, 4, 5}
	assert("cdemo1021\000", cdemo1021(a) == a)
	assert("demo1021\000", demo1021(a) == a)
}

type point1022 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int32
	x4 int16
}

//go:linkname cdemo1022 C.demo1022
func cdemo1022(a point1022) point1022

func demo1022(a point1022) point1022 {
	return a
}

func init() {
	a := point1022{1, 2, 3, 4, 5}
	assert("cdemo1022\000", cdemo1022(a) == a)
	assert("demo1022\000", demo1022(a) == a)
}

type point1023 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int32
	x4 int32
}

//go:linkname cdemo1023 C.demo1023
func cdemo1023(a point1023) point1023

func demo1023(a point1023) point1023 {
	return a
}

func init() {
	a := point1023{1, 2, 3, 4, 5}
	assert("cdemo1023\000", cdemo1023(a) == a)
	assert("demo1023\000", demo1023(a) == a)
}

type point1024 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 int32
	x4 float32
}

//go:linkname cdemo1024 C.demo1024
func cdemo1024(a point1024) point1024

func demo1024(a point1024) point1024 {
	return a
}

func init() {
	a := point1024{1, 2, 3, 4, 5}
	assert("cdemo1024\000", cdemo1024(a) == a)
	assert("demo1024\000", demo1024(a) == a)
}

type point1025 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 float32
}

//go:linkname cdemo1025 C.demo1025
func cdemo1025(a point1025) point1025

func demo1025(a point1025) point1025 {
	return a
}

func init() {
	a := point1025{1, 2, 3, 4}
	assert("cdemo1025\000", cdemo1025(a) == a)
	assert("demo1025\000", demo1025(a) == a)
}

type point1026 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 float32
	x4 int8
}

//go:linkname cdemo1026 C.demo1026
func cdemo1026(a point1026) point1026

func demo1026(a point1026) point1026 {
	return a
}

func init() {
	a := point1026{1, 2, 3, 4, 5}
	assert("cdemo1026\000", cdemo1026(a) == a)
	assert("demo1026\000", demo1026(a) == a)
}

type point1027 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 float32
	x4 int16
}

//go:linkname cdemo1027 C.demo1027
func cdemo1027(a point1027) point1027

func demo1027(a point1027) point1027 {
	return a
}

func init() {
	a := point1027{1, 2, 3, 4, 5}
	assert("cdemo1027\000", cdemo1027(a) == a)
	assert("demo1027\000", demo1027(a) == a)
}

type point1028 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 float32
	x4 int32
}

//go:linkname cdemo1028 C.demo1028
func cdemo1028(a point1028) point1028

func demo1028(a point1028) point1028 {
	return a
}

func init() {
	a := point1028{1, 2, 3, 4, 5}
	assert("cdemo1028\000", cdemo1028(a) == a)
	assert("demo1028\000", demo1028(a) == a)
}

type point1029 struct {
	x0 float32
	x1 int32
	x2 int32
	x3 float32
	x4 float32
}

//go:linkname cdemo1029 C.demo1029
func cdemo1029(a point1029) point1029

func demo1029(a point1029) point1029 {
	return a
}

func init() {
	a := point1029{1, 2, 3, 4, 5}
	assert("cdemo1029\000", cdemo1029(a) == a)
	assert("demo1029\000", demo1029(a) == a)
}

type point1030 struct {
	x0 float32
	x1 int32
	x2 float32
}

//go:linkname cdemo1030 C.demo1030
func cdemo1030(a point1030) point1030

func demo1030(a point1030) point1030 {
	return a
}

func init() {
	a := point1030{1, 2, 3}
	assert("cdemo1030\000", cdemo1030(a) == a)
	assert("demo1030\000", demo1030(a) == a)
}

type point1031 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int8
}

//go:linkname cdemo1031 C.demo1031
func cdemo1031(a point1031) point1031

func demo1031(a point1031) point1031 {
	return a
}

func init() {
	a := point1031{1, 2, 3, 4}
	assert("cdemo1031\000", cdemo1031(a) == a)
	assert("demo1031\000", demo1031(a) == a)
}

type point1032 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int8
	x4 int8
}

//go:linkname cdemo1032 C.demo1032
func cdemo1032(a point1032) point1032

func demo1032(a point1032) point1032 {
	return a
}

func init() {
	a := point1032{1, 2, 3, 4, 5}
	assert("cdemo1032\000", cdemo1032(a) == a)
	assert("demo1032\000", demo1032(a) == a)
}

type point1033 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int8
	x4 int16
}

//go:linkname cdemo1033 C.demo1033
func cdemo1033(a point1033) point1033

func demo1033(a point1033) point1033 {
	return a
}

func init() {
	a := point1033{1, 2, 3, 4, 5}
	assert("cdemo1033\000", cdemo1033(a) == a)
	assert("demo1033\000", demo1033(a) == a)
}

type point1034 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int8
	x4 int32
}

//go:linkname cdemo1034 C.demo1034
func cdemo1034(a point1034) point1034

func demo1034(a point1034) point1034 {
	return a
}

func init() {
	a := point1034{1, 2, 3, 4, 5}
	assert("cdemo1034\000", cdemo1034(a) == a)
	assert("demo1034\000", demo1034(a) == a)
}

type point1035 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int8
	x4 float32
}

//go:linkname cdemo1035 C.demo1035
func cdemo1035(a point1035) point1035

func demo1035(a point1035) point1035 {
	return a
}

func init() {
	a := point1035{1, 2, 3, 4, 5}
	assert("cdemo1035\000", cdemo1035(a) == a)
	assert("demo1035\000", demo1035(a) == a)
}

type point1036 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int16
}

//go:linkname cdemo1036 C.demo1036
func cdemo1036(a point1036) point1036

func demo1036(a point1036) point1036 {
	return a
}

func init() {
	a := point1036{1, 2, 3, 4}
	assert("cdemo1036\000", cdemo1036(a) == a)
	assert("demo1036\000", demo1036(a) == a)
}

type point1037 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int16
	x4 int8
}

//go:linkname cdemo1037 C.demo1037
func cdemo1037(a point1037) point1037

func demo1037(a point1037) point1037 {
	return a
}

func init() {
	a := point1037{1, 2, 3, 4, 5}
	assert("cdemo1037\000", cdemo1037(a) == a)
	assert("demo1037\000", demo1037(a) == a)
}

type point1038 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int16
	x4 int16
}

//go:linkname cdemo1038 C.demo1038
func cdemo1038(a point1038) point1038

func demo1038(a point1038) point1038 {
	return a
}

func init() {
	a := point1038{1, 2, 3, 4, 5}
	assert("cdemo1038\000", cdemo1038(a) == a)
	assert("demo1038\000", demo1038(a) == a)
}

type point1039 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int16
	x4 int32
}

//go:linkname cdemo1039 C.demo1039
func cdemo1039(a point1039) point1039

func demo1039(a point1039) point1039 {
	return a
}

func init() {
	a := point1039{1, 2, 3, 4, 5}
	assert("cdemo1039\000", cdemo1039(a) == a)
	assert("demo1039\000", demo1039(a) == a)
}

type point1040 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int16
	x4 float32
}

//go:linkname cdemo1040 C.demo1040
func cdemo1040(a point1040) point1040

func demo1040(a point1040) point1040 {
	return a
}

func init() {
	a := point1040{1, 2, 3, 4, 5}
	assert("cdemo1040\000", cdemo1040(a) == a)
	assert("demo1040\000", demo1040(a) == a)
}

type point1041 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int32
}

//go:linkname cdemo1041 C.demo1041
func cdemo1041(a point1041) point1041

func demo1041(a point1041) point1041 {
	return a
}

func init() {
	a := point1041{1, 2, 3, 4}
	assert("cdemo1041\000", cdemo1041(a) == a)
	assert("demo1041\000", demo1041(a) == a)
}

type point1042 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int32
	x4 int8
}

//go:linkname cdemo1042 C.demo1042
func cdemo1042(a point1042) point1042

func demo1042(a point1042) point1042 {
	return a
}

func init() {
	a := point1042{1, 2, 3, 4, 5}
	assert("cdemo1042\000", cdemo1042(a) == a)
	assert("demo1042\000", demo1042(a) == a)
}

type point1043 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int32
	x4 int16
}

//go:linkname cdemo1043 C.demo1043
func cdemo1043(a point1043) point1043

func demo1043(a point1043) point1043 {
	return a
}

func init() {
	a := point1043{1, 2, 3, 4, 5}
	assert("cdemo1043\000", cdemo1043(a) == a)
	assert("demo1043\000", demo1043(a) == a)
}

type point1044 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int32
	x4 int32
}

//go:linkname cdemo1044 C.demo1044
func cdemo1044(a point1044) point1044

func demo1044(a point1044) point1044 {
	return a
}

func init() {
	a := point1044{1, 2, 3, 4, 5}
	assert("cdemo1044\000", cdemo1044(a) == a)
	assert("demo1044\000", demo1044(a) == a)
}

type point1045 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 int32
	x4 float32
}

//go:linkname cdemo1045 C.demo1045
func cdemo1045(a point1045) point1045

func demo1045(a point1045) point1045 {
	return a
}

func init() {
	a := point1045{1, 2, 3, 4, 5}
	assert("cdemo1045\000", cdemo1045(a) == a)
	assert("demo1045\000", demo1045(a) == a)
}

type point1046 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 float32
}

//go:linkname cdemo1046 C.demo1046
func cdemo1046(a point1046) point1046

func demo1046(a point1046) point1046 {
	return a
}

func init() {
	a := point1046{1, 2, 3, 4}
	assert("cdemo1046\000", cdemo1046(a) == a)
	assert("demo1046\000", demo1046(a) == a)
}

type point1047 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 float32
	x4 int8
}

//go:linkname cdemo1047 C.demo1047
func cdemo1047(a point1047) point1047

func demo1047(a point1047) point1047 {
	return a
}

func init() {
	a := point1047{1, 2, 3, 4, 5}
	assert("cdemo1047\000", cdemo1047(a) == a)
	assert("demo1047\000", demo1047(a) == a)
}

type point1048 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 float32
	x4 int16
}

//go:linkname cdemo1048 C.demo1048
func cdemo1048(a point1048) point1048

func demo1048(a point1048) point1048 {
	return a
}

func init() {
	a := point1048{1, 2, 3, 4, 5}
	assert("cdemo1048\000", cdemo1048(a) == a)
	assert("demo1048\000", demo1048(a) == a)
}

type point1049 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 float32
	x4 int32
}

//go:linkname cdemo1049 C.demo1049
func cdemo1049(a point1049) point1049

func demo1049(a point1049) point1049 {
	return a
}

func init() {
	a := point1049{1, 2, 3, 4, 5}
	assert("cdemo1049\000", cdemo1049(a) == a)
	assert("demo1049\000", demo1049(a) == a)
}

type point1050 struct {
	x0 float32
	x1 int32
	x2 float32
	x3 float32
	x4 float32
}

//go:linkname cdemo1050 C.demo1050
func cdemo1050(a point1050) point1050

func demo1050(a point1050) point1050 {
	return a
}

func init() {
	a := point1050{1, 2, 3, 4, 5}
	assert("cdemo1050\000", cdemo1050(a) == a)
	assert("demo1050\000", demo1050(a) == a)
}
