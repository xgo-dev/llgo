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

package runner

import "time"

const (
	iterations = 40_000_000
	warmup     = 2_000_000
)

var sink [4096]*[16]byte

//go:noinline
func allocate(n int) {
	for i := 0; i < n; i++ {
		sink[i&(len(sink)-1)] = new([16]byte)
	}
}

func Run() time.Duration {
	allocate(warmup)
	start := time.Now()
	allocate(iterations)
	return time.Since(start)
}
