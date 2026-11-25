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
	"testing"

	"github.com/xgo-dev/llgo/internal/llvmgen"
	"github.com/xgo-dev/llgo/internal/packages"
)

func TestDropUnusedWindowsTestMain(t *testing.T) {
	const symbol = "example.com/cmd.main"
	newFixture := func(t *testing.T) (*context, *aPackage, llvmgen.Package) {
		t.Helper()
		prog := llvmgen.NewProgram(&llvmgen.Target{GOOS: "windows", GOARCH: "arm64"})
		t.Cleanup(prog.Dispose)
		lpkg := prog.NewPackage("main", "example.com/cmd")
		missing := lpkg.NewFunc("main.missing", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
		mainFn := lpkg.NewFunc(symbol, llvmgen.NoArgsNoRet, llvmgen.GoDecl)
		body := mainFn.MakeBody(1)
		body.Call(missing.Expr)
		body.Return()
		pkg := &aPackage{Package: &packages.Package{
			Name:    "main",
			PkgPath: "example.com/cmd",
			ForTest: "example.com/cmd",
		}, LPkg: lpkg}
		ctx := &context{
			mode: ModeTest,
			prog: prog,
			buildConf: &Config{
				BuildMode: BuildModeExe,
				Goos:      "windows",
				Goarch:    "arm64",
			},
		}
		return ctx, pkg, lpkg
	}

	t.Run("unused", func(t *testing.T) {
		ctx, pkg, lpkg := newFixture(t)
		dropUnusedWindowsTestMain(ctx, pkg, lpkg.Module())
		if !lpkg.Module().NamedFunction(symbol).IsNil() {
			t.Fatalf("unused original test main was retained:\n%s", lpkg.String())
		}
	})

	t.Run("local call", func(t *testing.T) {
		ctx, pkg, lpkg := newFixture(t)
		mainFn := lpkg.NewFunc(symbol, llvmgen.NoArgsNoRet, llvmgen.GoDecl)
		caller := lpkg.NewFunc("example.com/cmd.TestMain", llvmgen.NoArgsNoRet, llvmgen.GoDecl)
		body := caller.MakeBody(1)
		body.Call(mainFn.Expr)
		body.Return()
		dropUnusedWindowsTestMain(ctx, pkg, lpkg.Module())
		if lpkg.Module().NamedFunction(symbol).IsNil() {
			t.Fatal("locally referenced original test main was removed")
		}
	})

	t.Run("linkname reference", func(t *testing.T) {
		ctx, pkg, lpkg := newFixture(t)
		ctx.prog.SetLinkname("example.com/cmd_test.callMain", symbol)
		dropUnusedWindowsTestMain(ctx, pkg, lpkg.Module())
		if lpkg.Module().NamedFunction(symbol).IsNil() {
			t.Fatal("linkname-referenced original test main was removed")
		}
	})

	t.Run("export root", func(t *testing.T) {
		ctx, pkg, lpkg := newFixture(t)
		ctx.prog.SetPackageExport(symbol, "main")
		dropUnusedWindowsTestMain(ctx, pkg, lpkg.Module())
		if lpkg.Module().NamedFunction(symbol).IsNil() {
			t.Fatal("exported original test main was removed")
		}
	})

	t.Run("non-Windows", func(t *testing.T) {
		ctx, pkg, lpkg := newFixture(t)
		ctx.buildConf.Goos = "linux"
		dropUnusedWindowsTestMain(ctx, pkg, lpkg.Module())
		if lpkg.Module().NamedFunction(symbol).IsNil() {
			t.Fatal("non-Windows original test main was removed")
		}
	})
}
