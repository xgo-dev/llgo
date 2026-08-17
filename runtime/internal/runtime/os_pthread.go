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

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/clite/pthread"
)

// mOS is the pthread-specific part of an M.
type mOS struct {
	thread pthread.Thread
}

// newosproc provides the current host-thread backend for newm.
func newosproc(mp *m, stackSize uintptr) int {
	var attr pthread.Attr
	if ret := initThreadAttr(&attr, stackSize); ret != 0 {
		return int(ret)
	}
	ret := pthread.Create(
		&mp.os.thread,
		&attr,
		pthread.RoutineFunc(mstart),
		c.Pointer(unsafe.Pointer(mp)),
	)
	// Once Create succeeds, mp belongs to the detached thread. A destroy
	// failure cannot be reported as creation failure without freeing live data.
	_ = attr.Destroy()
	return int(ret)
}

func initThreadAttr(attr *pthread.Attr, stackSize uintptr) c.Int {
	if ret := attr.Init(); ret != 0 {
		return ret
	}
	if ret := attr.SetDetached(pthread.CreateDetached); ret != 0 {
		_ = attr.Destroy()
		return ret
	}
	if stackSize != 0 {
		if ret := attr.SetStackSize(stackSize); ret != 0 {
			_ = attr.Destroy()
			return ret
		}
	}
	return 0
}

func exitCurrentM() {
	mp := getg().m
	mexit(mp)
	pthread.Exit(nil)
}
