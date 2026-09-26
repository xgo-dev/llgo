//go:build llgo && js && wasm && llgo.wasm.workers

package runtime

import "unsafe"

// adoptWasmWorkerLocalContext transfers any package blocks created during the
// native entry preamble to storage owned by the long-lived worker. The entry
// stack can be discarded when an idle worker unwinds to JavaScript, so its
// stack-local LocalContext must not remain installed in TLS.
func adoptWasmWorkerLocalContext(dst *LocalContext) {
	if dst == nil {
		panic("runtime: nil WebAssembly worker local context")
	}
	current := (*LocalContext)(unsafe.Pointer(currentLocalContext))
	if current == dst {
		return
	}
	if current != nil {
		if dst.blocks != nil {
			panic("runtime: WebAssembly worker local context already populated")
		}
		dst.blocks = current.blocks
		current.blocks = nil
	}
	currentLocalContext = uintptr(unsafe.Pointer(dst))
}

// GoroutineLocalPackage returns the package-local block owned by the current
// logical G. Unlike LocalPackage, key is an identity only: caching the block in
// native TLS would leak one goroutine's GLS values into the next goroutine run
// by the same bounded worker.
//
//go:noinline
func GoroutineLocalPackage(key *uintptr, size, align uintptr) unsafe.Pointer {
	gp := getg()
	if gp == nil || gp.context == nil {
		panic("runtime: goroutine-local variable accessed without a current G")
	}
	if key == nil {
		panic("runtime: nil goroutine-local package key")
	}
	ctx := &gp.context.platform.glsContext
	for data := ctx.blocks; data != nil; data = localBlockHeader(data).next {
		if localBlockHeader(data).cacheSlot == key {
			return data
		}
	}
	if align == 0 || align&(align-1) != 0 {
		panic("runtime: invalid goroutine-local package alignment")
	}
	data := newLocalBlock(key, size, align)
	localBlockHeader(data).next = ctx.blocks
	ctx.blocks = data
	return data
}

// releaseGoroutineLocalBlocks drops one G's ownership links. The block header
// stores the process-global package key only as an identity; unlike a physical
// LocalPackage cache slot, that shared key must never be cleared as Gs exit.
func releaseGoroutineLocalBlocks(ctx *LocalContext) {
	data := ctx.blocks
	ctx.blocks = nil
	for data != nil {
		block := localBlockHeader(data)
		next := block.next
		block.next = nil
		block.cacheSlot = nil
		data = next
	}
}
