//go:build windows

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

import _ "unsafe"

const (
	LLGoPackage = "link: -lkernel32"
	LLGoFiles   = "_wrap/runtime_windows.c; _wrap/profile_windows.c; _wrap/syscall_windows.S; _wrap/stdcall_windows_go127.c; _wrap/debugtrap.c"
)

//go:linkname c_maxprocs C.llgo_maxprocs
func c_maxprocs() int32

//go:linkname c_debugtrap C.llgo_debugtrap
func c_debugtrap()

//go:linkname c_nanotime C.llgo_nanotime
func c_nanotime() int64

//go:linkname c_nanotimeInit C.llgo_nanotime_init
func c_nanotimeInit() int32

//go:linkname c_walltime C.llgo_walltime
func c_walltime(sec *int64, nsec *int32)
