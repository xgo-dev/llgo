package wasmtest

import (
	"runtime"
	"testing"
)

// Both the implicit copy and the indirect result exceed the backend's 64 KiB
// stack limit. Their heap storage does not exist in the frontend Go SSA.
type gcAggregate [128 << 10]byte

// Keep the non-local return in a nested call: the aggregate owner's root
// frame must survive replay without acquiring stale or cyclic chain links.
//
//go:noinline
func collectGCAggregate() {
	defer func() {
		if recover() != "aggregate" {
			panic("missing aggregate recovery")
		}
		runtime.Gosched()
		runtime.GC()
	}()
	panic("aggregate")
}

//go:noinline
func mutateGCAggregate(source *gcAggregate) gcAggregate {
	source[0] = 23
	collectGCAggregate()
	return *source
}

//go:noinline
func snapshotGCAggregate(source *gcAggregate) gcAggregate {
	value := *source
	source[0] = 47
	collectGCAggregate()
	return value
}

func TestLargeAggregateGCRoots(t *testing.T) {
	source := new(gcAggregate)
	source[0], source[len(source)-1] = 11, 97
	var before, after gcAggregate
	// Calls are evaluated left to right. The first result must survive both
	// its own collection and the second call's mutation/collection.
	before, after = snapshotGCAggregate(source), mutateGCAggregate(source)
	if before[0] != 11 || after[0] != 23 {
		t.Fatalf("assignment snapshots: before=%d after=%d", before[0], after[0])
	}
	result := snapshotGCAggregate(source)
	runtime.Gosched()
	runtime.GC()
	if result[0] != 23 || source[0] != 47 {
		t.Fatalf("return snapshot: result=%d source=%d", result[0], source[0])
	}
	for _, value := range []*gcAggregate{&before, &after, &result} {
		if value[len(value)-1] != 97 {
			t.Fatal("collection lost an aggregate")
		}
	}
}
