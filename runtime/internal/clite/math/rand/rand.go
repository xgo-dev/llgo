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

package rand

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

const (
	LLGoPackage = "decl"
)

// -----------------------------------------------------------------------------

//go:linkname Rand C.rand
func Rand() c.Int

//go:linkname RandR C.rand_r
func RandR(*c.Uint) c.Int

//go:linkname Srand C.srand
func Srand(c.Uint)

//go:linkname Sranddev C.sranddev
func Sranddev()

// -----------------------------------------------------------------------------

//go:linkname Random C.random
func Random() c.Long

//go:linkname Srandom C.srandom
func Srandom(c.Uint)

//go:linkname Srandomdev C.srandomdev
func Srandomdev()

// -----------------------------------------------------------------------------
