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

	buildfuncinfo "github.com/goplus/llgo/internal/build/funcinfo"
	llssa "github.com/goplus/llgo/ssa"
)

const (
	funcInfoTableSymbol             = "__llgo_funcinfo_table"
	funcInfoCountSymbol             = "__llgo_funcinfo_count"
	funcInfoStringsSymbol           = "__llgo_funcinfo_strings"
	funcInfoStringOffsetsSymbol     = "__llgo_funcinfo_string_offsets"
	funcInfoStringCountSymbol       = "__llgo_funcinfo_string_count"
	funcInfoHashSymbol              = "__llgo_funcinfo_hash"
	funcInfoHashMaskSymbol          = "__llgo_funcinfo_hash_mask"
	funcInfoStubIndexesSymbol       = "__llgo_funcinfo_stub_indexes"
	funcInfoStubCountSymbol         = "__llgo_funcinfo_stub_count"
	pcLineTableSymbol               = "__llgo_pcline_table"
	pcLineCountSymbol               = "__llgo_pcline_count"
	pcSiteStartPtrSymbol            = "__llgo_pcsite_start"
	pcSiteEndPtrSymbol              = "__llgo_pcsite_end"
	pcSiteStartSymbol               = "__start_llgo_pcline"
	pcSiteEndSymbol                 = "__stop_llgo_pcline"
	funcInfoDataSymbol              = "__llgo_funcinfo_table$data"
	pcLineDataSymbol                = "__llgo_pcline_table$data"
	funcInfoStringsDataSymbol       = "__llgo_funcinfo_strings$data"
	funcInfoStringOffsetsDataSymbol = "__llgo_funcinfo_string_offsets$data"
	funcInfoHashDataSymbol          = "__llgo_funcinfo_hash$data"
	funcInfoStubIndexesDataSymbol   = "__llgo_funcinfo_stub_indexes$data"
	closureStubPrefix               = "__llgo_stub."
)

type funcInfoRecord struct {
	symbol string
	name   string
	file   string
	line   uint32
	column uint32
}

type pcLineRecord struct {
	id     uint64
	symbol string
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

func collectPCLineInfo(pkgs []Package) []pcLineRecord {
	var out []pcLineRecord
	seen := make(map[uint64]none)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.LPkg == nil {
			continue
		}
		for _, rec := range readPCLineInfo(pkg.LPkg.Module()) {
			if rec.id == 0 || rec.symbol == "" {
				continue
			}
			if _, ok := seen[rec.id]; ok {
				continue
			}
			seen[rec.id] = none{}
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].symbol != out[j].symbol {
			return out[i].symbol < out[j].symbol
		}
		if out[i].line != out[j].line {
			return out[i].line < out[j].line
		}
		return out[i].id < out[j].id
	})
	return out
}

func collectFuncInfoStubIndexes(pkgs []Package, records []funcInfoRecord) []uint32 {
	if len(records) == 0 {
		return nil
	}
	recordBySymbol := make(map[string]uint32, len(records))
	for i, rec := range records {
		if rec.symbol != "" {
			recordBySymbol[rec.symbol] = uint32(i + 1)
		}
	}
	seen := make(map[uint32]none)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.LPkg == nil {
			continue
		}
		fn := pkg.LPkg.Module().FirstFunction()
		for !fn.IsNil() {
			name := fn.Name()
			if target, ok := strings.CutPrefix(name, closureStubPrefix); ok {
				if idx := recordBySymbol[target]; idx != 0 {
					seen[idx] = none{}
				}
			}
			fn = llvm.NextFunction(fn)
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]uint32, 0, len(seen))
	for idx := range seen {
		out = append(out, idx)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
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

func readPCLineInfo(mod llvm.Module) []pcLineRecord {
	rows := mod.NamedMetadataOperands(llssa.PCLineMetadataName)
	if len(rows) == 0 {
		return nil
	}
	out := make([]pcLineRecord, 0, len(rows))
	for _, row := range rows {
		fields := row.MDNodeOperands()
		if len(fields) != 6 || fields[0].ZExtValue() != 1 {
			continue
		}
		if !fields[2].IsAMDString() || !fields[3].IsAMDString() {
			continue
		}
		out = append(out, pcLineRecord{
			id:     fields[1].ZExtValue(),
			symbol: fields[2].MDString(),
			file:   fields[3].MDString(),
			line:   uint32(fields[4].ZExtValue()),
			column: uint32(fields[5].ZExtValue()),
		})
	}
	return out
}

func emitFuncInfoTable(ctx *context, pkg llssa.Package, records []funcInfoRecord, pcLines []pcLineRecord, stubIndexes []uint32) {
	mod := pkg.Module()
	llvmCtx := mod.Context()
	i8Type := llvmCtx.Int8Type()
	i16Type := llvmCtx.Int16Type()
	i32Type := llvmCtx.Int32Type()
	i64Type := llvmCtx.Int64Type()
	countType := llvmCtx.IntType(ctx.prog.PointerSize() * 8)
	recordType := llvmCtx.StructType([]llvm.Type{
		i16Type,
		i16Type,
		i16Type,
		i16Type,
		i16Type,
		i16Type,
		i32Type,
	}, false)
	pcLineRecordType := llvmCtx.StructType([]llvm.Type{
		i64Type,
		i32Type,
		i32Type,
		i32Type,
	}, false)
	pcSiteRecordType := llvmCtx.StructType([]llvm.Type{
		llvm.PointerType(i8Type, 0),
		i64Type,
	}, false)

	tablePtr := llvm.AddGlobal(mod, llvm.PointerType(recordType, 0), funcInfoTableSymbol)
	pcLinePtr := llvm.AddGlobal(mod, llvm.PointerType(pcLineRecordType, 0), pcLineTableSymbol)
	pcSiteStartPtr := llvm.AddGlobal(mod, llvm.PointerType(pcSiteRecordType, 0), pcSiteStartPtrSymbol)
	pcSiteEndPtr := llvm.AddGlobal(mod, llvm.PointerType(pcSiteRecordType, 0), pcSiteEndPtrSymbol)
	stringsPtr := llvm.AddGlobal(mod, llvm.PointerType(i8Type, 0), funcInfoStringsSymbol)
	stringOffsetsPtr := llvm.AddGlobal(mod, llvm.PointerType(i32Type, 0), funcInfoStringOffsetsSymbol)
	stringCount := llvm.AddGlobal(mod, countType, funcInfoStringCountSymbol)
	hashPtr := llvm.AddGlobal(mod, llvm.PointerType(i16Type, 0), funcInfoHashSymbol)
	count := llvm.AddGlobal(mod, countType, funcInfoCountSymbol)
	stubIndexesPtr := llvm.AddGlobal(mod, llvm.PointerType(i32Type, 0), funcInfoStubIndexesSymbol)
	stubCount := llvm.AddGlobal(mod, countType, funcInfoStubCountSymbol)
	pcLineCount := llvm.AddGlobal(mod, countType, pcLineCountSymbol)
	hashMask := llvm.AddGlobal(mod, countType, funcInfoHashMaskSymbol)
	if len(records) == 0 && len(pcLines) == 0 {
		tablePtr.SetInitializer(llvm.ConstPointerNull(tablePtr.GlobalValueType()))
		pcLinePtr.SetInitializer(llvm.ConstPointerNull(pcLinePtr.GlobalValueType()))
		pcSiteStartPtr.SetInitializer(llvm.ConstPointerNull(pcSiteStartPtr.GlobalValueType()))
		pcSiteEndPtr.SetInitializer(llvm.ConstPointerNull(pcSiteEndPtr.GlobalValueType()))
		stringsPtr.SetInitializer(llvm.ConstPointerNull(stringsPtr.GlobalValueType()))
		stringOffsetsPtr.SetInitializer(llvm.ConstPointerNull(stringOffsetsPtr.GlobalValueType()))
		stringCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		hashPtr.SetInitializer(llvm.ConstPointerNull(hashPtr.GlobalValueType()))
		count.SetInitializer(llvm.ConstInt(countType, 0, false))
		stubIndexesPtr.SetInitializer(llvm.ConstPointerNull(stubIndexesPtr.GlobalValueType()))
		stubCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		pcLineCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		hashMask.SetInitializer(llvm.ConstInt(countType, 0, false))
		return
	}

	encoded, err := buildfuncinfo.EncodeWithPCLines(toFuncInfoRecords(records), toPCLineRecords(pcLines))
	if err != nil {
		panic(err)
	}
	if len(encoded.Records) == 0 && len(encoded.PCLines) == 0 {
		tablePtr.SetInitializer(llvm.ConstPointerNull(tablePtr.GlobalValueType()))
		pcLinePtr.SetInitializer(llvm.ConstPointerNull(pcLinePtr.GlobalValueType()))
		pcSiteStartPtr.SetInitializer(llvm.ConstPointerNull(pcSiteStartPtr.GlobalValueType()))
		pcSiteEndPtr.SetInitializer(llvm.ConstPointerNull(pcSiteEndPtr.GlobalValueType()))
		stringsPtr.SetInitializer(llvm.ConstPointerNull(stringsPtr.GlobalValueType()))
		stringOffsetsPtr.SetInitializer(llvm.ConstPointerNull(stringOffsetsPtr.GlobalValueType()))
		stringCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		hashPtr.SetInitializer(llvm.ConstPointerNull(hashPtr.GlobalValueType()))
		count.SetInitializer(llvm.ConstInt(countType, 0, false))
		stubIndexesPtr.SetInitializer(llvm.ConstPointerNull(stubIndexesPtr.GlobalValueType()))
		stubCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		pcLineCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		hashMask.SetInitializer(llvm.ConstInt(countType, 0, false))
		return
	}

	values := make([]llvm.Value, 0, len(encoded.Records))
	for _, rec := range encoded.Records {
		values = append(values, llvm.ConstNamedStruct(recordType, []llvm.Value{
			llvm.ConstInt(i16Type, uint64(rec.SymbolPkg), false),
			llvm.ConstInt(i16Type, uint64(rec.SymbolName), false),
			llvm.ConstInt(i16Type, uint64(rec.NamePkg), false),
			llvm.ConstInt(i16Type, uint64(rec.NameName), false),
			llvm.ConstInt(i16Type, uint64(rec.FileRoot), false),
			llvm.ConstInt(i16Type, uint64(rec.FileName), false),
			llvm.ConstInt(i32Type, uint64(rec.Line), false),
		}))
	}
	arrayType := llvm.ArrayType(recordType, len(values))
	data := llvm.AddGlobal(mod, arrayType, funcInfoDataSymbol)
	data.SetInitializer(llvm.ConstArray(recordType, values))
	data.SetLinkage(llvm.PrivateLinkage)
	data.SetGlobalConstant(true)
	data.SetUnnamedAddr(true)
	data.SetAlignment(4)

	pcLineValues := make([]llvm.Value, 0, len(encoded.PCLines))
	for _, rec := range encoded.PCLines {
		pcLineValues = append(pcLineValues, llvm.ConstNamedStruct(pcLineRecordType, []llvm.Value{
			llvm.ConstInt(i64Type, rec.ID, false),
			llvm.ConstInt(i32Type, uint64(rec.Func), false),
			llvm.ConstInt(i32Type, uint64(rec.File), false),
			llvm.ConstInt(i32Type, uint64(rec.Line), false),
		}))
	}
	if len(pcLineValues) == 0 {
		pcLinePtr.SetInitializer(llvm.ConstPointerNull(pcLinePtr.GlobalValueType()))
		pcLineCount.SetInitializer(llvm.ConstInt(countType, 0, false))
		pcSiteStartPtr.SetInitializer(llvm.ConstPointerNull(pcSiteStartPtr.GlobalValueType()))
		pcSiteEndPtr.SetInitializer(llvm.ConstPointerNull(pcSiteEndPtr.GlobalValueType()))
	} else {
		pcLineArrayType := llvm.ArrayType(pcLineRecordType, len(pcLineValues))
		pcLineData := llvm.AddGlobal(mod, pcLineArrayType, pcLineDataSymbol)
		pcLineData.SetInitializer(llvm.ConstArray(pcLineRecordType, pcLineValues))
		pcLineData.SetLinkage(llvm.PrivateLinkage)
		pcLineData.SetGlobalConstant(true)
		pcLineData.SetUnnamedAddr(true)
		pcLineData.SetAlignment(8)
		pcLinePtr.SetInitializer(llvm.ConstInBoundsGEP(pcLineArrayType, pcLineData, []llvm.Value{
			llvm.ConstInt(countType, 0, false),
			llvm.ConstInt(countType, 0, false),
		}))
		pcLineCount.SetInitializer(llvm.ConstInt(countType, uint64(len(encoded.PCLines)), false))
		if shouldEmitRuntimeELFSites(ctx) {
			pcSiteStart := llvm.AddGlobal(mod, pcSiteRecordType, pcSiteStartSymbol)
			pcSiteEnd := llvm.AddGlobal(mod, pcSiteRecordType, pcSiteEndSymbol)
			pcSiteStartPtr.SetInitializer(pcSiteStart)
			pcSiteEndPtr.SetInitializer(pcSiteEnd)
		} else {
			pcSiteStartPtr.SetInitializer(llvm.ConstPointerNull(pcSiteStartPtr.GlobalValueType()))
			pcSiteEndPtr.SetInitializer(llvm.ConstPointerNull(pcSiteEndPtr.GlobalValueType()))
		}
	}
	emitRuntimeFuncInfoSentinels(mod, ctx.prog.PointerSize(), shouldEmitRuntimeELFSites(ctx) && len(pcLineValues) != 0)

	stringArrayType := llvm.ArrayType(i8Type, len(encoded.Strings))
	stringData := llvm.AddGlobal(mod, stringArrayType, funcInfoStringsDataSymbol)
	stringData.SetInitializer(llvmCtx.ConstString(string(encoded.Strings), false))
	stringData.SetLinkage(llvm.PrivateLinkage)
	stringData.SetGlobalConstant(true)
	stringData.SetUnnamedAddr(true)
	stringData.SetAlignment(1)

	stringOffsetValues := make([]llvm.Value, 0, len(encoded.StringOffsets))
	for _, off := range encoded.StringOffsets {
		stringOffsetValues = append(stringOffsetValues, llvm.ConstInt(i32Type, uint64(off), false))
	}
	stringOffsetsArrayType := llvm.ArrayType(i32Type, len(stringOffsetValues))
	stringOffsetsData := llvm.AddGlobal(mod, stringOffsetsArrayType, funcInfoStringOffsetsDataSymbol)
	stringOffsetsData.SetInitializer(llvm.ConstArray(i32Type, stringOffsetValues))
	stringOffsetsData.SetLinkage(llvm.PrivateLinkage)
	stringOffsetsData.SetGlobalConstant(true)
	stringOffsetsData.SetUnnamedAddr(true)
	stringOffsetsData.SetAlignment(4)

	tablePtr.SetInitializer(llvm.ConstInBoundsGEP(arrayType, data, []llvm.Value{
		llvm.ConstInt(countType, 0, false),
		llvm.ConstInt(countType, 0, false),
	}))
	stringsPtr.SetInitializer(llvm.ConstInBoundsGEP(stringArrayType, stringData, []llvm.Value{
		llvm.ConstInt(countType, 0, false),
		llvm.ConstInt(countType, 0, false),
	}))
	stringOffsetsPtr.SetInitializer(llvm.ConstInBoundsGEP(stringOffsetsArrayType, stringOffsetsData, []llvm.Value{
		llvm.ConstInt(countType, 0, false),
		llvm.ConstInt(countType, 0, false),
	}))
	stringCount.SetInitializer(llvm.ConstInt(countType, uint64(len(encoded.StringOffsets)), false))
	if len(encoded.Hash) == 0 {
		hashPtr.SetInitializer(llvm.ConstPointerNull(hashPtr.GlobalValueType()))
		hashMask.SetInitializer(llvm.ConstInt(countType, 0, false))
	} else {
		hashValues := make([]llvm.Value, 0, len(encoded.Hash))
		for _, idx := range encoded.Hash {
			hashValues = append(hashValues, llvm.ConstInt(i16Type, uint64(idx), false))
		}
		hashArrayType := llvm.ArrayType(i16Type, len(hashValues))
		hashData := llvm.AddGlobal(mod, hashArrayType, funcInfoHashDataSymbol)
		hashData.SetInitializer(llvm.ConstArray(i16Type, hashValues))
		hashData.SetLinkage(llvm.PrivateLinkage)
		hashData.SetGlobalConstant(true)
		hashData.SetUnnamedAddr(true)
		hashData.SetAlignment(2)
		hashPtr.SetInitializer(llvm.ConstInBoundsGEP(hashArrayType, hashData, []llvm.Value{
			llvm.ConstInt(countType, 0, false),
			llvm.ConstInt(countType, 0, false),
		}))
		hashMask.SetInitializer(llvm.ConstInt(countType, uint64(len(encoded.Hash)-1), false))
	}
	count.SetInitializer(llvm.ConstInt(countType, uint64(len(encoded.Records)), false))
	stubIndexValues := make([]llvm.Value, 0, len(stubIndexes))
	for _, idx := range stubIndexes {
		if idx == 0 || int(idx) > len(encoded.Records) {
			continue
		}
		stubIndexValues = append(stubIndexValues, llvm.ConstInt(i32Type, uint64(idx), false))
	}
	if len(stubIndexValues) == 0 {
		stubIndexesPtr.SetInitializer(llvm.ConstPointerNull(stubIndexesPtr.GlobalValueType()))
		stubCount.SetInitializer(llvm.ConstInt(countType, 0, false))
	} else {
		stubIndexArrayType := llvm.ArrayType(i32Type, len(stubIndexValues))
		stubIndexData := llvm.AddGlobal(mod, stubIndexArrayType, funcInfoStubIndexesDataSymbol)
		stubIndexData.SetInitializer(llvm.ConstArray(i32Type, stubIndexValues))
		stubIndexData.SetLinkage(llvm.PrivateLinkage)
		stubIndexData.SetGlobalConstant(true)
		stubIndexData.SetUnnamedAddr(true)
		stubIndexData.SetAlignment(4)
		stubIndexesPtr.SetInitializer(llvm.ConstInBoundsGEP(stubIndexArrayType, stubIndexData, []llvm.Value{
			llvm.ConstInt(countType, 0, false),
			llvm.ConstInt(countType, 0, false),
		}))
		stubCount.SetInitializer(llvm.ConstInt(countType, uint64(len(stubIndexValues)), false))
	}
}

func shouldEmitRuntimeELFSites(ctx *context) bool {
	return ctx != nil &&
		ctx.buildConf != nil &&
		ctx.buildConf.Goos == "linux" &&
		ctx.buildConf.Target == ""
}

func emitRuntimeFuncInfoSentinels(mod llvm.Module, pointerSize int, pcSite bool) {
	if !pcSite {
		return
	}
	ptrDirective := ".quad"
	align := "3"
	if pointerSize == 4 {
		ptrDirective = ".long"
		align = "2"
	}
	var asm strings.Builder
	if pcSite {
		asm.WriteString(".section llgo_pcline,\"aR\",@progbits\n")
		asm.WriteString(".p2align " + align + "\n")
		asm.WriteString(ptrDirective + " 0\n")
		asm.WriteString(".quad 0\n")
	}
	mod.SetInlineAsm(asm.String())
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

func toPCLineRecords(records []pcLineRecord) []buildfuncinfo.PCLineRecord {
	out := make([]buildfuncinfo.PCLineRecord, len(records))
	for i, rec := range records {
		out[i] = buildfuncinfo.PCLineRecord{
			ID:     rec.id,
			Symbol: rec.symbol,
			File:   rec.file,
			Line:   rec.line,
			Column: rec.column,
		}
	}
	return out
}
