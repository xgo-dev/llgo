// Copyright 2025 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.26

package sync

import _ "unsafe"

//go:linkname runtime_SemacquireMutex internal/sync.runtime_SemacquireMutex
func runtime_SemacquireMutex(s *uint32, lifo bool, skipframes int)

//go:linkname runtime_Semrelease internal/sync.runtime_Semrelease
func runtime_Semrelease(s *uint32, handoff bool, skipframes int)

//go:linkname runtime_canSpin internal/sync.runtime_canSpin
func runtime_canSpin(i int) bool

//go:linkname runtime_doSpin internal/sync.runtime_doSpin
func runtime_doSpin()

//go:linkname runtime_nanotime internal/sync.runtime_nanotime
func runtime_nanotime() int64

//go:linkname throw internal/sync.throw
func throw(string)

//go:linkname fatal internal/sync.fatal
func fatal(string)
