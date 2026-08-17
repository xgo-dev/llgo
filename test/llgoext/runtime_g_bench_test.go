//go:build llgo

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

package llgoext

import (
	"testing"
	"unsafe"
)

var runtimeDeferSink unsafe.Pointer

//go:linkname runtimeGetThreadDefer github.com/xgo-dev/llgo/runtime/internal/runtime.GetThreadDefer
func runtimeGetThreadDefer() unsafe.Pointer

func BenchmarkRuntimeGetThreadDefer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		runtimeDeferSink = runtimeGetThreadDefer()
	}
}

func BenchmarkRuntimeGoroutineEntryWithDefer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		done := make(chan struct{})
		go func() {
			defer close(done)
		}()
		<-done
	}
}
