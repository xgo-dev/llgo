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

import (
	_ "unsafe"

	llruntime "github.com/xgo-dev/llgo/runtime/internal/runtime"
)

// os/proc.go declares this runtime entry point on every platform. Its Windows
// init path returns before calling it because the official os/exec_windows.go
// parser initializes os.Args from GetCommandLineW instead. Keep the symbol so
// unoptimized builds do not retain an unresolved reference from the dead arm.
//
//go:linkname os_runtime_args os.runtime_args
func os_runtime_args() []string { return nil }

//go:linkname c_queryPerformanceCounter C.llgo_query_performance_counter
func c_queryPerformanceCounter() int64

//go:linkname c_queryPerformanceFrequency C.llgo_query_performance_frequency
func c_queryPerformanceFrequency() int64

// These entry points implement the runtime hooks declared by the official
// internal/syscall/windows package.

//go:linkname c_getSystemDirectory stdcall.GetSystemDirectoryA
func c_getSystemDirectory(buffer *byte, size uint32) uint32

//go:linkname windows_GetSystemDirectory internal/syscall/windows.GetSystemDirectory
func windows_GetSystemDirectory() string {
	const maxPath = 260
	var directory [maxPath + 1]byte
	length := c_getSystemDirectory(&directory[0], maxPath)
	if length == 0 || length > maxPath {
		throw("Unable to determine system directory")
	}
	directory[length] = '\\'
	return string(directory[:length+1])
}

//go:linkname windows_QueryPerformanceCounter internal/syscall/windows.QueryPerformanceCounter
func windows_QueryPerformanceCounter() int64 {
	return c_queryPerformanceCounter()
}

//go:linkname windows_QueryPerformanceFrequency internal/syscall/windows.QueryPerformanceFrequency
func windows_QueryPerformanceFrequency() int64 {
	return c_queryPerformanceFrequency()
}

// syscall.Setenv and syscall.Unsetenv have already updated the Win32
// environment before calling these hooks. LLGo only needs to propagate the
// runtime-observed GODEBUG change.

//go:linkname syscall_runtimeSetenv syscall.runtimeSetenv
func syscall_runtimeSetenv(key, value string) {
	if key == "GODEBUG" {
		godebugEnvChanged(value)
	}
}

//go:linkname syscall_runtimeUnsetenv syscall.runtimeUnsetenv
func syscall_runtimeUnsetenv(key string) {
	if key == "GODEBUG" {
		godebugEnvChanged("")
	}
}

//go:linkname os_beforeExit os.runtime_beforeExit
func os_beforeExit(exitCode int) {
	runCoverageExitHook(exitCode)
}

//go:linkname c_getpagesize C.llgo_getpagesize
func c_getpagesize() int32

//go:linkname syscall_Getpagesize syscall.Getpagesize
func syscall_Getpagesize() int {
	return int(c_getpagesize())
}

//go:linkname syscall_Exit syscall.Exit
//go:nosplit
func syscall_Exit(code int) {
	llruntime.ExitProcess(uint32(code))
}
