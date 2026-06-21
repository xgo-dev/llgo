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
	"fmt"
	"testing"
)

func TestMakeSlicePanicText(t *testing.T) {
	tests := []struct {
		name string
		want string
		f    func()
	}{
		{
			name: "cap out of range from propagated local",
			want: "runtime error: makeslice: cap out of range",
			f: func() {
				i := 2
				s := make([]int, i, 1)
				s[0] = 1
			},
		},
		{
			name: "cap out of range from parameter",
			want: "runtime error: makeslice: cap out of range",
			f: func() {
				makeSliceCapPanic(2)
			},
		},
		{
			name: "len out of range from propagated local",
			want: "runtime error: makeslice: len out of range",
			f: func() {
				i := -1
				s := make([]int, i, 3)
				s[0] = 1
			},
		},
		{
			name: "len out of range from parameter",
			want: "runtime error: makeslice: len out of range",
			f: func() {
				makeSliceLenPanic(-1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectMakeSlicePanic(t, tt.want, tt.f)
		})
	}
}

func makeSliceCapPanic(i int) {
	s := make([]int, i, 1)
	s[0] = 1
}

func makeSliceLenPanic(i int) {
	s := make([]int, i, 3)
	s[0] = 1
}

func expectMakeSlicePanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic %q", want)
		}
		if got := makeSlicePanicString(err); got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	f()
}

func makeSlicePanicString(v any) string {
	if err, ok := v.(interface{ Error() string }); ok {
		return err.Error()
	}
	return fmt.Sprint(v)
}
