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
	"go/types"
	"reflect"
	"testing"

	"github.com/xgo-dev/llvm"

	"github.com/goplus/llgo/internal/packages"
	llssa "github.com/goplus/llgo/ssa"
)

func TestPackageSummaryCapturesLinkerFacts(t *testing.T) {
	prog := llssa.NewProgram(nil)
	defer prog.Dispose()
	lpkg := prog.NewPackage("example.com/p", "p")
	lpkg.NeedAbiInit = 3
	lpkg.RecordReflectMethodByIndex("example.com/p.Method", 4)
	lpkg.RecordReflectMethodByIndex("example.com/p.Method", 1)
	lpkg.RecordReflectMethodByName("example.com/p.MethodByName", "B")
	lpkg.RecordReflectMethodByName("example.com/p.MethodByName", "A")
	lpkg.SetExport("example.com/p.Export", "Export")
	lpkg.EmitFuncInfo("example.com/p.live", "example.com/p.Live", "p.go", 17, 2)
	lpkg.EmitPCLineInfo(42, "example.com/p.live", "p.go", 18, 3)
	lpkg.NewFunc(closureStubPrefix+"example.com/p.live", llssa.NoArgsNoRet, llssa.InGo).MakeBody(1).Return()

	i32 := lpkg.Module().Context().Int32Type()
	defined := llvm.AddGlobal(lpkg.Module(), i32, "example.com/p.defined")
	defined.SetInitializer(llvm.ConstInt(i32, 1, false))
	llvm.AddGlobal(lpkg.Module(), i32, "example.com/p.declared")

	pkg := &aPackage{
		Package: &packages.Package{
			ID:      "example.com/p",
			PkgPath: "example.com/p",
			Types:   types.NewPackage("example.com/p", "p"),
		},
		LPkg:        lpkg,
		NeedRt:      true,
		NeedPyInit:  true,
		LinkArgs:    []string{"-lp"},
		ArchiveFile: "p.a",
	}
	summary := summarizePackage(pkg)
	if got, want := summary.MethodByIndex, []int{1, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("MethodByIndex = %v, want %v", got, want)
	}
	if got, want := summary.MethodByName, []string{"A", "B"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("MethodByName = %v, want %v", got, want)
	}
	if got, want := summary.GlobalSymbols, []string{"example.com/p.defined"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("GlobalSymbols = %v, want %v", got, want)
	}
	if got, want := summary.FuncInfoStubs, []string{closureStubPrefix + "example.com/p.live"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("FuncInfoStubs = %v, want %v", got, want)
	}
	if got := collectFuncInfoSummaries([]*PackageSummary{summary}); len(got) != 1 || got[0].symbol != "example.com/p.live" {
		t.Fatalf("func info from summary = %+v, want live record", got)
	}
	if got := linkedPackageGlobals([]*PackageSummary{summary}); len(got) != 1 {
		t.Fatalf("globals from summary = %#v, want one defined global", got)
	}

	loadedPkg := &aPackage{Package: pkg.Package, LinkArgs: []string{"-lp"}, ArchiveFile: "p.a", NeedRt: true, NeedPyInit: true}
	loaded := summaryFromMetadata(loadedPkg, &cacheArchiveMetadata{Summary: summary.metadata()})
	if !reflect.DeepEqual(loaded, summary) {
		t.Fatalf("cache summary round trip = %#v, want %#v", loaded, summary)
	}
}
