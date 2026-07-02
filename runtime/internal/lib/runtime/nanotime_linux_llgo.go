//go:build linux && !baremetal

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
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	ct "github.com/goplus/llgo/runtime/internal/clite/time"
)

// Linux CLOCK_MONOTONIC (see <linux/time.h>), which has nanosecond
// resolution. Deliberately a local constant: ct.CLOCK_MONOTONIC carries
// Darwin's id (6), which Linux interprets as CLOCK_MONOTONIC_COARSE — a
// millisecond-granularity clock that quantized every monotonic timestamp
// the runtime produced.
const _CLOCK_MONOTONIC = 1

// nanotime1 mirrors Go's runtime.nanotime1 on Linux.
func nanotime1() int64 {
	tv := (*ct.Timespec)(c.Alloca(unsafe.Sizeof(ct.Timespec{})))
	ct.ClockGettime(ct.ClockidT(_CLOCK_MONOTONIC), tv)
	return int64(tv.Sec)*1e9 + int64(tv.Nsec)
}
