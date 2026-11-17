package runtime

// Sampling and recursion suppression belong to G rather than M: browser
// fibers share an M and can suspend while a different G allocates.
type heapProfileGState struct {
	rate  int
	next  uint64
	rand  uint64
	depth uint32
	pcs   *[32]uintptr
}
