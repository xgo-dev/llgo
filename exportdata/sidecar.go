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

package exportdata

import (
	"io"
	"os"
	"sort"

	"github.com/xgo-dev/llgo/ssa"
)

// ParamEscapeFlags summarizes how a function parameter escapes.
type ParamEscapeFlags uint8

const (
	ParamNoEscape ParamEscapeFlags = 0
	ParamToHeap   ParamEscapeFlags = 1 << iota
	ParamToReturn
	ParamToCall
)

// FuncEscape records per-function escape analysis summary facts.
type FuncEscape struct {
	Name        string
	Params      []ParamEscapeFlags
	StackAllocs int
	HeapAllocs  int
}

// EscapeSummary holds per-package escape summaries for incremental builds.
type EscapeSummary struct {
	PkgPath string
	Funcs   []FuncEscape
}

// FuncInline records per-function inlining eligibility and cost.
type FuncInline struct {
	Name       string
	Cost       int
	Inlineable bool
	Reason     string
	Callees    []string
}

// InlineSummary holds per-package cross-package inlining summaries.
type InlineSummary struct {
	PkgPath string
	Budget  int
	Funcs   []FuncInline
}

// CallEdge records a summarized call-graph edge for incremental RTA/DCE.
type CallEdge struct {
	Caller string
	Callee string
	Kind   uint8 // 0=direct, 1=closure, 2=interface
}

// PackageMeta bundles all compact binary analysis summaries persisted alongside
// a compiled package archive so cache hits skip AST/SSA/escape recomputation.
type PackageMeta struct {
	Summary        ssa.PackageSummary
	DefinedGlobals []string
	Escape         EscapeSummary
	Inline         InlineSummary
	CallEdges      []CallEdge
	LiveTypes      []string
	ExportFuncs    []string
}

// MarshalMeta encodes meta into compact deterministic binary bytes.
func MarshalMeta(meta *PackageMeta) ([]byte, error) {
	if meta == nil {
		return nil, ErrCorrupt
	}
	e := newEncoder()
	s := &meta.Summary
	e.writeStr(s.Path)
	e.writeStr(s.TargetTriple)
	exportKeys := make([]string, 0, len(s.Exports))
	for k := range s.Exports {
		exportKeys = append(exportKeys, k)
	}
	sort.Strings(exportKeys)
	e.writeUvarint(uint64(len(exportKeys)))
	for _, k := range exportKeys {
		e.writeStr(k)
		e.writeStr(s.Exports[k])
	}
	e.writeStringSlice(sortedCopy(s.DefinedFuncs))
	e.writeStringSlice(sortedCopy(meta.DefinedGlobals))
	e.writeStringSlice(sortedCopy(s.DirectCalls))
	e.writeStringSlice(sortedCopy(s.ClosureRefs))
	e.writeStringSlice(sortedCopy(s.InterfaceCalls))
	e.writeBool(s.HasSafepoints)
	e.writeBool(s.HasWriteBarrier)

	// Escape summary.
	e.writeStr(meta.Escape.PkgPath)
	escFuncs := append([]FuncEscape(nil), meta.Escape.Funcs...)
	sort.Slice(escFuncs, func(i, j int) bool { return escFuncs[i].Name < escFuncs[j].Name })
	e.writeUvarint(uint64(len(escFuncs)))
	for _, f := range escFuncs {
		e.writeStr(f.Name)
		e.writeUvarint(uint64(len(f.Params)))
		for _, p := range f.Params {
			e.writeByte(byte(p))
		}
		e.writeVarint(int64(f.StackAllocs))
		e.writeVarint(int64(f.HeapAllocs))
	}

	// Inline summary.
	e.writeStr(meta.Inline.PkgPath)
	e.writeVarint(int64(meta.Inline.Budget))
	inlFuncs := append([]FuncInline(nil), meta.Inline.Funcs...)
	sort.Slice(inlFuncs, func(i, j int) bool { return inlFuncs[i].Name < inlFuncs[j].Name })
	e.writeUvarint(uint64(len(inlFuncs)))
	for _, f := range inlFuncs {
		e.writeStr(f.Name)
		e.writeVarint(int64(f.Cost))
		e.writeBool(f.Inlineable)
		e.writeStr(f.Reason)
		e.writeStringSlice(sortedCopy(f.Callees))
	}

	// Call edges & live types.
	edges := append([]CallEdge(nil), meta.CallEdges...)
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		if edges[i].Callee != edges[j].Callee {
			return edges[i].Callee < edges[j].Callee
		}
		return edges[i].Kind < edges[j].Kind
	})
	e.writeUvarint(uint64(len(edges)))
	for _, ed := range edges {
		e.writeStr(ed.Caller)
		e.writeStr(ed.Callee)
		e.writeByte(ed.Kind)
	}
	e.writeStringSlice(sortedCopy(meta.LiveTypes))
	e.writeStringSlice(sortedCopy(meta.ExportFuncs))
	return e.finish(kindMeta), nil
}

// WriteMeta writes the binary encoding of meta to w.
func WriteMeta(w io.Writer, meta *PackageMeta) error {
	raw, err := MarshalMeta(meta)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

// UnmarshalMeta decodes a binary PackageMeta.
func UnmarshalMeta(raw []byte) (*PackageMeta, error) {
	d, err := newDecoder(raw, kindMeta)
	if err != nil {
		return nil, err
	}
	m := &PackageMeta{}
	s := &m.Summary
	if s.Path, err = d.readStr(); err != nil {
		return nil, err
	}
	if s.TargetTriple, err = d.readStr(); err != nil {
		return nil, err
	}
	nExp, err := d.readUvarint()
	if err != nil {
		return nil, err
	}
	if nExp > 0 {
		s.Exports = make(map[string]string, nExp)
		for i := uint64(0); i < nExp; i++ {
			k, err := d.readStr()
			if err != nil {
				return nil, err
			}
			v, err := d.readStr()
			if err != nil {
				return nil, err
			}
			s.Exports[k] = v
		}
	}
	if s.DefinedFuncs, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if m.DefinedGlobals, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.DirectCalls, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.ClosureRefs, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.InterfaceCalls, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.HasSafepoints, err = d.readBool(); err != nil {
		return nil, err
	}
	if s.HasWriteBarrier, err = d.readBool(); err != nil {
		return nil, err
	}

	if m.Escape.PkgPath, err = d.readStr(); err != nil {
		return nil, err
	}
	nEsc, err := d.readUvarint()
	if err != nil {
		return nil, err
	}
	if nEsc > 0 {
		m.Escape.Funcs = make([]FuncEscape, nEsc)
		for i := uint64(0); i < nEsc; i++ {
			f := &m.Escape.Funcs[i]
			if f.Name, err = d.readStr(); err != nil {
				return nil, err
			}
			np, err := d.readUvarint()
			if err != nil {
				return nil, err
			}
			if np > 0 {
				f.Params = make([]ParamEscapeFlags, np)
				for j := uint64(0); j < np; j++ {
					b, err := d.readByte()
					if err != nil {
						return nil, err
					}
					f.Params[j] = ParamEscapeFlags(b)
				}
			}
			sa, err := d.readVarint()
			if err != nil {
				return nil, err
			}
			ha, err := d.readVarint()
			if err != nil {
				return nil, err
			}
			f.StackAllocs = int(sa)
			f.HeapAllocs = int(ha)
		}
	}

	if m.Inline.PkgPath, err = d.readStr(); err != nil {
		return nil, err
	}
	budget, err := d.readVarint()
	if err != nil {
		return nil, err
	}
	m.Inline.Budget = int(budget)
	nInl, err := d.readUvarint()
	if err != nil {
		return nil, err
	}
	if nInl > 0 {
		m.Inline.Funcs = make([]FuncInline, nInl)
		for i := uint64(0); i < nInl; i++ {
			f := &m.Inline.Funcs[i]
			if f.Name, err = d.readStr(); err != nil {
				return nil, err
			}
			cost, err := d.readVarint()
			if err != nil {
				return nil, err
			}
			f.Cost = int(cost)
			if f.Inlineable, err = d.readBool(); err != nil {
				return nil, err
			}
			if f.Reason, err = d.readStr(); err != nil {
				return nil, err
			}
			if f.Callees, err = d.readStringSlice(); err != nil {
				return nil, err
			}
		}
	}

	nEdges, err := d.readUvarint()
	if err != nil {
		return nil, err
	}
	if nEdges > 0 {
		m.CallEdges = make([]CallEdge, nEdges)
		for i := uint64(0); i < nEdges; i++ {
			if m.CallEdges[i].Caller, err = d.readStr(); err != nil {
				return nil, err
			}
			if m.CallEdges[i].Callee, err = d.readStr(); err != nil {
				return nil, err
			}
			if m.CallEdges[i].Kind, err = d.readByte(); err != nil {
				return nil, err
			}
		}
	}
	if m.LiveTypes, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if m.ExportFuncs, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	return m, nil
}

// ReadMeta reads and decodes a binary PackageMeta from r.
func ReadMeta(r io.Reader) (*PackageMeta, error) {
	raw, err := readAll(r)
	if err != nil {
		return nil, err
	}
	return UnmarshalMeta(raw)
}

// PCLNEntry describes a single compact PC-to-line record for a function.
type PCLNEntry struct {
	PC   uint32
	Line uint32
	File string
}

// PCLNFunc describes compact binary PCLN metadata for one compiled function.
type PCLNFunc struct {
	Symbol   string
	Name     string
	File     string
	Line     uint32
	StartPC  uint64
	EndPC    uint64
	Entries  []PCLNEntry
}

// LinkSnapshot persists the package-level link & PCLN metadata alongside cached
// .a archives so warm builds skip cl.NewPackageEx and LLVM context creation.
type LinkSnapshot struct {
	PkgPath         string
	PkgName         string
	NeedRuntime     bool
	NeedPyInit      bool
	NeedAbiInit     bool
	HasLocalExports bool
	MethodByIndex   bool
	MethodByName    []string
	AbiSymbols      []string
	AbiTypes        []string
	CExports        []string
	Imports         []string
	PCLNFuncs       []PCLNFunc
}

// MarshalLinkSnapshot encodes snap into compact deterministic binary bytes.
func MarshalLinkSnapshot(snap *LinkSnapshot) ([]byte, error) {
	if snap == nil {
		return nil, ErrCorrupt
	}
	e := newEncoder()
	e.writeStr(snap.PkgPath)
	e.writeStr(snap.PkgName)
	e.writeBool(snap.NeedRuntime)
	e.writeBool(snap.NeedPyInit)
	e.writeBool(snap.NeedAbiInit)
	e.writeBool(snap.HasLocalExports)
	e.writeBool(snap.MethodByIndex)
	e.writeStringSlice(sortedCopy(snap.MethodByName))
	e.writeStringSlice(sortedCopy(snap.AbiSymbols))
	e.writeStringSlice(sortedCopy(snap.AbiTypes))
	e.writeStringSlice(sortedCopy(snap.CExports))
	e.writeStringSlice(sortedCopy(snap.Imports))

	fns := append([]PCLNFunc(nil), snap.PCLNFuncs...)
	sort.Slice(fns, func(i, j int) bool { return fns[i].Symbol < fns[j].Symbol })
	e.writeUvarint(uint64(len(fns)))
	for _, fn := range fns {
		e.writeStr(fn.Symbol)
		e.writeStr(fn.Name)
		e.writeStr(fn.File)
		e.writeUvarint(uint64(fn.Line))
		e.writeUvarint(fn.StartPC)
		e.writeUvarint(fn.EndPC)
		e.writeUvarint(uint64(len(fn.Entries)))
		for _, ent := range fn.Entries {
			e.writeUvarint(uint64(ent.PC))
			e.writeUvarint(uint64(ent.Line))
			e.writeStr(ent.File)
		}
	}
	return e.finish(kindLinkSnapshot), nil
}

// WriteLinkSnapshot writes the binary encoding of snap to w.
func WriteLinkSnapshot(w io.Writer, snap *LinkSnapshot) error {
	raw, err := MarshalLinkSnapshot(snap)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

// UnmarshalLinkSnapshot decodes a binary LinkSnapshot.
func UnmarshalLinkSnapshot(raw []byte) (*LinkSnapshot, error) {
	d, err := newDecoder(raw, kindLinkSnapshot)
	if err != nil {
		return nil, err
	}
	s := &LinkSnapshot{}
	if s.PkgPath, err = d.readStr(); err != nil {
		return nil, err
	}
	if s.PkgName, err = d.readStr(); err != nil {
		return nil, err
	}
	if s.NeedRuntime, err = d.readBool(); err != nil {
		return nil, err
	}
	if s.NeedPyInit, err = d.readBool(); err != nil {
		return nil, err
	}
	if s.NeedAbiInit, err = d.readBool(); err != nil {
		return nil, err
	}
	if s.HasLocalExports, err = d.readBool(); err != nil {
		return nil, err
	}
	if s.MethodByIndex, err = d.readBool(); err != nil {
		return nil, err
	}
	if s.MethodByName, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.AbiSymbols, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.AbiTypes, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.CExports, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	if s.Imports, err = d.readStringSlice(); err != nil {
		return nil, err
	}
	nFns, err := d.readUvarint()
	if err != nil {
		return nil, err
	}
	if nFns > 0 {
		s.PCLNFuncs = make([]PCLNFunc, nFns)
		for i := uint64(0); i < nFns; i++ {
			fn := &s.PCLNFuncs[i]
			if fn.Symbol, err = d.readStr(); err != nil {
				return nil, err
			}
			if fn.Name, err = d.readStr(); err != nil {
				return nil, err
			}
			if fn.File, err = d.readStr(); err != nil {
				return nil, err
			}
			line, err := d.readUvarint()
			if err != nil {
				return nil, err
			}
			fn.Line = uint32(line)
			if fn.StartPC, err = d.readUvarint(); err != nil {
				return nil, err
			}
			if fn.EndPC, err = d.readUvarint(); err != nil {
				return nil, err
			}
			ne, err := d.readUvarint()
			if err != nil {
				return nil, err
			}
			if ne > 0 {
				fn.Entries = make([]PCLNEntry, ne)
				for j := uint64(0); j < ne; j++ {
					pc, err := d.readUvarint()
					if err != nil {
						return nil, err
					}
					ln, err := d.readUvarint()
					if err != nil {
						return nil, err
					}
					fl, err := d.readStr()
					if err != nil {
						return nil, err
					}
					fn.Entries[j] = PCLNEntry{PC: uint32(pc), Line: uint32(ln), File: fl}
				}
			}
		}
	}
	return s, nil
}

// ReadLinkSnapshot reads and decodes a binary LinkSnapshot from r.
func ReadLinkSnapshot(r io.Reader) (*LinkSnapshot, error) {
	raw, err := readAll(r)
	if err != nil {
		return nil, err
	}
	return UnmarshalLinkSnapshot(raw)
}

// FileStat captures pure-Go os.Stat metadata for one C source/header file.
type FileStat struct {
	Path            string
	Size            int64
	ModTimeUnixNano int64
}

// Depfile persists -MMD -MF header dependencies and preprocessed digest for
// LLGoFiles C wrappers so warm builds validate headers via pure-Go os.Stat
// instead of spawning clang -E.
type Depfile struct {
	Key              string
	PreprocessedHash string
	Files            []FileStat
}

// MarshalDepfile encodes df into compact deterministic binary bytes.
func MarshalDepfile(df *Depfile) ([]byte, error) {
	if df == nil {
		return nil, ErrCorrupt
	}
	e := newEncoder()
	e.writeStr(df.Key)
	e.writeStr(df.PreprocessedHash)
	files := append([]FileStat(nil), df.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	e.writeUvarint(uint64(len(files)))
	for _, f := range files {
		e.writeStr(f.Path)
		e.writeVarint(f.Size)
		e.writeVarint(f.ModTimeUnixNano)
	}
	return e.finish(kindDepfile), nil
}

// UnmarshalDepfile decodes a binary Depfile.
func UnmarshalDepfile(raw []byte) (*Depfile, error) {
	d, err := newDecoder(raw, kindDepfile)
	if err != nil {
		return nil, err
	}
	df := &Depfile{}
	if df.Key, err = d.readStr(); err != nil {
		return nil, err
	}
	if df.PreprocessedHash, err = d.readStr(); err != nil {
		return nil, err
	}
	n, err := d.readUvarint()
	if err != nil {
		return nil, err
	}
	if n > 0 {
		df.Files = make([]FileStat, n)
		for i := uint64(0); i < n; i++ {
			if df.Files[i].Path, err = d.readStr(); err != nil {
				return nil, err
			}
			if df.Files[i].Size, err = d.readVarint(); err != nil {
				return nil, err
			}
			if df.Files[i].ModTimeUnixNano, err = d.readVarint(); err != nil {
				return nil, err
			}
		}
	}
	return df, nil
}

// ValidateDepfile verifies every recorded file matches current os.Stat size and
// modification time. Returns false if any file is missing or modified.
func ValidateDepfile(df *Depfile) bool {
	if df == nil || len(df.Files) == 0 || df.PreprocessedHash == "" {
		return false
	}
	for _, f := range df.Files {
		info, err := os.Stat(f.Path)
		if err != nil || info.IsDir() {
			return false
		}
		if info.Size() != f.Size || info.ModTime().UnixNano() != f.ModTimeUnixNano {
			return false
		}
	}
	return true
}

// StatFiles captures FileStat entries for paths.
func StatFiles(paths []string) ([]FileStat, error) {
	seen := make(map[string]bool, len(paths))
	out := make([]FileStat, 0, len(paths))
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		info, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		out = append(out, FileStat{
			Path:            p,
			Size:            info.Size(),
			ModTimeUnixNano: info.ModTime().UnixNano(),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func sortedCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}
