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

package build

import (
	"errors"
	"go/types"
	"runtime"
	"testing"

	llssa "github.com/goplus/llgo/ssa"
)

func TestNewBackendSessionCreatesIndependentLLVMState(t *testing.T) {
	coordinator := llssa.NewProgram(&llssa.Target{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH})
	defer coordinator.Dispose()
	ctx := &context{
		prog: coordinator,
		buildConf: &Config{
			Goos:   runtime.GOOS,
			Goarch: runtime.GOARCH,
		},
	}
	first := ctx.newBackendSession()
	defer first.prog.Dispose()
	second := ctx.newBackendSession()
	defer second.prog.Dispose()
	if first.transformer == nil || second.transformer == nil {
		t.Fatal("backend session missing C ABI transformer")
	}
	firstModule := first.prog.NewPackage("first", "example.com/first").Module()
	secondModule := second.prog.NewPackage("second", "example.com/second").Module()
	if firstModule.Context().C == secondModule.Context().C {
		t.Fatal("backend sessions share an LLVM context")
	}
}

func TestRetainedBackendProgramsKeepModulesUntilExplicitDispose(t *testing.T) {
	ctx := &context{}
	pkgs := make([]*aPackage, 2)
	for i := range pkgs {
		prog := llssa.NewProgram(nil)
		pkg := &aPackage{LPkg: prog.NewPackage("p", "example.com/p")}
		pkgs[i] = pkg
		ctx.retainBackendProgram(pkg, prog)
	}

	if got := len(ctx.retained.programs); got != len(pkgs) {
		t.Fatalf("retained Programs = %d, want %d", got, len(pkgs))
	}
	for _, pkg := range pkgs {
		if pkg.LPkg == nil || pkg.LPkg.Module().IsNil() {
			t.Fatal("retained package module was cleared before whole-program use")
		}
	}

	ctx.disposeRetainedBackendPrograms()
	if got := len(ctx.retained.programs); got != 0 {
		t.Fatalf("retained Programs after dispose = %d, want 0", got)
	}
	for _, pkg := range pkgs {
		if pkg.LPkg != nil {
			t.Fatal("retained package still references LPkg after dispose")
		}
	}
	ctx.disposeRetainedBackendPrograms()
}

func TestRetainedBackendProgramsReleaseOnErrorAndPanic(t *testing.T) {
	retain := func(ctx *context) *aPackage {
		prog := llssa.NewProgram(nil)
		pkg := &aPackage{LPkg: prog.NewPackage("p", "example.com/p")}
		ctx.retainBackendProgram(pkg, prog)
		return pkg
	}

	ctx := &context{}
	errorPkg := retain(ctx)
	wantErr := errors.New("link failed")
	runError := func() (err error) {
		defer ctx.disposeRetainedBackendPrograms()
		return wantErr
	}
	if err := runError(); !errors.Is(err, wantErr) {
		t.Fatalf("error unwind = %v, want %v", err, wantErr)
	}
	if errorPkg.LPkg != nil || len(ctx.retained.programs) != 0 {
		t.Fatal("error unwind retained a backend Program")
	}

	panicPkg := retain(ctx)
	func() {
		defer func() {
			if got := recover(); got != "link panic" {
				t.Fatalf("panic unwind recovered %v", got)
			}
		}()
		defer ctx.disposeRetainedBackendPrograms()
		panic("link panic")
	}()
	if panicPkg.LPkg != nil || len(ctx.retained.programs) != 0 {
		t.Fatal("panic unwind retained a backend Program")
	}
}
