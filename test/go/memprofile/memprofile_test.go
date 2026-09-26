package memprofile

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"runtime/pprof"
	"strings"
	"testing"
)

var tinySink []*int32
var mixedSizeSink [][]byte
var memProfileWarmup []byte

type profiledClosure struct {
	fn func(int) int
}

func makeProfiledClosure(base int) profiledClosure {
	return profiledClosure{fn: func(v int) int { return base + v }}
}

func TestSamplingPreservesReflectCallFrames(t *testing.T) {
	oldRate := runtime.MemProfileRate
	runtime.MemProfileRate = 1
	defer func() {
		runtime.MemProfileRate = oldRate
	}()

	makeFn := reflect.ValueOf(makeProfiledClosure)
	for i := 0; i < 64; i++ {
		out := makeFn.Call([]reflect.Value{reflect.ValueOf(i)})
		closure := out[0].Interface().(profiledClosure)
		if got, want := closure.fn(2), i+2; got != want {
			t.Fatalf("sampled reflect call returned %d, want %d", got, want)
		}
	}
}

func TestRuntimeMemProfileReportsTinyAllocations(t *testing.T) {
	oldRate := runtime.MemProfileRate
	runtime.MemProfileRate = 1
	defer func() {
		runtime.MemProfileRate = oldRate
	}()

	const n = 4096
	tinySink = make([]*int32, 0, n)
	for i := 0; i < n; i++ {
		p := new(int32)
		*p = int32(i)
		tinySink = append(tinySink, p)
	}
	runtime.GC()
	runtime.GC()

	records := readMemProfile(t)
	wantBytes := int64(n * 4)
	for _, r := range records {
		inUseObjects := r.InUseObjects()
		inUseBytes := r.InUseBytes()
		if inUseObjects <= 0 || inUseBytes <= 0 {
			continue
		}
		if got := len(r.Stack()); got > len(r.Stack0) {
			t.Fatalf("MemProfileRecord.Stack length = %d, want <= %d", got, len(r.Stack0))
		}
		if inUseBytes/inUseObjects == 16 && inUseBytes >= wantBytes {
			return
		}
	}
	t.Fatalf("MemProfile did not report tiny allocations totaling at least %d bytes: %#v", wantBytes, records)
}

func TestRuntimeMemProfileSeparatesSizesAtOneStack(t *testing.T) {
	oldRate := runtime.MemProfileRate
	runtime.MemProfileRate = 1
	defer func() {
		runtime.MemProfileRate = oldRate
	}()

	// gc caches the active rate in each mcache. Allocate enough to force the
	// current cache to observe the change before checking the interesting
	// allocation stacks (the Go runtime's own profiler tests do the same).
	for range 1024 {
		memProfileWarmup = make([]byte, 1024)
	}

	mixedSizeSink = make([][]byte, 128)
	mixedSizeProfileAlloc(mixedSizeSink)
	// The gc runtime publishes heap profile samples up to two GC cycles
	// after allocation. Materialize both sizes before reading them.
	runtime.GC()
	runtime.GC()
	sizes := make(map[int64]bool)
	for _, record := range readMemProfile(t) {
		if record.AllocObjects == 0 {
			continue
		}
		frames := runtime.CallersFrames(record.Stack())
		for {
			frame, more := frames.Next()
			if strings.HasSuffix(frame.Function, ".mixedSizeProfileAlloc") {
				sizes[record.AllocBytes/record.AllocObjects] = true
				break
			}
			if !more {
				break
			}
		}
	}
	if !sizes[64] || !sizes[256] {
		t.Fatalf("same-stack memory profile sizes = %v, want 64 and 256", sizes)
	}
}

//go:noinline
func mixedSizeProfileAlloc(dst [][]byte) {
	for i := range dst {
		size := 64
		if i&1 != 0 {
			size = 256
		}
		dst[i] = make([]byte, size)
	}
}

func TestRuntimePprofHeapProfileReportsTinyAllocations(t *testing.T) {
	oldRate := runtime.MemProfileRate
	runtime.MemProfileRate = 1
	defer func() {
		runtime.MemProfileRate = oldRate
	}()

	const n = 4096
	allocateTinyObjects(n)
	runtime.GC()
	runtime.GC()

	var buf bytes.Buffer
	if err := pprof.Lookup("heap").WriteTo(&buf, 1); err != nil {
		t.Fatalf("heap profile WriteTo failed: %v", err)
	}

	var inUseObjects, inUseBytes, allocObjects, allocBytes, rate int64
	if _, err := fmt.Fscanf(bytes.NewReader(buf.Bytes()), "heap profile: %d: %d [%d: %d] @ heap/%d",
		&inUseObjects, &inUseBytes, &allocObjects, &allocBytes, &rate); err != nil {
		t.Fatalf("failed to parse heap profile header: %v\n%s", err, buf.String())
	}
	wantBytes := int64(n * 4)
	if inUseObjects <= 0 || allocObjects <= 0 || inUseBytes < wantBytes || allocBytes < wantBytes {
		t.Fatalf("heap profile totals = %d: %d [%d: %d], want live allocation bytes >= %d\n%s",
			inUseObjects, inUseBytes, allocObjects, allocBytes, wantBytes, buf.String())
	}
}

func readMemProfile(t *testing.T) []runtime.MemProfileRecord {
	t.Helper()
	var records []runtime.MemProfileRecord
	for {
		n, ok := runtime.MemProfile(records, false)
		if ok {
			return records[:n]
		}
		records = make([]runtime.MemProfileRecord, n+10)
	}
}

func allocateTinyObjects(n int) {
	tinySink = make([]*int32, 0, n)
	for i := 0; i < n; i++ {
		p := new(int32)
		*p = int32(i)
		tinySink = append(tinySink, p)
	}
}
