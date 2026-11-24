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

package types

import (
	gotypes "go/types"

	"github.com/xgo-dev/llgo/target"
)

// PassMode classifies how a function parameter or return value crosses a call
// boundary under a target ABI.
type PassMode uint8

const (
	PassIgnore PassMode = iota
	PassDirect
	PassIndirect
	PassCoerce
	PassExpand
)

func (m PassMode) String() string {
	switch m {
	case PassDirect:
		return "direct"
	case PassIndirect:
		return "indirect"
	case PassCoerce:
		return "coerce"
	case PassExpand:
		return "expand"
	default:
		return "ignore"
	}
}

// ArgABI describes the ABI lowering decision for one parameter or return value.
type ArgABI struct {
	Mode     PassMode
	Type     Type
	Layout   *Layout
	SRet     bool              // return value passed via hidden caller pointer
	ByVal    bool              // argument passed by-value in stack memory
	ZExt     bool              // small unsigned integer zero-extended by caller
	SExt     bool              // small signed integer sign-extended by caller
	HFACount int               // non-zero for AAPCS64/ARM HFA (1..4 homogeneous floats)
	HFAElem  gotypes.BasicKind // Float32 or Float64 when HFACount > 0
}

// FuncABI holds the complete pure-Go ABI classification for a Func prototype.
type FuncABI struct {
	CallConv CallConv
	HasEnv   bool
	HasSRet  bool
	Return   ArgABI
	Params   []ArgABI
}

// ClassifyFuncABI computes target-accurate ABI passing modes directly from
// LLGo's physical Types and Layouts without inspecting LLVM IR.
func ClassifyFuncABI(spec target.Spec, sig *Func, layoutFn func(Type) *Layout) *FuncABI {
	if sig == nil {
		return nil
	}
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	cc := sig.CallConv
	if cc == CallConvUnknown {
		cc = GoDecl
	}
	res := &FuncABI{
		CallConv: cc,
		HasEnv:   sig.HasEnv,
	}

	// Build parameter list: receiver (if any) followed by formal params.
	totalParams := len(sig.Params)
	if sig.Recv != nil {
		totalParams++
	}
	res.Params = make([]ArgABI, 0, totalParams)
	if sig.Recv != nil {
		res.Params = append(res.Params, classifyArg(spec, cc, sig.Recv.Type, false, layoutFn))
	}
	for _, p := range sig.Params {
		res.Params = append(res.Params, classifyArg(spec, cc, p.Type, false, layoutFn))
	}

	// Classify return value(s).
	switch len(sig.Results) {
	case 0:
		res.Return = ArgABI{Mode: PassIgnore}
	case 1:
		res.Return = classifyArg(spec, cc, sig.Results[0].Type, true, layoutFn)
		res.HasSRet = res.Return.SRet
	default:
		tup := &Tuple{Elems: sig.Results}
		res.Return = classifyArg(spec, cc, tup, true, layoutFn)
		res.HasSRet = res.Return.SRet
	}
	return res
}

func classifyArg(spec target.Spec, cc CallConv, t Type, isReturn bool, layoutFn func(Type) *Layout) ArgABI {
	l := layoutFn(t)
	arg := ArgABI{Type: t, Layout: l}
	if l.Size == 0 {
		arg.Mode = PassIgnore
		return arg
	}
	// Go internal ABI passes values directly (with backend register/tuple lowering).
	if !cc.IsNative() {
		arg.Mode = PassDirect
		return arg
	}
	// Native C / Stdcall scalar integer extension flags (matching Clang C ABI).
	if b, ok := t.Underlying().(*Basic); ok {
		arg.Mode = PassDirect
		if needsSmallIntExt(spec) {
			switch b.Kind {
			case gotypes.Bool, gotypes.Uint8, gotypes.Uint16:
				arg.ZExt = true
			case gotypes.Int8, gotypes.Int16:
				arg.SExt = true
			}
		}
		return arg
	}
	switch t.Underlying().(type) {
	case *Pointer, *Func, *SIMDVec, *Map, *Chan:
		arg.Mode = PassDirect
		return arg
	}

	// Aggregate C ABI classification by target architecture / OS.
	switch spec.GOARCH {
	case "arm64":
		if elem, count, ok := detectHFA(t); ok {
			arg.Mode = PassDirect
			arg.HFACount = count
			arg.HFAElem = elem
			return arg
		}
		if spec.GOOS == "windows" {
			return classifyWindowsAggregate(arg, isReturn)
		}
		// AAPCS64: aggregates <= 16 bytes coerced into up to 2 x i64; > 16 bytes indirect.
		if l.Size <= 16 {
			arg.Mode = PassCoerce
		} else {
			arg.Mode = PassIndirect
			arg.SRet = isReturn
			arg.ByVal = !isReturn
		}
		return arg
	case "amd64":
		if spec.GOOS == "windows" {
			return classifyWindowsAggregate(arg, isReturn)
		}
		// SysV AMD64: aggregates <= 16 bytes coerced into up to 2 eightbytes; > 16 bytes indirect.
		if l.Size <= 16 {
			arg.Mode = PassCoerce
		} else {
			arg.Mode = PassIndirect
			arg.SRet = isReturn
			arg.ByVal = !isReturn
		}
		return arg
	case "wasm":
		// Wasm C ABI: single-scalar-like aggregates pass direct/coerce; multi-field indirect for return.
		if isReturn && l.Size > int64(spec.PtrSize) {
			arg.Mode = PassIndirect
			arg.SRet = true
		} else {
			arg.Mode = PassDirect
		}
		return arg
	default:
		// 32-bit / riscv64 / other targets.
		maxDirect := int64(2 * spec.PtrSize)
		if spec.GOOS == "windows" {
			maxDirect = int64(spec.PtrSize)
		}
		if l.Size <= maxDirect {
			arg.Mode = PassCoerce
		} else {
			arg.Mode = PassIndirect
			arg.SRet = isReturn
			arg.ByVal = !isReturn
		}
		return arg
	}
}

func classifyWindowsAggregate(arg ArgABI, isReturn bool) ArgABI {
	sz := arg.Layout.Size
	// Win64 ABI passes power-of-two aggregates (1, 2, 4, 8 bytes) directly in a single GPR.
	if sz == 1 || sz == 2 || sz == 4 || sz == 8 {
		arg.Mode = PassCoerce
		return arg
	}
	arg.Mode = PassIndirect
	arg.SRet = isReturn
	arg.ByVal = !isReturn
	return arg
}

func needsSmallIntExt(spec target.Spec) bool {
	// Darwin/Linux arm64/amd64/wasm follow Clang signext/zeroext conventions on small integer params/returns.
	return spec.GOOS != "windows"
}

// detectHFA detects an AAPCS64 Homogeneous Floating-point Aggregate (1 to 4
// identically-typed float32 or float64 leaf fields).
func detectHFA(t Type) (gotypes.BasicKind, int, bool) {
	var kind gotypes.BasicKind
	count := 0
	var walk func(Type) bool
	walk = func(cur Type) bool {
		switch u := cur.Underlying().(type) {
		case *Basic:
			if u.Kind != gotypes.Float32 && u.Kind != gotypes.Float64 {
				return false
			}
			if kind == gotypes.Invalid {
				kind = u.Kind
			} else if kind != u.Kind {
				return false
			}
			count++
			return count <= 4
		case *Array:
			if u.Len <= 0 || u.Len > 4 {
				return false
			}
			for i := int64(0); i < u.Len; i++ {
				if !walk(u.Elem) {
					return false
				}
			}
			return true
		case *Struct:
			if len(u.Fields) == 0 {
				return false
			}
			for _, f := range u.Fields {
				if !walk(f.Type) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
	if !walk(t) || count == 0 || count > 4 {
		return gotypes.Invalid, 0, false
	}
	return kind, count, true
}
