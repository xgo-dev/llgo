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

package setjmp

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = "decl"
)

type (
	SigjmpBuf [SigjmpBufSize]byte
	JmpBuf    [JmpBufSize]byte
)

// -----------------------------------------------------------------------------

//go:linkname Setjmp C.setjmp
func Setjmp(env *JmpBuf) c.Int

//go:linkname Longjmp C.longjmp
func Longjmp(env *JmpBuf, val c.Int)

// -----------------------------------------------------------------------------

//go:linkname Siglongjmp C.siglongjmp
func Siglongjmp(env *SigjmpBuf, val c.Int)

// -----------------------------------------------------------------------------
