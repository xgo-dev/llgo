package wasmtest

import (
	"runtime"
	"testing"
)

// Both the implicit copy and the indirect result exceed the backend's 64 KiB
// stack limit. Their heap storage does not exist in the frontend Go SSA.
type gcPointerAggregate struct {
	pointer *int
	data    [128 << 10]byte
}

//go:noinline
func mutateGCPointerAggregate(source *gcPointerAggregate) gcPointerAggregate {
	source.data[0] = 23
	runtime.Gosched()
	runtime.GC()
	return *source
}

//go:noinline
func deferredGCPointerAggregate(source *gcPointerAggregate) gcPointerAggregate {
	defer func() {
		source.data[0] = 47
		runtime.Gosched()
		runtime.GC()
	}()
	return *source
}

func TestLargeStructGCRoots(t *testing.T) {
	source := new(gcPointerAggregate)
	source.pointer = new(int)
	*source.pointer = 101
	source.data[0], source.data[len(source.data)-1] = 11, 97
	var before, after gcPointerAggregate
	// Calls are evaluated left to right. The first result must survive both
	// its own deferred collection and the second call's mutation/collection.
	before, after = deferredGCPointerAggregate(source), mutateGCPointerAggregate(source)
	if before.data[0] != 11 || after.data[0] != 23 {
		t.Fatalf("assignment snapshots: before=%d after=%d", before.data[0], after.data[0])
	}
	result := deferredGCPointerAggregate(source)
	runtime.Gosched()
	runtime.GC()
	if result.data[0] != 23 || source.data[0] != 47 {
		t.Fatalf("return snapshot: result=%d source=%d", result.data[0], source.data[0])
	}
	for _, value := range []*gcPointerAggregate{&before, &after, &result} {
		if value.data[len(value.data)-1] != 97 || value.pointer == nil || *value.pointer != 101 {
			t.Fatal("collection lost an aggregate or its pointer member")
		}
	}
}
