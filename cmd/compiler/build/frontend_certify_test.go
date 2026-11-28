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
	"go/token"
	"testing"

	"github.com/xgo-dev/llgo/cmd/compiler/cl"
	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
	"github.com/xgo-dev/llgo/ssa"
)

func TestFrontendCertifySkipsUnfingerprintableUnusedPackage(t *testing.T) {
	badPkg := llgoFilesTestPackage(t, t.TempDir(), "nonexistent_target_only.c")
	ap := &aPackage{Package: badPkg}

	prog := llvmgen.NewProgram(nil)
	defer prog.Dispose()
	ssaProg := ssa.NewProgram(token.NewFileSet(), 0)
	ctx := &context{
		prog:           prog,
		progSSA:        ssaProg,
		buildConf:      &Config{},
		pkgByID:        map[string]Package{badPkg.ID: ap},
		callerTracking: cl.NewCallerTracking(),
	}
	f := &frontendCache{
		ctx:     ctx,
		records: map[string]*frontendExport{},
		owners:  map[*ssa.Package]Package{},
		built:   map[*ssa.Package]bool{},
		keys:    map[string]string{},
		locals:  map[string]string{},
	}

	if err := f.certify(); err != nil {
		t.Fatalf("certify aborted on unfingerprintable package: %v", err)
	}
	if !f.uncacheable[ap] {
		t.Fatal("unfingerprintable package was not marked uncacheable")
	}
	// A subsequent escapeBodyKeys call should also skip it cleanly without error.
	_ = ctx.escapeBodyKeys()
}
