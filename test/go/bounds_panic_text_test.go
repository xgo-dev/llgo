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
	"runtime"
	"testing"
)

func TestBoundsPanicText(t *testing.T) {
	tests := []struct {
		name string
		want string
		f    func()
	}{
		{
			name: "signed negative index",
			want: "runtime error: index out of range [-1]",
			f: func() {
				s := []int{1, 2, 3}
				i := -1
				_ = s[i]
			},
		},
		{
			name: "signed index length",
			want: "runtime error: index out of range [3] with length 3",
			f: func() {
				s := []int{1, 2, 3}
				i := 3
				_ = s[i]
			},
		},
		{
			name: "unsigned index length",
			want: "runtime error: index out of range [18446744073709551615] with length 3",
			f: func() {
				s := []int{1, 2, 3}
				i := ^uint64(0)
				_ = s[i]
			},
		},
		{
			name: "slice high capacity",
			want: "runtime error: slice bounds out of range [:4] with capacity 3",
			f: func() {
				s := []int{1, 2, 3}
				j := 4
				_ = s[:j]
			},
		},
		{
			name: "array high length",
			want: "runtime error: slice bounds out of range [:4] with length 3",
			f: func() {
				a := [3]int{1, 2, 3}
				j := 4
				_ = a[:j]
			},
		},
		{
			name: "string high length",
			want: "runtime error: slice bounds out of range [:4] with length 3",
			f: func() {
				s := "123"
				j := 4
				_ = s[:j]
			},
		},
		{
			name: "unsigned low above high",
			want: "runtime error: slice bounds out of range [18446744073709551615:0]",
			f: func() {
				s := []int{1, 2, 3}
				i := ^uint64(0)
				_ = s[i:0]
			},
		},
		{
			name: "full slice max capacity",
			want: "runtime error: slice bounds out of range [::4] with capacity 3",
			f: func() {
				s := []int{1, 2, 3}
				k := 4
				_ = s[:0:k]
			},
		},
		{
			name: "full array max length",
			want: "runtime error: slice bounds out of range [::4] with length 3",
			f: func() {
				a := [3]int{1, 2, 3}
				k := 4
				_ = a[:0:k]
			},
		},
		{
			name: "full slice unsigned max",
			want: "runtime error: slice bounds out of range [::18446744073709551615] with capacity 3",
			f: func() {
				s := []int{1, 2, 3}
				k := ^uint64(0)
				_ = s[:0:k]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectRuntimeBoundsPanic(t, tt.want, tt.f)
		})
	}
}

func expectRuntimeBoundsPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic %q", want)
		}
		rerr, ok := err.(runtime.Error)
		if !ok {
			t.Fatalf("panic type = %T, want runtime.Error", err)
		}
		if got := rerr.Error(); got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	f()
}
