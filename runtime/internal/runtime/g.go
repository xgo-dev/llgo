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

// SetPanicOnFault updates the fault behavior requested by the current
// goroutine and reports its previous value.
func SetPanicOnFault(in bool) (out bool) {
	gp := getg()
	out = gp.paniconfault
	gp.paniconfault = in
	return out
}

// SetThreadDefer associates the current goroutine with the given defer chain.
func SetThreadDefer(head *Defer) {
	getg().defer_ = head
}

// GetThreadDefer returns the current goroutine's defer chain head.
func GetThreadDefer() *Defer {
	return getg().defer_
}

// ClearThreadDefer resets the current goroutine's defer chain to nil.
func ClearThreadDefer() {
	getg().defer_ = nil
}

func goid() uint64 {
	return getg().goid
}
