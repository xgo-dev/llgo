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

package directive

import (
	"fmt"
	"go/ast"
	"go/types"
	"sync"

	"golang.org/x/tools/go/ssa"
)

// Function contains prepared source attributes. Only NoInline has migrated to
// preparation; other pragmas and compiler-inferred policies remain separate.
type Function struct {
	NoInline bool
}

// FunctionRecords is an immutable batch collected from source declarations.
// Declaration identity and the backing map are private to this package.
type FunctionRecords struct {
	records map[*ast.FuncDecl]Function
}

// CollectFunctions snapshots every function declaration, including declarations
// without attributes. Callers publish the batch after syntax validation succeeds.
func CollectFunctions(files []*ast.File) FunctionRecords {
	records := make(map[*ast.FuncDecl]Function)
	for _, file := range files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				records[fn] = Function{NoInline: ParseFunctionPragmas(fn.Doc).NoInline}
			}
		}
	}
	return FunctionRecords{records: records}
}

// FunctionIndex owns the prepared function records for one compilation.
// Backends share it for read-only queries after preparation. Its zero value is
// ready for use; it must not be copied after first use.
type FunctionIndex struct {
	mu      sync.RWMutex
	records map[*ast.FuncDecl]Function
}

// Add publishes a batch without changing already prepared declarations.
func (i *FunctionIndex) Add(batch FunctionRecords) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.records == nil {
		i.records = make(map[*ast.FuncDecl]Function)
	}
	for decl, record := range batch.records {
		if _, exists := i.records[decl]; !exists {
			i.records[decl] = record
		}
	}
}

// PrepareSSA supplies the one-shot compiler's source dependencies, whose
// generic bodies may be lowered without compiling their declaring packages.
// Existing records, including empty attributes, remain authoritative. The
// preloaded build driver prepares syntax directly and does not need this pass.
func (i *FunctionIndex) PrepareSSA(prog *ssa.Program) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.records == nil {
		i.records = make(map[*ast.FuncDecl]Function)
	}
	prepare := func(fn *ssa.Function) {
		decl := functionDecl(fn)
		if decl == nil {
			return
		}
		if _, exists := i.records[decl]; !exists {
			i.records[decl] = Function{NoInline: ParseFunctionPragmas(decl.Doc).NoInline}
		}
	}
	for _, pkg := range prog.AllPackages() {
		for _, member := range pkg.Members {
			switch member := member.(type) {
			case *ssa.Function:
				prepare(member)
			case *ssa.Type:
				if named, ok := member.Type().(*types.Named); ok {
					for n := 0; n < named.NumMethods(); n++ {
						prepare(prog.FuncValue(named.Method(n)))
					}
				}
			}
		}
	}
}

// Function reads the attributes of fn's own source declaration. It does not
// follow origins, wrappers or enclosing functions, and never parses comments.
// Functions without a declaration have empty attributes. A source declaration
// missing from the prepared index is a compiler lifecycle error, not an empty
// result; reporting it here keeps consumers independent of source association.
func (i *FunctionIndex) Function(fn *ssa.Function) Function {
	decl := functionDecl(fn)
	if decl == nil {
		return Function{}
	}
	i.mu.RLock()
	record, prepared := i.records[decl]
	i.mu.RUnlock()
	if !prepared {
		panic(fmt.Sprintf("function directives were not prepared for %s", fn))
	}
	return record
}

func functionDecl(fn *ssa.Function) *ast.FuncDecl {
	if fn == nil {
		return nil
	}
	decl, _ := fn.Syntax().(*ast.FuncDecl)
	return decl
}
