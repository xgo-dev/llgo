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
	"sort"
	"strings"

	"github.com/xgo-dev/llvm"
)

// PackageSummary is the immutable, LLVM-free output one package contributes
// to the final link. A backend can emit its object, capture this summary, and
// release its Program and LLVM context before whole-program linking starts.
//
// C archive/shared header declarations still consume LPkg and therefore stay
// on the serial compatibility path.
type PackageSummary struct {
	ID      string
	PkgPath string
	Name    string

	LinkArgs    []string
	ArchiveFile string
	NeedRuntime bool
	NeedPyInit  bool

	NeedAbiInit   int
	MethodByIndex []int
	MethodByName  []string
	GlobalSymbols []string

	FuncInfo       []funcInfoRecord
	PCLineInfo     []pcLineRecord
	FuncInfoStubs  []string
	CSharedExports []string
}

type packageSummaryMetadata struct {
	NeedAbiInit   int      `yaml:"need_abi_init,omitempty"`
	MethodByIndex []int    `yaml:"method_by_index,omitempty"`
	MethodByName  []string `yaml:"method_by_name,omitempty"`
	GlobalSymbols []string `yaml:"global_symbols,omitempty"`

	FuncInfo       []funcInfoMetadata `yaml:"func_info,omitempty"`
	PCLineInfo     []pcLineMetadata   `yaml:"pcline_info,omitempty"`
	FuncInfoStubs  []string           `yaml:"func_info_stubs,omitempty"`
	CSharedExports []string           `yaml:"c_shared_exports,omitempty"`
}

type funcInfoMetadata struct {
	Symbol string `yaml:"symbol"`
	Name   string `yaml:"name,omitempty"`
	File   string `yaml:"file,omitempty"`
	Line   uint32 `yaml:"line,omitempty"`
	Column uint32 `yaml:"column,omitempty"`
}

type pcLineMetadata struct {
	ID     uint64 `yaml:"id"`
	Symbol string `yaml:"symbol"`
	File   string `yaml:"file,omitempty"`
	Line   uint32 `yaml:"line,omitempty"`
	Column uint32 `yaml:"column,omitempty"`
}

func summarizePackage(pkg *aPackage) *PackageSummary {
	if pkg == nil {
		return nil
	}
	summary := &PackageSummary{
		LinkArgs:    append([]string(nil), pkg.LinkArgs...),
		ArchiveFile: pkg.ArchiveFile,
		NeedRuntime: pkg.NeedRt,
		NeedPyInit:  pkg.NeedPyInit,
	}
	if pkg.Package != nil {
		summary.ID = pkg.ID
		summary.PkgPath = pkg.PkgPath
		summary.Name = pkg.Name
	}
	if pkg.LPkg == nil {
		return summary
	}

	lpkg := pkg.LPkg
	if summary.PkgPath == "" {
		summary.PkgPath = lpkg.Path()
	}
	summary.NeedAbiInit = lpkg.NeedAbiInit
	for method := range lpkg.MethodByIndex {
		summary.MethodByIndex = append(summary.MethodByIndex, method)
	}
	sort.Ints(summary.MethodByIndex)
	for method := range lpkg.MethodByName {
		summary.MethodByName = append(summary.MethodByName, method)
	}
	sort.Strings(summary.MethodByName)

	mod := lpkg.Module()
	for global := mod.FirstGlobal(); !global.IsNil(); global = llvm.NextGlobal(global) {
		if !global.IsDeclaration() {
			summary.GlobalSymbols = append(summary.GlobalSymbols, global.Name())
		}
	}
	sort.Strings(summary.GlobalSymbols)
	summary.FuncInfo = readFuncInfo(mod)
	summary.PCLineInfo = readPCLineInfo(mod)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() || fn.BasicBlocksCount() == 0 {
			continue
		}
		if _, ok := strings.CutPrefix(fn.Name(), closureStubPrefix); ok {
			summary.FuncInfoStubs = append(summary.FuncInfoStubs, fn.Name())
		}
	}
	sort.Strings(summary.FuncInfoStubs)
	for _, name := range lpkg.ExportFuncs() {
		if name != "" {
			summary.CSharedExports = append(summary.CSharedExports, name)
		}
	}
	sort.Strings(summary.CSharedExports)
	return summary
}

func (s *PackageSummary) metadata() *packageSummaryMetadata {
	if s == nil {
		return nil
	}
	meta := &packageSummaryMetadata{
		NeedAbiInit:    s.NeedAbiInit,
		MethodByIndex:  append([]int(nil), s.MethodByIndex...),
		MethodByName:   append([]string(nil), s.MethodByName...),
		GlobalSymbols:  append([]string(nil), s.GlobalSymbols...),
		FuncInfoStubs:  append([]string(nil), s.FuncInfoStubs...),
		CSharedExports: append([]string(nil), s.CSharedExports...),
	}
	for _, rec := range s.FuncInfo {
		meta.FuncInfo = append(meta.FuncInfo, funcInfoMetadata{
			Symbol: rec.symbol,
			Name:   rec.name,
			File:   rec.file,
			Line:   rec.line,
			Column: rec.column,
		})
	}
	for _, rec := range s.PCLineInfo {
		meta.PCLineInfo = append(meta.PCLineInfo, pcLineMetadata{
			ID:     rec.id,
			Symbol: rec.symbol,
			File:   rec.file,
			Line:   rec.line,
			Column: rec.column,
		})
	}
	return meta
}

func summaryFromMetadata(pkg *aPackage, meta *cacheArchiveMetadata) *PackageSummary {
	if pkg == nil || pkg.Package == nil || meta == nil || meta.Summary == nil {
		return nil
	}
	summary := &PackageSummary{
		ID:             pkg.ID,
		PkgPath:        pkg.PkgPath,
		Name:           pkg.Name,
		LinkArgs:       append([]string(nil), pkg.LinkArgs...),
		ArchiveFile:    pkg.ArchiveFile,
		NeedRuntime:    pkg.NeedRt,
		NeedPyInit:     pkg.NeedPyInit,
		NeedAbiInit:    meta.Summary.NeedAbiInit,
		MethodByIndex:  append([]int(nil), meta.Summary.MethodByIndex...),
		MethodByName:   append([]string(nil), meta.Summary.MethodByName...),
		GlobalSymbols:  append([]string(nil), meta.Summary.GlobalSymbols...),
		FuncInfoStubs:  append([]string(nil), meta.Summary.FuncInfoStubs...),
		CSharedExports: append([]string(nil), meta.Summary.CSharedExports...),
	}
	for _, rec := range meta.Summary.FuncInfo {
		summary.FuncInfo = append(summary.FuncInfo, funcInfoRecord{
			symbol: rec.Symbol,
			name:   rec.Name,
			file:   rec.File,
			line:   rec.Line,
			column: rec.Column,
		})
	}
	for _, rec := range meta.Summary.PCLineInfo {
		summary.PCLineInfo = append(summary.PCLineInfo, pcLineRecord{
			id:     rec.ID,
			symbol: rec.Symbol,
			file:   rec.File,
			line:   rec.Line,
			column: rec.Column,
		})
	}
	return summary
}

func summariesForPackages(pkgs []Package) []*PackageSummary {
	summaries := make([]*PackageSummary, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg == nil {
			continue
		}
		summary := pkg.Summary
		if summary == nil {
			summary = summarizePackage(pkg)
		}
		if summary != nil {
			summaries = append(summaries, summary)
		}
	}
	return summaries
}
