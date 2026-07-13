//go:build !llgo || baremetal

/*
 * Copyright (c) 2025 The XGo Authors (xgo.dev). All rights reserved.
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

package tls

// Handle is a no-op TLS handle used when building without the llgo tag.
type Handle[T any] struct{}

// Alloc returns a stub TLS handle that ignores all operations.
func Alloc[T any](func(*T)) Handle[T] { return Handle[T]{} }

// Get always returns the zero value.
func (Handle[T]) Get() (zero T) { return zero }

// Local returns an isolated zero value in non-LLGo builds.
func (Handle[T]) Local() *T { return new(T) }

// Set is a no-op.
func (Handle[T]) Set(T) {}

// Clear is a no-op.
func (Handle[T]) Clear() {}
