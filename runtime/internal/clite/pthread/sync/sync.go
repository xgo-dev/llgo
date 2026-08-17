/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package sync

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/time"
)

const (
	LLGoPackage = "link"
)

const (
	PTHREAD_MUTEX_NORMAL     = 0
	PTHREAD_MUTEX_ERRORCHECK = 1
	PTHREAD_MUTEX_RECURSIVE  = 2
	PTHREAD_MUTEX_DEFAULT    = PTHREAD_MUTEX_NORMAL
)

// -----------------------------------------------------------------------------

type align4 struct {
	_ [0]uint32
}

type align8 struct {
	_ [0]uint64
}

type alignPtr struct {
	_ [0]uintptr
}

// Once is an object that will perform exactly one action.
// pthread_once_t
type Once struct {
	_      alignOnce
	Unused [PthreadOnceSize]c.Char
}

//go:linkname OnceInit once_control
var OnceInit Once

// llgo:link (*Once).Do C.pthread_once
func (o *Once) Do(f func()) c.Int { return 0 }

// -----------------------------------------------------------------------------

type MutexType c.Int

const (
	MUTEX_NORMAL     MutexType = PTHREAD_MUTEX_NORMAL
	MUTEX_ERRORCHECK MutexType = PTHREAD_MUTEX_ERRORCHECK
	MUTEX_RECURSIVE  MutexType = PTHREAD_MUTEX_RECURSIVE
	MUTEX_DEFAULT    MutexType = PTHREAD_MUTEX_DEFAULT
)

// MutexAttr is a mutex attribute object.
// pthread_mutexattr_t
type MutexAttr struct {
	_      alignMutexAttr
	Unused [PthreadMutexAttrSize]c.Char
}

// llgo:link (*MutexAttr).Init C.pthread_mutexattr_init
func (a *MutexAttr) Init(attr *MutexAttr) c.Int { return 0 }

// llgo:link (*MutexAttr).Destroy C.pthread_mutexattr_destroy
func (a *MutexAttr) Destroy() {}

// llgo:link (*MutexAttr).SetType C.pthread_mutexattr_settype
func (a *MutexAttr) SetType(typ MutexType) c.Int { return 0 }

// -----------------------------------------------------------------------------

//go:linkname c_pthread_mutex_init C.pthread_mutex_init
func c_pthread_mutex_init(m *Mutex, attr *MutexAttr) c.Int

//go:linkname c_pthread_mutex_destroy C.pthread_mutex_destroy
func c_pthread_mutex_destroy(m *Mutex) c.Int

//go:linkname c_pthread_mutex_lock C.pthread_mutex_lock
func c_pthread_mutex_lock(m *Mutex) c.Int

//go:linkname c_pthread_mutex_unlock C.pthread_mutex_unlock
func c_pthread_mutex_unlock(m *Mutex) c.Int

//go:linkname c_pthread_mutex_trylock C.pthread_mutex_trylock
func c_pthread_mutex_trylock(m *Mutex) c.Int

// Mutex is a mutual exclusion lock.
// pthread_mutex_t
type Mutex struct {
	_      alignMutex
	Unused [PthreadMutexSize]c.Char
}

func (m *Mutex) Init(attr *MutexAttr) c.Int {
	return c_pthread_mutex_init(m, attr)
}

func (m *Mutex) Destroy() {
	c_pthread_mutex_destroy(m)
}

func (m *Mutex) TryLock() c.Int {
	return c_pthread_mutex_trylock(m)
}

func (m *Mutex) Lock() {
	c_pthread_mutex_lock(m)
}

func (m *Mutex) Unlock() {
	c_pthread_mutex_unlock(m)
}

// -----------------------------------------------------------------------------

// RWLockAttr is a read-write lock attribute object.
// pthread_rwlockattr_t
type RWLockAttr struct {
	_      alignRWLockAttr
	Unused [PthreadRWLockAttrSize]c.Char
}

// llgo:link (*RWLockAttr).Init C.pthread_rwlockattr_init
func (a *RWLockAttr) Init(attr *RWLockAttr) c.Int { return 0 }

// llgo:link (*RWLockAttr).Destroy C.pthread_rwlockattr_destroy
func (a *RWLockAttr) Destroy() {}

// llgo:link (*RWLockAttr).SetPShared C.pthread_rwlockattr_setpshared
func (a *RWLockAttr) SetPShared(pshared c.Int) c.Int { return 0 }

// llgo:link (*RWLockAttr).GetPShared C.pthread_rwlockattr_getpshared
func (a *RWLockAttr) GetPShared(pshared *c.Int) c.Int { return 0 }

// -----------------------------------------------------------------------------

//go:linkname c_pthread_rwlock_init C.pthread_rwlock_init
func c_pthread_rwlock_init(rw *RWLock, attr *RWLockAttr) c.Int

//go:linkname c_pthread_rwlock_destroy C.pthread_rwlock_destroy
func c_pthread_rwlock_destroy(rw *RWLock) c.Int

//go:linkname c_pthread_rwlock_rdlock C.pthread_rwlock_rdlock
func c_pthread_rwlock_rdlock(rw *RWLock) c.Int

//go:linkname c_pthread_rwlock_wrlock C.pthread_rwlock_wrlock
func c_pthread_rwlock_wrlock(rw *RWLock) c.Int

//go:linkname c_pthread_rwlock_unlock C.pthread_rwlock_unlock
func c_pthread_rwlock_unlock(rw *RWLock) c.Int

//go:linkname c_pthread_rwlock_tryrdlock C.pthread_rwlock_tryrdlock
func c_pthread_rwlock_tryrdlock(rw *RWLock) c.Int

//go:linkname c_pthread_rwlock_trywrlock C.pthread_rwlock_trywrlock
func c_pthread_rwlock_trywrlock(rw *RWLock) c.Int

// RWLock is a read-write lock.
// pthread_rwlock_t
type RWLock struct {
	_      alignRWLock
	Unused [PthreadRWLockSize]c.Char
}

// llgo:link (*RWLock).Init C.pthread_rwlock_init
func (rw *RWLock) Init(attr *RWLockAttr) c.Int { return 0 }

func (rw *RWLock) Destroy() {
	c_pthread_rwlock_destroy(rw)
}

func (rw *RWLock) RLock() {
	c_pthread_rwlock_rdlock(rw)
}

// llgo:link (*RWLock).TryRLock C.pthread_rwlock_tryrdlock
func (rw *RWLock) TryRLock() c.Int { return 0 }

func (rw *RWLock) RUnlock() {
	c_pthread_rwlock_unlock(rw)
}

func (rw *RWLock) Lock() {
	c_pthread_rwlock_wrlock(rw)
}

// llgo:link (*RWLock).TryLock C.pthread_rwlock_trywrlock
func (rw *RWLock) TryLock() c.Int { return 0 }

func (rw *RWLock) Unlock() {
	c_pthread_rwlock_unlock(rw)
}

// -----------------------------------------------------------------------------

// CondAttr is a condition variable attribute object.
// pthread_condattr_t
type CondAttr struct {
	_      alignCondAttr
	Unused [PthreadCondAttrSize]c.Char
}

// llgo:link (*CondAttr).Init C.pthread_condattr_init
func (a *CondAttr) Init(attr *CondAttr) c.Int { return 0 }

// llgo:link (*CondAttr).Destroy C.pthread_condattr_destroy
func (a *CondAttr) Destroy() {}

// // llgo:link (*CondAttr).SetClock C.pthread_condattr_setclock
// func (a *CondAttr) SetClock(clock time.ClockidT) c.Int { return 0 }

// // llgo:link (*CondAttr).GetClock C.pthread_condattr_getclock
// func (a *CondAttr) GetClock(clock *time.ClockidT) c.Int { return 0 }

// -----------------------------------------------------------------------------

//go:linkname c_pthread_cond_init C.pthread_cond_init
func c_pthread_cond_init(c *Cond, attr *CondAttr) c.Int

//go:linkname c_pthread_cond_destroy C.pthread_cond_destroy
func c_pthread_cond_destroy(c *Cond) c.Int

//go:linkname c_pthread_cond_signal C.pthread_cond_signal
func c_pthread_cond_signal(c *Cond) c.Int

//go:linkname c_pthread_cond_broadcast C.pthread_cond_broadcast
func c_pthread_cond_broadcast(c *Cond) c.Int

//go:linkname c_pthread_cond_wait C.pthread_cond_wait
func c_pthread_cond_wait(c *Cond, m *Mutex) c.Int

//go:linkname c_pthread_cond_timedwait C.pthread_cond_timedwait
func c_pthread_cond_timedwait(c *Cond, m *Mutex, abstime *time.Timespec) c.Int

// Cond is a condition variable.
// pthread_cond_t
type Cond struct {
	_      alignCond
	Unused [PthreadCondSize]c.Char
}

func (c *Cond) Init(attr *CondAttr) c.Int {
	return c_pthread_cond_init(c, attr)
}

func (c *Cond) Destroy() {
	c_pthread_cond_destroy(c)
}

func (c *Cond) Signal() c.Int {
	return c_pthread_cond_signal(c)
}

func (c *Cond) Broadcast() c.Int {
	return c_pthread_cond_broadcast(c)
}

func (c *Cond) Wait(m *Mutex) c.Int {
	return c_pthread_cond_wait(c, m)
}

func (c *Cond) TimedWait(m *Mutex, abstime *time.Timespec) c.Int {
	return c_pthread_cond_timedwait(c, m, abstime)
}

// -----------------------------------------------------------------------------
