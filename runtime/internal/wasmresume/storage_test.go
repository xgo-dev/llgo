package wasmresume

import (
	"testing"
	"unsafe"
)

type testFrameRoots struct {
	blocks map[unsafe.Pointer][]byte
	allocs int
	frees  int
}

type testUnwindFrame struct {
	Frame
	deferFrame unsafe.Pointer
}

func (r *testFrameRoots) allocate(size uintptr) unsafe.Pointer {
	block := make([]byte, size)
	if len(block) == 0 {
		return nil
	}
	ptr := unsafe.Pointer(&block[0])
	if r.blocks == nil {
		r.blocks = make(map[unsafe.Pointer][]byte)
	}
	r.blocks[ptr] = block
	r.allocs++
	return ptr
}

func (r *testFrameRoots) release(ptr unsafe.Pointer) {
	if _, ok := r.blocks[ptr]; !ok {
		panic("released unknown root block")
	}
	delete(r.blocks, ptr)
	r.frees++
}

func TestFrameStorageAlignsAndReusesFrames(t *testing.T) {
	var (
		storage frameStorage
		roots   testFrameRoots
	)
	first := storage.allocate(31, 8, roots.allocate)
	second := storage.allocate(64, 64, roots.allocate)
	if first == nil || second == nil {
		t.Fatal("frame allocation failed")
	}
	if uintptr(first)%8 != 0 || uintptr(second)%64 != 0 {
		t.Fatalf("unaligned frames: first=%p second=%p", first, second)
	}
	if roots.allocs != 1 {
		t.Fatalf("root block allocations = %d, want 1", roots.allocs)
	}

	storage.releaseFrame(second, 64, roots.release)
	reused := storage.allocate(64, 64, roots.allocate)
	if reused != second {
		t.Fatalf("frame was not reused: got %p, want %p", reused, second)
	}
	storage.releaseFrame(reused, 64, roots.release)
	storage.releaseFrame(first, 31, roots.release)
	storage.close(roots.release)
	if roots.frees != 1 || len(roots.blocks) != 0 {
		t.Fatalf("released roots = %d, remaining = %d", roots.frees, len(roots.blocks))
	}
}

func TestFrameStorageAddsAndReleasesSegments(t *testing.T) {
	var (
		storage frameStorage
		roots   testFrameRoots
	)
	first := storage.allocate(defaultFrameBlockSize, 16, roots.allocate)
	second := storage.allocate(128, 16, roots.allocate)
	if first == nil || second == nil || roots.allocs != 2 {
		t.Fatalf("allocations = %d, first=%p second=%p", roots.allocs, first, second)
	}
	storage.releaseFrame(second, 128, roots.release)
	if roots.frees != 1 {
		t.Fatalf("released child segments = %d, want 1", roots.frees)
	}
	storage.releaseFrame(first, defaultFrameBlockSize, roots.release)
	storage.close(roots.release)
	if roots.frees != 2 || len(roots.blocks) != 0 {
		t.Fatalf("released roots = %d, remaining = %d", roots.frees, len(roots.blocks))
	}
}

func TestContextOwnsGeneratedFrameStorage(t *testing.T) {
	var (
		ctx   Context
		roots testFrameRoots
	)
	size := unsafe.Sizeof(testLeafFrame{})
	raw := ctx.AllocateFrame(size, unsafe.Alignof(testLeafFrame{}), roots.allocate)
	if raw == nil {
		t.Fatal("Context.AllocateFrame failed")
	}
	frame := (*testLeafFrame)(raw)
	frame.Descriptor = &Descriptor{FrameSize: size}
	ctx.ReleaseFrame(&frame.Frame, roots.release)
	ctx.Close(roots.release)
	if roots.allocs != 1 || roots.frees != 1 {
		t.Fatalf("root lifecycle = %d allocs, %d frees", roots.allocs, roots.frees)
	}
}

func TestContextReleaseFrameDiscardsDynamicStorage(t *testing.T) {
	var (
		ctx   Context
		roots testFrameRoots
	)
	const frameSize = uintptr(32)
	raw := ctx.AllocateFrame(frameSize, 8, roots.allocate)
	frame := (*Frame)(raw)
	frame.Descriptor = &Descriptor{FrameSize: frameSize}
	if ctx.AllocateFrame(64, 16, roots.allocate) == nil ||
		ctx.AllocateFrame(defaultFrameBlockSize, 16, roots.allocate) == nil {
		t.Fatal("dynamic frame storage allocation failed")
	}

	ctx.ReleaseFrame(frame, roots.release)
	reused := ctx.AllocateFrame(frameSize, 8, roots.allocate)
	if reused != raw {
		t.Fatalf("frame storage was not rewound: got %p, want %p", reused, raw)
	}
	ctx.Close(roots.release)
	if roots.allocs != roots.frees || len(roots.blocks) != 0 {
		t.Fatalf("root lifecycle = %d allocs, %d frees, %d remaining",
			roots.allocs, roots.frees, len(roots.blocks))
	}
}

func TestContextUnwindReclaimsChildrenAndRedirectsOwner(t *testing.T) {
	var (
		ctx   Context
		roots testFrameRoots
		token byte
	)
	size := unsafe.Sizeof(testUnwindFrame{})
	align := unsafe.Alignof(testUnwindFrame{})
	owner := (*testUnwindFrame)(ctx.AllocateFrame(size, align, roots.allocate))
	child := (*testUnwindFrame)(ctx.AllocateFrame(size, align, roots.allocate))
	ownerDescriptor := &Descriptor{
		FrameSize:    size,
		UnwindOffset: unsafe.Offsetof(testUnwindFrame{}.deferFrame),
		UnwindPC:     7,
	}
	childDescriptor := &Descriptor{FrameSize: size}
	owner.Descriptor = ownerDescriptor
	owner.deferFrame = unsafe.Pointer(&token)
	child.Parent = &owner.Frame
	child.Descriptor = childDescriptor
	ctx.top = &child.Frame

	if !ctx.Unwind(unsafe.Pointer(&token), roots.release) {
		t.Fatal("Context.Unwind did not find the defer owner")
	}
	if ctx.top != &owner.Frame || owner.PC != ownerDescriptor.UnwindPC {
		t.Fatalf("unwind result: top=%p PC=%d", ctx.top, owner.PC)
	}
	reused := ctx.AllocateFrame(size, align, roots.allocate)
	if reused != unsafe.Pointer(child) {
		t.Fatalf("discarded child storage was not reused: got %p, want %p", reused, child)
	}
	ctx.Close(roots.release)
}

func TestContextUnwindRejectsMissingOwner(t *testing.T) {
	var ctx Context
	if ctx.Unwind(nil, nil) || ctx.Unwind(unsafe.Pointer(new(byte)), nil) {
		t.Fatal("Context.Unwind accepted a missing defer owner")
	}
}

func TestContextUnwindIgnoresIncompleteDescriptor(t *testing.T) {
	var (
		ctx   Context
		token byte
		frame testUnwindFrame
	)
	frame.deferFrame = unsafe.Pointer(&token)
	frame.Descriptor = &Descriptor{
		FrameSize:    unsafe.Sizeof(frame),
		UnwindOffset: unsafe.Offsetof(frame.deferFrame),
	}
	ctx.top = &frame.Frame
	if ctx.Unwind(unsafe.Pointer(&token), nil) {
		t.Fatal("Context.Unwind accepted a descriptor without an unwind PC")
	}
}

func TestContextKeepsGeneratedABIPrefix(t *testing.T) {
	if got, want := unsafe.Offsetof(Context{}.storage), 2*unsafe.Sizeof(uintptr(0)); got != want {
		t.Fatalf("Context storage offset = %d, want %d", got, want)
	}
}

func TestFrameStorageRejectsInvalidOperations(t *testing.T) {
	var (
		storage frameStorage
		roots   testFrameRoots
	)
	if storage.allocate(0, 8, roots.allocate) != nil ||
		storage.allocate(8, 3, roots.allocate) != nil ||
		storage.allocate(^uintptr(0), 8, roots.allocate) != nil ||
		storage.allocate(8, 8, nil) != nil {
		t.Fatal("invalid allocation was accepted")
	}
	if storage.allocate(8, 8, func(uintptr) unsafe.Pointer { return nil }) != nil {
		t.Fatal("failed root allocation returned a frame")
	}

	storage.close(roots.release)
}

func TestFrameStorageRejectsInvalidReleaseState(t *testing.T) {
	assertPanic := func(name string, operation func()) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("operation did not panic")
				}
			}()
			operation()
		})
	}

	assertPanic("empty", func() {
		var storage frameStorage
		storage.releaseFrame(unsafe.Pointer(new(byte)), 1, nil)
	})

	var (
		storage frameStorage
		roots   testFrameRoots
	)
	frame := storage.allocate(8, 8, roots.allocate)
	assertPanic("nil frame", func() {
		storage.releaseFrame(nil, 8, roots.release)
	})
	assertPanic("zero size", func() {
		storage.releaseFrame(frame, 0, roots.release)
	})
	assertPanic("foreign frame", func() {
		storage.releaseFrame(unsafe.Pointer(new(byte)), 1, roots.release)
	})
	assertPanic("invalid header", func() {
		header := unsafe.Pointer(uintptr(frame) - unsafe.Sizeof(uintptr(0)))
		*(*uintptr)(header) = 0
		storage.releaseFrame(frame, 8, roots.release)
	})
	storage.close(roots.release)

	var noRelease frameStorage
	noRelease.allocate(8, 8, roots.allocate)
	assertPanic("close without reclaimer", func() {
		noRelease.close(nil)
	})
	noRelease.close(roots.release)
}

func TestContextRejectsFrameWithoutDescriptor(t *testing.T) {
	var ctx Context
	defer func() {
		if recover() == nil {
			t.Fatal("ReleaseFrame accepted an untyped frame")
		}
	}()
	ctx.ReleaseFrame(&Frame{}, nil)
}

func BenchmarkFrameStorageHotAllocateRelease(b *testing.B) {
	var (
		storage frameStorage
		roots   testFrameRoots
	)
	frame := storage.allocate(64, 16, roots.allocate)
	storage.releaseFrame(frame, 64, roots.release)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame = storage.allocate(64, 16, roots.allocate)
		storage.releaseFrame(frame, 64, roots.release)
	}
}
