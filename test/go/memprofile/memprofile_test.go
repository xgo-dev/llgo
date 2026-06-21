package memprofile

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"testing"
)

var tinySink []*int32

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
