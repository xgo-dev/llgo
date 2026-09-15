//go:build !llgo || baremetal

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

// Bare-metal runtimes have one execution context and must not introduce a
// native TLS relocation for the locality package cache.
var callerLocationStoreCurrent *callerLocationStore

// This source set has one global caller store, so its local index is already
// process-unique and does not require an atomic operation.
func nextCallerPCBase(store *callerLocationStore) uintptr {
	return uintptr(len(store.synthetic)+1) << 2
}
