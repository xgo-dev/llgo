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
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/xgo-dev/llgo/test/llgoext/testdata/localitybench"
	"github.com/xgo-dev/llgo/test/llgoext/testdata/localityfailure"
	"github.com/xgo-dev/llgo/test/llgoext/testdata/localityscope"
)

var initializerSequence int

func nextLocalValue(base int) int {
	initializerSequence++
	return base + initializerSequence
}

//llgo:tls
var tlsCounter int

//llgo:gls
var glsCounter int

//llgo:tls
var initializedTLS = nextLocalValue(100)

//llgo:gls
var initializedGLS = nextLocalValue(200)

type localitySnapshot struct {
	tlsCounter     int
	glsCounter     int
	initializedTLS int
	initializedGLS int
}

func snapshotLocality() localitySnapshot {
	return localitySnapshot{
		tlsCounter:     tlsCounter,
		glsCounter:     glsCounter,
		initializedTLS: initializedTLS,
		initializedGLS: initializedGLS,
	}
}

func TestTLSAndGLSIsolation(t *testing.T) {
	parentInitial := snapshotLocality()
	if parentInitial.initializedTLS <= 100 || parentInitial.initializedGLS <= 200 {
		t.Fatalf("parent initializers did not run: %+v", parentInitial)
	}

	tlsCounter = 11
	glsCounter = 22
	parentSet := snapshotLocality()

	type childResult struct {
		first localitySnapshot
		again localitySnapshot
		set   localitySnapshot
	}
	done := make(chan childResult)
	go func() {
		first := snapshotLocality()
		again := snapshotLocality()
		tlsCounter = 31
		glsCounter = 32
		done <- childResult{first: first, again: again, set: snapshotLocality()}
	}()
	child := <-done

	if child.first.tlsCounter != 0 || child.first.glsCounter != 0 {
		t.Fatalf("child inherited parent local values: %+v", child.first)
	}
	if child.first != child.again {
		t.Fatalf("child initializer ran more than once: first=%+v again=%+v", child.first, child.again)
	}
	if child.first.initializedTLS == parentInitial.initializedTLS || child.first.initializedGLS == parentInitial.initializedGLS {
		t.Fatalf("child reused parent initialization: parent=%+v child=%+v", parentInitial, child.first)
	}
	if child.set.tlsCounter != 31 || child.set.glsCounter != 32 {
		t.Fatalf("child local writes were lost: %+v", child.set)
	}
	if got := snapshotLocality(); got != parentSet {
		t.Fatalf("child changed parent local values: got=%+v want=%+v", got, parentSet)
	}
}

func TestLocalPackageDirectCaches(t *testing.T) {
	type result struct {
		local    int
		imported int
	}
	done := make(chan result)
	go func() {
		glsCounter = 41
		localityscope.SetFirst(51)
		glsCounter++
		imported := localityscope.IncrementFirst()
		done <- result{local: glsCounter, imported: imported}
	}()
	if got := <-done; got != (result{local: 42, imported: 52}) {
		t.Fatalf("local package values through direct caches = %+v", got)
	}
}

func TestPanickingInitializerIsSticky(t *testing.T) {
	for attempt := 0; attempt < 2; attempt++ {
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = localityfailure.Value()
		}()
		if recovered == nil {
			t.Fatalf("attempt %d did not re-panic", attempt)
		}
		if recovered != localityfailure.Failure {
			t.Fatalf("attempt %d panic = %v, want %v", attempt, recovered, localityfailure.Failure)
		}
	}
	if attempts := localityfailure.Attempts(); attempts != 2 {
		t.Fatalf("initializer attempts = %d, want 2", attempts)
	}
}

func TestNilPanickingInitializerIsSticky(t *testing.T) {
	for attempt := 0; attempt < 2; attempt++ {
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = localityfailure.NilValue()
		}()
		if recovered == nil {
			t.Fatalf("attempt %d did not re-panic", attempt)
		}
	}
	if attempts := localityfailure.NilAttempts(); attempts != 2 {
		t.Fatalf("initializer attempts = %d, want 2", attempts)
	}
}

type recursiveInitializer interface {
	value() int
}

type recursiveSourceValue struct{}

func (recursiveSourceValue) value() int {
	return recursiveValue + 1
}

var recursiveSource recursiveInitializer = recursiveSourceValue{}

//llgo:gls
var recursiveValue = recursiveSource.value()

func TestRecursiveInitializerObservesPartialValue(t *testing.T) {
	if recursiveValue != 1 {
		t.Fatalf("recursive initializer value = %d, want 1", recursiveValue)
	}
	if recursiveValue != 1 {
		t.Fatal("recursive initializer ran more than once")
	}
}

var lateInitializerAttempts int

func nextLateValue() int {
	lateInitializerAttempts++
	return 100 + lateInitializerAttempts
}

// zLate sorts after the package init function in SSA member order.
//
//llgo:tls
var zLate = nextLateValue()

func TestLateSortedInitializer(t *testing.T) {
	value, attempts := zLate, lateInitializerAttempts
	if value != 100+attempts {
		t.Fatalf("late initializer value = %d, attempts = %d", value, attempts)
	}
	if again := zLate; again != value || lateInitializerAttempts != attempts {
		t.Fatalf("late initializer repeated: value=%d/%d attempts=%d/%d", again, value, lateInitializerAttempts, attempts)
	}
}

type rootedValue struct {
	value int
	pad   [256]byte
}

func newRootedValue() *rootedValue {
	return &rootedValue{value: 73}
}

//llgo:gls
var rootedPointer = newRootedValue()

//go:noinline
func touchRootedPointer() {
	if rootedPointer == nil || rootedPointer.value != 73 {
		panic("invalid rooted pointer")
	}
}

//go:noinline
func collectWithoutLocalPointer() {
	for i := 0; i < 3; i++ {
		_ = make([]byte, 1<<20)
		runtime.GC()
	}
}

//go:noinline
func readRootedPointer() int {
	return rootedPointer.value
}

func TestLocalPointerIsGCRoot(t *testing.T) {
	touchRootedPointer()
	collectWithoutLocalPointer()
	if got := readRootedPointer(); got != 73 {
		t.Fatalf("rooted pointer value = %d, want 73", got)
	}
}

func TestLocalContextCleanupAfterThreadExit(t *testing.T) {
	for i := 0; i < 16; i++ {
		exited := make(chan struct{})
		go func() {
			defer close(exited)
			touchRootedPointer()
		}()
		<-exited
	}
	runtime.GC()
	runtime.GC()
}

//llgo:gls
var atomicGLS int64

func TestLocalAddressAndAtomicSemantics(t *testing.T) {
	first := &atomicGLS
	second := &atomicGLS
	if first != second {
		t.Fatalf("repeated GLS address changed: %p != %p", first, second)
	}
	atomic.StoreInt64(first, 7)
	if got := atomic.AddInt64(&atomicGLS, 5); got != 12 {
		t.Fatalf("atomic GLS value = %d, want 12", got)
	}
}

func localClosure() func() int {
	glsCounter = 40
	return func() int {
		glsCounter++
		return glsCounter
	}
}

func TestClosureUsesInvocationContext(t *testing.T) {
	closure := localClosure()
	done := make(chan int)
	go func() {
		done <- closure()
	}()
	if got := <-done; got != 1 {
		t.Fatalf("closure used creator GLS value: got %d, want 1", got)
	}
	if glsCounter != 40 {
		t.Fatalf("child closure changed parent GLS value: %d", glsCounter)
	}
}

type escapedBlockValue struct {
	pointer *int
	value   int
}

//llgo:gls
var escapedBlock escapedBlockValue

func TestEscapedPackageBlockAddressSurvivesOwnerExit(t *testing.T) {
	addresses := make(chan *escapedBlockValue)
	exited := make(chan struct{})
	go func() {
		value := 71
		escapedBlock = escapedBlockValue{pointer: &value, value: 71}
		addresses <- &escapedBlock
		close(exited)
	}()
	address := <-addresses
	<-exited
	runtime.GC()
	runtime.GC()
	if address.value != 71 || address.pointer == nil || *address.pointer != 71 {
		t.Fatalf("escaped package block after owner exit = %+v", address)
	}
	done := make(chan bool)
	go func(block *escapedBlockValue) {
		block.value = 72
		*block.pointer = 72
		done <- true
	}(address)
	<-done
	if address.value != 72 || *address.pointer != 72 {
		t.Fatalf("escaped package block after cross-goroutine write = %+v", address)
	}
}

func TestEscapedPackageBlockAddressSurvivesGoexit(t *testing.T) {
	addresses := make(chan *escapedBlockValue)
	exited := make(chan struct{})
	go func() {
		value := 81
		escapedBlock = escapedBlockValue{pointer: &value, value: 81}
		address := &escapedBlock
		defer close(exited)
		defer func() {
			addresses <- address
		}()
		runtime.Goexit()
	}()
	address := <-addresses
	<-exited
	runtime.GC()
	if address.value != 81 || address.pointer == nil || *address.pointer != 81 {
		t.Fatalf("escaped package block after Goexit = %+v", address)
	}
}

//llgo:gls
var zeroSizedGLS struct{}

func TestZeroSizedNativeLocalAddressIsStable(t *testing.T) {
	first := &zeroSizedGLS
	second := &zeroSizedGLS
	if first == nil || first != second {
		t.Fatalf("zero-sized GLS address changed: %p != %p", first, second)
	}
}

func TestInitializerScopeRunsOncePerPackageKind(t *testing.T) {
	firstBefore := localityscope.FirstCalls()
	secondBefore := localityscope.SecondCalls()
	type result struct {
		firstValue       int
		firstCalls       int
		secondCalls      int
		firstCallsAgain  int
		secondValue      int
		secondCallsAfter int
	}
	done := make(chan result)
	go func() {
		firstValue := localityscope.First()
		firstCalls := localityscope.FirstCalls()
		secondCalls := localityscope.SecondCalls()
		_ = localityscope.First()
		firstCallsAgain := localityscope.FirstCalls()
		secondValue := localityscope.Second()
		done <- result{
			firstValue:       firstValue,
			firstCalls:       firstCalls,
			secondCalls:      secondCalls,
			firstCallsAgain:  firstCallsAgain,
			secondValue:      secondValue,
			secondCallsAfter: localityscope.SecondCalls(),
		}
	}()
	got := <-done
	if got.firstValue == 0 || got.secondValue == 0 {
		t.Fatalf("lazy initializer values = %+v", got)
	}
	if got.firstCalls != firstBefore+1 || got.firstCallsAgain != got.firstCalls {
		t.Fatalf("first initializer calls = %+v, baseline %d", got, firstBefore)
	}
	if got.secondCalls != secondBefore+1 || got.secondCallsAfter != got.secondCalls {
		t.Fatalf("package GLS initializers did not run together once: %+v, baseline %d", got, secondBefore)
	}
}

func TestMultiValueInitializerUsesOneGroup(t *testing.T) {
	before := localityscope.PairCalls()
	type result struct {
		first, second int
		afterFirst    int
		afterSecond   int
	}
	done := make(chan result)
	go func() {
		first := localityscope.PairFirst()
		afterFirst := localityscope.PairCalls()
		second := localityscope.PairSecond()
		done <- result{first, second, afterFirst, localityscope.PairCalls()}
	}()
	got := <-done
	if got.first == 0 || got.second == 0 || got.afterFirst != before+1 || got.afterSecond != got.afterFirst {
		t.Fatalf("multi-value initializer group = %+v, baseline %d", got, before)
	}
}

func TestCrossPackageMixedInitializerGroup(t *testing.T) {
	before := localityscope.MixedCalls()
	type result struct {
		scalar        int
		pointer       *int
		addressStable bool
		calls         int
	}
	done := make(chan result)
	go func() {
		scalar := localityscope.MixedScalar()
		address := localityscope.MixedScalarAddress()
		done <- result{
			scalar:        scalar,
			pointer:       localityscope.MixedPointer(),
			addressStable: address == localityscope.MixedScalarAddress(),
			calls:         localityscope.MixedCalls(),
		}
	}()
	got := <-done
	if got.scalar == 0 || got.pointer == nil || !got.addressStable || got.calls != before+1 {
		t.Fatalf("cross-package mixed initializer = %+v, baseline %d", got, before)
	}
}

var benchmarkOrdinary int

//llgo:tls
var benchmarkTLS int

//llgo:gls
var benchmarkGLS int

var benchmarkSink int
var benchmarkReadSink uintptr

//go:noinline
func bumpOrdinaryGlobal() int {
	benchmarkOrdinary++
	return benchmarkOrdinary
}

//go:noinline
func bumpNativeTLS() int {
	benchmarkTLS++
	return benchmarkTLS
}

//go:noinline
func bumpNativeGLS() int {
	benchmarkGLS++
	return benchmarkGLS
}

type benchmarkPackageValue struct {
	pointer *int
	value   int
}

//llgo:tls
var benchmarkTLSPackage benchmarkPackageValue

//llgo:gls
var benchmarkGLSPackage benchmarkPackageValue

//go:noinline
func bumpTLSPackageBlock() int {
	benchmarkTLSPackage.value++
	return benchmarkTLSPackage.value
}

//go:noinline
func bumpGLSPackageBlock() int {
	benchmarkGLSPackage.value++
	return benchmarkGLSPackage.value
}

//go:noinline
func readGLSPackageBlock() uintptr {
	return uintptr(benchmarkGLSPackage.value)
}

func BenchmarkOrdinaryGlobal(b *testing.B) {
	value := 0
	for i := 0; i < b.N; i++ {
		value += bumpOrdinaryGlobal()
	}
	benchmarkSink = value
}

func BenchmarkNativeTLS(b *testing.B) {
	value := 0
	for i := 0; i < b.N; i++ {
		value += bumpNativeTLS()
	}
	benchmarkSink = value
}

func BenchmarkNativeGLS(b *testing.B) {
	value := 0
	for i := 0; i < b.N; i++ {
		value += bumpNativeGLS()
	}
	benchmarkSink = value
}

func BenchmarkTLSPackageBlock(b *testing.B) {
	benchmarkTLSPackage.pointer = &benchmarkSink
	b.ResetTimer()
	value := 0
	for i := 0; i < b.N; i++ {
		value += bumpTLSPackageBlock()
	}
	benchmarkSink = value
}

func BenchmarkGLSPackageBlock(b *testing.B) {
	benchmarkGLSPackage.pointer = &benchmarkSink
	b.ResetTimer()
	value := 0
	for i := 0; i < b.N; i++ {
		value += bumpGLSPackageBlock()
	}
	benchmarkSink = value
}

func TestComparableLocalityReads(t *testing.T) {
	localitybench.PrepareReads()
	ordinary := localitybench.ReadOrdinaryGlobal()
	native := localitybench.ReadNativeTLS()
	local := localitybench.ReadGLSPackage()
	if ordinary == 0 || native != ordinary || local != ordinary {
		t.Fatalf("comparable locality reads = ordinary:%#x native:%#x GLS:%#x", ordinary, native, local)
	}
}

func BenchmarkComparableOrdinaryGlobalRead(b *testing.B) {
	localitybench.PrepareReads()
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += localitybench.ReadOrdinaryGlobal()
	}
	benchmarkReadSink = value
}

func BenchmarkComparableNativeTLSRead(b *testing.B) {
	localitybench.PrepareReads()
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += localitybench.ReadNativeTLS()
	}
	benchmarkReadSink = value
}

func BenchmarkComparableGLSPackageRead(b *testing.B) {
	localitybench.PrepareReads()
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += localitybench.ReadGLSPackage()
	}
	benchmarkReadSink = value
}

func BenchmarkAlternatingGLSPackageRead(b *testing.B) {
	localitybench.PrepareReads()
	benchmarkGLSPackage.pointer = &benchmarkSink
	benchmarkGLSPackage.value = 1
	b.ResetTimer()
	var value uintptr
	for i := 0; i < b.N; i++ {
		value += readGLSPackageBlock()
		value += localitybench.ReadGLSPackage()
	}
	benchmarkReadSink = value
}

func BenchmarkGoroutineEntry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		go func() { close(done) }()
		<-done
	}
}

func BenchmarkGoroutinePackageBlockFirstTouch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		go func() {
			localitybench.Touch()
			close(done)
		}()
		<-done
	}
}
