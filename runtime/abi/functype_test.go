package abi

import "testing"

func TestFuncTypeParameterTable(t *testing.T) {
	arg, result := &Type{Kind_: uint8(Int)}, &Type{Kind_: uint8(String)}
	args := []*Type{arg, result}
	ft := FuncType{Types: &args[0], InCount: 1, OutCount: 1}
	in, out := ft.In(), ft.Out()
	if len(in) != 1 || cap(in) != 1 || in[0] != arg || len(out) != 1 || out[0] != result {
		t.Fatal("signature table did not distinguish input and output types")
	}
	ft.InCount, ft.OutCount = 0, 2
	if ft.In() != nil || len(ft.Out()) != 2 || ft.Out()[0] != arg {
		t.Fatal("signature with only results did not use the start of the table")
	}
	ft.InCount, ft.OutCount = 2, 0
	if ft.Out() != nil || len(ft.In()) != 2 || ft.In()[1] != result {
		t.Fatal("signature with only inputs did not use the whole table")
	}
}
