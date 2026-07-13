//go:build llgo && !baremetal && !nogc

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

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/bdwgc"
	"github.com/goplus/llgo/runtime/internal/clite/pthread"
)

type localRoot struct {
	start c.Pointer
	end   c.Pointer
	next  *localRoot
}

var (
	localRootKey      pthread.Key
	localRootKeyReady bool
)

func init() {
	ensureLocalRootKey()
}

func ensureLocalRootKey() {
	if localRootKeyReady {
		return
	}
	localRootKey = newLocalRootKey()
	localRootKeyReady = true
}

func newLocalRootKey() pthread.Key {
	var key pthread.Key
	ret := key.Create(pthread.KeyDestructor(destroyLocalRoots))
	if ret != 0 {
		c.Fprintf(c.Stderr, c.Str("runtime: pthread_key_create for local roots failed (errno=%d)\n"), ret)
		c.Exit(2)
	}
	return key
}

// RegisterLocalRoot keeps pointers stored in a native TLS range visible to
// BDWGC until the current thread exits.
func RegisterLocalRoot(start unsafe.Pointer, size uintptr) {
	if size == 0 {
		return
	}
	ensureLocalRootKey()
	addr := uintptr(start)
	if addr > ^uintptr(0)-size {
		panic("runtime: local root range overflow")
	}
	root := (*localRoot)(c.Calloc(1, unsafe.Sizeof(localRoot{})))
	if root == nil {
		panic("runtime: failed to allocate local root")
	}
	root.start = c.Pointer(start)
	root.end = c.Pointer(unsafe.Pointer(addr + size))
	root.next = (*localRoot)(unsafe.Pointer(localRootKey.Get()))
	bdwgc.AddRoots(root.start, root.end)
	if ret := localRootKey.Set(c.Pointer(unsafe.Pointer(root))); ret != 0 {
		bdwgc.RemoveRoots(root.start, root.end)
		c.Free(unsafe.Pointer(root))
		c.Fprintf(c.Stderr, c.Str("runtime: pthread_setspecific for local roots failed (errno=%d)\n"), ret)
		c.Exit(2)
	}
}

func destroyLocalRoots(ptr c.Pointer) {
	for root := (*localRoot)(unsafe.Pointer(ptr)); root != nil; {
		next := root.next
		bdwgc.RemoveRoots(root.start, root.end)
		*root = localRoot{}
		c.Free(unsafe.Pointer(root))
		root = next
	}
}
