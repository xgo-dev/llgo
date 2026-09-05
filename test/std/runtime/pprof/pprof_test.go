package pprof_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"runtime/pprof"
	"testing"
	"time"
)

var cpuProfileSink uint64 = 1

// Keep the sampled function entirely computational. Checking the clock here
// can put samples in Windows time helpers instead of this function; recovering
// this caller from a foreign frame is a separate stack-unwinding contract.
// Carry a nonlinear result between batches so optimization cannot discard the
// work or replace a batch with one affine operation.
//
//go:noinline
func cpuProfileHotLoop(x uint64) uint64 {
	for i := 0; i < 100000; i++ {
		x ^= x >> 12
		x ^= x << 25
		x ^= x >> 27
		x *= 2685821657736338717
	}
	return x
}

func cpuProfileWork(d time.Duration) uint64 {
	start := time.Now()
	x := cpuProfileSink
	// Like Go's pprof CPU hog, require actual work as well as elapsed time.
	// A busy runner must not spend the whole sampling interval descheduled and
	// then finish after only checking the wall clock.
	for batches := 0; batches < 500 || time.Since(start) < d; batches++ {
		x = cpuProfileHotLoop(x)
	}
	cpuProfileSink = x
	return x
}

func TestCPUProfileWorkload(t *testing.T) {
	const want uint64 = 0x2a08f1edef4e04ba
	if got := cpuProfileHotLoop(1); got != want {
		t.Fatalf("CPU profile workload = %#x, want %#x", got, want)
	}
}

func readCPUProfile(t *testing.T, data []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("CPU profile is not valid gzip: %v", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read CPU profile: %v", err)
	}
	if err := zr.Close(); err != nil {
		t.Fatalf("close CPU profile reader: %v", err)
	}
	return raw
}

// cpuProfileContains reports a statistical result, not profile validity. A
// false result must be retried by the caller rather than treated as immediate
// evidence that profiling is broken.
func cpuProfileContains(t *testing.T, data []byte, function string) bool {
	t.Helper()
	return bytes.Contains(readCPUProfile(t, data), []byte(function))
}

// collectCPUProfile retries progressively longer runs because CPU profiling is
// statistical. In particular, a loaded Windows host can produce a valid
// profile without sampling the function under test during a short run. This
// follows the retry strategy used by the Go runtime's own pprof tests while
// keeping the common fast path at 500 ms.
//
// Do not replace this with an acknowledgement that the sampler observed the
// current OS thread. Such a sample may interrupt a native helper, and its frame
// walk need not contain the Go function that this test is trying to verify.
// Only the finished profile can establish that the target function was both
// sampled and symbolized.
func collectCPUProfile(t *testing.T, work func(time.Duration)) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	duration := 500 * time.Millisecond
	for {
		var buf bytes.Buffer
		if err := pprof.StartCPUProfile(&buf); err != nil {
			t.Fatalf("StartCPUProfile failed: %v", err)
		}
		work(duration)
		pprof.StopCPUProfile()
		if cpuProfileContains(t, buf.Bytes(), "cpuProfileHotLoop") {
			return
		}

		duration *= 2
		if time.Until(deadline) < duration {
			t.Fatalf("CPU profile does not contain sampled function %q (last compressed profile=%d bytes)", "cpuProfileHotLoop", buf.Len())
		}
		t.Logf("CPU profile missed cpuProfileHotLoop; retrying with %s duration", duration)
	}
}

func TestStartStopCPUProfile(t *testing.T) {
	collectCPUProfile(t, func(duration time.Duration) {
		_ = cpuProfileWork(duration)
	})
}

func TestStartCPUProfileTwice(t *testing.T) {
	var buf bytes.Buffer
	err := pprof.StartCPUProfile(&buf)
	if err != nil {
		t.Fatalf("StartCPUProfile failed: %v", err)
	}
	defer pprof.StopCPUProfile()

	err = pprof.StartCPUProfile(&buf)
	if err == nil {
		t.Error("StartCPUProfile should fail when already profiling")
	}

	pprof.StopCPUProfile()

	var restarted bytes.Buffer
	if err := pprof.StartCPUProfile(&restarted); err != nil {
		t.Fatalf("StartCPUProfile after stop failed: %v", err)
	}
	_ = cpuProfileWork(300 * time.Millisecond)
	pprof.StopCPUProfile()
	// This test owns the profiler lifecycle contract: a second start must fail,
	// and a profile stopped afterward must be restartable. Requiring this short
	// restarted run to sample a particular function would duplicate the
	// statistical checks above and below, and has caused loaded Windows runners
	// to fail despite producing a complete profile.
	readCPUProfile(t, restarted.Bytes())
}

func TestCPUProfileGoroutine(t *testing.T) {
	collectCPUProfile(t, func(duration time.Duration) {
		done := make(chan uint64, 1)
		go func() {
			done <- cpuProfileWork(duration)
		}()
		if result := <-done; result == 0 {
			t.Fatal("CPU profile workload lost its nonzero state")
		}
	})
}

func TestWriteHeapProfile(t *testing.T) {
	var buf bytes.Buffer
	err := pprof.WriteHeapProfile(&buf)
	if err != nil {
		t.Fatalf("WriteHeapProfile failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Heap profile is empty")
	}
}

func TestLookup(t *testing.T) {
	profiles := []string{
		"goroutine",
		"heap",
		"allocs",
		"threadcreate",
		"block",
		"mutex",
	}

	for _, name := range profiles {
		p := pprof.Lookup(name)
		if p == nil {
			t.Errorf("Lookup(%q) returned nil", name)
		} else if p.Name() != name {
			t.Errorf("Profile.Name() = %q, want %q", p.Name(), name)
		}
	}

	nonExistent := pprof.Lookup("nonexistent")
	if nonExistent != nil {
		t.Error("Lookup for non-existent profile should return nil")
	}
}

func TestProfiles(t *testing.T) {
	profiles := pprof.Profiles()
	if len(profiles) == 0 {
		t.Fatal("Profiles() returned empty slice")
	}

	foundGoroutine := false
	for _, p := range profiles {
		if p.Name() == "goroutine" {
			foundGoroutine = true
			break
		}
	}
	if !foundGoroutine {
		t.Error("goroutine profile not found in Profiles()")
	}
}

func TestNewProfile(t *testing.T) {
	name := "test-profile"
	p := pprof.NewProfile(name)
	if p == nil {
		t.Fatal("NewProfile returned nil")
	}
	if p.Name() != name {
		t.Errorf("Profile.Name() = %q, want %q", p.Name(), name)
	}

	lookup := pprof.Lookup(name)
	if lookup != p {
		t.Error("Lookup did not return the newly created profile")
	}
}

func TestProfileCount(t *testing.T) {
	p := pprof.Lookup("goroutine")
	if p == nil {
		t.Fatal("goroutine profile not found")
	}

	count := p.Count()
	if count <= 0 {
		t.Errorf("Profile.Count() = %d, want > 0", count)
	}
}

func TestProfileWriteTo(t *testing.T) {
	p := pprof.Lookup("goroutine")
	if p == nil {
		t.Fatal("goroutine profile not found")
	}

	var buf bytes.Buffer
	err := p.WriteTo(&buf, 0)
	if err != nil {
		t.Fatalf("Profile.WriteTo failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Profile.WriteTo produced empty output")
	}
}

func TestLabels(t *testing.T) {
	labels := pprof.Labels("key1", "value1", "key2", "value2")
	_ = labels
}

func TestWithLabels(t *testing.T) {
	labels := pprof.Labels("testkey", "testvalue")
	ctx := pprof.WithLabels(context.Background(), labels)

	value, ok := pprof.Label(ctx, "testkey")
	if !ok {
		t.Error("Label not found in context")
	}
	if value != "testvalue" {
		t.Errorf("Label value = %q, want testvalue", value)
	}

	_, ok = pprof.Label(ctx, "nonexistent")
	if ok {
		t.Error("Label should not be found for non-existent key")
	}
}

func TestForLabels(t *testing.T) {
	labels := pprof.Labels("key1", "value1", "key2", "value2")
	ctx := pprof.WithLabels(context.Background(), labels)

	found := make(map[string]string)
	pprof.ForLabels(ctx, func(key, value string) bool {
		found[key] = value
		return true
	})

	if len(found) != 2 {
		t.Errorf("ForLabels found %d labels, want 2", len(found))
	}
	if found["key1"] != "value1" {
		t.Errorf("found[key1] = %q, want value1", found["key1"])
	}
	if found["key2"] != "value2" {
		t.Errorf("found[key2] = %q, want value2", found["key2"])
	}
}

func TestSetGoroutineLabels(t *testing.T) {
	labels := pprof.Labels("goroutine-key", "goroutine-value")
	ctx := pprof.WithLabels(context.Background(), labels)

	pprof.SetGoroutineLabels(ctx)
}

func TestDo(t *testing.T) {
	labels := pprof.Labels("do-key", "do-value")

	executed := false
	pprof.Do(context.Background(), labels, func(ctx context.Context) {
		executed = true

		value, ok := pprof.Label(ctx, "do-key")
		if !ok {
			t.Error("Label not found in Do context")
		}
		if value != "do-value" {
			t.Errorf("Label value = %q, want do-value", value)
		}
	})

	if !executed {
		t.Error("Do function was not executed")
	}
}

func TestProfileRemove(t *testing.T) {
	name := "test-removable-profile"
	p := pprof.NewProfile(name)
	if p == nil {
		t.Fatal("NewProfile returned nil")
	}

	dummy := 0
	p.Add(&dummy, 0)

	count := p.Count()
	if count != 1 {
		t.Errorf("Profile.Count() = %d, want 1", count)
	}

	p.Remove(&dummy)

	count = p.Count()
	if count != 0 {
		t.Errorf("After Remove, Profile.Count() = %d, want 0", count)
	}
}

func TestProfileName(t *testing.T) {
	p := pprof.Lookup("goroutine")
	if p == nil {
		t.Fatal("goroutine profile not found")
	}

	name := p.Name()
	if name != "goroutine" {
		t.Errorf("Profile.Name() = %q, want goroutine", name)
	}
}

func TestLabelSet(t *testing.T) {
	var ls pprof.LabelSet
	_ = ls
}
