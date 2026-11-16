//go:build darwin

// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package sync

import _ "unsafe"

// UnfairMutex has the native layout of os_unfair_lock and starts unlocked.
// Its owner must unlock on the same native thread; condition-variable waits
// continue to use Mutex. Darwin's hosted goroutines each own a native thread.
type UnfairMutex struct{ state uint32 }

//go:linkname unfairMutexLock C.os_unfair_lock_lock
func unfairMutexLock(m *UnfairMutex)

//go:linkname unfairMutexUnlock C.os_unfair_lock_unlock
func unfairMutexUnlock(m *UnfairMutex)

func (m *UnfairMutex) Lock()   { unfairMutexLock(m) }
func (m *UnfairMutex) Unlock() { unfairMutexUnlock(m) }
