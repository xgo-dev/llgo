//go:build !llgo

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
	"fmt"
	"go/types"
	"strings"
	"testing"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llgo/ssa/ssatest"
	"github.com/xgo-dev/llvm"
)

func TestCompileRuntimeGoroutineStackSize(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		size       uint64
		injected   bool
	}{
		{"default", llssa.PkgRuntime, 0, true},
		{"configured", llssa.PkgRuntime, 32 << 20, true},
		{"ordinary_package", "example.com/runtime", 32 << 20, false},
		{"public_runtime", "runtime", 32 << 20, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Even the negative cases use the package name runtime: injection
			// must match the full core-runtime path, not its name or suffix.
			ssaPkg, files := buildCallerFrameSSAPackage(t, tc.path, `package runtime
var goroutineStackSize uintptr
func Size() uintptr { return goroutineStackSize }
`)
			prog := ssatest.NewProgram(t, &llssa.Target{GOOS: "linux", GOARCH: "amd64"})
			defer prog.Dispose()
			prog.TypeSizes(types.SizesFor("gc", "amd64"))
			prog.SetPthreadStackSize(tc.size)
			pkg, err := compileTestPackage(prog, ssaPkg, files)
			if err != nil {
				t.Fatal(err)
			}
			mod := pkg.Module()
			global := mod.NamedGlobal(tc.path + ".goroutineStackSize")
			if global.IsNil() {
				t.Fatalf("missing stack-size global:\n%s", mod.String())
			}
			if got := global.IsGlobalConstant(); got != tc.injected {
				t.Fatalf("global constant = %v, want %v: %s", got, tc.injected, global.String())
			}
			want := uint64(0)
			if tc.injected {
				want = tc.size
			}
			if initializer := global.Initializer(); initializer.IsNil() || initializer.ZExtValue() != want {
				t.Fatalf("global initializer = %s, want %d", global.String(), want)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
			if !tc.injected {
				return
			}

			// No package init execution or whole-program LTO is needed: a
			// normal function pass folds the immutable configuration load.
			options := llvm.NewPassBuilderOptions()
			defer options.Dispose()
			if err := mod.RunPasses("function(instcombine)", prog.TargetMachine(), options); err != nil {
				t.Fatal(err)
			}
			body := mod.NamedFunction(tc.path + ".Size").String()
			if strings.Contains(body, "load ") || !strings.Contains(body, fmt.Sprintf("ret i64 %d", want)) {
				t.Fatalf("stack-size load did not fold to %d:\n%s", want, body)
			}
			if err := llvm.VerifyModule(mod, llvm.ReturnStatusAction); err != nil {
				t.Fatal(err)
			}
		})
	}
}
