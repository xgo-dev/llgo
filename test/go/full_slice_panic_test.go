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

func TestFullSliceBoundsPanicText(t *testing.T) {
	tests := []struct {
		name string
		want string
		f    func()
	}{
		{
			name: "max exceeds cap",
			want: "runtime error: slice bounds out of range [::2] with capacity 1",
			f: func() {
				s := []int{1}
				k := 2
				_ = s[:1:k]
			},
		},
		{
			name: "high exceeds max",
			want: "runtime error: slice bounds out of range [:2:1]",
			f: func() {
				s := []int{1, 2}
				j, k := 2, 1
				_ = s[:j:k]
			},
		},
		{
			name: "low exceeds high",
			want: "runtime error: slice bounds out of range [1:0:]",
			f: func() {
				s := []int{1, 2}
				i, j := 1, 0
				_ = s[i:j:1]
			},
		},
		{
			name: "negative low",
			want: "runtime error: slice bounds out of range [-1::]",
			f: func() {
				s := []int{1, 2}
				i := -1
				_ = s[i:0:1]
			},
		},
		{
			name: "negative high",
			want: "runtime error: slice bounds out of range [:-1:]",
			f: func() {
				s := []int{1, 2}
				j := -1
				_ = s[0:j:1]
			},
		},
		{
			name: "negative max",
			want: "runtime error: slice bounds out of range [::-1]",
			f: func() {
				s := []int{1, 2}
				k := -1
				_ = s[0:0:k]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectFullSlicePanic(t, tt.want, tt.f)
		})
	}
}

func expectFullSlicePanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic %q", want)
		}
		if got := fullSlicePanicString(err); got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	f()
}

func fullSlicePanicString(v any) string {
	if err, ok := v.(interface{ Error() string }); ok {
		return err.Error()
	}
	return fmt.Sprint(v)
}
