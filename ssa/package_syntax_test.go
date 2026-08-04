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
	"go/token"
	"go/types"
	"sync"
	"testing"
)

func TestFrozenPackageSyntaxStateIsSharedWithoutWorkerOverlay(t *testing.T) {
	source := NewProgram(nil)
	defer source.Dispose()
	pkg := types.NewPackage("example.com/p", "p")
	fset := token.NewFileSet()
	const name = "example.com/p.Entry"
	source.SetLinkname(name, "entry")
	source.SetPackageExport(name, "entry")
	source.SetClosureEnvDirective(fset, name, token.Pos(7))
	source.SetNoInterfaceMethod(name)
	source.SetTypeBackground("example.com/p.CType", InC)
	source.SetClosureEnvDirective(fset, name, token.Pos(7))
	source.SetNoInterfaceMethod(name)
	source.SetTypeBackground("example.com/p.CType", InC)
	source.MarkPackageSyntaxParsed(pkg)

	state := source.FreezePackageSyntaxState()
	first := NewProgram(nil)
	defer first.Dispose()
	first.UsePackageSyntaxState(state)
	second := NewProgram(nil)
	defer second.Dispose()
	second.UsePackageSyntaxState(state)
	if first.packageSyntax != source.packageSyntax || first.packageSyntax != second.packageSyntax {
		t.Fatal("backend Programs did not share package syntax state")
	}
	if !first.PackageSyntaxParsed(pkg) {
		t.Fatal("shared state lost parsed package")
	}
	if link, ok := first.Linkname(name); !ok || link != "entry" {
		t.Fatalf("shared linkname = (%q, %v), want (entry, true)", link, ok)
	}
	if links := first.packageSyntax.linknamesSnapshot(); links[name] != "entry" {
		t.Fatalf("shared linkname snapshot = %#v", links)
	}
	if export, ok := first.PackageExport(name); !ok || export != "entry" {
		t.Fatalf("shared export = (%q, %v), want (entry, true)", export, ok)
	}
	if !first.HasClosureEnvDirective(fset, name, token.Pos(7)) {
		t.Fatal("shared state lost closure environment directive")
	}
	if _, ok := first.packageSyntax.noInterfaceMethod(name); !ok {
		t.Fatal("shared state lost nointerface directive")
	}
	if bg, ok := first.packageSyntax.typeBackground("example.com/p.CType"); !ok || bg != InC {
		t.Fatalf("shared type background = (%v, %v), want (InC, true)", bg, ok)
	}

	// Repeated lowering-time registration is allowed only when it exactly
	// matches metadata that the coordinator already collected.
	first.SetLinkname(name, "entry")
	first.SetPackageExport(name, "entry")
	first.MarkPackageSyntaxParsed(pkg)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		prog := first
		if i%2 != 0 {
			prog = second
		}
		wg.Add(1)
		go func(prog Program) {
			defer wg.Done()
			for range 100 {
				prog.SetLinkname(name, "entry")
				if link, ok := prog.Linkname(name); !ok || link != "entry" {
					t.Errorf("concurrent shared linkname = (%q, %v)", link, ok)
				}
			}
		}(prog)
	}
	wg.Wait()

	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("worker added metadata to frozen package syntax state")
		}
	}()
	first.SetLinkname("example.com/p.Missing", "missing")
}

func TestZeroPackageSyntaxStateIsFrozen(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	prog.UsePackageSyntaxState(PackageSyntaxState{})
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("zero package syntax state remained mutable")
		}
	}()
	prog.MarkPackageSyntaxParsed(types.NewPackage("example.com/missing", "missing"))
}
