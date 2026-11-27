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

// Package pkgmeta computes and persists compact binary per-package compiler
// analysis summaries (call edges, escape facts, inlining cost, reachability)
// over github.com/xgo-dev/llgo/ssa for incremental compiler caching.
package pkgmeta

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/ssa"
)

const (
	magicHeader   = "LLGOPM"
	formatVersion = uint32(1)
	DefaultBudget = 80
)

// ParamEscapeFlags summarizes how a function parameter escapes.
type ParamEscapeFlags uint8

const (
	ParamNoEscape ParamEscapeFlags = 0
	ParamToHeap   ParamEscapeFlags = 1 << iota
	ParamToReturn
	ParamToCall
)

type FuncEscape struct {
	Name        string
	Params      []ParamEscapeFlags
	StackAllocs int
	HeapAllocs  int
}

type EscapeSummary struct {
	PkgPath string
	Funcs   []FuncEscape
}

type FuncInline struct {
	Name       string
	Cost       int
	Inlineable bool
	Reason     string
	Callees    []string
}

type InlineSummary struct {
	PkgPath string
	Budget  int
	Funcs   []FuncInline
}

type CallEdge struct {
	Caller string
	Callee string
	Kind   uint8 // 0=direct, 1=closure, 2=interface
}

// PackageMeta bundles compact binary analysis summaries persisted alongside
// a cached package archive.
type PackageMeta struct {
	PkgPath        string
	TargetTriple   string
	DefinedFuncs   []string
	DefinedGlobals []string
	DirectCalls    []string
	ClosureRefs    []string
	InterfaceCalls []string
	Escape         EscapeSummary
	Inline         InlineSummary
	CallEdges      []CallEdge
	ExportFuncs    []string
}

// DeadcodeResult holds deterministic reachability sets across live packages
// and cached PackageMeta summaries.
type DeadcodeResult struct {
	ReachableFuncs   []string
	DeadFuncs        []string
	ReachableGlobals []string
	DeadGlobals      []string
	ReachableMethods []string
}

func collectPackageFuncs(pkg *ssa.Package) []*ssa.Function {
	if pkg == nil {
		return nil
	}
	seen := make(map[*ssa.Function]bool)
	var funcs []*ssa.Function
	var add func(fn *ssa.Function)
	add = func(fn *ssa.Function) {
		if fn == nil || seen[fn] {
			return
		}
		seen[fn] = true
		funcs = append(funcs, fn)
		for _, anon := range fn.AnonFuncs {
			add(anon)
		}
	}
	for _, m := range pkg.Members {
		if fn, ok := m.(*ssa.Function); ok {
			add(fn)
		}
	}
	if pkg.Prog != nil {
		for _, m := range pkg.Members {
			if t, ok := m.(*ssa.Type); ok {
				for _, typ := range []interface{ String() string }{t.Type()} {
					_ = typ
				}
				mset := pkg.Prog.MethodSets.MethodSet(t.Type())
				for i := 0; i < mset.Len(); i++ {
					if fn := pkg.Prog.MethodValue(mset.At(i)); fn != nil && fn.Pkg == pkg {
						add(fn)
					}
				}
			}
		}
	}
	sort.Slice(funcs, func(i, j int) bool { return funcs[i].String() < funcs[j].String() })
	return funcs
}

// BuildPackageMeta runs pure-Go summary extraction (call edges, escape, inline)
// over an llgo/ssa.Package.
func BuildPackageMeta(pkg *ssa.Package, allFuncs []*ssa.Function, importedEsc map[string]*EscapeSummary, inlineBudget int) *PackageMeta {
	if pkg == nil {
		return nil
	}
	if len(allFuncs) == 0 {
		allFuncs = collectPackageFuncs(pkg)
	}
	pkgPath := ""
	if pkg.Pkg != nil {
		pkgPath = pkg.Pkg.Path()
	}
	triple := ""
	if pkg.Prog != nil {
		triple = pkg.Prog.Target.Triple
	}

	meta := &PackageMeta{
		PkgPath:      pkgPath,
		TargetTriple: triple,
	}

	directSet := make(map[string]bool)
	closureSet := make(map[string]bool)
	ifaceSet := make(map[string]bool)
	edgeSet := make(map[CallEdge]bool)

	for _, m := range pkg.Members {
		if g, ok := m.(*ssa.Global); ok {
			meta.DefinedGlobals = append(meta.DefinedGlobals, g.String())
		}
	}
	sort.Strings(meta.DefinedGlobals)

	for _, fn := range allFuncs {
		fnName := fn.String()
		meta.DefinedFuncs = append(meta.DefinedFuncs, fnName)
		if fn.ExportName != "" {
			meta.ExportFuncs = append(meta.ExportFuncs, fnName)
		}
		for _, blk := range fn.Blocks {
			for _, ins := range blk.Instrs {
				switch ins := ins.(type) {
				case *ssa.MakeClosure:
					if target, ok := ins.Fn.(*ssa.Function); ok {
						tName := target.String()
						closureSet[tName] = true
						e := CallEdge{Caller: fnName, Callee: tName, Kind: 1}
						if !edgeSet[e] {
							edgeSet[e] = true
							meta.CallEdges = append(meta.CallEdges, e)
						}
					}
				case ssa.CallInstruction:
					common := ins.Common()
					if common.IsInvoke() {
						mName := common.Method.Name()
						ifaceSet[mName] = true
						e := CallEdge{Caller: fnName, Callee: mName, Kind: 2}
						if !edgeSet[e] {
							edgeSet[e] = true
							meta.CallEdges = append(meta.CallEdges, e)
						}
					} else if callee := common.StaticCallee(); callee != nil {
						cName := callee.String()
						directSet[cName] = true
						e := CallEdge{Caller: fnName, Callee: cName, Kind: 0}
						if !edgeSet[e] {
							edgeSet[e] = true
							meta.CallEdges = append(meta.CallEdges, e)
						}
					}
				}
			}
		}
	}

	meta.DirectCalls = sortedKeys(directSet)
	meta.ClosureRefs = sortedKeys(closureSet)
	meta.InterfaceCalls = sortedKeys(ifaceSet)
	sort.Strings(meta.DefinedFuncs)
	sort.Strings(meta.ExportFuncs)
	sort.Slice(meta.CallEdges, func(i, j int) bool {
		if meta.CallEdges[i].Caller != meta.CallEdges[j].Caller {
			return meta.CallEdges[i].Caller < meta.CallEdges[j].Caller
		}
		if meta.CallEdges[i].Callee != meta.CallEdges[j].Callee {
			return meta.CallEdges[i].Callee < meta.CallEdges[j].Callee
		}
		return meta.CallEdges[i].Kind < meta.CallEdges[j].Kind
	})

	meta.Escape = *AnalyzeEscape(pkgPath, allFuncs, importedEsc)
	meta.Inline = *AnalyzeInline(pkgPath, allFuncs, inlineBudget)
	return meta
}

// AnalyzeEscape computes bounded-fixpoint escape summaries over llgo/ssa functions.
func AnalyzeEscape(pkgPath string, funcs []*ssa.Function, imports map[string]*EscapeSummary) *EscapeSummary {
	known := make(map[string]FuncEscape)
	for _, imp := range imports {
		if imp == nil {
			continue
		}
		for _, fe := range imp.Funcs {
			known[fe.Name] = fe
		}
	}
	local := make(map[string]FuncEscape, len(funcs))
	for _, fn := range funcs {
		name := fn.String()
		if len(fn.Blocks) == 0 {
			params := make([]ParamEscapeFlags, len(fn.Params))
			for i, p := range fn.Params {
				if p.Type() != nil && p.Type().HasPtrData() {
					params[i] = ParamToHeap
				}
			}
			known[name] = FuncEscape{Name: name, Params: params}
			continue
		}
		local[name] = FuncEscape{Name: name, Params: make([]ParamEscapeFlags, len(fn.Params))}
	}
	lookup := func(name string) (FuncEscape, bool) {
		if fe, ok := local[name]; ok {
			return fe, true
		}
		fe, ok := known[name]
		return fe, ok
	}

	for round := 0; round < 16; round++ {
		changed := false
		for _, fn := range funcs {
			if len(fn.Blocks) == 0 {
				continue
			}
			name := fn.String()
			next := analyzeFuncEscape(fn, lookup)
			if !funcEscapeEqual(local[name], next) {
				local[name] = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	res := &EscapeSummary{PkgPath: pkgPath, Funcs: make([]FuncEscape, 0, len(funcs))}
	for _, fn := range funcs {
		name := fn.String()
		if fe, ok := local[name]; ok {
			res.Funcs = append(res.Funcs, fe)
		} else if fe, ok := known[name]; ok {
			res.Funcs = append(res.Funcs, fe)
		}
	}
	sort.Slice(res.Funcs, func(i, j int) bool { return res.Funcs[i].Name < res.Funcs[j].Name })
	return res
}

func analyzeFuncEscape(fn *ssa.Function, lookup func(string) (FuncEscape, bool)) FuncEscape {
	paramOrigin := make(map[ssa.Value][]int)
	allocOrigin := make(map[ssa.Value][]*ssa.Alloc)
	paramFlags := make([]ParamEscapeFlags, len(fn.Params))
	allocEscapes := make(map[*ssa.Alloc]bool)

	for i, p := range fn.Params {
		paramOrigin[p] = []int{i}
	}
	addFlag := func(v ssa.Value, flag ParamEscapeFlags) {
		for _, idx := range paramOrigin[v] {
			paramFlags[idx] |= flag
		}
		if flag&(ParamToHeap|ParamToReturn) != 0 {
			for _, a := range allocOrigin[v] {
				allocEscapes[a] = true
			}
		}
	}
	propagate := func(dst, src ssa.Value) {
		if ps := paramOrigin[src]; len(ps) > 0 {
			paramOrigin[dst] = appendUniqueInts(paramOrigin[dst], ps)
		}
		if as := allocOrigin[src]; len(as) > 0 {
			allocOrigin[dst] = appendUniqueAllocs(allocOrigin[dst], as)
		}
	}

	var allAllocs []*ssa.Alloc
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			if a, ok := ins.(*ssa.Alloc); ok {
				allAllocs = append(allAllocs, a)
				allocOrigin[a] = []*ssa.Alloc{a}
			}
		}
	}
	for pass := 0; pass < 4; pass++ {
		for _, blk := range fn.Blocks {
			for _, ins := range blk.Instrs {
				switch ins := ins.(type) {
				case *ssa.Phi:
					for _, e := range ins.Edges {
						propagate(ins, e)
					}
				case *ssa.Convert:
					propagate(ins, ins.X)
				case *ssa.ChangeType:
					propagate(ins, ins.X)
				case *ssa.ChangeInterface:
					propagate(ins, ins.X)
				case *ssa.MakeInterface:
					propagate(ins, ins.X)
				case *ssa.SliceToArrayPointer:
					propagate(ins, ins.X)
				case *ssa.FieldAddr:
					propagate(ins, ins.X)
				case *ssa.IndexAddr:
					propagate(ins, ins.X)
				case *ssa.Slice:
					propagate(ins, ins.X)
				case *ssa.Extract:
					propagate(ins, ins.Tuple)
				case *ssa.Store:
					if _, isGlobal := ins.Addr.(*ssa.Global); isGlobal || len(paramOrigin[ins.Addr]) > 0 {
						addFlag(ins.Val, ParamToHeap)
					} else {
						propagate(ins.Addr, ins.Val)
					}
				case *ssa.MakeClosure:
					for _, b := range ins.Bindings {
						addFlag(b, ParamToHeap)
					}
				case *ssa.Return:
					for _, r := range ins.Results {
						addFlag(r, ParamToReturn)
					}
				case *ssa.Panic:
					addFlag(ins.X, ParamToHeap)
				case *ssa.Send:
					addFlag(ins.X, ParamToHeap)
				case *ssa.MapUpdate:
					addFlag(ins.Key, ParamToHeap)
					addFlag(ins.Value, ParamToHeap)
				case ssa.CallInstruction:
					common := ins.Common()
					if common.IsInvoke() {
						addFlag(common.Value, ParamToHeap)
						for _, a := range common.Args {
							addFlag(a, ParamToHeap)
						}
					} else if callee := common.StaticCallee(); callee != nil {
						fe, ok := lookup(callee.String())
						for i, a := range common.Args {
							if !ok || i >= len(fe.Params) {
								addFlag(a, ParamToHeap|ParamToCall)
								continue
							}
							if fe.Params[i]&ParamToHeap != 0 {
								addFlag(a, ParamToHeap|ParamToCall)
							}
							if fe.Params[i]&ParamToReturn != 0 {
								if callVal := ins.Value(); callVal != nil {
									propagate(callVal, a)
								}
							}
						}
					} else if _, isBuiltin := common.Value.(*ssa.Builtin); !isBuiltin {
						for _, a := range common.Args {
							addFlag(a, ParamToHeap|ParamToCall)
						}
					}
				}
			}
		}
	}

	stackN, heapN := 0, 0
	for _, a := range allAllocs {
		if allocEscapes[a] {
			heapN++
		} else {
			stackN++
		}
	}
	return FuncEscape{
		Name:        fn.String(),
		Params:      paramFlags,
		StackAllocs: stackN,
		HeapAllocs:  heapN,
	}
}

// AnalyzeInline computes per-function inlining cost summaries.
func AnalyzeInline(pkgPath string, funcs []*ssa.Function, budget int) *InlineSummary {
	if budget <= 0 {
		budget = DefaultBudget
	}
	res := &InlineSummary{PkgPath: pkgPath, Budget: budget, Funcs: make([]FuncInline, 0, len(funcs))}
	for _, fn := range funcs {
		res.Funcs = append(res.Funcs, evalInline(fn, budget))
	}
	sort.Slice(res.Funcs, func(i, j int) bool { return res.Funcs[i].Name < res.Funcs[j].Name })
	return res
}

func evalInline(fn *ssa.Function, budget int) FuncInline {
	fi := FuncInline{Name: fn.String()}
	if len(fn.Blocks) == 0 {
		fi.Reason = "declaration"
		return fi
	}
	if fn.NoInline {
		fi.Reason = "noinline"
		return fi
	}
	cost := 0
	callees := make(map[string]bool)
	disqualify := ""
	for _, blk := range fn.Blocks {
		for _, ins := range blk.Instrs {
			switch ins := ins.(type) {
			case *ssa.DebugRef:
			case *ssa.Jump, *ssa.Return, *ssa.Extract, *ssa.BinOp, *ssa.UnOp, *ssa.Convert, *ssa.ChangeType, *ssa.ChangeInterface, *ssa.MakeInterface, *ssa.FieldAddr, *ssa.IndexAddr, *ssa.Field, *ssa.Index, *ssa.Store, *ssa.Phi:
				cost++
			case *ssa.If:
				cost += 2
			case *ssa.Alloc:
				if ins.Heap {
					cost += 12
				} else {
					cost += 2
				}
			case *ssa.MakeClosure:
				cost += 8
				if target, ok := ins.Fn.(*ssa.Function); ok {
					callees[target.String()] = true
				}
			case *ssa.Call:
				if ins.Call.IsInvoke() {
					cost += 25
				} else if callee := ins.Call.StaticCallee(); callee != nil {
					cName := callee.String()
					callees[cName] = true
					if cName == fi.Name {
						disqualify = "recursive"
					}
					cost += 15
				} else if bi, ok := ins.Call.Value.(*ssa.Builtin); ok && bi.Name() == "recover" {
					disqualify = "recover"
				} else {
					cost += 25
				}
			case *ssa.Defer, *ssa.RunDefers:
				disqualify = "defer"
			case *ssa.Go:
				disqualify = "go"
			case *ssa.Panic:
				cost += 10
			default:
				cost += 4
			}
		}
	}
	fi.Callees = sortedKeys(callees)
	fi.Cost = cost
	if disqualify != "" {
		fi.Reason = disqualify
		return fi
	}
	if cost > budget {
		fi.Reason = "over-budget"
		return fi
	}
	fi.Inlineable = true
	return fi
}

// AnalyzeDeadcode computes cross-package reachability combining live PackageMetas
// and cached PackageMetas from entry roots.
func AnalyzeDeadcode(metas []*PackageMeta, roots []string) *DeadcodeResult {
	allFuncs := make(map[string]bool)
	allGlobals := make(map[string]bool)
	outEdges := make(map[string][]CallEdge)
	methodsByShort := make(map[string][]string)

	registerMethod := func(fnName string) {
		if idx := strings.LastIndexByte(fnName, '.'); idx >= 0 && idx+1 < len(fnName) {
			short := fnName[idx+1:]
			methodsByShort[short] = append(methodsByShort[short], fnName)
		}
	}

	allRoots := append([]string(nil), roots...)
	for _, m := range metas {
		if m == nil {
			continue
		}
		for _, fn := range m.DefinedFuncs {
			allFuncs[fn] = true
			registerMethod(fn)
		}
		for _, g := range m.DefinedGlobals {
			allGlobals[g] = true
		}
		allRoots = append(allRoots, m.ExportFuncs...)
		for _, e := range m.CallEdges {
			outEdges[e.Caller] = append(outEdges[e.Caller], e)
		}
	}

	reachable := make(map[string]bool)
	liveMethods := make(map[string]bool)
	queue := make([]string, 0, len(allRoots))
	push := func(fn string) {
		if fn != "" && !reachable[fn] {
			reachable[fn] = true
			queue = append(queue, fn)
		}
	}
	for _, r := range allRoots {
		push(r)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, e := range outEdges[cur] {
			if e.Kind == 2 {
				if !liveMethods[e.Callee] {
					liveMethods[e.Callee] = true
					for _, impl := range methodsByShort[e.Callee] {
						push(impl)
					}
				}
				continue
			}
			push(e.Callee)
		}
	}

	res := &DeadcodeResult{}
	for fn := range allFuncs {
		if reachable[fn] {
			res.ReachableFuncs = append(res.ReachableFuncs, fn)
		} else {
			res.DeadFuncs = append(res.DeadFuncs, fn)
		}
	}
	for g := range allGlobals {
		res.DeadGlobals = append(res.DeadGlobals, g)
	}
	res.ReachableMethods = sortedKeys(liveMethods)
	sort.Strings(res.ReachableFuncs)
	sort.Strings(res.DeadFuncs)
	sort.Strings(res.DeadGlobals)
	return res
}

// Marshal encodes meta into compact deterministic binary bytes.
func Marshal(meta *PackageMeta) ([]byte, error) {
	if meta == nil {
		return nil, errors.New("pkgmeta: nil metadata")
	}
	var buf bytes.Buffer
	buf.WriteString(magicHeader)
	var ver [4]byte
	binary.LittleEndian.PutUint32(ver[:], formatVersion)
	buf.Write(ver[:])

	writeStr(&buf, meta.PkgPath)
	writeStr(&buf, meta.TargetTriple)
	writeStrSlice(&buf, sortedCopy(meta.DefinedFuncs))
	writeStrSlice(&buf, sortedCopy(meta.DefinedGlobals))
	writeStrSlice(&buf, sortedCopy(meta.DirectCalls))
	writeStrSlice(&buf, sortedCopy(meta.ClosureRefs))
	writeStrSlice(&buf, sortedCopy(meta.InterfaceCalls))

	writeStr(&buf, meta.Escape.PkgPath)
	escFuncs := append([]FuncEscape(nil), meta.Escape.Funcs...)
	sort.Slice(escFuncs, func(i, j int) bool { return escFuncs[i].Name < escFuncs[j].Name })
	writeUvarint(&buf, uint64(len(escFuncs)))
	for _, f := range escFuncs {
		writeStr(&buf, f.Name)
		writeUvarint(&buf, uint64(len(f.Params)))
		for _, p := range f.Params {
			buf.WriteByte(byte(p))
		}
		writeVarint(&buf, int64(f.StackAllocs))
		writeVarint(&buf, int64(f.HeapAllocs))
	}

	writeStr(&buf, meta.Inline.PkgPath)
	writeVarint(&buf, int64(meta.Inline.Budget))
	inlFuncs := append([]FuncInline(nil), meta.Inline.Funcs...)
	sort.Slice(inlFuncs, func(i, j int) bool { return inlFuncs[i].Name < inlFuncs[j].Name })
	writeUvarint(&buf, uint64(len(inlFuncs)))
	for _, f := range inlFuncs {
		writeStr(&buf, f.Name)
		writeVarint(&buf, int64(f.Cost))
		if f.Inlineable {
			buf.WriteByte(1)
		} else {
			buf.WriteByte(0)
		}
		writeStr(&buf, f.Reason)
		writeStrSlice(&buf, sortedCopy(f.Callees))
	}

	writeUvarint(&buf, uint64(len(meta.CallEdges)))
	for _, e := range meta.CallEdges {
		writeStr(&buf, e.Caller)
		writeStr(&buf, e.Callee)
		buf.WriteByte(e.Kind)
	}
	writeStrSlice(&buf, sortedCopy(meta.ExportFuncs))
	return buf.Bytes(), nil
}

// Unmarshal decodes binary bytes produced by Marshal.
func Unmarshal(raw []byte) (*PackageMeta, error) {
	if len(raw) < len(magicHeader)+4 || string(raw[:len(magicHeader)]) != magicHeader {
		return nil, errors.New("pkgmeta: invalid magic")
	}
	if binary.LittleEndian.Uint32(raw[len(magicHeader):len(magicHeader)+4]) != formatVersion {
		return nil, errors.New("pkgmeta: unsupported version")
	}
	r := &reader{data: raw, pos: len(magicHeader) + 4}
	m := &PackageMeta{}
	var err error
	if m.PkgPath, err = r.readStr(); err != nil {
		return nil, err
	}
	if m.TargetTriple, err = r.readStr(); err != nil {
		return nil, err
	}
	if m.DefinedFuncs, err = r.readStrSlice(); err != nil {
		return nil, err
	}
	if m.DefinedGlobals, err = r.readStrSlice(); err != nil {
		return nil, err
	}
	if m.DirectCalls, err = r.readStrSlice(); err != nil {
		return nil, err
	}
	if m.ClosureRefs, err = r.readStrSlice(); err != nil {
		return nil, err
	}
	if m.InterfaceCalls, err = r.readStrSlice(); err != nil {
		return nil, err
	}
	if m.Escape.PkgPath, err = r.readStr(); err != nil {
		return nil, err
	}
	nEsc, err := r.readUvarint()
	if err != nil || nEsc > uint64(len(raw)) {
		return nil, errors.New("pkgmeta: corrupt escape table")
	}
	if nEsc > 0 {
		m.Escape.Funcs = make([]FuncEscape, nEsc)
		for i := uint64(0); i < nEsc; i++ {
			f := &m.Escape.Funcs[i]
			if f.Name, err = r.readStr(); err != nil {
				return nil, err
			}
			np, err := r.readUvarint()
			if err != nil || np > uint64(len(raw)) {
				return nil, errors.New("pkgmeta: corrupt params")
			}
			if np > 0 {
				f.Params = make([]ParamEscapeFlags, np)
				for j := uint64(0); j < np; j++ {
					b, err := r.readByte()
					if err != nil {
						return nil, err
					}
					f.Params[j] = ParamEscapeFlags(b)
				}
			}
			sa, err := r.readVarint()
			if err != nil {
				return nil, err
			}
			ha, err := r.readVarint()
			if err != nil {
				return nil, err
			}
			f.StackAllocs, f.HeapAllocs = int(sa), int(ha)
		}
	}
	if m.Inline.PkgPath, err = r.readStr(); err != nil {
		return nil, err
	}
	budget, err := r.readVarint()
	if err != nil {
		return nil, err
	}
	m.Inline.Budget = int(budget)
	nInl, err := r.readUvarint()
	if err != nil || nInl > uint64(len(raw)) {
		return nil, errors.New("pkgmeta: corrupt inline table")
	}
	if nInl > 0 {
		m.Inline.Funcs = make([]FuncInline, nInl)
		for i := uint64(0); i < nInl; i++ {
			f := &m.Inline.Funcs[i]
			if f.Name, err = r.readStr(); err != nil {
				return nil, err
			}
			cost, err := r.readVarint()
			if err != nil {
				return nil, err
			}
			f.Cost = int(cost)
			b, err := r.readByte()
			if err != nil {
				return nil, err
			}
			f.Inlineable = b != 0
			if f.Reason, err = r.readStr(); err != nil {
				return nil, err
			}
			if f.Callees, err = r.readStrSlice(); err != nil {
				return nil, err
			}
		}
	}
	nEdge, err := r.readUvarint()
	if err != nil || nEdge > uint64(len(raw)) {
		return nil, errors.New("pkgmeta: corrupt call edges")
	}
	if nEdge > 0 {
		m.CallEdges = make([]CallEdge, nEdge)
		for i := uint64(0); i < nEdge; i++ {
			if m.CallEdges[i].Caller, err = r.readStr(); err != nil {
				return nil, err
			}
			if m.CallEdges[i].Callee, err = r.readStr(); err != nil {
				return nil, err
			}
			if m.CallEdges[i].Kind, err = r.readByte(); err != nil {
				return nil, err
			}
		}
	}
	if m.ExportFuncs, err = r.readStrSlice(); err != nil {
		return nil, err
	}
	return m, nil
}

func sortedKeys(m map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func funcEscapeEqual(a, b FuncEscape) bool {
	if a.Name != b.Name || a.StackAllocs != b.StackAllocs || a.HeapAllocs != b.HeapAllocs || len(a.Params) != len(b.Params) {
		return false
	}
	for i := range a.Params {
		if a.Params[i] != b.Params[i] {
			return false
		}
	}
	return true
}

func appendUniqueInts(dst, src []int) []int {
	for _, v := range src {
		found := false
		for _, e := range dst {
			if e == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

func appendUniqueAllocs(dst, src []*ssa.Alloc) []*ssa.Alloc {
	for _, v := range src {
		found := false
		for _, e := range dst {
			if e == v {
				found = true
				break
			}
		}
		if !found {
			dst = append(dst, v)
		}
	}
	return dst
}

func writeUvarint(buf *bytes.Buffer, v uint64) {
	var tmp [binary.MaxVarintLen64]byte
	buf.Write(tmp[:binary.PutUvarint(tmp[:], v)])
}

func writeVarint(buf *bytes.Buffer, v int64) {
	var tmp [binary.MaxVarintLen64]byte
	buf.Write(tmp[:binary.PutVarint(tmp[:], v)])
}

func writeStr(buf *bytes.Buffer, s string) {
	writeUvarint(buf, uint64(len(s)))
	buf.WriteString(s)
}

func writeStrSlice(buf *bytes.Buffer, ss []string) {
	writeUvarint(buf, uint64(len(ss)))
	for _, s := range ss {
		writeStr(buf, s)
	}
}

type reader struct {
	data []byte
	pos  int
}

func (r *reader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, errors.New("pkgmeta: EOF")
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *reader) readUvarint() (uint64, error) {
	v, n := binary.Uvarint(r.data[r.pos:])
	if n <= 0 {
		return 0, errors.New("pkgmeta: corrupt uvarint")
	}
	r.pos += n
	return v, nil
}

func (r *reader) readVarint() (int64, error) {
	v, n := binary.Varint(r.data[r.pos:])
	if n <= 0 {
		return 0, errors.New("pkgmeta: corrupt varint")
	}
	r.pos += n
	return v, nil
}

func (r *reader) readStr() (string, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return "", errors.New("pkgmeta: corrupt string")
	}
	s := string(r.data[r.pos : r.pos+int(n)])
	r.pos += int(n)
	return s, nil
}

func (r *reader) readStrSlice() ([]string, error) {
	n, err := r.readUvarint()
	if err != nil || n > uint64(len(r.data)-r.pos) {
		return nil, errors.New("pkgmeta: corrupt slice")
	}
	if n == 0 {
		return nil, nil
	}
	out := make([]string, n)
	for i := uint64(0); i < n; i++ {
		s, err := r.readStr()
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}
