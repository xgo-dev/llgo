package gotest

import "testing"

type methodDispatchEmbedded int

func (methodDispatchEmbedded) value() int { return 11 }

func (*methodDispatchEmbedded) pointer() int { return 12 }

type methodDispatchOuter struct {
	*methodDispatchEmbedded
}

type methodDispatchNumber int

func (n methodDispatchNumber) twice() int { return int(n) * 2 }

type methodDispatchPointer struct{}

func (*methodDispatchPointer) marker() int { return 31 }

type methodDispatchValuer interface {
	twice() int
}

type methodDispatchPointerIface interface {
	marker() int
}

func expectMethodDispatchPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected nil pointer dereference panic")
		}
	}()
	f()
}

func TestPromotedValueMethodNilEmbeddedPointerPanics(t *testing.T) {
	var outer methodDispatchOuter

	expectMethodDispatchPanic(t, func() { _ = outer.value() })
	expectMethodDispatchPanic(t, func() { _ = methodDispatchOuter(outer).value() })
	expectMethodDispatchPanic(t, func() { _ = (&outer).value() })
}

func TestPromotedPointerMethodKeepsNilReceiver(t *testing.T) {
	var outer methodDispatchOuter

	if got := outer.pointer(); got != 12 {
		t.Fatalf("outer.pointer() = %d, want 12", got)
	}
	if got := methodDispatchOuter(outer).pointer(); got != 12 {
		t.Fatalf("methodDispatchOuter(outer).pointer() = %d, want 12", got)
	}
	if got := (&outer).pointer(); got != 12 {
		t.Fatalf("(&outer).pointer() = %d, want 12", got)
	}
}

func TestValueReceiverMethodExpressionAndValueNilPointerPanics(t *testing.T) {
	var n *methodDispatchNumber

	expectMethodDispatchPanic(t, func() { _ = (*methodDispatchNumber).twice(n) })
	expectMethodDispatchPanic(t, func() { _ = n.twice })
}

func TestInterfaceCallValueReceiverOnNilPointerPanics(t *testing.T) {
	var n *methodDispatchNumber
	var v methodDispatchValuer = n

	expectMethodDispatchPanic(t, func() { _ = v.twice() })
}

func TestInterfaceCallPointerReceiverKeepsNilReceiver(t *testing.T) {
	var p *methodDispatchPointer
	var v methodDispatchPointerIface = p

	if got := v.marker(); got != 31 {
		t.Fatalf("v.marker() = %d, want 31", got)
	}
}
