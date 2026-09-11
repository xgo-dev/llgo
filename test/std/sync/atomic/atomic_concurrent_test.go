package atomic_test

import (
	"sync/atomic"
	"testing"
)

func TestConcurrentOperations(t *testing.T) {
	var counter atomic.Int32

	// Test concurrent increments with smaller goroutine count
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 1000; j++ {
				counter.Add(1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	if counter.Load() != 5000 {
		t.Fatalf("Concurrent Add operations: got %d, want 5000", counter.Load())
	}
}

func TestConcurrentCAS(t *testing.T) {
	var counter atomic.Int32
	done := make(chan struct{}, 5)
	for range 5 {
		go func() {
			for range 100 {
				for {
					old := counter.Load()
					if counter.CompareAndSwap(old, old+1) {
						break
					}
				}
			}
			done <- struct{}{}
		}()
	}
	for range 5 {
		<-done
	}
	if got := counter.Load(); got != 500 {
		t.Fatalf("Concurrent CAS operations: got %d, want 500", got)
	}
}

// Benchmark functions
func BenchmarkAddInt32(b *testing.B) {
	var i int32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt32(&i, 1)
		}
	})
}

func BenchmarkLoadInt32(b *testing.B) {
	var i int32 = 42
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.LoadInt32(&i)
		}
	})
}

func BenchmarkCASInt32(b *testing.B) {
	var i int32 = 0
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.CompareAndSwapInt32(&i, 0, 1)
		}
	})
}

func BenchmarkInt32Type(b *testing.B) {
	var i atomic.Int32
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			i.Add(1)
		}
	})
}

func BenchmarkValueType(b *testing.B) {
	var v atomic.Value
	data := struct{ value int }{value: 42}
	v.Store(data)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v.Load()
		}
	})
}
