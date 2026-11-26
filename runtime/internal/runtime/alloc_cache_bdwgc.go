//go:build llgo && !nogc && !baremetal && !wasm

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
	"github.com/xgo-dev/llgo/runtime/internal/clite/bdwgc"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// BDWGC's built-in thread-local allocator (GC_TINY_FREELISTS = 25 granules)
// covers small allocations up to ~368 bytes; larger objects acquire the global
// allocation lock on every call. Cache per-M batches fetched via GC_malloc_many
// so common 369..1024 B scanned allocations (structs, map buckets, buffers)
// amortize the global allocation lock. Because GC_malloc_many returns normal
// pointer-scanned objects linked through their first word, holding the list
// head inside m (an uncollectable GC root) keeps every queued object reachable
// across concurrent collections without disabling GC.
//
// Also cache small (<= 64 B) uncollectable root blocks per M and in a small
// cross-thread bridge pool so frequent goroutine startup records avoid
// GC_malloc_uncollectable / GC_free global lock contention.
type mediumAllocCache struct {
	normal512  unsafe.Pointer
	normal1024 unsafe.Pointer
	root64     [4]unsafe.Pointer
	nRoot64    uint8
}

const rootCacheMaxReq = 64

// rootCacheBlockSize records the actual BDWGC block size returned for a
// rootCacheMaxReq-byte MallocUncollectable request (including any collector
// rounding or end-byte padding). Every cached block is zeroed across its full
// actual size on FreeRoot so it never retains stale pointers while cached.
var rootCacheBlockSize uintptr

// root64Pool bridges cross-thread producer/consumer root allocations (for
// example AllocRoot on a caller M and FreeRoot on a worker M) using a
// lightweight chanMutex instead of BDWGC's global GC_allocate_lock.
var root64Pool struct {
	mu     chanMutex
	inited uint32
	n      int
	ptrs   [32]unsafe.Pointer
}

func ensureRoot64Pool() {
	if atomic.Load(&root64Pool.inited) == 2 {
		return
	}
	if _, won := atomic.CompareAndExchange(&root64Pool.inited, uint32(0), uint32(1)); won {
		root64Pool.mu.init()
		atomic.Store(&root64Pool.inited, uint32(2))
		return
	}
	for atomic.Load(&root64Pool.inited) != 2 {
		c.Usleep(1)
	}
}

func currentAllocCache() *mediumAllocCache {
	gp := (*g)(unsafe.Pointer(currentG))
	if gp == nil || gp.m == nil {
		return nil
	}
	return &gp.m.allocCache
}

func clearMediumAllocCache(mp *m) {
	if mp == nil {
		return
	}
	for i := uint8(0); i < mp.allocCache.nRoot64; i++ {
		if p := mp.allocCache.root64[i]; p != nil {
			bdwgc.Free(p)
		}
	}
	mp.allocCache = mediumAllocCache{}
}

func allocRootCached(size uintptr) unsafe.Pointer {
	if size == 0 || size > rootCacheMaxReq {
		return nil
	}
	cache := currentAllocCache()
	if cache != nil && cache.nRoot64 > 0 {
		n := cache.nRoot64 - 1
		cache.nRoot64 = n
		p := cache.root64[n]
		cache.root64[n] = nil
		return p
	}
	if atomic.Load(&rootCacheBlockSize) != 0 {
		ensureRoot64Pool()
		root64Pool.mu.Lock()
		if root64Pool.n > 0 {
			root64Pool.n--
			p := root64Pool.ptrs[root64Pool.n]
			root64Pool.ptrs[root64Pool.n] = nil
			if cache != nil {
				for root64Pool.n > 0 && int(cache.nRoot64) < len(cache.root64)/2 {
					root64Pool.n--
					cache.root64[cache.nRoot64] = root64Pool.ptrs[root64Pool.n]
					root64Pool.ptrs[root64Pool.n] = nil
					cache.nRoot64++
				}
			}
			root64Pool.mu.Unlock()
			return p
		}
		root64Pool.mu.Unlock()
	}
	p := bdwgc.MallocUncollectable(rootCacheMaxReq)
	if p == nil {
		return nil
	}
	if atomic.Load(&rootCacheBlockSize) == 0 {
		if actual := bdwgc.Size(p); actual >= rootCacheMaxReq {
			atomic.Store(&rootCacheBlockSize, actual)
		}
	}
	return p
}

func freeRootCached(ptr unsafe.Pointer) bool {
	if ptr == nil {
		return false
	}
	slotSize := atomic.Load(&rootCacheBlockSize)
	if slotSize == 0 || bdwgc.Size(ptr) != slotSize {
		return false
	}
	cache := currentAllocCache()
	if cache != nil && int(cache.nRoot64) < len(cache.root64) {
		c.Memset(ptr, 0, slotSize)
		cache.root64[cache.nRoot64] = ptr
		cache.nRoot64++
		return true
	}
	ensureRoot64Pool()
	c.Memset(ptr, 0, slotSize)
	root64Pool.mu.Lock()
	if root64Pool.n < len(root64Pool.ptrs) {
		root64Pool.ptrs[root64Pool.n] = ptr
		root64Pool.n++
		if cache != nil {
			for cache.nRoot64 > uint8(len(cache.root64)/2) && root64Pool.n < len(root64Pool.ptrs) {
				cache.nRoot64--
				root64Pool.ptrs[root64Pool.n] = cache.root64[cache.nRoot64]
				cache.root64[cache.nRoot64] = nil
				root64Pool.n++
			}
		}
		root64Pool.mu.Unlock()
		return true
	}
	root64Pool.mu.Unlock()
	return false
}

func allocNormalMedium(size uintptr) unsafe.Pointer {
	if size <= 368 || size > 1024 {
		return nil
	}
	cache := currentAllocCache()
	if cache == nil {
		return nil
	}
	if size <= 512 {
		if p := cache.normal512; p != nil {
			cache.normal512 = *(*unsafe.Pointer)(p)
			*(*unsafe.Pointer)(p) = nil
			return p
		}
		return refillNormal(&cache.normal512, 512)
	}
	if p := cache.normal1024; p != nil {
		cache.normal1024 = *(*unsafe.Pointer)(p)
		*(*unsafe.Pointer)(p) = nil
		return p
	}
	return refillNormal(&cache.normal1024, 1024)
}

//go:noinline
func refillNormal(head *unsafe.Pointer, size uintptr) unsafe.Pointer {
	fl := bdwgc.MallocMany(size)
	if fl == nil {
		return nil
	}
	next := *(*unsafe.Pointer)(fl)
	*(*unsafe.Pointer)(fl) = nil
	*head = next
	return fl
}
