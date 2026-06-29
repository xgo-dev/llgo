/*
 * Copyright (c) 2026 The XGo Authors (xgo.dev). All rights reserved.
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

package gotest

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRuntimeSetFinalizerTinyObjects(t *testing.T) {
	const n = 32
	finalized := make(chan int32, n)
	makeFinalizerTinyObjects(n, finalized)

	done := make([]bool, n)
	count := 0
	deadline := time.After(3 * time.Second)
	for count <= n/2 {
		runGCWithTimeout(t)
		for {
			select {
			case v := <-finalized:
				if v < 0 || v >= n {
					t.Fatalf("finalizer got %d, want [0,%d)", v, n)
				}
				if done[v] {
					t.Fatalf("finalizer got duplicate value %d", v)
				}
				done[v] = true
				count++
				if count > n/2 {
					return
				}
			default:
				goto wait
			}
		}
	wait:
		select {
		case <-deadline:
			t.Fatalf("only %d/%d finalizers ran", count, n)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func makeFinalizerTinyObjects(n int, finalized chan<- int32) {
	for i := 0; i < n; i++ {
		x := new(int32)
		*x = int32(i)
		runtime.SetFinalizer(x, func(p *int32) {
			finalized <- *p
		})
	}
}

func TestRuntimeSetFinalizerCancel(t *testing.T) {
	finalized := make(chan struct{}, 1)
	func() {
		x := new(int)
		runtime.SetFinalizer(x, func(*int) {
			finalized <- struct{}{}
		})
		runtime.SetFinalizer(x, nil)
	}()

	for i := 0; i < 3; i++ {
		runGCWithTimeout(t)
	}
	select {
	case <-finalized:
		t.Fatal("canceled finalizer ran")
	case <-time.After(50 * time.Millisecond):
	}
}

const finalizerStackLivenessProbe = `package main

import (
	"fmt"
	"runtime"
)

type HeapObj [8]int64

type StkObj struct {
	h *HeapObj
}

var n int
var c int = -1
var null StkObj
var sink *HeapObj

func gc() {
	runtime.GC()
	runtime.GC()
	runtime.GC()
	n++
}

func keepAliveCase() {
	c = -1
	n = 0
	f()
	gc()
	if c != 1 {
		panic(fmt.Sprintf("keepalive collection phase = %d, want 1", c))
	}
}

func f() {
	var s StkObj
	s.h = new(HeapObj)
	runtime.SetFinalizer(s.h, func(h *HeapObj) {
		c = n
	})
	g(&s)
	gc()
}

func g(s *StkObj) {
	gc()
	runtime.KeepAlive(s)
	gc()
}

//go:noinline
func use(p *StkObj) {
}

//go:noinline
func ambiguousArgCase(s StkObj, b bool) {
	var p *StkObj
	if b {
		p = &s
	} else {
		p = &null
	}
	use(p)
	gc()
	sink = p.h
	gc()
	sink = nil
	gc()
}

func runAmbiguousArgCase(b bool, want int) {
	var s StkObj
	s.h = new(HeapObj)
	c = -1
	n = 0
	runtime.SetFinalizer(s.h, func(h *HeapObj) {
		c = n
	})
	ambiguousArgCase(s, b)
	if c != want {
		panic(fmt.Sprintf("ambiguous arg b=%v collection phase = %d, want %d", b, c, want))
	}
}

func main() {
	keepAliveCase()
	runAmbiguousArgCase(true, 2)
	runAmbiguousArgCase(false, 0)
}
`

func TestRuntimeSetFinalizerStackObjectLiveness(t *testing.T) {
	dir, err := os.MkdirTemp("", "llgo-finalizer-stack-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(finalizerStackLivenessProbe), 0644); err != nil {
		t.Fatal(err)
	}

	runGoCmd(t, dir, "run", mainFile)

	root := findLLGoRoot(t)
	t.Setenv("LLGO_ROOT", root)
	runGoCmd(t, root, "run", "./cmd/llgo", "run", mainFile)
}

func runGCWithTimeout(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		runtime.GC()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("runtime.GC did not return")
	}
}
