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

package wasmevent

var (
	pollEvents func() int
	waitEvent  func() bool
)

// Poll runs host events that are ready without blocking.
func Poll() int {
	if pollEvents == nil {
		return 0
	}
	return pollEvents()
}

// Wait blocks until a host event is ready. It reports false when no event
// source has been activated.
func Wait() bool {
	if waitEvent == nil {
		return false
	}
	return waitEvent()
}

func installEventLoop(poll func() int, wait func() bool) {
	if pollEvents != nil {
		return
	}
	pollEvents = poll
	waitEvent = wait
}
