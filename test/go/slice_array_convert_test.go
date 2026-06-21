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

func TestSliceToArrayConversionSemantics(t *testing.T) {
	s := []byte{0, 1, 2, 3}

	if p := (*[4]byte)(s); &p[0] != &s[0] {
		t.Fatal("slice to array pointer did not preserve backing array")
	}

	a := [4]byte(s)
	if a != [4]byte{0, 1, 2, 3} {
		t.Fatalf("slice to array = %v", a)
	}
	a[0] = 9
	if s[0] != 0 {
		t.Fatalf("slice to array did not make a copy; slice[0] = %d", s[0])
	}

	var nilSlice []byte
	if p := (*[0]byte)(nilSlice); p != nil {
		t.Fatal("nil slice converted to *[0]byte should be nil")
	}
	_ = [0]byte(nilSlice)

	empty := make([]byte, 0)
	if p := (*[0]byte)(empty); p == nil {
		t.Fatal("empty non-nil slice converted to *[0]byte should be non-nil")
	}
	_ = [0]byte(empty)
}

func TestSliceToArrayConversionPanicText(t *testing.T) {
	s := []byte{0, 1, 2, 3}
	want := "runtime error: cannot convert slice with length 4 to array or pointer to array with length 5"

	expectSliceToArrayPanic(t, want, func() {
		_ = (*[5]byte)(s)
	})
	expectSliceToArrayPanic(t, want, func() {
		_ = [5]byte(s)
	})
}

func TestSliceToArrayConversionEvaluatesOperand(t *testing.T) {
	var p *[]byte
	expectSliceToArrayPanic(t, "runtime error: invalid memory address or nil pointer dereference", func() {
		_ = [0]byte(*p)
	})
}

func expectSliceToArrayPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatalf("expected panic %q", want)
		}
		if got := sliceToArrayPanicString(err); got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	f()
}

func sliceToArrayPanicString(v any) string {
	if err, ok := v.(interface{ Error() string }); ok {
		return err.Error()
	}
	return fmt.Sprint(v)
}
