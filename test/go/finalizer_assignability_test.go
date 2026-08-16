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
	"runtime"
	"strings"
	"testing"
)

type finalizerAssignableValue struct {
	value int
	keep  *int
}

type finalizerAssignablePointer *finalizerAssignableValue

type finalizerInterfaceValue struct {
	value int
	keep  *int
}

type finalizerAssignableInterface interface {
	finalizerValue() int
}

func (p *finalizerInterfaceValue) finalizerValue() int {
	return p.value
}

func TestRuntimeSetFinalizerAssignableArgumentTypes(t *testing.T) {
	tests := []struct {
		name     string
		register func(chan<- int, int)
	}{
		{
			name: "identical pointer",
			register: func(done chan<- int, value int) {
				p := &finalizerAssignableValue{value: value, keep: new(int)}
				runtime.SetFinalizer(p, func(p *finalizerAssignableValue) { done <- p.value })
			},
		},
		{
			name: "identical named pointer",
			register: func(done chan<- int, value int) {
				p := finalizerAssignablePointer(&finalizerAssignableValue{value: value, keep: new(int)})
				runtime.SetFinalizer(p, func(p finalizerAssignablePointer) { done <- p.value })
			},
		},
		{
			name: "named to unnamed pointer",
			register: func(done chan<- int, value int) {
				p := finalizerAssignablePointer(&finalizerAssignableValue{value: value, keep: new(int)})
				runtime.SetFinalizer(p, func(p *finalizerAssignableValue) { done <- p.value })
			},
		},
		{
			name: "empty interface with aggregate result",
			register: func(done chan<- int, value int) {
				p := &finalizerAssignableValue{value: value, keep: new(int)}
				runtime.SetFinalizer(p, func(v any) (unused [4]int64) {
					done <- v.(*finalizerAssignableValue).value
					return
				})
			},
		},
		{
			name: "implemented interface",
			register: func(done chan<- int, value int) {
				p := &finalizerInterfaceValue{value: value, keep: new(int)}
				runtime.SetFinalizer(p, func(v finalizerAssignableInterface) { done <- v.finalizerValue() })
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// BDWGC conservatively scans stacks and registers, so a stale
			// pointer can keep any one object reachable. Register several
			// objects and require one real finalizer call to verify the ABI.
			const registrations = 8
			const wantBase = 97531
			want := wantBase + i
			done := make(chan int, registrations)
			registerFinalizerForTest(func() {
				for range registrations {
					tt.register(done, want)
				}
			})
			waitForFinalizerValue(t, done, want)
		})
	}
}

const finalizerInvalidCaseEnv = "LLGO_TEST_FINALIZER_INVALID_CASE"

func TestRuntimeSetFinalizerRejectsInvalidArgumentTypes(t *testing.T) {
	if name := os.Getenv(finalizerInvalidCaseEnv); name != "" {
		p := &finalizerAssignableValue{keep: new(int)}
		switch name {
		case "non-function":
			runtime.SetFinalizer(p, 1)
		case "no parameters":
			runtime.SetFinalizer(p, func() {})
		case "two parameters":
			runtime.SetFinalizer(p, func(*finalizerAssignableValue, int) {})
		case "variadic":
			runtime.SetFinalizer(p, func(...*finalizerAssignableValue) {})
		case "wrong type":
			runtime.SetFinalizer(p, func(*int) {})
		default:
			panic("unknown invalid finalizer case: " + name)
		}
		os.Exit(0)
	}

	tests := []string{
		"non-function",
		"no parameters",
		"two parameters",
		"variadic",
		"wrong type",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestRuntimeSetFinalizerRejectsInvalidArgumentTypes$")
			cmd.Env = append(os.Environ(), finalizerInvalidCaseEnv+"="+name)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("SetFinalizer accepted invalid %s", name)
			}
			if !strings.Contains(string(out), "runtime.SetFinalizer:") {
				t.Fatalf("SetFinalizer error for %s:\n%s", name, out)
			}
		})
	}
}
