//go:build !wasm && !baremetal && !nogc

package gotest

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

type mediumObj512 struct {
	head *int
	data [58]uint64
	tail *int
}

type mediumObj640 struct {
	head *int
	data [74]uint64
	tail *int
}

type mediumObj768 struct {
	head *int
	data [90]uint64
	tail *int
}

type mediumObj1024 struct {
	head *int
	data [120]uint64
	tail *int
}

func TestMediumAllocZeroInitAfterGC(t *testing.T) {
	// Deterministically verify recycled medium objects across all cached size
	// classes are fully zero-initialized after dirtying and collection.
	const batch = 64
	dirty := func() {
		a := make([]*mediumObj512, batch)
		b := make([]*mediumObj640, batch)
		c := make([]*mediumObj768, batch)
		d := make([]*mediumObj1024, batch)
		for i := 0; i < batch; i++ {
			v := i + 1
			a[i] = &mediumObj512{head: &v, tail: &v}
			for j := range a[i].data {
				a[i].data[j] = 0xa5a5a5a5a5a5a5a5
			}
			b[i] = &mediumObj640{head: &v, tail: &v}
			for j := range b[i].data {
				b[i].data[j] = 0xa5a5a5a5a5a5a5a5
			}
			c[i] = &mediumObj768{head: &v, tail: &v}
			for j := range c[i].data {
				c[i].data[j] = 0xa5a5a5a5a5a5a5a5
			}
			d[i] = &mediumObj1024{head: &v, tail: &v}
			for j := range d[i].data {
				d[i].data[j] = 0xa5a5a5a5a5a5a5a5
			}
		}
		runtime.KeepAlive(a)
		runtime.KeepAlive(b)
		runtime.KeepAlive(c)
		runtime.KeepAlive(d)
	}
	dirty()
	runtime.GC()
	runtime.GC()

	for i := 0; i < batch; i++ {
		a := new(mediumObj512)
		if a.head != nil || a.tail != nil {
			t.Fatalf("mediumObj512[%d] pointer field non-nil", i)
		}
		for j, v := range a.data {
			if v != 0 {
				t.Fatalf("mediumObj512[%d].data[%d] = %#x, want 0", i, j, v)
			}
		}
		b := new(mediumObj640)
		if b.head != nil || b.tail != nil {
			t.Fatalf("mediumObj640[%d] pointer field non-nil", i)
		}
		for j, v := range b.data {
			if v != 0 {
				t.Fatalf("mediumObj640[%d].data[%d] = %#x, want 0", i, j, v)
			}
		}
		c := new(mediumObj768)
		if c.head != nil || c.tail != nil {
			t.Fatalf("mediumObj768[%d] pointer field non-nil", i)
		}
		for j, v := range c.data {
			if v != 0 {
				t.Fatalf("mediumObj768[%d].data[%d] = %#x, want 0", i, j, v)
			}
		}
		d := new(mediumObj1024)
		if d.head != nil || d.tail != nil {
			t.Fatalf("mediumObj1024[%d] pointer field non-nil", i)
		}
		for j, v := range d.data {
			if v != 0 {
				t.Fatalf("mediumObj1024[%d].data[%d] = %#x, want 0", i, j, v)
			}
		}
	}
}

func TestMediumAllocConcurrentGC(t *testing.T) {
	stopGC := make(chan struct{})
	gcDone := make(chan struct{})
	go func() {
		defer close(gcDone)
		for round := 0; round < 64; round++ {
			select {
			case <-stopGC:
				return
			default:
				runtime.GC()
				runtime.Gosched()
				time.Sleep(time.Millisecond)
			}
		}
	}()

	const workers = 8
	const iters = 300
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		workerID := w
		go func() {
			defer wg.Done()
			keep512 := make([]*mediumObj512, 16)
			keep640 := make([]*mediumObj640, 16)
			keep768 := make([]*mediumObj768, 16)
			keep1024 := make([]*mediumObj1024, 16)
			for i := 0; i < iters; i++ {
				v1 := workerID*10000 + i
				v2 := v1 + 1
				v3 := v1 + 2
				v4 := v1 + 3

				a := new(mediumObj512)
				if a.head != nil || a.tail != nil || a.data[0] != 0 || a.data[len(a.data)-1] != 0 {
					t.Errorf("mediumObj512 not zero-initialized")
					return
				}
				a.head, a.tail = &v1, &v2
				a.data[0], a.data[len(a.data)-1] = uint64(v1), uint64(v2)
				keep512[i%len(keep512)] = a

				b := new(mediumObj640)
				if b.head != nil || b.tail != nil || b.data[0] != 0 || b.data[len(b.data)-1] != 0 {
					t.Errorf("mediumObj640 not zero-initialized")
					return
				}
				b.head, b.tail = &v2, &v3
				b.data[0], b.data[len(b.data)-1] = uint64(v2), uint64(v3)
				keep640[i%len(keep640)] = b

				c := new(mediumObj768)
				if c.head != nil || c.tail != nil || c.data[0] != 0 || c.data[len(c.data)-1] != 0 {
					t.Errorf("mediumObj768 not zero-initialized")
					return
				}
				c.head, c.tail = &v3, &v4
				c.data[0], c.data[len(c.data)-1] = uint64(v3), uint64(v4)
				keep768[i%len(keep768)] = c

				d := new(mediumObj1024)
				if d.head != nil || d.tail != nil || d.data[0] != 0 || d.data[len(d.data)-1] != 0 {
					t.Errorf("mediumObj1024 not zero-initialized")
					return
				}
				d.head, d.tail = &v1, &v4
				d.data[0], d.data[len(d.data)-1] = uint64(v1), uint64(v4)
				keep1024[i%len(keep1024)] = d

				if prev := keep512[(i+len(keep512)-1)%len(keep512)]; prev != nil {
					if *prev.head != int(prev.data[0]) || *prev.tail != int(prev.data[len(prev.data)-1]) {
						t.Errorf("mediumObj512 corrupted under concurrent GC")
						return
					}
				}
				if prev := keep640[(i+len(keep640)-1)%len(keep640)]; prev != nil {
					if *prev.head != int(prev.data[0]) || *prev.tail != int(prev.data[len(prev.data)-1]) {
						t.Errorf("mediumObj640 corrupted under concurrent GC")
						return
					}
				}
				if prev := keep768[(i+len(keep768)-1)%len(keep768)]; prev != nil {
					if *prev.head != int(prev.data[0]) || *prev.tail != int(prev.data[len(prev.data)-1]) {
						t.Errorf("mediumObj768 corrupted under concurrent GC")
						return
					}
				}
				if prev := keep1024[(i+len(keep1024)-1)%len(keep1024)]; prev != nil {
					if *prev.head != int(prev.data[0]) || *prev.tail != int(prev.data[len(prev.data)-1]) {
						t.Errorf("mediumObj1024 corrupted under concurrent GC")
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	close(stopGC)
	<-gcDone
}

func TestSmallGoroutineStartArgRootCacheUnderGC(t *testing.T) {
	// Launching short-lived goroutines with pointer-bearing captures exercises
	// cross-thread AllocRoot (caller) / FreeRoot (worker -> clearMediumAllocCache
	// flush -> root64Pool -> caller) caching under GC.
	const waves = 12
	const perWave = 16
	for wave := 0; wave < waves; wave++ {
		done := make(chan int, perWave)
		for i := 0; i < perWave; i++ {
			payload := new(int)
			*payload = wave*100 + i
			go func(p *int, want int) {
				if *p != want {
					done <- -1
					return
				}
				done <- *p
			}(payload, wave*100+i)
		}
		if wave%3 == 0 {
			runtime.GC()
		}
		sum := 0
		wantSum := 0
		for i := 0; i < perWave; i++ {
			v := <-done
			if v < 0 {
				t.Fatalf("wave %d: startup arg pointer corrupted", wave)
			}
			sum += v
			wantSum += wave*100 + i
		}
		if sum != wantSum {
			t.Fatalf("wave %d: sum = %d, want %d", wave, sum, wantSum)
		}
	}
}

func TestSmallGoroutineStartArgBurstRetirementUnderGC(t *testing.T) {
	// Exercise waves larger than the native worker pool idle cap (max 32) so
	// excess recyclable workers retire through finishRetiredM and flush any
	// cached small root startup blocks instead of leaking them.
	const waves = 6
	const burst = 96
	for wave := 0; wave < waves; wave++ {
		ready := make(chan struct{}, burst)
		release := make(chan struct{})
		done := make(chan int, burst)
		for i := 0; i < burst; i++ {
			payload := new(int)
			*payload = wave*1000 + i
			go func(p *int, want int) {
				ready <- struct{}{}
				<-release
				if *p != want {
					done <- -1
					return
				}
				done <- *p
			}(payload, wave*1000+i)
		}
		for i := 0; i < burst; i++ {
			<-ready
		}
		close(release)
		sum := 0
		wantSum := 0
		for i := 0; i < burst; i++ {
			v := <-done
			if v < 0 {
				t.Fatalf("wave %d: startup arg pointer corrupted", wave)
			}
			sum += v
			wantSum += wave*1000 + i
		}
		if sum != wantSum {
			t.Fatalf("wave %d: sum = %d, want %d", wave, sum, wantSum)
		}
		if wave%2 == 0 {
			runtime.GC()
		}
	}
}
