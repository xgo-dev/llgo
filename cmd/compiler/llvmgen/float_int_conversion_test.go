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

package llvmgen

import (
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func TestAMD64FloatToIntegerConversionIR(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "linux", GOARCH: "amd64"})
	defer prog.Dispose()
	pkg := prog.NewPackage("floatconvert", "floatconvert")

	tests := []struct {
		name  string
		typ   *types.Basic
		wants []string
	}{
		{name: "I8", typ: types.Typ[types.Int8], wants: []string{"fptosi double", "to i32", "trunc i32", "to i8"}},
		{name: "I32", typ: types.Typ[types.Int32], wants: []string{"fcmp olt double", "fcmp oge double", "fptosi double", "i32 -2147483648"}},
		{name: "I64", typ: types.Typ[types.Int64], wants: []string{"fptosi double", "to i64", "i64 -9223372036854775808"}},
		{name: "U8", typ: types.Typ[types.Uint8], wants: []string{"fptosi double", "to i32", "trunc i32", "to i8"}},
		{name: "U32", typ: types.Typ[types.Uint32], wants: []string{"fptosi double", "to i64", "trunc i64", "to i32"}},
		{name: "U64", typ: types.Typ[types.Uint64], wants: []string{"fcmp oge double", "fsub double", "fptosi double", "or i64"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := types.NewTuple(types.NewParam(0, nil, "x", types.Typ[types.Float64]))
			results := types.NewTuple(types.NewParam(0, nil, "", tt.typ))
			sig := types.NewSignatureType(nil, nil, nil, params, results, false)
			fn := pkg.NewFunc(tt.name, sig, GoDecl)
			b := fn.MakeBody(1)
			b.Return(b.Convert(prog.Type(tt.typ, GoDecl), fn.Param(0)))

			ir := fn.impl.String()
			for _, want := range tt.wants {
				if !strings.Contains(ir, want) {
					t.Errorf("conversion IR missing %q:\n%s", want, ir)
				}
			}
		})
	}
}

func Test386FloatToIntegerConversionIR(t *testing.T) {
	prog := NewProgram(&Target{GOOS: "windows", GOARCH: "386"})
	defer prog.Dispose()
	runtimePkg := types.NewPackage(PkgRuntime, "runtime")
	params := types.NewTuple(types.NewVar(token.NoPos, runtimePkg, "d", types.Typ[types.Float64]))
	for _, resultType := range []struct {
		name string
		typ  types.Type
	}{
		{name: "Float64ToInt64", typ: types.Typ[types.Int64]},
		{name: "Float64ToUint64", typ: types.Typ[types.Uint64]},
	} {
		results := types.NewTuple(types.NewVar(token.NoPos, runtimePkg, "", resultType.typ))
		sig := types.NewSignatureType(nil, nil, nil, params, results, false)
		if err := runtimePkg.Scope().Insert(types.NewFunc(token.NoPos, runtimePkg, resultType.name, sig)); err != nil {
			t.Fatal(err)
		}
	}
	prog.SetRuntime(runtimePkg)
	pkg := prog.NewPackage("floatconvert386", "floatconvert386")

	tests := []struct {
		name  string
		src   *types.Basic
		dst   *types.Basic
		wants []string
	}{
		{name: "I32", src: types.Typ[types.Float64], dst: types.Typ[types.Int32], wants: []string{"fptosi double", "to i32", "i32 -2147483648"}},
		{name: "I64", src: types.Typ[types.Float64], dst: types.Typ[types.Int64], wants: []string{"Float64ToInt64", "(double"}},
		{name: "U32", src: types.Typ[types.Float64], dst: types.Typ[types.Uint32], wants: []string{"fcmp olt double", "fcmp oge double", "fptosi double", "i64 -9223372036854775808", "trunc i64", "to i32"}},
		{name: "Uint", src: types.Typ[types.Float64], dst: types.Typ[types.Uint], wants: []string{"fptosi double", "i64 -9223372036854775808", "trunc i64", "to i32"}},
		{name: "Uintptr", src: types.Typ[types.Float64], dst: types.Typ[types.Uintptr], wants: []string{"fptosi double", "i64 -9223372036854775808", "trunc i64", "to i32"}},
		{name: "F32ToU32", src: types.Typ[types.Float32], dst: types.Typ[types.Uint32], wants: []string{"fptosi float", "i64 -9223372036854775808", "trunc i64", "to i32"}},
		{name: "U64", src: types.Typ[types.Float64], dst: types.Typ[types.Uint64], wants: []string{"Float64ToUint64", "(double"}},
		{name: "F32ToI64", src: types.Typ[types.Float32], dst: types.Typ[types.Int64], wants: []string{"fpext float", "to double", "Float64ToInt64"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := types.NewTuple(types.NewParam(0, nil, "x", tt.src))
			results := types.NewTuple(types.NewParam(0, nil, "", tt.dst))
			sig := types.NewSignatureType(nil, nil, nil, params, results, false)
			fn := pkg.NewFunc(tt.name, sig, GoDecl)
			b := fn.MakeBody(1)
			b.Return(b.Convert(prog.Type(tt.dst, GoDecl), fn.Param(0)))

			ir := fn.impl.String()
			for _, want := range tt.wants {
				if !strings.Contains(ir, want) {
					t.Errorf("conversion IR missing %q:\n%s", want, ir)
				}
			}
		})
	}
}

// Check the conversion width as well as signedness: directly saturating to a
// narrow Go type would change the required wide-conversion-then-truncation.
func TestSaturatingFloatToIntegerConversionIR(t *testing.T) {
	for _, arch := range []string{"amd64", "386", "arm64", "riscv64", "wasm"} {
		t.Run(arch, func(t *testing.T) {
			target := &Target{GOOS: "linux", GOARCH: arch}
			if arch == "wasm" {
				target.GOOS = "wasip1"
			}
			if arch == "amd64" || arch == "386" {
				target.SaturatingFloatToInt = true
			}
			prog := NewProgram(target)
			defer prog.Dispose()
			pkg := prog.NewPackage("floatconvert", "floatconvert")
			for _, src := range []struct {
				typ    *types.Basic
				suffix string
			}{
				{types.Typ[types.Float32], "f32"},
				{types.Typ[types.Float64], "f64"},
			} {
				for _, dst := range []struct {
					typ       *types.Basic
					intrinsic string
					trunc     string
				}{
					{types.Typ[types.Int8], "fptosi.sat.i32", "trunc i32"},
					{types.Typ[types.Int16], "fptosi.sat.i32", "trunc i32"},
					{types.Typ[types.Int32], "fptosi.sat.i32", ""},
					{types.Typ[types.Int64], "fptosi.sat.i64", ""},
					{types.Typ[types.Uint8], "fptosi.sat.i32", "trunc i32"},
					{types.Typ[types.Uint16], "fptosi.sat.i32", "trunc i32"},
					{types.Typ[types.Uint32], "fptosi.sat.i64", "trunc i64"},
					{types.Typ[types.Uint64], "fptoui.sat.i64", ""},
				} {
					name := src.suffix + "To" + dst.typ.Name()
					t.Run(name, func(t *testing.T) {
						params := types.NewTuple(types.NewParam(0, nil, "x", src.typ))
						results := types.NewTuple(types.NewParam(0, nil, "", dst.typ))
						fn := pkg.NewFunc(name, types.NewSignatureType(nil, nil, nil, params, results, false), GoDecl)
						b := fn.MakeBody(1)
						b.Return(b.Convert(prog.Type(dst.typ, GoDecl), fn.Param(0)))
						ir := fn.impl.String()
						intrinsic, trunc := dst.intrinsic, dst.trunc
						if dst.typ == types.Typ[types.Uint32] && target.SaturatingFloatToInt {
							intrinsic, trunc = "fptoui.sat.i32", ""
						}
						want := "@llvm." + intrinsic + "." + src.suffix + "("
						if !strings.Contains(ir, want) || !strings.Contains(ir, trunc) {
							t.Fatalf("conversion IR missing %q or %q:\n%s", want, trunc, ir)
						}
						if strings.Contains(ir, "fcmp") || strings.Contains(ir, "select") {
							t.Fatalf("conversion still emits manual saturation guards:\n%s", ir)
						}
					})
				}
			}
		})
	}
}
