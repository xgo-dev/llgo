//go:build llgo && !baremetal

/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/pthread"
)

// currentG is the scheduler's physical-thread slot for locating the G that is
// running on the current M. It must be available before a goroutine's
// LocalContext is installed, so this is TLS rather than GLS.
//
// The slot deliberately stores a uintptr. The runtimeContext allocation is the
// GC root; making this a pointer-bearing local variable would require the
// LocalContext that getg helps bootstrap.
//
//llgo:tls
var currentG uintptr

// currentGHasLifecycle records whether the current G was installed in the
// pthread destructor sidecar. Runtime-owned M threads leave this false.
//
//llgo:tls
var currentGHasLifecycle bool

// gLifecycleKey is not used to locate the current G. It only retains the
// pthread destructor needed for contexts lazily created on main or foreign
// threads, which have no runtime-owned mexit path.
var gLifecycleKey = newGLifecycleKey()

func newGLifecycleKey() pthread.Key {
	var key pthread.Key
	if ret := key.Create(pthread.KeyDestructor(destroyG)); ret != 0 {
		c.Fprintf(c.Stderr, c.Str("runtime: pthread_key_create failed (errno=%d)\n"), ret)
		panic("runtime: failed to create getg lifecycle key")
	}
	return key
}

func getg() *g {
	if gp := (*g)(unsafe.Pointer(currentG)); gp != nil {
		return gp
	}
	gp := initRuntimeContext(allocRuntimeContext(), nil, _Grunning)
	if ret := setAutoG(gp); ret != 0 {
		destroyG(c.Pointer(unsafe.Pointer(gp)))
		c.Fprintf(c.Stderr, c.Str("runtime: pthread_setspecific failed (errno=%d)\n"), ret)
		panic("runtime: failed to install g")
	}
	return gp
}

func setg(gp *g) {
	if currentGHasLifecycle {
		old := (*g)(unsafe.Pointer(currentG))
		if ret := gLifecycleKey.Set(nil); ret != 0 {
			c.Fprintf(c.Stderr, c.Str("runtime: pthread_setspecific failed (errno=%d)\n"), ret)
			panic("runtime: failed to clear g lifecycle key")
		}
		currentGHasLifecycle = false
		currentG = 0
		// A lazy main/foreign-thread context may be replaced when the thread
		// becomes runtime-owned. Transfer ownership if it is the same G;
		// otherwise release the root no longer covered by the destructor.
		if old != nil && old != gp {
			destroyG(c.Pointer(unsafe.Pointer(old)))
		}
	}
	currentG = uintptr(unsafe.Pointer(gp))
}

func setAutoG(gp *g) c.Int {
	if ret := gLifecycleKey.Set(c.Pointer(unsafe.Pointer(gp))); ret != 0 {
		return ret
	}
	currentG = uintptr(unsafe.Pointer(gp))
	currentGHasLifecycle = true
	return 0
}

func currentGUsesLifecycle() bool {
	return currentGHasLifecycle
}

func destroyG(ptr c.Pointer) {
	gp := (*g)(ptr)
	if gp == nil {
		return
	}
	releaseGAndCheckDeadlock()
	if gp.panic_ != nil {
		c.Free(gp.panic_)
	}
	ctx := gp.context
	if ctx != nil && ctx.root != nil {
		root := ctx.root
		ctx.root = nil
		FreeRoot(root)
	}
}
