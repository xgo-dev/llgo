//go:build darwin && !baremetal

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

import (
	_ "unsafe"
)

// Mirrors Go's runtime.nanotime1 on Darwin (sys_darwin.go): read
// CLOCK_UPTIME_RAW through clock_gettime_nsec_np. Darwin serves
// clock_gettime(CLOCK_MONOTONIC) with only microsecond granularity, while
// CLOCK_UPTIME_RAW is mach_absolute_time with full nanosecond resolution.
const _CLOCK_UPTIME_RAW = 8

//go:linkname c_clock_gettime_nsec_np C.clock_gettime_nsec_np
func c_clock_gettime_nsec_np(clockID int32) uint64

func nanotime1() int64 {
	return int64(c_clock_gettime_nsec_np(_CLOCK_UPTIME_RAW))
}
