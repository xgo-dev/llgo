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
	"os/exec"
	"path/filepath"
	"testing"
)

const finalizerLivenessProbe = `package main

import (
	"os"
	"runtime"
	"time"
	"unsafe"
)

type Box struct {
	p *int
}

type HeapObject [8]int64

type StackSlots [8]*HeapObject

var (
	savedClosure func()
	savedBox     *Box
	expected     uintptr
	evalSlot     *int
)

func loopCase() {
	x := 42
	var box Box
	box.p = &x
	for i := 0; i < 3; i++ {
		if box.p == nil || *box.p != 42 {
			panic("box was cleared while live across loop backedge")
		}
	}
}

func closureCase() {
	x := 42
	box := Box{p: &x}
	savedClosure = func() {
		if box.p == nil || *box.p != 42 {
			panic("captured heap allocation was cleared")
		}
	}
	savedClosure()
}

func globalEscapeCase() {
	x := 42
	savedBox = &Box{p: &x}
	if savedBox.p == nil || *savedBox.p != 42 {
		panic("globally escaped heap allocation was cleared")
	}
}

//go:noinline
func checkDeferred(box *Box) {
	if box.p == nil || *box.p != 42 {
		panic("deferred argument was cleared before RunDefers")
	}
}

func deferCase() {
	x := 42
	box := Box{p: &x}
	defer checkDeferred(&box)
}

//go:noinline
func consumeBox(*Box) {}

//go:noinline
func checkAlias(p *int) {
	if p == nil || *p != 42 {
		panic("independent live alias was cleared")
	}
}

func aliasCase() {
	h := new(int)
	*h = 42
	box := Box{p: h}
	alias := h
	consumeBox(&box)
	checkAlias(alias)
}

func goroutineCase() {
	x := 42
	box := Box{p: &x}
	start := make(chan struct{})
	done := make(chan struct{})
	go func(p *Box) {
		<-start
		if p.p == nil || *p.p != 42 {
			panic("goroutine argument was cleared before use")
		}
		close(done)
	}(&box)
	close(start)
	<-done
}

func uintptrCase() {
	h := new(int)
	box := Box{p: h}
	bits := uintptr(unsafe.Pointer(h))
	expected = bits
	consumeBox(&box)
	if bits != expected {
		panic("live uintptr bits were rewritten by stack scan")
	}
}

func clearEvalSlot() any {
	evalSlot = nil
	return func(*int) {}
}

//go:noinline
func loadEvalSlot() *int {
	return evalSlot
}

func evalOrderCase() {
	p := new(int)
	evalSlot = p
	runtime.SetFinalizer(loadEvalSlot(), clearEvalSlot())
	runtime.KeepAlive(p)
}

//go:noinline
func sameBlockFinalizationCase(writeIndex, readIndex int) {
	finalized := make(chan struct{}, 1)
	var slots StackSlots
	// Keep the dynamic store and load distinct in SSA while the caller supplies
	// the same index, so this exercises clearing the exact dead stack allocation.
	slots[writeIndex] = new(HeapObject)
	runtime.SetFinalizer(slots[readIndex], func(*HeapObject) {
		finalized <- struct{}{}
	})

	for i := 0; i < 100; i++ {
		runtime.GC()
		select {
		case <-finalized:
			return
		default:
		}
		runtime.Gosched()
		time.Sleep(10 * time.Millisecond)
	}
	panic("same-block dead stack slot kept finalizer object alive")
}

func storedAliasCase() {
	x := 42
	var box Box
	var alias **int
	box.p = &x
	aliasSlot := &alias
	*aliasSlot = &box.p
	if **aliasSlot == nil || ***aliasSlot != 42 {
		panic("stack slot was cleared before a stored alias read")
	}
}

func main() {
	if len(os.Args) != 2 {
		panic("missing case name")
	}
	activation := new(int)
	runtime.SetFinalizer(activation, func(*int) {})
	switch os.Args[1] {
	case "loop":
		loopCase()
	case "closure":
		closureCase()
	case "global-escape":
		globalEscapeCase()
	case "defer":
		deferCase()
	case "alias":
		aliasCase()
	case "goroutine":
		goroutineCase()
	case "uintptr":
		uintptrCase()
	case "eval-order":
		evalOrderCase()
	case "same-block-finalization":
		index := len(os.Args[1]) & (len(StackSlots{}) - 1)
		sameBlockFinalizationCase(index, index)
	case "stored-alias":
		storedAliasCase()
	default:
		panic("unknown case")
	}
	runtime.KeepAlive(activation)
}
`

func buildFinalizerLivenessProbe(t *testing.T) (hostBin, llgoBin string) {
	t.Helper()
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(finalizerLivenessProbe), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/finalizerprobe\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	hostBin = filepath.Join(dir, "host-probe")
	runGoCmd(t, dir, "build", "-o", hostBin, ".")

	llgoBin = filepath.Join(dir, "llgo-probe")
	out, err := runLLGoInModule(t, dir, "build", "-o", llgoBin, ".")
	if err != nil {
		t.Fatalf("llgo build failed: %v\n%s", err, out)
	}
	return hostBin, llgoBin
}

func runFinalizerLivenessProbe(t *testing.T, bin, caseName string) {
	t.Helper()
	out, err := exec.Command(bin, caseName).CombinedOutput()
	if err != nil {
		t.Fatalf("%s failed: %v\n%s", filepath.Base(bin), err, out)
	}
}

func TestRuntimeSetFinalizerPreservesLiveValues(t *testing.T) {
	hostBin, llgoBin := buildFinalizerLivenessProbe(t)
	for _, caseName := range []string{
		"loop",
		"closure",
		"global-escape",
		"defer",
		"alias",
		"goroutine",
		"uintptr",
		"eval-order",
		"same-block-finalization",
		"stored-alias",
	} {
		t.Run(caseName, func(t *testing.T) {
			runFinalizerLivenessProbe(t, hostBin, caseName)
			runFinalizerLivenessProbe(t, llgoBin, caseName)
		})
	}
}
