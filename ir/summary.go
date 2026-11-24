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

package ir

import (
	"encoding/json"
	"sort"
)

// PackageSummary is a compact, backend-independent summary extracted from a
// pure-Go IR Package. It is designed to be persisted alongside cached archives
// so incremental builds can run whole-program call-graph, DCE, and link
// planning on cache hits without materializing LLVM modules.
//
// DirectCalls records statically named targets of CallDirect instructions.
// ClosureRefs records function symbols referenced by MakeClosure instructions
// (including closures later invoked via CallClosure, Go, or Defer). Dynamic
// callees passed as arbitrary SSA values to CallClosure, CallCFunc, Go, or
// Defer are not statically enumerated here.
type PackageSummary struct {
	Path            string            `json:"path"`
	TargetTriple    string            `json:"target_triple"`
	Exports         map[string]string `json:"exports,omitempty"`
	DefinedFuncs    []string          `json:"defined_funcs,omitempty"`
	DirectCalls     []string          `json:"direct_calls,omitempty"`
	ClosureRefs     []string          `json:"closure_refs,omitempty"`
	InterfaceCalls  []string          `json:"interface_calls,omitempty"`
	HasSafepoints   bool              `json:"has_safepoints,omitempty"`
	HasWriteBarrier bool              `json:"has_write_barrier,omitempty"`
}

// Summarize extracts a deterministic PackageSummary from pkg.
func Summarize(pkg *Package) PackageSummary {
	s := PackageSummary{
		Path:         pkg.Path,
		TargetTriple: pkg.Target.Triple,
	}
	if len(pkg.Exports) > 0 {
		s.Exports = make(map[string]string, len(pkg.Exports))
		for k, v := range pkg.Exports {
			s.Exports[k] = v
		}
	}
	directSet := make(map[string]struct{})
	closureSet := make(map[string]struct{})
	ifaceSet := make(map[string]struct{})
	for _, fn := range pkg.Funcs {
		if !fn.IsDeclaration() {
			s.DefinedFuncs = append(s.DefinedFuncs, fn.Name)
		}
		for _, blk := range fn.Blocks {
			for _, ins := range blk.Instrs {
				switch ins := ins.(type) {
				case *CallDirect:
					directSet[ins.Callee] = struct{}{}
				case *MakeClosure:
					closureSet[ins.FnName] = struct{}{}
				case *CallInterface:
					ifaceSet[ins.Method] = struct{}{}
				case *Safepoint:
					s.HasSafepoints = true
				case *WriteBarrier:
					s.HasWriteBarrier = true
				}
			}
		}
	}
	sort.Strings(s.DefinedFuncs)
	for name := range directSet {
		s.DirectCalls = append(s.DirectCalls, name)
	}
	sort.Strings(s.DirectCalls)
	for name := range closureSet {
		s.ClosureRefs = append(s.ClosureRefs, name)
	}
	sort.Strings(s.ClosureRefs)
	for name := range ifaceSet {
		s.InterfaceCalls = append(s.InterfaceCalls, name)
	}
	sort.Strings(s.InterfaceCalls)
	return s
}

// MarshalSummary encodes summary deterministically as JSON for build caches.
func MarshalSummary(summary PackageSummary) ([]byte, error) {
	return json.Marshal(summary)
}

// UnmarshalSummary decodes a cached PackageSummary.
func UnmarshalSummary(data []byte) (PackageSummary, error) {
	var s PackageSummary
	err := json.Unmarshal(data, &s)
	return s, err
}
