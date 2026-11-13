// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

import (
	"fmt"
	"go/token"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/xgo-dev/llgo/internal/frontcache"
	llssa "github.com/xgo-dev/llgo/ssa"
)

type frontendABIType struct {
	Name string
	Type int
}

type frontendCExport struct {
	GoName, CName string
	Signature     int
}

type frontendFuncInfo struct {
	Symbol, Name, File  string
	Line, Column, Flags uint32
}

type frontendPCLine struct {
	ID           uint64
	Symbol, File string
	Line, Column uint32
}

// frontendLinkExport carries the metadata previously recovered by lowering a
// dependency again even when its archive was cached. The snapshot contains no
// LLVM-owned pointers and is validated against that exact archive's inputs.
type frontendLinkExport struct {
	NeedABI                 int
	MethodIndexes           []int
	MethodNames, ABISymbols []string
	Types                   frontcache.TypeGraph
	ABI                     []frontendABIType
	Functions               []frontendFuncInfo
	Lines                   []frontendPCLine
	LocalExports            bool
	CExports                []frontendCExport
}

func (f *frontendCache) exportLink(snapshot *packageLinkSnapshot) (*frontendLinkExport, error) {
	index := frontcache.NewTypes(f.ids, f.ctx.progSSA.Fset)
	graph := frontcache.NewTypeGraph(index, f.ctx.progSSA.Fset)
	result := &frontendLinkExport{NeedABI: snapshot.needAbiInit, LocalExports: snapshot.hasLocalExports}
	for key := range snapshot.methodByIndex {
		result.MethodIndexes = append(result.MethodIndexes, key)
	}
	for key := range snapshot.methodByName {
		result.MethodNames = append(result.MethodNames, key)
	}
	for key := range snapshot.abiSymbols {
		result.ABISymbols = append(result.ABISymbols, key)
	}
	sort.Ints(result.MethodIndexes)
	sort.Strings(result.MethodNames)
	sort.Strings(result.ABISymbols)
	for _, info := range snapshot.abiTypes {
		if _, owned := snapshot.abiSymbols[info.Name]; owned && info.Raw != nil {
			result.ABI = append(result.ABI, frontendABIType{Name: info.Name, Type: graph.Add(info.Raw)})
		}
	}
	for _, info := range snapshot.funcInfo {
		result.Functions = append(result.Functions, frontendFuncInfo{info.symbol, info.name, info.file, info.line, info.column, info.flags})
	}
	for _, line := range snapshot.pcLineInfo {
		result.Lines = append(result.Lines, frontendPCLine{line.id, line.symbol, line.file, line.line, line.column})
	}
	for _, export := range snapshot.cExports {
		ref := -1
		if export.sig != nil {
			ref = graph.Add(export.sig)
		}
		result.CExports = append(result.CExports, frontendCExport{export.goName, export.cName, ref})
	}
	result.Types = *graph
	return result, nil
}

func (f *frontendCache) position(key string) token.Pos {
	return frontendPosition(f.ctx.progSSA.Fset)(key)
}

func frontendPosition(fset *token.FileSet) func(string) token.Pos {
	return func(key string) token.Pos {
		colon := strings.LastIndexByte(key, ':')
		if colon < 0 {
			return token.NoPos
		}
		offset, err := strconv.Atoi(key[colon+1:])
		if err != nil || offset < 0 {
			return token.NoPos
		}
		var result token.Pos
		fset.Iterate(func(file *token.File) bool {
			if file.Name() == key[:colon] && offset <= file.Size() {
				result = file.Pos(offset)
				return false
			}
			return true
		})
		return result
	}
}

func (f *frontendCache) importLink(data *frontendLinkExport) (*packageLinkSnapshot, error) {
	if data == nil {
		return nil, fmt.Errorf("missing cached link export")
	}
	result := &packageLinkSnapshot{needAbiInit: data.NeedABI, hasLocalExports: data.LocalExports,
		methodByIndex: make(map[int]none), methodByName: make(map[string]none), abiSymbols: make(map[string]none)}
	for _, key := range data.MethodIndexes {
		if key < 0 {
			return nil, fmt.Errorf("invalid cached method index")
		}
		result.methodByIndex[key] = none{}
	}
	for _, key := range data.MethodNames {
		result.methodByName[key] = none{}
	}
	for _, key := range data.ABISymbols {
		result.abiSymbols[key] = none{}
	}
	importer := &frontcache.TypeImporter{Graph: &data.Types, Packages: f.packages, Existing: f.types, Position: f.position}
	for _, info := range data.ABI {
		typ, err := importer.Import(info.Type)
		if err != nil || info.Name == "" {
			return nil, fmt.Errorf("invalid cached ABI type %s: %v", info.Name, err)
		}
		result.abiTypes = append(result.abiTypes, llssa.AbiTypeInfo{Name: info.Name, Raw: typ})
	}
	for _, info := range data.Functions {
		result.funcInfo = append(result.funcInfo, funcInfoRecord{info.Symbol, info.Name, info.File, info.Line, info.Column, info.Flags})
	}
	for _, line := range data.Lines {
		result.pcLineInfo = append(result.pcLineInfo, pcLineRecord{line.ID, line.Symbol, line.File, line.Line, line.Column})
	}
	for _, export := range data.CExports {
		if export.Signature == -1 && export.GoName != "" && export.CName != "" {
			result.cExports = append(result.cExports, cExport{goName: export.GoName, cName: export.CName})
			continue
		}
		typ, err := importer.Import(export.Signature)
		sig, ok := typ.(*types.Signature)
		if err != nil || !ok || export.GoName == "" || export.CName == "" {
			return nil, fmt.Errorf("invalid cached C export %s: %v", export.GoName, err)
		}
		result.cExports = append(result.cExports, cExport{export.GoName, export.CName, sig})
	}
	return result, nil
}
