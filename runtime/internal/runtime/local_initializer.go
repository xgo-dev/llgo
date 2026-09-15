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

import "unsafe"

// The compiler emits localInitReady directly from cl/locality_lower.go. Keep
// these numeric values stable and update the compiler constant if they change.
const (
	localInitUninitialized uint8 = iota
	localInitInitializing
	localInitReady
	localInitFailed
)

// EnsureLocalInitializer executes one package/locality dispatcher at most once
// in the current owner. Recursive access observes partial initialization; a
// recovered failure remains failed and re-panics on every later access.
func EnsureLocalInitializer(state *uint8, failureCache *uintptr, initialize func()) {
	ensureLocalInitializer(state, nil, failureCache, initialize)
}

// EnsureGoroutineLocalInitializer uses failure storage from the logical G's
// package block. It must not route that state through LocalPackage, whose cache
// belongs to the physical worker.
func EnsureGoroutineLocalInitializer(state *uint8, failure *any, initialize func()) {
	ensureLocalInitializer(state, failure, nil, initialize)
}

func ensureLocalInitializer(state *uint8, failure *any, failureCache *uintptr, initialize func()) {
	switch *state {
	case localInitReady:
		return
	case localInitInitializing:
		return
	case localInitFailed:
		panic(*localInitializerFailureStorage(failure, failureCache))
	case localInitUninitialized:
	default:
		panic("runtime: invalid local initializer state")
	}
	*state = localInitInitializing
	completed := false
	defer func() {
		if completed {
			return
		}
		value := recover()
		*localInitializerFailureStorage(failure, failureCache) = value
		*state = localInitFailed
		panic(value)
	}()
	initialize()
	completed = true
	*state = localInitReady
}

func localInitializerFailureStorage(failure *any, cache *uintptr) *any {
	if failure != nil {
		return failure
	}
	return localInitializerFailure(cache)
}

func localInitializerFailure(cache *uintptr) *any {
	var value any
	return (*any)(LocalPackage(cache, unsafe.Sizeof(value), unsafe.Alignof(value)))
}
