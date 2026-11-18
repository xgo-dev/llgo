package gotest

import (
	"runtime"
	"testing"
)

type forwardRecover interface{ Catch() }
type forwardRecoverWrap struct{ forwardRecover }
type forwardRecoverLeaf struct{ result *any }

//go:noinline
func (leaf *forwardRecoverLeaf) Catch() { *leaf.result = recover() }

type forwardRecoverValue struct{ result *any }

//go:noinline
func (leaf forwardRecoverValue) Catch() {
	// A direct-interface value needs receiver storage that survives removal
	// of every forwarding frame, including while the collector scans it.
	runtime.GC()
	*leaf.result = recover()
}

func forwardRecoverChain(leaf forwardRecover) forwardRecover {
	for range 64 {
		leaf = &forwardRecoverWrap{leaf}
	}
	return leaf
}

func TestInterfaceForwardDeferredRecover(t *testing.T) {
	for _, value := range []bool{false, true} {
		var result any
		var leaf forwardRecover = &forwardRecoverLeaf{&result}
		if value {
			leaf = forwardRecoverValue{&result}
		}
		chain := forwardRecoverChain(leaf)
		func() {
			defer chain.Catch()
			panic("forwarded")
		}()
		if result != "forwarded" {
			t.Fatalf("value=%v: recover=%v", value, result)
		}
	}
}

func TestInterfaceForwardBoundRecover(t *testing.T) {
	var result any
	bound := forwardRecoverChain(&forwardRecoverLeaf{&result}).Catch
	func() {
		defer bound()
		panic("bound")
	}()
	if result != "bound" {
		t.Fatalf("recover=%v", result)
	}
}

func TestInterfaceForwardOrdinaryCallPreservesRecoverScope(t *testing.T) {
	var indirect, direct any
	chain := forwardRecoverChain(&forwardRecoverLeaf{&indirect})
	func() {
		defer func() {
			chain.Catch()
			direct = recover()
		}()
		panic("outer")
	}()
	if indirect != nil || direct != "outer" {
		t.Fatalf("indirect=%v direct=%v", indirect, direct)
	}
}
