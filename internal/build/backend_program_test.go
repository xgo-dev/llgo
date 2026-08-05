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
