package gotest

import (
	"runtime"
	"sync"
	"testing"
)

type mediumObj512 struct {
	head *int
	data [58]uint64
	tail *int
}

type mediumObj1024 struct {
	head *int
	data [120]uint64
	tail *int
}

func TestMediumAllocConcurrentGC(t *testing.T) {
	stopGC := make(chan struct{})
	gcDone := make(chan struct{})
	go func() {
		defer close(gcDone)
		for {
			select {
			case <-stopGC:
				return
			default:
				runtime.GC()
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
			keep1024 := make([]*mediumObj1024, 16)
			for i := 0; i < iters; i++ {
				a := new(mediumObj512)
				if a.head != nil || a.tail != nil || a.data[0] != 0 || a.data[len(a.data)-1] != 0 {
					t.Errorf("mediumObj512 not zero-initialized")
					return
				}
				v1 := workerID*10000 + i
				v2 := v1 + 1
				a.head = &v1
				a.tail = &v2
				a.data[0] = uint64(v1)
				a.data[len(a.data)-1] = uint64(v2)
				keep512[i%len(keep512)] = a

				b := new(mediumObj1024)
				if b.head != nil || b.tail != nil || b.data[0] != 0 || b.data[len(b.data)-1] != 0 {
					t.Errorf("mediumObj1024 not zero-initialized")
					return
				}
				v3 := v1 + 2
				v4 := v1 + 3
				b.head = &v3
				b.tail = &v4
				b.data[0] = uint64(v3)
				b.data[len(b.data)-1] = uint64(v4)
				keep1024[i%len(keep1024)] = b

				if prev := keep512[(i+len(keep512)-1)%len(keep512)]; prev != nil {
					if *prev.head != int(prev.data[0]) || *prev.tail != int(prev.data[len(prev.data)-1]) {
						t.Errorf("mediumObj512 corrupted under concurrent GC")
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
	// cross-thread AllocRoot (caller) / FreeRoot (worker) caching under GC.
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
