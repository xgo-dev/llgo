//go:build !baremetal && !(386 || arm || mips || mipsle)

package runtime

// Other targets already align uint64 adequately. Avoid introducing a stdlib
// dependency into their allocator bootstrap, including freestanding targets.
type memProfileAlignment struct{}
