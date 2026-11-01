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

package ssa

import (
	"github.com/xgo-dev/llgo/internal/directive"
	gossa "golang.org/x/tools/go/ssa"
)

// AddFunctionDirectives publishes validated source records for backend queries.
func (p Program) AddFunctionDirectives(records directive.FunctionRecords) {
	p.packageSyntax.functions.Add(records)
}

// PrepareSSAFunctionDirectives supplies source records for one-shot compilation
// before lowering or caller analysis begins. Preloaded builds do not need it.
func (p Program) PrepareSSAFunctionDirectives(prog *gossa.Program) {
	p.packageSyntax.functions.PrepareSSA(prog)
}

// FunctionDirectives returns prepared source attributes for a Go SSA function.
// Source association and preparation checks belong to internal/directive.
func (p Program) FunctionDirectives(fn *gossa.Function) directive.Function {
	return p.packageSyntax.functions.Function(fn)
}
