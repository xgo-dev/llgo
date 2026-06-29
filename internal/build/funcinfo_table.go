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

	"github.com/xgo-dev/llvm"

	buildfuncinfo "github.com/goplus/llgo/internal/build/funcinfo"
	llssa "github.com/goplus/llgo/ssa"
)

const (
	funcInfoTableSymbol       = "__llgo_funcinfo_table"
	funcInfoCountSymbol       = "__llgo_funcinfo_count"
	funcInfoStringsSymbol     = "__llgo_funcinfo_strings"
	funcInfoHashSymbol        = "__llgo_funcinfo_hash"
	funcInfoHashMaskSymbol    = "__llgo_funcinfo_hash_mask"
	funcInfoDataSymbol        = "__llgo_funcinfo_table$data"
	funcInfoStringsDataSymbol = "__llgo_funcinfo_strings$data"
	funcInfoHashDataSymbol    = "__llgo_funcinfo_hash$data"
)

type funcInfoRecord struct {
	symbol string
	name   string
	file   string
	line   uint32
	column uint32
}

func collectFuncInfo(pkgs []Package) []funcInfoRecord {
	seen := make(map[string]funcInfoRecord)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.LPkg == nil {
			continue
		}
		for _, rec := range readFuncInfo(pkg.LPkg.Module()) {
			if rec.symbol == "" {
				continue
			}
			if _, ok := seen[rec.symbol]; !ok {
				seen[rec.symbol] = rec
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]funcInfoRecord, 0, len(seen))
	for _, rec := range seen {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].symbol < out[j].symbol
	})
	return out
}

func prepareFuncInfoTableRecords(records []funcInfoRecord, liveSymbols map[string]none) []funcInfoRecord {
	if len(records) == 0 {
		return nil
	}
	// A nil liveSymbols means no post-DCE live symbol set is available yet.
	// The current table is still DCE-compatible because it stores only strings,
	// never function pointers or llvm.compiler.used references. Once the linker
	// or an LTO hook exposes a live-symbol set, pass it here to drop metadata for
	// functions removed by global DCE before materializing the runtime table.
	if liveSymbols == nil {
		return records
	}
	out := records[:0]
	for _, rec := range records {
		if _, ok := liveSymbols[rec.symbol]; ok {
			out = append(out, rec)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func readFuncInfo(mod llvm.Module) []funcInfoRecord {
	rows := mod.NamedMetadataOperands(llssa.FuncInfoMetadataName)
	if len(rows) == 0 {
		return nil
	}
	out := make([]funcInfoRecord, 0, len(rows))
	for _, row := range rows {
		fields := row.MDNodeOperands()
		if len(fields) != 6 || fields[0].ZExtValue() != 1 {
			continue
		}
		if !fields[1].IsAMDString() || !fields[2].IsAMDString() || !fields[3].IsAMDString() {
			continue
		}
		out = append(out, funcInfoRecord{
			symbol: fields[1].MDString(),
			name:   fields[2].MDString(),
			file:   fields[3].MDString(),
			line:   uint32(fields[4].ZExtValue()),
			column: uint32(fields[5].ZExtValue()),
		})
	}
	return out
}

func emitFuncInfoTable(ctx *context, pkg llssa.Package, records []funcInfoRecord) {
	mod := pkg.Module()
	llvmCtx := mod.Context()
	i8Type := llvmCtx.Int8Type()
	i32Type := llvmCtx.Int32Type()
	countType := llvmCtx.IntType(ctx.prog.PointerSize() * 8)
	recordType := llvmCtx.StructType([]llvm.Type{
		i32Type,
		i32Type,
		i32Type,
		i32Type,
		i32Type,
	}, false)

	tablePtr := llvm.AddGlobal(mod, llvm.PointerType(recordType, 0), funcInfoTableSymbol)
	stringsPtr := llvm.AddGlobal(mod, llvm.PointerType(i8Type, 0), funcInfoStringsSymbol)
	hashPtr := llvm.AddGlobal(mod, llvm.PointerType(i32Type, 0), funcInfoHashSymbol)
	count := llvm.AddGlobal(mod, countType, funcInfoCountSymbol)
	hashMask := llvm.AddGlobal(mod, countType, funcInfoHashMaskSymbol)
	if len(records) == 0 {
		tablePtr.SetInitializer(llvm.ConstPointerNull(tablePtr.GlobalValueType()))
		stringsPtr.SetInitializer(llvm.ConstPointerNull(stringsPtr.GlobalValueType()))
		hashPtr.SetInitializer(llvm.ConstPointerNull(hashPtr.GlobalValueType()))
		count.SetInitializer(llvm.ConstInt(countType, 0, false))
		hashMask.SetInitializer(llvm.ConstInt(countType, 0, false))
		return
	}

	encoded, err := buildfuncinfo.Encode(toFuncInfoRecords(records))
	if err != nil {
		panic(err)
	}

	values := make([]llvm.Value, 0, len(encoded.Records))
	for _, rec := range encoded.Records {
		values = append(values, llvm.ConstNamedStruct(recordType, []llvm.Value{
			llvm.ConstInt(i32Type, uint64(rec.Symbol), false),
			llvm.ConstInt(i32Type, uint64(rec.Name), false),
			llvm.ConstInt(i32Type, uint64(rec.File), false),
			llvm.ConstInt(i32Type, uint64(rec.Line), false),
			llvm.ConstInt(i32Type, uint64(rec.Column), false),
		}))
	}
	arrayType := llvm.ArrayType(recordType, len(values))
	data := llvm.AddGlobal(mod, arrayType, funcInfoDataSymbol)
	data.SetInitializer(llvm.ConstArray(recordType, values))
	data.SetLinkage(llvm.PrivateLinkage)
	data.SetGlobalConstant(true)
	data.SetUnnamedAddr(true)
	data.SetAlignment(4)

	stringArrayType := llvm.ArrayType(i8Type, len(encoded.Strings))
	stringData := llvm.AddGlobal(mod, stringArrayType, funcInfoStringsDataSymbol)
	stringData.SetInitializer(llvmCtx.ConstString(string(encoded.Strings), false))
	stringData.SetLinkage(llvm.PrivateLinkage)
	stringData.SetGlobalConstant(true)
	stringData.SetUnnamedAddr(true)
	stringData.SetAlignment(1)

	hashValues := make([]llvm.Value, 0, len(encoded.Hash))
	for _, idx := range encoded.Hash {
		hashValues = append(hashValues, llvm.ConstInt(i32Type, uint64(idx), false))
	}
	hashArrayType := llvm.ArrayType(i32Type, len(hashValues))
	hashData := llvm.AddGlobal(mod, hashArrayType, funcInfoHashDataSymbol)
	hashData.SetInitializer(llvm.ConstArray(i32Type, hashValues))
	hashData.SetLinkage(llvm.PrivateLinkage)
	hashData.SetGlobalConstant(true)
	hashData.SetUnnamedAddr(true)
	hashData.SetAlignment(4)

	tablePtr.SetInitializer(llvm.ConstInBoundsGEP(arrayType, data, []llvm.Value{
		llvm.ConstInt(countType, 0, false),
		llvm.ConstInt(countType, 0, false),
	}))
	stringsPtr.SetInitializer(llvm.ConstInBoundsGEP(stringArrayType, stringData, []llvm.Value{
		llvm.ConstInt(countType, 0, false),
		llvm.ConstInt(countType, 0, false),
	}))
	hashPtr.SetInitializer(llvm.ConstInBoundsGEP(hashArrayType, hashData, []llvm.Value{
		llvm.ConstInt(countType, 0, false),
		llvm.ConstInt(countType, 0, false),
	}))
	count.SetInitializer(llvm.ConstInt(countType, uint64(len(encoded.Records)), false))
	hashMask.SetInitializer(llvm.ConstInt(countType, uint64(len(encoded.Hash)-1), false))
}

func toFuncInfoRecords(records []funcInfoRecord) []buildfuncinfo.Record {
	out := make([]buildfuncinfo.Record, len(records))
	for i, rec := range records {
		out[i] = buildfuncinfo.Record{
			Symbol: rec.symbol,
			Name:   rec.name,
			File:   rec.file,
			Line:   rec.line,
			Column: rec.column,
		}
	}
	return out
}
