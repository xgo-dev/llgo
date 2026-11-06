//go:build llgo

package test

import (
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	rt "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func TestMapFastStringKeyHeaders(t *testing.T) {
	mt := abi.TypeFor[map[string]int]().MapType()
	// The long keys share their first and last four bytes, forcing the
	// one-bucket lookup to hash when more than one candidate remains.
	keys := []string{
		"", "short", "a\x00b", strings.Repeat("x", 31),
		"head" + strings.Repeat("a", 24) + "tail",
		"head" + strings.Repeat("b", 24) + "tail",
		("!unaligned string")[1:],
	}
	for _, hint := range []int{0, 64} {
		h := rt.MakeMap(mt, hint)
		for i, key := range keys {
			*(*int)(rt.MapAssignFastStr(mt, h, strings.Clone(key))) = i + 1
		}
		for i, key := range keys {
			query := strings.Clone(key)
			if got := *(*int)(rt.MapAccess1FastStr(mt, h, query)); got != i+1 {
				t.Fatalf("hint %d access1 %q = %d, want %d", hint, key, got, i+1)
			}
			p, ok := rt.MapAccess2FastStr(mt, h, query)
			if !ok || *(*int)(p) != i+1 {
				t.Fatalf("hint %d access2 %q returned an incorrect value", hint, key)
			}
			*(*int)(rt.MapAssignFastStr(mt, h, query)) = i + 101
		}
		for _, key := range []string{"missing", "head" + strings.Repeat("c", 24) + "tail"} {
			if p, ok := rt.MapAccess2FastStr(mt, h, key); ok || *(*int)(p) != 0 {
				t.Fatalf("hint %d missing key %q returned a value", hint, key)
			}
		}
		for i, key := range keys {
			if got := *(*int)(rt.MapAccess1FastStr(mt, h, key)); got != i+101 {
				t.Fatalf("hint %d updated key %q = %d, want %d", hint, key, got, i+101)
			}
			rt.MapDeleteFastStr(mt, h, strings.Clone(key))
			if _, ok := rt.MapAccess2FastStr(mt, h, key); ok {
				t.Fatalf("hint %d deleted key %q is still present", hint, key)
			}
		}
	}
}

//go:noinline
func mapFastStringFreshKey(i int) string {
	buf := make([]byte, 96)
	for j := range buf {
		buf[j] = 'x'
	}
	buf[12], buf[13] = byte(i), byte(i>>8)
	return string(buf)
}

func TestMapFastStringKeysSurviveGrowthAndGC(t *testing.T) {
	mt := abi.TypeFor[map[string]int]().MapType()
	h := rt.MakeMap(mt, 0)
	const n = 256
	for i := 0; i < n; i++ {
		// Only the map retains the backing bytes after this call returns.
		*(*int)(rt.MapAssignFastStr(mt, h, mapFastStringFreshKey(i))) = i + 1
	}
	runtime.GC()
	for i := 0; i < n; i++ {
		p, ok := rt.MapAccess2FastStr(mt, h, mapFastStringFreshKey(i))
		if !ok || *(*int)(p) != i+1 {
			t.Fatalf("key %d did not survive map growth and GC", i)
		}
	}
	runtime.KeepAlive(h)
}

func TestMapFastStringHasherInput(t *testing.T) {
	mt := new(abi.MapType)
	*mt = *abi.TypeFor[map[string]int]().MapType()
	hasher := mt.Hasher
	key := strings.Clone("a key long enough to take the hashed lookup path")
	calls := 0
	var seed uintptr
	mt.Hasher = func(p unsafe.Pointer, hashSeed uintptr) uintptr {
		if got := *(*string)(p); got != key {
			t.Fatalf("hasher received %q, want %q", got, key)
		}
		if calls != 0 && seed != hashSeed {
			t.Fatal("hasher received a different map seed")
		}
		seed = hashSeed
		calls++
		return hasher(p, hashSeed)
	}
	h := rt.MakeMap(mt, 64)
	*(*int)(rt.MapAssignFastStr(mt, h, key)) = 42
	if got := *(*int)(rt.MapAccess1FastStr(mt, h, key)); got != 42 {
		t.Fatalf("access1 = %d, want 42", got)
	}
	if p, ok := rt.MapAccess2FastStr(mt, h, key); !ok || *(*int)(p) != 42 {
		t.Fatal("access2 returned an incorrect value")
	}
	rt.MapDeleteFastStr(mt, h, key)
	if calls != 4 {
		t.Fatalf("hasher called %d times, want 4", calls)
	}
}

// LLGo's allocation counters record every runtime allocation, unlike
// ReadMemStats.Mallocs on its BDWGC backend. Keep the profile buffer outside
// the measured calls.
var mapFastStringProfileRecords [32]rt.MemProfileRecord

func mapFastStringAllocationCount() int64 {
	n, ok := rt.MemProfile(mapFastStringProfileRecords[:], true)
	if !ok {
		return -1
	}
	var total int64
	for i := 0; i < n; i++ {
		total += mapFastStringProfileRecords[i].AllocObjects
	}
	return total
}

//go:noinline
func exerciseMapFastStringOperations(mt *abi.MapType, h *rt.Map, n int) bool {
	const key = "existing key"
	for i := 0; i < n; i++ {
		if *(*int)(rt.MapAccess1FastStr(mt, h, key)) != 42 {
			return false
		}
		if p, ok := rt.MapAccess2FastStr(mt, h, key); !ok || *(*int)(p) != 42 {
			return false
		}
		// Updating an existing entry and deleting an absent key need no
		// bucket allocation, so all four paths can be measured together.
		*(*int)(rt.MapAssignFastStr(mt, h, key)) = 42
		rt.MapDeleteFastStr(mt, h, "absent key")
	}
	return true
}

func TestMapFastStringOperationsDoNotAllocateKeyHeaders(t *testing.T) {
	mt := abi.TypeFor[map[string]int]().MapType()
	for _, hint := range []int{0, 64} {
		h := rt.MakeMap(mt, hint)
		*(*int)(rt.MapAssignFastStr(mt, h, "existing key")) = 42
		if !exerciseMapFastStringOperations(mt, h, 1) {
			t.Fatal("string map warmup returned an incorrect value")
		}
		for _, n := range []int{512, 1024} {
			// These counters are process-wide: test timers or another runtime
			// thread may allocate during a measurement. Require a zero-allocation
			// run, allowing a few attempts to exclude that unrelated activity.
			best := int64(1<<63 - 1)
			for attempt := 0; attempt < 4; attempt++ {
				before := mapFastStringAllocationCount()
				ok := exerciseMapFastStringOperations(mt, h, n)
				after := mapFastStringAllocationCount()
				if !ok {
					t.Fatal("string map operations returned an incorrect value")
				}
				if before < 0 || after < 0 {
					t.Fatal("allocation profile buffer is too small")
				}
				if delta := after - before; delta < best {
					best = delta
				}
			}
			if best != 0 {
				t.Fatalf("hint %d: %d rounds of string map operations allocated at least %d objects", hint, n, best)
			}
		}
	}
}
