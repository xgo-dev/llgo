/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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
	"sync/atomic"
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// llgo:skipall
type _runtime struct{}

var defaultGOROOT string // set by cmd/link

func GOROOT() string {
	return defaultGOROOT
}

var buildVersion string

func Version() string {
	return buildVersion
}

func GOMAXPROCS(n int) int {
	return int(c_maxprocs())
}

func Goexit() {
	runtime.Goexit()
}

func KeepAlive(x any) {
}

//go:linkname c_write C.write
func c_write(fd c.Int, p unsafe.Pointer, n c.SizeT) c.SsizeT

func write(fd uintptr, p unsafe.Pointer, n int32) int32 {
	return int32(c_write(c.Int(fd), p, c.SizeT(n)))
}

const heapArenaBytes = 1024 * 1024

// panicking is non-zero when crashing the program for an unrecovered panic.
var panicking atomic.Uint32

// crashFD is an optional file descriptor to use for fatal panics, as
// set by debug.SetCrashOutput (see #42888). If it is a valid fd (not
// all ones), writeErr and related functions write to it in addition
// to standard error.
//
// Initialized to -1 in schedinit.
var crashFD atomic.Uintptr

//go:linkname setCrashFD
func setCrashFD(fd uintptr) uintptr {
	// Don't change the crash FD if a crash is already in progress.
	//
	// Unlike the case below, this is not required for correctness, but it
	// is generally nicer to have all of the crash output go to the same
	// place rather than getting split across two different FDs.
	if panicking.Load() > 0 {
		return ^uintptr(0)
	}

	old := crashFD.Swap(fd)

	// If we are panicking, don't return the old FD to runtime/debug for
	// closing. writeErrData may have already read the old FD from crashFD
	// before the swap and closing it would cause the write to be lost [1].
	// The old FD will never be closed, but we are about to crash anyway.
	//
	// On the writeErrData thread, panicking.Add(1) happens-before
	// crashFD.Load() [2].
	//
	// On this thread, swapping old FD for new in crashFD happens-before
	// panicking.Load() > 0.
	//
	// Therefore, if panicking.Load() == 0 here (old FD will be closed), it
	// is impossible for the writeErrData thread to observe
	// crashFD.Load() == old FD.
	//
	// [1] Or, if really unlucky, another concurrent open could reuse the
	// FD, sending the write into an unrelated file.
	//
	// [2] If gp != nil, it occurs when incrementing gp.m.dying in
	// startpanic_m. If gp == nil, we read panicking.Load() > 0, so an Add
	// must have happened-before.
	if panicking.Load() > 0 {
		return ^uintptr(0)
	}
	return old
}
