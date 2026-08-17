//go:build llgo

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

package llgoext

import (
	"testing"
	"unsafe"
)

//go:linkname runtimeGetGForTest github.com/xgo-dev/llgo/runtime/internal/runtime.getg
func runtimeGetGForTest() unsafe.Pointer

//go:linkname runtimeGetThreadDeferForGTest github.com/xgo-dev/llgo/runtime/internal/runtime.GetThreadDefer
func runtimeGetThreadDeferForGTest() unsafe.Pointer

//go:linkname runtimeSetThreadDeferForGTest github.com/xgo-dev/llgo/runtime/internal/runtime.SetThreadDefer
func runtimeSetThreadDeferForGTest(unsafe.Pointer)

//go:linkname runtimeClearThreadDeferForGTest github.com/xgo-dev/llgo/runtime/internal/runtime.ClearThreadDefer
func runtimeClearThreadDeferForGTest()

type runtimeGPointerResult struct {
	first  unsafe.Pointer
	second unsafe.Pointer
}

func TestRuntimeGetGIsolation(t *testing.T) {
	mainG := runtimeGetGForTest()
	if mainG == nil || runtimeGetGForTest() != mainG {
		t.Fatalf("main getg is not stable: %p", mainG)
	}

	ready := make(chan unsafe.Pointer, 2)
	start := make(chan struct{})
	results := make(chan runtimeGPointerResult, 2)
	for i := 0; i < 2; i++ {
		go func() {
			first := runtimeGetGForTest()
			ready <- first
			<-start
			results <- runtimeGPointerResult{first: first, second: runtimeGetGForTest()}
		}()
	}
	firstReady := <-ready
	secondReady := <-ready
	close(start)

	first := <-results
	second := <-results
	for i, result := range []runtimeGPointerResult{first, second} {
		if result.first == nil || result.first != result.second {
			t.Fatalf("goroutine %d getg values = (%p, %p), want one stable non-nil pointer", i, result.first, result.second)
		}
		if result.first == mainG {
			t.Fatalf("goroutine %d shared main g %p", i, mainG)
		}
	}
	if firstReady == secondReady {
		t.Fatalf("goroutines shared g %p", firstReady)
	}
}

//go:linkname runtimeGMPForTesting github.com/xgo-dev/llgo/runtime/internal/runtime.GMPForTesting
func runtimeGMPForTesting() (goid, parentGoid uint64, mid int64, pid int32, gstatus, pstatus uint32, linked bool)

const (
	runtimeGRunning = 2
	runtimePRunning = 1
)

type runtimeGMPState struct {
	goid       uint64
	parentGoid uint64
	mid        int64
	pid        int32
	gstatus    uint32
	pstatus    uint32
	linked     bool
}

func currentRuntimeGMPState() runtimeGMPState {
	goid, parentGoid, mid, pid, gstatus, pstatus, linked := runtimeGMPForTesting()
	return runtimeGMPState{
		goid:       goid,
		parentGoid: parentGoid,
		mid:        mid,
		pid:        pid,
		gstatus:    gstatus,
		pstatus:    pstatus,
		linked:     linked,
	}
}

func checkRunningRuntimeGMP(t *testing.T, state runtimeGMPState) {
	t.Helper()
	if state.goid == 0 {
		t.Fatal("current G has no ID")
	}
	if state.mid == 0 {
		t.Fatal("current G has no M")
	}
	if state.pid < 0 {
		t.Fatalf("current G has invalid P id %d", state.pid)
	}
	if state.gstatus != runtimeGRunning {
		t.Fatalf("G status = %d, want running (%d)", state.gstatus, runtimeGRunning)
	}
	if state.pstatus != runtimePRunning {
		t.Fatalf("P status = %d, want running (%d)", state.pstatus, runtimePRunning)
	}
	if !state.linked {
		t.Fatal("G, M, and P links are inconsistent")
	}
}

func TestRuntimeGMPLinks(t *testing.T) {
	parent := currentRuntimeGMPState()
	checkRunningRuntimeGMP(t, parent)

	results := make(chan runtimeGMPState, 2)
	for i := 0; i < cap(results); i++ {
		go func() {
			results <- currentRuntimeGMPState()
		}()
	}

	seenG := map[uint64]bool{parent.goid: true}
	seenM := map[int64]bool{parent.mid: true}
	seenP := map[int32]bool{parent.pid: true}
	for i := 0; i < cap(results); i++ {
		state := <-results
		checkRunningRuntimeGMP(t, state)
		if state.parentGoid != parent.goid {
			t.Fatalf("G %d parent = %d, want %d", state.goid, state.parentGoid, parent.goid)
		}
		if seenG[state.goid] {
			t.Fatalf("duplicate G id %d", state.goid)
		}
		if seenM[state.mid] {
			t.Fatalf("duplicate M id %d", state.mid)
		}
		if seenP[state.pid] {
			t.Fatalf("duplicate P id %d", state.pid)
		}
		seenG[state.goid] = true
		seenM[state.mid] = true
		seenP[state.pid] = true
	}
}

type runtimeDeferProbeResult struct {
	before unsafe.Pointer
	inside unsafe.Pointer
	after  unsafe.Pointer
}

func runtimeDeferProbe(ready chan<- unsafe.Pointer, start <-chan struct{}, results chan<- runtimeDeferProbeResult) {
	result := runtimeDeferProbeResult{before: runtimeGetThreadDeferForGTest()}
	func() {
		defer func() {}()
		result.inside = runtimeGetThreadDeferForGTest()
		ready <- result.inside
		<-start
	}()
	result.after = runtimeGetThreadDeferForGTest()
	results <- result
}

func TestRuntimeDeferHeadIsolation(t *testing.T) {
	mainHead := runtimeGetThreadDeferForGTest()
	ready := make(chan unsafe.Pointer, 2)
	start := make(chan struct{})
	results := make(chan runtimeDeferProbeResult, 2)

	go runtimeDeferProbe(ready, start, results)
	go runtimeDeferProbe(ready, start, results)
	first := <-ready
	second := <-ready
	close(start)

	if first == nil || second == nil {
		t.Fatalf("active defer heads = (%p, %p), want two non-nil heads", first, second)
	}
	if first == second {
		t.Fatalf("concurrent goroutines shared defer head %p", first)
	}
	for i := 0; i < 2; i++ {
		result := <-results
		if result.inside == nil {
			t.Fatal("active defer head is nil")
		}
		if result.after != result.before {
			t.Fatalf("defer head after return = %p, want previous %p", result.after, result.before)
		}
	}
	if got := runtimeGetThreadDeferForGTest(); got != mainHead {
		t.Fatalf("main defer head changed to %p, want %p", got, mainHead)
	}
}

type runtimeDeferAccessorResult struct {
	want    unsafe.Pointer
	got     unsafe.Pointer
	cleared unsafe.Pointer
}

func runtimeDeferAccessorProbe(token unsafe.Pointer, results chan<- runtimeDeferAccessorResult) {
	previous := runtimeGetThreadDeferForGTest()
	runtimeSetThreadDeferForGTest(token)
	got := runtimeGetThreadDeferForGTest()
	runtimeClearThreadDeferForGTest()
	cleared := runtimeGetThreadDeferForGTest()
	runtimeSetThreadDeferForGTest(previous)
	results <- runtimeDeferAccessorResult{want: token, got: got, cleared: cleared}
}

func TestRuntimeDeferAccessors(t *testing.T) {
	results := make(chan runtimeDeferAccessorResult, 2)
	first := new(byte)
	second := new(byte)
	go runtimeDeferAccessorProbe(unsafe.Pointer(first), results)
	go runtimeDeferAccessorProbe(unsafe.Pointer(second), results)

	for i := 0; i < 2; i++ {
		result := <-results
		if result.got != result.want {
			t.Fatalf("defer head = %p, want %p", result.got, result.want)
		}
		if result.cleared != nil {
			t.Fatalf("cleared defer head = %p, want nil", result.cleared)
		}
	}
}

var runtimeGSink unsafe.Pointer

func BenchmarkRuntimeGetG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runtimeGSink = runtimeGetGForTest()
	}
}
