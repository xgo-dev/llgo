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

package pthread

import (
	_ "unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
)

func __noop__() c.Int {
	return 0
}

// -----------------------------------------------------------------------------

type aThread struct {
	Unused [8]byte
}

//llgo:type C
type RoutineFunc func(c.Pointer) c.Pointer

// Thread represents a POSIX thread.
type Thread = *aThread

// The pthread_exit() function terminates the calling thread and
// returns a value via retval that (if the thread is joinable) is
// available to another thread in the same process that calls
// pthread_join(3).
//
// See https://man7.org/linux/man-pages/man3/pthread_exit.3.html
//
//go:linkname Exit C.pthread_exit
func Exit(retval c.Pointer)

// The pthread_cancel() function sends a cancelation request to the
// thread thread.
//
// See https://man7.org/linux/man-pages/man3/pthread_cancel.3.html
//
//go:linkname Cancel C.pthread_cancel
func Cancel(thread Thread) c.Int

// The pthread_self() function returns the ID of the calling thread.
// This is the same value that is returned in *thread in the
// pthread_create(3) call that created this thread.
//
//go:linkname Self C.pthread_self
func Self() Thread

// The pthread_equal() function compares two thread identifiers.
// If the two thread IDs are equal, pthread_equal() returns a nonzero
// value; otherwise, it returns 0.
//
//go:linkname Equal C.pthread_equal
func Equal(t1, t2 Thread) c.Int

// -----------------------------------------------------------------------------

// Attr represents a POSIX thread attributes.
//
// pthread_attr_t is opaque and its exact layout varies across libc/OS targets.
// Keep enough uintptr-aligned storage for the native pthread functions instead
// of modeling individual fields.
type Attr struct {
	_ [16]uintptr
}

const (
	CreateJoinable = 0
	CreateDetached = 1
)

// llgo:link (*Attr).Init C.pthread_attr_init
func (attr *Attr) Init() c.Int { return 0 }

// llgo:link (*Attr).Destroy C.pthread_attr_destroy
func (attr *Attr) Destroy() c.Int { return 0 }

// llgo:link (*Attr).GetDetached C.pthread_attr_getdetachstate
func (attr *Attr) GetDetached(detached *c.Int) c.Int { return 0 }

// llgo:link (*Attr).SetDetached C.pthread_attr_setdetachstate
func (attr *Attr) SetDetached(detached c.Int) c.Int { return 0 }

// llgo:link (*Attr).GetStackSize C.pthread_attr_getstacksize
func (attr *Attr) GetStackSize(stackSize *uintptr) c.Int { return 0 }

// llgo:link (*Attr).SetStackSize C.pthread_attr_setstacksize
func (attr *Attr) SetStackSize(stackSize uintptr) c.Int { return 0 }

// llgo:link (*Attr).GetStackAddr C.pthread_attr_getstackaddr
func (attr *Attr) GetStackAddr(stackAddr *c.Pointer) c.Int { return 0 }

// llgo:link (*Attr).SetStackAddr C.pthread_attr_setstackaddr
func (attr *Attr) SetStackAddr(stackAddr c.Pointer) c.Int { return 0 }

// -----------------------------------------------------------------------------
// Thread Local Storage

//llgo:type C
type KeyDestructor func(c.Pointer)

// llgo:link (*Key).Create C.pthread_key_create
func (key *Key) Create(destructor KeyDestructor) c.Int { return 0 }

// llgo:link Key.Delete C.pthread_key_delete
func (key Key) Delete() c.Int { return 0 }

// llgo:link Key.Get C.pthread_getspecific
func (key Key) Get() c.Pointer { return nil }

// llgo:link Key.Set C.pthread_setspecific
func (key Key) Set(value c.Pointer) c.Int { return __noop__() }

// -----------------------------------------------------------------------------
