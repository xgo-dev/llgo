//go:build linux || darwin

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
	"syscall"
	"testing"
)

var boundsLoadPageSize = syscall.Getpagesize()
var boundsLoadOne = 1

func TestBoundsCheckBeforeWiderByteLoad(t *testing.T) {
	b, err := syscall.Mmap(-1, 0, 2*boundsLoadPageSize, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_ANON|syscall.MAP_PRIVATE)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.Munmap(b)
	if err := syscall.Mprotect(b[boundsLoadPageSize:], syscall.PROT_NONE); err != nil {
		t.Fatal(err)
	}
	x := b[boundsLoadPageSize-boundsLoadOne : boundsLoadPageSize]

	tests := []struct {
		name string
		f    func([]byte)
	}{
		{"uint16 constant indexes", func(x []byte) { _ = loadUint16Const(x) }},
		{"uint16 variable indexes", func(x []byte) { _ = loadUint16Index(x, 0) }},
		{"uint32 constant indexes", func(x []byte) { _ = loadUint32Const(x) }},
		{"uint32 variable indexes", func(x []byte) { _ = loadUint32Index(x, 0) }},
		{"uint64 constant indexes", func(x []byte) { _ = loadUint64Const(x) }},
		{"uint64 variable indexes", func(x []byte) { _ = loadUint64Index(x, 0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectIndexOnePanic(t, func() { tt.f(x) })
		})
	}
}

func expectIndexOnePanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		err := recover()
		if err == nil {
			t.Fatal("expected bounds panic")
		}
		if got, want := fmt.Sprint(err), "runtime error: index out of range [1] with length 1"; got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	f()
}

func loadUint16Const(x []byte) uint16 {
	return uint16(x[0]) | uint16(x[1])<<8
}

func loadUint16Index(x []byte, i int) uint16 {
	return uint16(x[i]) | uint16(x[i+1])<<8
}

func loadUint32Const(x []byte) uint32 {
	return uint32(x[0]) | uint32(x[1])<<8 | uint32(x[2])<<16 | uint32(x[3])<<24
}

func loadUint32Index(x []byte, i int) uint32 {
	return uint32(x[i]) | uint32(x[i+1])<<8 | uint32(x[i+2])<<16 | uint32(x[i+3])<<24
}

func loadUint64Const(x []byte) uint64 {
	return uint64(x[0]) | uint64(x[1])<<8 | uint64(x[2])<<16 | uint64(x[3])<<24 |
		uint64(x[4])<<32 | uint64(x[5])<<40 | uint64(x[6])<<48 | uint64(x[7])<<56
}

func loadUint64Index(x []byte, i int) uint64 {
	return uint64(x[i]) | uint64(x[i+1])<<8 | uint64(x[i+2])<<16 | uint64(x[i+3])<<24 |
		uint64(x[i+4])<<32 | uint64(x[i+5])<<40 | uint64(x[i+6])<<48 | uint64(x[i+7])<<56
}
