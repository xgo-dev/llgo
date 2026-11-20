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

// Package thread exposes the hosted runtime's native thread and TLS backend.
package thread

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const LLGoPackage = "link"

//llgo:cdecl
type RoutineFunc func(c.Pointer) c.Pointer

//llgo:cdecl
type KeyDestructor func(c.Pointer)

type nativeThread *struct {
	unused [8]byte
}

// pthread_attr_t is opaque and varies across Unix targets. Sixteen pointer
// slots preserve the existing conservative storage bound and native alignment.
type nativeAttr struct {
	_ [16]uintptr
}

// llgo:link (*nativeAttr).Init C.pthread_attr_init
func (attr *nativeAttr) Init() c.Int { return 0 }

// llgo:link (*nativeAttr).Destroy C.pthread_attr_destroy
func (attr *nativeAttr) Destroy() c.Int { return 0 }

// llgo:link (*nativeAttr).SetDetached C.pthread_attr_setdetachstate
func (attr *nativeAttr) SetDetached(detached c.Int) c.Int { return 0 }

// llgo:link (*nativeAttr).SetStackSize C.pthread_attr_setstacksize
func (attr *nativeAttr) SetStackSize(stackSize uintptr) c.Int { return 0 }

//go:linkname pthreadExit C.pthread_exit
func pthreadExit(retval c.Pointer)

func Exit() {
	pthreadExit(nil)
}

// llgo:link (*Key).Create C.pthread_key_create
func (key *Key) Create(destructor KeyDestructor) c.Int { return 0 }

// llgo:link Key.Delete C.pthread_key_delete
func (key Key) Delete() c.Int { return 0 }

// llgo:link Key.Get C.pthread_getspecific
func (key Key) Get() c.Pointer { return nil }

// llgo:link Key.Set C.pthread_setspecific
func (key Key) Set(value c.Pointer) c.Int { return 0 }
