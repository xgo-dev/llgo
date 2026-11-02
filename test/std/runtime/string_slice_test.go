// Copyright 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0 license.

package runtime_test

import (
	"runtime"
	"strconv"
	"testing"
)

// Keep values at a call boundary so both Go and LLGo execute the bounds checks.
//
//go:noinline
func stringSliceSigned(s string, low, high int) string { return s[low:high] }

//go:noinline
func stringSliceNarrow(s string, low, high int8) string { return s[low:high] }

//go:noinline
func stringSliceWideLow(s string, low int64) string { return s[low:] }

//go:noinline
func stringSliceWideHigh(s string, high int64) string { return s[:high] }

//go:noinline
func stringSliceUnsignedLow(s string, low uint64) string { return s[low:] }

//go:noinline
func stringSliceUnsignedHigh(s string, high uint64) string { return s[:high] }

//go:noinline
func stringSliceNativeUnsigned(s string, low uint) string { return s[low:] }

func TestStringSliceIndices(t *testing.T) {
	for _, test := range []struct {
		name, want string
		call       func() string
	}{
		{"native", "bc", func() string { return stringSliceSigned("abcd", 1, 3) }},
		{"narrow", "bc", func() string { return stringSliceNarrow("abcd", 1, 3) }},
		{"wide low", "bcd", func() string { return stringSliceWideLow("abcd", 1) }},
		{"wide high", "ab", func() string { return stringSliceWideHigh("abcd", 2) }},
		{"unsigned low", "bcd", func() string { return stringSliceUnsignedLow("abcd", 1) }},
		{"unsigned high", "ab", func() string { return stringSliceUnsignedHigh("abcd", 2) }},
		{"empty suffix", "", func() string { return stringSliceSigned("abcd", 4, 4) }},
		{"empty string", "", func() string { return stringSliceSigned("", 0, 0) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.call(); got != test.want {
				t.Fatalf("slice = %q, want %q", got, test.want)
			}
		})
	}
	for _, test := range []struct {
		name, detail string
		call         func() string
	}{
		{"negative low", "[-1:]", func() string { return stringSliceSigned("abc", -1, 2) }},
		{"negative high", "[:-1]", func() string { return stringSliceSigned("abc", 0, -1) }},
		{"high length", "[:4] with length 3", func() string { return stringSliceSigned("abc", 0, 4) }},
		{"low above high", "[2:1]", func() string { return stringSliceSigned("abc", 2, 1) }},
		{"narrow negative", "[-1:]", func() string { return stringSliceNarrow("abc", -1, 2) }},
		{"wide low", "[4294967297:3]", func() string { return stringSliceWideLow("abc", 1<<32|1) }},
		{"wide high", "[:4294967298] with length 3", func() string { return stringSliceWideHigh("abc", 1<<32|2) }},
		{"wide negative", "[-4294967295:]", func() string { return stringSliceWideLow("abc", -1<<32|1) }},
		{"unsigned low", "[18446744073709551615:3]", func() string { return stringSliceUnsignedLow("abc", ^uint64(0)) }},
		{"unsigned high", "[:18446744073709551615] with length 3", func() string { return stringSliceUnsignedHigh("abc", ^uint64(0)) }},
		{"native unsigned", "[" + strconv.FormatUint(uint64(^uint(0)), 10) + ":3]", func() string { return stringSliceNativeUnsigned("abc", ^uint(0)) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				value := recover()
				err, ok := value.(runtime.Error)
				want := "runtime error: slice bounds out of range " + test.detail
				if !ok || err.Error() != want {
					t.Fatalf("panic = %v (%T), want runtime.Error %q", value, value, want)
				}
			}()
			_ = test.call()
		})
	}
}
