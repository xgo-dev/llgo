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
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/runtime"
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
