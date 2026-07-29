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

import "testing"

func TestEventLoopDispatch(t *testing.T) {
	oldPoll, oldWait := pollEvents, waitEvent
	defer func() {
		pollEvents, waitEvent = oldPoll, oldWait
	}()
	pollEvents, waitEvent = nil, nil

	if got := Poll(); got != 0 {
		t.Fatalf("Poll without an event loop = %d, want 0", got)
	}
	if Wait() {
		t.Fatal("Wait without an event loop returned true")
	}

	polls, waits := 0, 0
	installEventLoop(
		func() int {
			polls++
			return 3
		},
		func() bool {
			waits++
			return true
		},
	)
	installEventLoop(func() int { return -1 }, func() bool { return false })

	if got := Poll(); got != 3 || polls != 1 {
		t.Fatalf("Poll = %d, calls = %d, want 3, 1", got, polls)
	}
	if !Wait() || waits != 1 {
		t.Fatalf("Wait calls = %d, want true, 1", waits)
	}
}
