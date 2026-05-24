package gotest

import "testing"

type noInterfaceE struct{}

//go:nointerface
func (noInterfaceE) EBad() int { return 1 }

func (noInterfaceE) EGood() int { return 2 }

type noInterfaceX[T any] struct {
	noInterfaceE
}

//go:nointerface
func (noInterfaceX[T]) XBad() int { return 3 }

func (noInterfaceX[T]) XGood() int { return 4 }

type noInterfaceW struct {
	noInterfaceX[int]
}

type noInterfacePtrBase struct{}

//go:nointerface
func (*noInterfacePtrBase) PBad() int { return 5 }

func (*noInterfacePtrBase) PGood() int { return 6 }

type noInterfacePtrWrap struct {
	noInterfacePtrBase
}

func TestNoInterfaceMethodsDoNotImplementInterfaces(t *testing.T) {
	if got := (noInterfaceE{}).EBad(); got != 1 {
		t.Fatalf("direct nointerface method call = %d, want 1", got)
	}
	if got := (noInterfaceX[int]{}).XBad(); got != 3 {
		t.Fatalf("direct generic nointerface method call = %d, want 3", got)
	}
	ptrWrap := noInterfacePtrWrap{}
	if got := ptrWrap.PBad(); got != 5 {
		t.Fatalf("direct promoted pointer nointerface method call = %d, want 5", got)
	}

	checkNoInterface[noInterfaceE, interface{ EBad() int }, interface{ EGood() int }](t, "E")
	checkNoInterface[noInterfaceX[int], interface{ EBad() int }, interface{ EGood() int }](t, "X.E")
	checkNoInterface[noInterfaceX[int], interface{ XBad() int }, interface{ XGood() int }](t, "X")
	checkNoInterface[noInterfaceW, interface{ EBad() int }, interface{ EGood() int }](t, "W.E")
	checkNoInterface[noInterfaceW, interface{ XBad() int }, interface{ XGood() int }](t, "W.X")
	checkNoInterface[noInterfacePtrWrap, interface{ PBad() int }, interface{ PGood() int }](t, "promoted pointer")
}

func checkNoInterface[T any, Bad any, Good any](t *testing.T, name string) {
	t.Helper()
	v := any(new(T))
	_, badOK := v.(Bad)
	if want := !noInterfaceMethodsFiltered; badOK != want {
		t.Fatalf("%s: nointerface method assertion = %v, want %v", name, badOK, want)
	}
	if _, goodOK := v.(Good); !goodOK {
		t.Fatalf("%s: normal method did not satisfy interface", name)
	}
}
