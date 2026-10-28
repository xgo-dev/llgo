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
	"go/types"

	"github.com/xgo-dev/llgo/internal/directive"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// NewFunctionSet connects prepared source directives to this compilation's
// patch selection and explicit linker bindings. Construction and inheritance
// rules belong to directive; cl consumes only the resulting function values.
func NewFunctionSet(prog llssa.Program, patches Patches) *directive.FunctionSet {
	replacements := make(map[string]*types.Package, len(patches))
	for path, patch := range patches {
		replacements[path] = patch.Types
	}
	return directive.NewFunctionSet(prog.FunctionSources(), replacements, prog.Linkname)
}

func (p *context) sourceFunction(fn *ssa.Function) *directive.Function {
	if p.functions == nil {
		p.functions = NewFunctionSet(p.prog, p.patches)
	}
	return p.functions.Function(fn)
}
