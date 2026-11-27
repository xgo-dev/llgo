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

// BDWGC's built-in thread-local allocator (GC_TINY_FREELISTS = 25 granules of
// 16 bytes on 64-bit platforms) covers allocations up to ~384 bytes (minus
// collector end-byte padding); larger objects acquire the global allocation
// lock on every call. Cache per-M batches fetched via GC_malloc_many across
// four size classes (512, 640, 768, 1024 B, keeping worst-case internal
// fragmentation <= 1.33x) so common medium scanned allocations amortize the
// global allocation lock. Because GC_malloc_many returns normal pointer-scanned
// objects linked through their first word, holding the list heads inside m (an
// uncollectable GC root) keeps queued objects reachable across concurrent
// collections without disabling GC.
//
// Also cache small (<= 64 B) uncollectable root blocks per M and in a bounded
// cross-thread pool so frequent goroutine startup records avoid
// GC_malloc_uncollectable / GC_free global lock contention.
const (
	// Leave requests <= 384 B on BDWGC's own lock-free tiny freelists.
	mediumAllocMinBytes = 384
	mediumAllocMaxBytes = 1024
	mediumClass512      = 512
	mediumClass640      = 640
	mediumClass768      = 768
	mediumClass1024     = 1024
	mediumClassCount    = 4
	// Cap the retained freelist chain per class per M so platforms with larger
	// BDWGC heap blocks do not pin unbounded scanned memory per goroutine/M.
	mediumBatchCap  = 8
	rootCacheMaxReq = 64
)

type mediumAllocCache struct {
	normal  [mediumClassCount]unsafe.Pointer
	root64  [4]unsafe.Pointer
	nRoot64 uint8
}

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

// clearMediumAllocCache releases per-M caches on thread teardown. Remaining
// root64 blocks are flushed into root64Pool up to its capacity so exiting
// child goroutines recycle their startup argument blocks back to future callers,
// and only overflow entries are returned to BDWGC via Free.
func clearMediumAllocCache(mp *m) {
	if mp == nil {
		return
	}
	n := mp.allocCache.nRoot64
	if n > 0 {
		ensureRoot64Pool()
		root64Pool.mu.Lock()
		for n > 0 && root64Pool.n < len(root64Pool.ptrs) {
			n--
			p := mp.allocCache.root64[n]
			mp.allocCache.root64[n] = nil
			if p != nil {
				root64Pool.ptrs[root64Pool.n] = p
				root64Pool.n++
			}
		}
		root64Pool.mu.Unlock()
		for i := uint8(0); i < n; i++ {
			if p := mp.allocCache.root64[i]; p != nil {
				bdwgc.Free(p)
			}
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

// freeRootCached caches a block previously returned by AllocRoot.
// Callers must pass only pointers returned by AllocRoot and free each block
// at most once; duplicate frees into the cache are detected and rejected.
func freeRootCached(ptr unsafe.Pointer) bool {
	if ptr == nil {
		return false
	}
	slotSize := atomic.Load(&rootCacheBlockSize)
	if slotSize == 0 || bdwgc.Size(ptr) != slotSize {
		return false
	}
	cache := currentAllocCache()
	if cache != nil {
		for i := uint8(0); i < cache.nRoot64; i++ {
			if cache.root64[i] == ptr {
				fatal("runtime: double FreeRoot")
			}
		}
		if int(cache.nRoot64) < len(cache.root64) {
			c.Memset(ptr, 0, slotSize)
			cache.root64[cache.nRoot64] = ptr
			cache.nRoot64++
			return true
		}
	}
	ensureRoot64Pool()
	c.Memset(ptr, 0, slotSize)
	root64Pool.mu.Lock()
	for i := 0; i < root64Pool.n; i++ {
		if root64Pool.ptrs[i] == ptr {
			root64Pool.mu.Unlock()
			fatal("runtime: double FreeRoot")
		}
	}
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
	if size <= mediumAllocMinBytes || size > mediumAllocMaxBytes {
		return nil
	}
	cache := currentAllocCache()
	if cache == nil {
		return nil
	}
	var idx int
	var classSize uintptr
	switch {
	case size <= mediumClass512:
		idx, classSize = 0, mediumClass512
	case size <= mediumClass640:
		idx, classSize = 1, mediumClass640
	case size <= mediumClass768:
		idx, classSize = 2, mediumClass768
	default:
		idx, classSize = 3, mediumClass1024
	}
	if p := cache.normal[idx]; p != nil {
		cache.normal[idx] = *(*unsafe.Pointer)(p)
		*(*unsafe.Pointer)(p) = nil
		return p
	}
	return refillNormal(&cache.normal[idx], classSize)
}

//go:noinline
func refillNormal(head *unsafe.Pointer, size uintptr) unsafe.Pointer {
	fl := bdwgc.MallocMany(size)
	if fl == nil {
		return nil
	}
	next := *(*unsafe.Pointer)(fl)
	*(*unsafe.Pointer)(fl) = nil
	// Truncate the cached chain to mediumBatchCap so per-M retention stays
	// bounded even on platforms with larger BDWGC heap blocks.
	if next != nil {
		cur := next
		for kept := 1; kept < mediumBatchCap; kept++ {
			n := *(*unsafe.Pointer)(cur)
			if n == nil {
				break
			}
			cur = n
		}
		*(*unsafe.Pointer)(cur) = nil
	}
	*head = next
	return fl
}
