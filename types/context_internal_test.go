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

package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	gotypes "go/types"
	"testing"
)

func TestLookupErasedAliasCallConv(t *testing.T) {
	ctx := NewContext()
	fset := token.NewFileSet()
	src := `package demo
//llgo:type C
type AliasC = func(int32)
`
	f, err := parser.ParseFile(fset, "demo.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	// Calling prescan twice exercises the existing-entry update path.
	ctx.prescanTypeDirectives("example.com/demo", []*ast.File{f})
	ctx.prescanTypeDirectives("example.com/demo", []*ast.File{f})

	// Simulate a Go 1.20/1.21 go/types package where AliasC's TypeName holds an
	// erased *gotypes.Signature directly rather than a *gotypes.Alias wrapper.
	goPkg := gotypes.NewPackage("example.com/demo", "demo")
	erasedSig := gotypes.NewSignatureType(
		nil, nil, nil,
		gotypes.NewTuple(gotypes.NewParam(token.NoPos, goPkg, "", gotypes.Typ[gotypes.Int32])),
		nil, false,
	)
	otherSig := gotypes.NewSignatureType(nil, nil, nil, nil, nil, false)
	goPkg.Scope().Insert(gotypes.NewTypeName(token.NoPos, goPkg, "AliasC", erasedSig))
	ctx.registerCheckedGoPackage("example.com/demo", goPkg)
	ctx.registerCheckedGoPackage("example.com/empty", nil)

	ctx.mu.Lock()
	gotType := ctx.fromGoLocked(erasedSig, Go, false)
	otherType := ctx.fromGoLocked(otherSig, Go, false)
	ctx.mu.Unlock()

	gotSig, ok := gotType.(*Signature)
	if !ok || gotSig.CallConv() != CDecl {
		t.Fatalf("erased alias signature = %v (cc=%v), want CDecl calling convention", gotType, gotSig.CallConv())
	}
	if otherSigLL, ok := otherType.(*Signature); !ok || otherSigLL.CallConv() != Go {
		t.Fatalf("unrelated signature = %v, want Go calling convention", otherType)
	}
}
