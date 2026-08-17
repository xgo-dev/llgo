// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	_ "unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
)

// These functions provide runtime/cgo.Handle without pulling in the gc
// runtime's C callback bridge. LLGo supplies its own C callback trampoline.

//go:linkname cgoNewHandle runtime/cgo.NewHandle
func cgoNewHandle(v any) uintptr {
	cgoHandleState.once.Do(initCgoHandleState)
	cgoHandleState.mu.Lock()
	cgoHandleState.next++
	h := cgoHandleState.next
	if h == 0 {
		cgoHandleState.mu.Unlock()
		panic("runtime/cgo: ran out of handle space")
	}
	cgoHandleState.handles[h] = v
	cgoHandleState.mu.Unlock()
	return h
}

//go:linkname cgoHandleValue runtime/cgo.Handle.Value
func cgoHandleValue(h uintptr) any {
	cgoHandleState.once.Do(initCgoHandleState)
	cgoHandleState.mu.Lock()
	v, ok := cgoHandleState.handles[h]
	cgoHandleState.mu.Unlock()
	if !ok {
		panic("runtime/cgo: misuse of an invalid Handle")
	}
	return v
}

//go:linkname cgoHandleDelete runtime/cgo.Handle.Delete
func cgoHandleDelete(h uintptr) {
	cgoHandleState.once.Do(initCgoHandleState)
	cgoHandleState.mu.Lock()
	_, ok := cgoHandleState.handles[h]
	if ok {
		delete(cgoHandleState.handles, h)
	}
	cgoHandleState.mu.Unlock()
	if !ok {
		panic("runtime/cgo: misuse of an invalid Handle")
	}
}

var cgoHandleState struct {
	once    psync.Once
	mu      psync.Mutex
	next    uintptr
	handles map[uintptr]any
}

func initCgoHandleState() {
	cgoHandleState.mu.Init(nil)
	cgoHandleState.handles = make(map[uintptr]any)
}
