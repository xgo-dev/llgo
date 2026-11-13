//go:build llgo

/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package tls_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/xgo-dev/llgo/runtime/internal/clite/tls"
)

func TestAllocReadWrite(t *testing.T) {
	h := tls.Alloc[int](nil)
	if got := h.Get(); got != 0 {
		t.Fatalf("zero slot = %d, want 0", got)
	}
	h.Set(42)
	if got := h.Get(); got != 42 {
		t.Fatalf("Set/Get mismatch: got %d", got)
	}
	h.Clear()
	if got := h.Get(); got != 0 {
		t.Fatalf("Clear() did not reset slot, got %d", got)
	}
}

func TestAllocThreadLocalIsolation(t *testing.T) {
	h := tls.Alloc[int](nil)
	h.Set(7)

	const want = 99
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if got := h.Get(); got != 0 {
			t.Errorf("new goroutine initial value = %d, want 0", got)
		}
		h.Set(want)
		if got := h.Get(); got != want {
			t.Errorf("goroutine value = %d, want %d", got, want)
		}
	}()
	wg.Wait()

	if got := h.Get(); got != 7 {
		t.Fatalf("main goroutine value changed to %d", got)
	}
}

func TestDestructorRuns(t *testing.T) {
	values := make(chan int, 1)
	h := tls.Alloc[*int](func(p **int) {
		value := 0
		if p != nil && *p != nil {
			value = **p
		}
		values <- value
	})
	go func() {
		val := new(int)
		*val = 123
		h.Set(val)
	}()

	// A goroutine's deferred WaitGroup.Done runs before native thread-local
	// destructors. Wait for the destructor itself, rather than that earlier signal.
	select {
	case value := <-values:
		if value != 123 {
			t.Fatalf("destructor value = %d, want 123", value)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("expected destructor to be invoked")
	}
}

func TestAllocStress(t *testing.T) {
	const sequentialIterations = 200_000

	h := tls.Alloc[int](nil)
	for i := 0; i < sequentialIterations; i++ {
		h.Set(i)
		if got := h.Get(); got != i {
			t.Fatalf("stress iteration %d: got %d want %d", i, got, i)
		}
	}

	var wg sync.WaitGroup
	const (
		goroutines             = 32
		iterationsPerGoroutine = 1_000
	)
	errs := make(chan error, goroutines)
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(offset int) {
			defer wg.Done()
			local := tls.Alloc[int](nil)
			for i := 0; i < iterationsPerGoroutine; i++ {
				v := offset*iterationsPerGoroutine + i
				local.Set(v)
				if got := local.Get(); got != v {
					errs <- fmt.Errorf("goroutine %d iteration %d: got %d want %d", offset, i, got, v)
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}
