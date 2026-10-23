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

package build

import (
	"runtime"
	"testing"
)

func TestShouldReleaseGoHeapForLLVM(t *testing.T) {
	const threshold = uint64(256 << 20)
	const released = uint64(64 << 20)
	const maximum = ^uint64(0)

	for _, test := range []struct {
		name           string
		idle, released uint64
		want           bool
	}{
		{"zero", 0, 0, false},
		{"all-released", maximum, maximum, false},
		{"underflow", 1, maximum, false},
		{"below-threshold", threshold - 1, 0, false},
		{"at-threshold", threshold, 0, true},
		{"above-threshold", threshold + 1, 0, true},
		{"subtract-released-below", released + threshold - 1, released, false},
		{"subtract-released-at", released + threshold, released, true},
		{"subtract-released-above", released + threshold + 1, released, true},
		{"maximum-unreleased", maximum, 0, true},
		{"maximum-idle-below", maximum, maximum - threshold + 1, false},
		{"maximum-idle-at", maximum, maximum - threshold, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stats := runtime.MemStats{HeapIdle: test.idle, HeapReleased: test.released}
			if got := shouldReleaseGoHeapForLLVM(stats); got != test.want {
				t.Fatalf("idle=%d released=%d: release=%t, want %t",
					test.idle, test.released, got, test.want)
			}
		})
	}
}
