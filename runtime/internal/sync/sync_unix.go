//go:build !windows

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
)

const LLGoPackage = "link"

type align4 struct {
	_ [0]uint32
}

type align8 struct {
	_ [0]uint64
}

type alignPtr struct {
	_ [0]uintptr
}

// Once has the native layout of pthread_once_t.
type Once struct {
	_      alignOnce
	unused [pthreadOnceSize]c.Char
}

// MutexAttr has the native layout of pthread_mutexattr_t.
type MutexAttr struct {
	_      alignMutexAttr
	unused [pthreadMutexAttrSize]c.Char
}

// Mutex has the native layout of pthread_mutex_t.
type Mutex struct {
	_      alignMutex
	unused [pthreadMutexSize]c.Char
}

func (m *Mutex) Init(attr *MutexAttr) c.Int {
	return pthreadMutexInit(m, attr)
}

func (m *Mutex) Destroy() {
	pthreadMutexDestroy(m)
}

func (m *Mutex) Lock() {
	mutexLock(m)
}

func (m *Mutex) TryLock() bool {
	return pthreadMutexTryLock(m) == 0
}

func (m *Mutex) Unlock() {
	pthreadMutexUnlock(m)
}

// CondAttr has the native layout of pthread_condattr_t.
type CondAttr struct {
	_      alignCondAttr
	unused [pthreadCondAttrSize]c.Char
}

// Cond has the native layout of pthread_cond_t.
type Cond struct {
	_      alignCond
	unused [pthreadCondSize]c.Char
}

func (cond *Cond) Init(attr *CondAttr) c.Int {
	return pthreadCondInit(cond, attr)
}

func (cond *Cond) Destroy() {
	pthreadCondDestroy(cond)
}

func (cond *Cond) Signal() c.Int {
	return pthreadCondSignal(cond)
}

func (cond *Cond) Broadcast() c.Int {
	return pthreadCondBroadcast(cond)
}

func (cond *Cond) Wait(m *Mutex) c.Int {
	return condWait(cond, m)
}
