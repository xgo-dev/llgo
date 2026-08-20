//go:build llgo && !baremetal

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

package runtime

import (
	psync "github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
	"github.com/xgo-dev/llgo/runtime/internal/clite/sync/atomic"
)

const (
	memProfileLockUninitialized uint32 = iota
	memProfileLockInitializing
	memProfileLockReady
)

type memProfileLock struct {
	state uint32
	mu    psync.Mutex
}

func (l *memProfileLock) lock() {
	l.ensureInitialized()
	l.mu.Lock()
}

func (l *memProfileLock) unlock() {
	l.mu.Unlock()
}

func (l *memProfileLock) ensureInitialized() {
	if atomic.Load(&l.state) == memProfileLockReady {
		return
	}
	if _, won := atomic.CompareAndExchange(&l.state, memProfileLockUninitialized, memProfileLockInitializing); won {
		if l.mu.Init(nil) != 0 {
			panic("runtime: failed to initialize memory profile lock")
		}
		atomic.Store(&l.state, memProfileLockReady)
		return
	}
	for atomic.Load(&l.state) != memProfileLockReady {
	}
}
