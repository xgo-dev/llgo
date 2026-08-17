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

// #include <semaphore.h>
import "C"

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// Sem represents a semaphore.
type Sem C.sem_t

// initializes the unnamed semaphore at the address
// pointed to by sem.  The value argument specifies the initial
// value for the semaphore.
//
// The pshared argument indicates whether this semaphore is to be
// shared between the threads of a process, or between processes.
//
// If pshared has the value 0, then the semaphore is shared between
// the threads of a process, and should be located at some address
// that is visible to all threads (e.g., a global variable, or a
// variable allocated dynamically on the heap).
//
// If pshared is nonzero, then the semaphore is shared between
// processes, and should be located in a region of shared memory
// (see shm_open(3), mmap(2), and shmget(2)).  (Since a child
// created by fork(2) inherits its parent's memory mappings, it can
// also access the semaphore.)  Any process that can access the
// shared memory region can operate on the semaphore using
// sem_post(3), sem_wait(3), and so on.
//
// Initializing a semaphore that has already been initialized
// results in undefined behavior.
//
// llgo:link (*Sem).Init C.sem_init
func (*Sem) Init(pshared c.Int, value c.Uint) c.Int { return 0 }

// llgo:link (*Sem).Destroy C.sem_destroy
func (*Sem) Destroy() c.Int { return 0 }

// llgo:link (*Sem).Post C.sem_post
func (*Sem) Post() c.Int { return 0 }

// llgo:link (*Sem).Wait C.sem_wait
func (*Sem) Wait() c.Int { return 0 }

// llgo:link (*Sem).TryWait C.sem_trywait
func (*Sem) TryWait() c.Int { return 0 }

// llgo:link (*Sem).GetValue C.sem_getvalue
func (*Sem) GetValue(sval *c.Int) c.Int { return 0 }
