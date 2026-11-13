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

package atomic

import (
	"unsafe"
)

const (
	LLGoPackage = "decl"
)

type valtype interface {
	~int | ~uint | ~uintptr | ~int32 | ~uint32 | ~int64 | ~uint64 | ~unsafe.Pointer
}

// llgo:link Add llgo.atomicAdd
func Add[T valtype](ptr *T, v T) T { return v }

// llgo:link Sub llgo.atomicSub
func Sub[T valtype](ptr *T, v T) T { return v }

// llgo:link And llgo.atomicAnd
func And[T valtype](ptr *T, v T) T { return v }

// llgo:link NotAnd llgo.atomicNand
func NotAnd[T valtype](ptr *T, v T) T { return v }

// llgo:link Or llgo.atomicOr
func Or[T valtype](ptr *T, v T) T { return v }

// llgo:link Xor llgo.atomicXor
func Xor[T valtype](ptr *T, v T) T { return v }

// llgo:link Max llgo.atomicMax
func Max[T valtype](ptr *T, v T) T { return v }

// llgo:link Min llgo.atomicMin
func Min[T valtype](ptr *T, v T) T { return v }

// llgo:link UMax llgo.atomicUMax
func UMax[T valtype](ptr *T, v T) T { return v }

// llgo:link UMin llgo.atomicUMin
func UMin[T valtype](ptr *T, v T) T { return v }

// llgo:link Load llgo.atomicLoad
func Load[T valtype](ptr *T) T { return *ptr }

// llgo:link Store llgo.atomicStore
func Store[T valtype](ptr *T, v T) {}

// StoreUnordered performs an indivisible store without synchronization or
// ordering guarantees. It is intended for runtime memory operations that must
// prevent torn words; it must not be used to publish other memory accesses.
//
// llgo:link StoreUnordered llgo.atomicStoreUnordered
func StoreUnordered[T valtype](ptr *T, v T) {}

// llgo:link Exchange llgo.atomicXchg
func Exchange[T valtype](ptr *T, v T) T { return v }

// llgo:link CompareAndExchange llgo.atomicCmpXchg
func CompareAndExchange[T valtype](ptr *T, old, new T) (T, bool) { return old, false }
