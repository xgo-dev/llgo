package wasmresume

import (
	"testing"
	"unsafe"
)

type testRootFrame struct {
	Frame
	direct   testLeafFrame
	indirect testLeafFrame
	value    int
}

type testLeafFrame struct {
	Frame
	value int
}

var (
	testRootDescriptor = Descriptor{Resume: resumeTestRoot}
	testAddDescriptor  = Descriptor{Resume: resumeTestAdd}
	testMulDescriptor  = Descriptor{Resume: resumeTestMul}
)

func resumeTestRoot(ctx *Context, raw *Frame) Action {
	frame := (*testRootFrame)(unsafe.Pointer(raw))
	switch frame.PC {
	case 0:
		frame.PC = 1
		frame.direct.value = 4
		ctx.Push(&frame.direct.Frame, &testAddDescriptor)
		return Continue
	case 1:
		if returned := ctx.TakeReturned(); returned != &frame.direct.Frame {
			panic("unexpected direct child frame")
		}
		frame.value = frame.direct.value
		frame.PC = 2
		frame.indirect.value = frame.value
		descriptor := &testAddDescriptor
		if frame.value == 7 {
			descriptor = &testMulDescriptor
		}
		ctx.Push(&frame.indirect.Frame, descriptor)
		return Continue
	case 2:
		if returned := ctx.TakeReturned(); returned != &frame.indirect.Frame {
			panic("unexpected indirect child frame")
		}
		frame.value = frame.indirect.value
		return Return
	default:
		panic("unexpected root resume PC")
	}
}

func resumeTestAdd(_ *Context, raw *Frame) Action {
	frame := (*testLeafFrame)(unsafe.Pointer(raw))
	frame.value += 3
	return Return
}

func resumeTestMul(_ *Context, raw *Frame) Action {
	frame := (*testLeafFrame)(unsafe.Pointer(raw))
	switch frame.PC {
	case 0:
		frame.PC = 1
		return Suspend
	case 1:
		frame.value *= 2
		return Return
	default:
		panic("unexpected leaf resume PC")
	}
}

func TestContextRunDirectAndIndirectCalls(t *testing.T) {
	var (
		ctx   Context
		frame testRootFrame
	)
	ctx.Push(&frame.Frame, &testRootDescriptor)

	if action := ctx.Run(); action != Suspend {
		t.Fatalf("first Run action = %d, want Suspend", action)
	}
	if ctx.Top() != &frame.indirect.Frame {
		t.Fatal("suspended child is not the active frame")
	}
	if frame.indirect.Parent != &frame.Frame {
		t.Fatal("child frame is not linked to its caller")
	}

	if action := ctx.Run(); action != Return {
		t.Fatalf("second Run action = %d, want Return", action)
	}
	if ctx.Top() != nil {
		t.Fatal("completed frame chain remains active")
	}
	if returned := ctx.TakeReturned(); returned != &frame.Frame {
		t.Fatal("completed root frame was not returned to its owner")
	}
	if frame.value != 14 {
		t.Fatalf("result = %d, want 14", frame.value)
	}
}

func TestContextRunEmpty(t *testing.T) {
	var ctx Context
	if action := ctx.Run(); action != Return {
		t.Fatalf("empty Run action = %d, want Return", action)
	}
	if returned := ctx.TakeReturned(); returned != nil {
		t.Fatalf("empty Run returned frame %p", returned)
	}
}

func TestSuspendCurrentRequiresCompilerLowering(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("SuspendCurrent fallback did not panic")
		}
	}()
	SuspendCurrent()
}

func TestContextStart(t *testing.T) {
	descriptor := &Descriptor{}
	root := &Frame{Descriptor: descriptor}
	var context Context
	context.Start(root)
	if context.Top() != root {
		t.Fatalf("Top() = %p, want %p", context.Top(), root)
	}
	for _, frame := range []*Frame{
		nil,
		{Descriptor: descriptor},
		{Parent: root, Descriptor: descriptor},
		{},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("Start(%+v) did not panic", frame)
				}
			}()
			context.Start(frame)
		}()
	}
}

func TestContextPushInitializesHeader(t *testing.T) {
	parent := Frame{}
	child := Frame{Parent: &parent, Descriptor: &testMulDescriptor, PC: 9}
	ctx := Context{top: &parent}
	ctx.Push(&child, &testAddDescriptor)
	if child.Parent != &parent {
		t.Fatal("Push did not link the parent frame")
	}
	if child.Descriptor != &testAddDescriptor {
		t.Fatal("Push did not set the descriptor")
	}
	if child.PC != 0 {
		t.Fatalf("Push PC = %d, want 0", child.PC)
	}
}

func TestDescriptorCarriesFrameLayout(t *testing.T) {
	descriptor := Descriptor{
		Resume:       resumeTestAdd,
		FrameSize:    unsafe.Sizeof(testLeafFrame{}),
		FrameAlign:   unsafe.Alignof(testLeafFrame{}),
		UnwindOffset: unsafe.Sizeof(Frame{}),
		UnwindPC:     3,
	}
	if descriptor.Resume == nil || descriptor.FrameSize != unsafe.Sizeof(testLeafFrame{}) ||
		descriptor.FrameAlign != unsafe.Alignof(testLeafFrame{}) ||
		descriptor.UnwindOffset != unsafe.Sizeof(Frame{}) || descriptor.UnwindPC != 3 {
		t.Fatalf("descriptor = %+v", descriptor)
	}
}

func TestContextPushClearsCompletedChain(t *testing.T) {
	var (
		ctx   Context
		first Frame
		next  Frame
	)
	ctx.Push(&first, &testAddDescriptor)
	if action := ctx.Run(); action != Return {
		t.Fatalf("first Run action = %d, want Return", action)
	}
	ctx.Push(&next, &testAddDescriptor)
	if returned := ctx.TakeReturned(); returned != nil {
		t.Fatalf("new root retained completed frame %p", returned)
	}
}

func TestContextRejectsInvalidAction(t *testing.T) {
	descriptor := Descriptor{Resume: func(*Context, *Frame) Action {
		return Action(255)
	}}
	var (
		ctx   Context
		frame Frame
	)
	ctx.Push(&frame, &descriptor)
	defer func() {
		if recover() == nil {
			t.Fatal("Run accepted an invalid resume action")
		}
	}()
	ctx.Run()
}

func BenchmarkContextDispatch(b *testing.B) {
	descriptor := Descriptor{Resume: func(_ *Context, frame *Frame) Action {
		if frame.PC == 0 {
			frame.PC = 1
			return Continue
		}
		return Return
	}}
	var (
		ctx   Context
		frame Frame
	)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ctx.Push(&frame, &descriptor)
		ctx.Run()
	}
}
