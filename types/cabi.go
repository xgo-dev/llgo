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
	"strings"

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

// NativeIntExt describes how a target's native C ABI requires narrow integer
// arguments to carry signext / zeroext attributes at call boundaries.
type NativeIntExt uint8

const (
	NativeIntNoExt     NativeIntExt = iota
	NativeIntNarrowExt              // extend integers narrower than 32 bits (Darwin arm64, SysV amd64, x86, wasm, arm, riscv32)
	NativeIntBoolExt                // zeroext bool only (Win64 amd64)
	NativeIntRV64Ext                // signext both signed and unsigned 32-bit ints + narrow ints (riscv64)
)

// NativeIntExtForSpec determines the C ABI narrow-integer extension policy for
// spec without inspecting LLVM IR. Named embedded targets may use GOARCH only
// for package selection (for example, RISC-V profiles use arm), so PhysicalTriple
// / LLVMTarget / Triple takes precedence over GOARCH/GOOS when present.
func NativeIntExtForSpec(spec target.Spec) NativeIntExt {
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	triple := spec.PhysicalTriple
	if triple == "" {
		triple = spec.LLVMTarget
	}
	if triple == "" {
		triple = spec.Triple
	}
	if triple != "" {
		triple = strings.ToLower(triple)
		arch, _, _ := strings.Cut(triple, "-")
		switch {
		case arch == "arm64", arch == "arm64_32", arch == "aarch64", arch == "aarch64_be":
			if target.IsDarwinTriple(triple) {
				return NativeIntNarrowExt
			}
			return NativeIntNoExt
		case arch == "x86_64", arch == "amd64":
			if target.IsWindowsTriple(triple) {
				return NativeIntBoolExt
			}
			return NativeIntNarrowExt
		case arch == "riscv64":
			return NativeIntRV64Ext
		case arch == "riscv32", arch == "wasm32", arch == "wasm64",
			arch == "x86", arch == "386", arch == "i386", arch == "i486", arch == "i586", arch == "i686",
			strings.HasPrefix(arch, "arm"), strings.HasPrefix(arch, "thumb"):
			return NativeIntNarrowExt
		default:
			return NativeIntNoExt
		}
	}
	arch := strings.ToLower(spec.GOARCH)
	goos := strings.ToLower(spec.GOOS)
	switch arch {
	case "arm64":
		if goos == "darwin" || goos == "ios" {
			return NativeIntNarrowExt
		}
		return NativeIntNoExt
	case "amd64":
		if goos == "windows" {
			return NativeIntBoolExt
		}
		return NativeIntNarrowExt
	case "riscv64":
		return NativeIntRV64Ext
	case "riscv32", "wasm", "386", "arm":
		return NativeIntNarrowExt
	}
	return NativeIntNoExt
}

// ArgABI describes the ABI lowering decision for one parameter or return value.
type ArgABI struct {
	Mode     PassMode
	Type     Type
	Layout   *Layout
	SRet     bool      // return value passed via hidden caller pointer
	ByVal    bool      // argument passed by-value in stack memory
	ZExt     bool      // small unsigned integer zero-extended by caller
	SExt     bool      // small signed integer sign-extended by caller
	HFACount int       // non-zero for AAPCS64/ARM HFA (1..4 homogeneous floats)
	HFAElem  BasicKind // Float32 or Float64 when HFACount > 0
}

// FuncABI holds the complete pure-Go ABI classification for a Signature.
type FuncABI struct {
	CallConv CallConv
	Variadic bool
	HasEnv   bool
	HasSRet  bool
	Return   ArgABI
	Params   []ArgABI
}

// ClassifyFuncABI computes target-accurate ABI passing modes directly from
// LLGo's canonical Types and Layouts without inspecting LLVM IR.
func ClassifyFuncABI(spec target.Spec, sig *Signature, layoutFn func(Type) *Layout) *FuncABI {
	if sig == nil {
		return nil
	}
	if spec.WordSize == 0 || spec.PtrSize == 0 {
		spec = target.ResolveSpec(spec)
	}
	cc := sig.callConv
	res := &FuncABI{
		CallConv: cc,
		Variadic: sig.variadic,
		HasEnv:   sig.IsClosure(),
	}

	totalParams := sig.params.Len()
	if sig.recv != nil {
		totalParams++
	}
	res.Params = make([]ArgABI, 0, totalParams)
	if sig.recv != nil {
		res.Params = append(res.Params, classifyArg(spec, cc, sig.recv.typ, false, layoutFn))
	}
	for i := 0; i < sig.params.Len(); i++ {
		res.Params = append(res.Params, classifyArg(spec, cc, sig.params.At(i).typ, false, layoutFn))
	}

	switch sig.results.Len() {
	case 0:
		res.Return = ArgABI{Mode: PassIgnore}
	case 1:
		res.Return = classifyArg(spec, cc, sig.results.At(0).typ, true, layoutFn)
		res.HasSRet = res.Return.SRet
	default:
		res.Return = classifyArg(spec, cc, sig.results, true, layoutFn)
		res.HasSRet = res.Return.SRet
	}
	return res
}

func classifyArg(spec target.Spec, cc CallConv, t Type, isReturn bool, layoutFn func(Type) *Layout) ArgABI {
	l := layoutFn(t)
	arg := ArgABI{
		Mode:   PassDirect,
		Type:   t,
		Layout: l,
	}
	if l == nil || l.Size == 0 {
		arg.Mode = PassIgnore
		return arg
	}
	if cc.IsGo() {
		return arg
	}

	if b, ok := t.Underlying().(*Basic); ok && b.kind != String {
		if !isReturn {
			applyNativeIntExt(NativeIntExtForSpec(spec), b.kind, &arg)
		}
		return arg
	}

	switch t.Underlying().(type) {
	case *Pointer, *Signature, *Map, *Chan, *SIMDVec:
		return arg
	}

	if spec.GOOS == "windows" && spec.GOARCH == "amd64" {
		return classifyWindowsAggregate(arg, isReturn)
	}

	switch spec.GOARCH {
	case "arm64":
		if elem, count, ok := detectHFA(t); ok {
			arg.Mode = PassDirect
			arg.HFACount = count
			arg.HFAElem = elem
			return arg
		}
		if l.Size <= 16 {
			arg.Mode = PassCoerce
			return arg
		}
		arg.Mode = PassIndirect
		if isReturn {
			arg.SRet = true
		}
		return arg

	case "amd64":
		if l.Size <= 16 {
			arg.Mode = PassCoerce
			return arg
		}
		arg.Mode = PassIndirect
		if isReturn {
			arg.SRet = true
		} else {
			arg.ByVal = true
		}
		return arg

	case "wasm":
		if l.Size <= int64(spec.PtrSize) {
			arg.Mode = PassDirect
			return arg
		}
		arg.Mode = PassIndirect
		if isReturn {
			arg.SRet = true
		}
		return arg

	default:
		if l.Size <= int64(spec.WordSize) {
			arg.Mode = PassCoerce
			return arg
		}
		arg.Mode = PassIndirect
		if isReturn {
			arg.SRet = true
		} else {
			arg.ByVal = true
		}
		return arg
	}
}

func applyNativeIntExt(policy NativeIntExt, kind BasicKind, arg *ArgABI) {
	switch policy {
	case NativeIntNoExt:
		return
	case NativeIntBoolExt:
		if kind == Bool {
			arg.ZExt = true
		}
	case NativeIntNarrowExt:
		switch kind {
		case Bool, Uint8, Uint16:
			arg.ZExt = true
		case Int8, Int16:
			arg.SExt = true
		}
	case NativeIntRV64Ext:
		switch kind {
		case Bool, Uint8, Uint16:
			arg.ZExt = true
		case Int8, Int16, Int32, Uint32:
			arg.SExt = true
		}
	}
}

func classifyWindowsAggregate(arg ArgABI, isReturn bool) ArgABI {
	switch arg.Layout.Size {
	case 1, 2, 4, 8:
		arg.Mode = PassCoerce
		return arg
	default:
		arg.Mode = PassIndirect
		if isReturn {
			arg.SRet = true
		}
		return arg
	}
}

// detectHFA checks whether t is a Homogeneous Floating-point Aggregate of
// 1..4 float32 or float64 scalars (AAPCS64/ARM HFA rule).
func detectHFA(t Type) (BasicKind, int, bool) {
	var kind BasicKind
	count := 0
	var walk func(Type) bool
	walk = func(cur Type) bool {
		if cur == nil {
			return false
		}
		switch u := cur.Underlying().(type) {
		case *Basic:
			if u.kind != Float32 && u.kind != Float64 {
				return false
			}
			if kind == Invalid {
				kind = u.kind
			} else if kind != u.kind {
				return false
			}
			count++
			return count <= 4
		case *Array:
			if u.length <= 0 || u.length > 4 {
				return false
			}
			for i := int64(0); i < u.length; i++ {
				if !walk(u.elem) {
					return false
				}
			}
			return true
		case *Struct:
			if len(u.fields) == 0 {
				return false
			}
			for _, f := range u.fields {
				if !walk(f.typ) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
	if !walk(t) || count == 0 || count > 4 {
		return Invalid, 0, false
	}
	return kind, count, true
}
