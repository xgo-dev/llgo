//go:build baremetal && llgo.idf && !llgo.baremetal.cstdio

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

package c

import _ "unsafe"

// On an llgo.idf target (ESP-IDF component link) the libc is the IDF
// toolchain's libc (picolibc), whose VFS has no /dev/stdout: the generic
// baremetal init Fopen("/dev/stdout", "w") would return NULL and every
// print would fwrite to a NULL FILE*. The IDF libc always defines the
// stdout/stderr/stdin globals, so reference them directly at link time -
// the same shape the hosted non-baremetal build uses in stdio_default.go.
// Aliasing the C variables here is what let app code drop its per-demo
// fixStreams() bootstrap call.

const (
	LLGoPackage = "decl"
)

//go:linkname Stdin C.stdin
var Stdin FilePtr

//go:linkname Stdout C.stdout
var Stdout FilePtr

//go:linkname Stderr C.stderr
var Stderr FilePtr
