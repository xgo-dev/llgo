// Copyright 2026 The XGo Authors (xgo.dev). All rights reserved.
// Use of this source code is governed by the Apache License, Version 2.0.

package build

// These records are shared by archive and frontend-cache serialization. Keep
// them free of LLVM values and Go types so archive hits need no backend replay.
type cachedFuncInfo struct {
	Symbol, Name, File  string
	Line, Column, Flags uint32
}

type cachedPCLine struct {
	ID           uint64
	Symbol, File string
	Line, Column uint32
}

const archivePCLNVersion = 1

// archivePCLNMetadata belongs to the emitted archive, not its reconstructed
// frontend module. Backend transforms such as FMV add physical functions and
// source mappings which must survive a cache hit even without frontend exports.
type archivePCLNMetadata struct {
	Version   int              `yaml:"version"`
	Functions []cachedFuncInfo `yaml:"functions,omitempty"`
	Lines     []cachedPCLine   `yaml:"lines,omitempty"`
}

type packagePCLNSnapshot struct {
	functions []funcInfoRecord
	lines     []pcLineRecord
}

func exportArchivePCLN(pkg Package) *archivePCLNMetadata {
	data := &archivePCLNMetadata{Version: archivePCLNVersion}
	for _, info := range packageFuncInfo(pkg) {
		data.Functions = append(data.Functions, cachedFuncInfo{info.symbol, info.name, info.file, info.line, info.column, info.flags})
	}
	for _, line := range packagePCLineInfo(pkg) {
		data.Lines = append(data.Lines, cachedPCLine{line.id, line.symbol, line.file, line.line, line.column})
	}
	return data
}

func (data *archivePCLNMetadata) snapshot() *packagePCLNSnapshot {
	result := &packagePCLNSnapshot{}
	for _, info := range data.Functions {
		result.functions = append(result.functions, funcInfoRecord{info.Symbol, info.Name, info.File, info.Line, info.Column, info.Flags})
	}
	for _, line := range data.Lines {
		result.lines = append(result.lines, pcLineRecord{line.ID, line.Symbol, line.File, line.Line, line.Column})
	}
	return result
}

func packageFuncInfo(pkg Package) []funcInfoRecord {
	if pkg.cachedPCLN != nil {
		return pkg.cachedPCLN.functions
	}
	if pkg.linkSnapshot != nil {
		return pkg.linkSnapshot.funcInfo
	}
	if pkg.LPkg != nil {
		return readFuncInfo(pkg.LPkg.Module())
	}
	return nil
}

func packagePCLineInfo(pkg Package) []pcLineRecord {
	if pkg.cachedPCLN != nil {
		return pkg.cachedPCLN.lines
	}
	if pkg.linkSnapshot != nil {
		return pkg.linkSnapshot.pcLineInfo
	}
	if pkg.LPkg != nil {
		return readPCLineInfo(pkg.LPkg.Module())
	}
	return nil
}
