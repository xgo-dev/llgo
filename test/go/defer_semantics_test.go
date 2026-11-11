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
	"errors"
	"reflect"
	"sync"
	"testing"
)

func recoverThenDeferPanic(events *[]string) {
	if recovered := recover(); recovered != nil {
		defer panic(recovered)
		*events = append(*events, "recovered")
	}
	*events = append(*events, "returning")
}

func TestNestedLoopDefersBeforeTail(t *testing.T) {
	for _, mode := range []string{"return", "panic", "break", "continue"} {
		t.Run(mode, func(t *testing.T) {
			var events []int
			mutex := new(sync.Mutex)
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				func() {
					mutex.Lock()
					defer mutex.Unlock()
					record := func(value int) { events = append(events, value) }
					for _, row := range [][]int{{1, 2}, {3, 4}} {
						for _, value := range row {
							if mode == "continue" && value == 2 {
								continue
							}
							defer record(value)
							if mode == "break" {
								break
							}
						}
					}
					defer record(5)
					if mode == "panic" {
						panic("nested defers")
					}
				}()
			}()
			want := []int{5, 4, 3, 2, 1}
			if mode == "break" {
				want = []int{5, 3, 1}
			}
			if mode == "continue" {
				want = []int{5, 4, 3, 1}
			}
			if !reflect.DeepEqual(events, want) {
				t.Fatalf("defer order = %v, want %v", events, want)
			}
			if mode == "panic" {
				if recovered != "nested defers" {
					t.Fatalf("panic = %v, want nested defers", recovered)
				}
			} else if recovered != nil {
				t.Fatalf("unexpected panic = %v", recovered)
			}
			if !mutex.TryLock() {
				t.Fatal("deferred method did not unlock the original receiver")
			}
			mutex.Unlock()
		})
	}
}

func TestRecoverThenDeferredPanic(t *testing.T) {
	sentinel := errors.New("panic in test")
	var events []string
	func() {
		defer func() {
			if recovered := recover(); recovered != sentinel {
				t.Fatalf("outer recover = %v, want %v", recovered, sentinel)
			}
			events = append(events, "repanicked")
		}()
		func() {
			defer recoverThenDeferPanic(&events)
			panic(sentinel)
		}()
		t.Fatal("re-panic unexpectedly returned")
	}()

	want := []string{"recovered", "returning", "repanicked"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("defer events = %v, want %v", events, want)
	}
}

type nestedDeferValue struct {
	Type uint8
	Bits uint64
}

//go:noinline
func nestedLoopDeferredReceiver() *sync.Mutex {
	s := new(sync.Mutex)
	s.Lock()
	defer s.Unlock()
	output := func(name string, initial nestedDeferValue, status int32, args ...nestedDeferValue) nestedDeferValue {
		slot := new(sync.Mutex)
		slot.Lock()
		defer slot.Unlock()
		return initial
	}
	for _, prefix := range []string{"q", "n"} {
		for _, class := range []string{"plain", "qualified"} {
			_ = prefix
			_ = class
			object := output("make", nestedDeferValue{Bits: 1}, 0, nestedDeferValue{Bits: 42})
			defer output("free", object, 0)
		}
	}
	object := output("make", nestedDeferValue{Bits: 5}, 0, nestedDeferValue{Bits: 42})
	defer func() { output("free", object, 0) }()
	template := output("make", nestedDeferValue{Bits: 6}, 0, nestedDeferValue{Bits: 42})
	defer output("free", template, 0)
	object = output("free", object, 0)
	return s
}

func TestNestedLoopDeferredMethodReceiver(t *testing.T) {
	mutex := nestedLoopDeferredReceiver()
	if !mutex.TryLock() {
		t.Fatal("deferred call did not unlock the original receiver")
	}
	mutex.Unlock()
}
