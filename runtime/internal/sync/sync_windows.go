//go:build windows

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

// Package sync exposes the synchronization primitives used by the hosted
// runtime, backed by pthreads on Unix and Win32 on Windows.
package sync

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	ctime "github.com/xgo-dev/llgo/runtime/internal/clite/time"
)

const (
	// MinGW exposes the address-wait imports used by this backend through
	// libsynchronization rather than libkernel32.
	LLGoPackage = "link: -lsynchronization"
)

// Once has the layout of Windows INIT_ONCE. Its zero value is ready for use.
type Once struct {
	state uintptr
}

//go:linkname winOnce C.llgo_win_once
func winOnce(once *Once, f OnceFunc) c.Int

func (o *Once) Do(f OnceFunc) c.Int {
	return winOnce(o, f)
}

type MutexAttr struct{}

// Mutex has the layout of Windows SRWLOCK. Its zero value is ready for use.
type Mutex struct {
	state uintptr
}

//go:linkname acquireSRWLockExclusive stdcall.AcquireSRWLockExclusive
func acquireSRWLockExclusive(m *Mutex)

//go:linkname releaseSRWLockExclusive stdcall.ReleaseSRWLockExclusive
func releaseSRWLockExclusive(m *Mutex)

func (m *Mutex) Init(_ *MutexAttr) c.Int {
	m.state = 0
	return 0
}

func (m *Mutex) Destroy() {}

func (m *Mutex) Lock() {
	acquireSRWLockExclusive(m)
}

func (m *Mutex) Unlock() {
	releaseSRWLockExclusive(m)
}

type CondAttr struct{}

// Cond has the layout of Windows CONDITION_VARIABLE. Its zero value is ready
// for use with a Mutex backed by an exclusive SRW lock.
type Cond struct {
	state uintptr
}

//go:linkname wakeConditionVariable stdcall.WakeConditionVariable
func wakeConditionVariable(cond *Cond)

//go:linkname wakeAllConditionVariable stdcall.WakeAllConditionVariable
func wakeAllConditionVariable(cond *Cond)

//go:linkname winCondWait C.llgo_win_cond_wait
func winCondWait(cond *Cond, m *Mutex) c.Int

//go:linkname winCondWaitInactive C.llgo_win_cond_wait_inactive
func winCondWaitInactive(cond *Cond, m *Mutex) c.Int

//go:linkname winCondTimedWait C.llgo_win_cond_timedwait
func winCondTimedWait(cond *Cond, m *Mutex, abstime *ctime.Timespec) c.Int

func (cond *Cond) Init(_ *CondAttr) c.Int {
	cond.state = 0
	return 0
}

func (cond *Cond) Destroy() {}

func (cond *Cond) Signal() c.Int {
	wakeConditionVariable(cond)
	return 0
}

func (cond *Cond) Broadcast() c.Int {
	wakeAllConditionVariable(cond)
	return 0
}

func (cond *Cond) Wait(m *Mutex) c.Int {
	return winCondWait(cond, m)
}

// WaitInactive blocks outside BDWGC's active thread set on every Windows ABI
// (including MSVC ARM64). Use only for detached runtime worker threads that
// carry no Go frames.
func (cond *Cond) WaitInactive(m *Mutex) c.Int {
	return winCondWaitInactive(cond, m)
}

func (cond *Cond) TimedWait(m *Mutex, abstime *ctime.Timespec) c.Int {
	return winCondTimedWait(cond, m, abstime)
}

//go:linkname winWaitUint32 C.llgo_win_wait_uint32
func winWaitUint32(addr *uint32, value uint32) c.Int

//go:linkname wakeByAddressSingle stdcall.WakeByAddressSingle
func wakeByAddressSingle(addr *uint32)

// WaitUint32 blocks while addr still contains value. Callers must recheck the
// value after it returns because Windows permits spurious wakeups.
func WaitUint32(addr *uint32, value uint32) c.Int {
	return winWaitUint32(addr, value)
}

// WakeUint32 wakes one thread waiting for addr.
func WakeUint32(addr *uint32) {
	wakeByAddressSingle(addr)
}
