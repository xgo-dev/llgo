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

package cl

import (
	"github.com/xgo-dev/llgo/cmd/compiler/ssaopt"
	"github.com/xgo-dev/llgo/ssa"
)

func uintptrEscapesRoots(fn *ssa.Function) map[ssa.Value]struct{} {
	return ssaopt.UintptrEscapesRoots(fn)
}

func uintptrEscapesBackingArray(alloc *ssa.Alloc) bool {
	return ssaopt.UintptrEscapesBackingArray(alloc)
}

func hasUintptrEscapesDirective(fn *ssa.Function) bool {
	return ssaopt.HasUintptrEscapesDirective(fn)
}
