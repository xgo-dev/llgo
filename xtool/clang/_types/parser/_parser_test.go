//go:build !llgo
// +build !llgo

/*
 * Copyright (c) 2022 The XGo Authors (xgo.dev). All rights reserved.
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

package parser

import (
	"go/token"
	"go/types"
	"testing"

	ctypes "github.com/xgo-dev/llgo/x/clang/types"
)

// -----------------------------------------------------------------------------

var (
	pkg   = types.NewPackage("", "foo")
	scope = pkg.Scope()
)

var (
	nameInt128  = types.NewTypeName(token.NoPos, pkg, "__int128", nil)
	nameUint128 = types.NewTypeName(token.NoPos, pkg, "__uint128", nil)
	tyInt128    = types.NewNamed(nameInt128, types.Typ[types.String], nil)
	tyUint128   = types.NewNamed(nameUint128, types.Typ[types.Rune], nil)
)

func init() {
	aliasType(scope, pkg, "char", types.Typ[types.Int8])
	aliasType(scope, pkg, "void", ctypes.Void)
	aliasType(scope, pkg, "float", types.Typ[types.Float32])
	aliasType(scope, pkg, "double", types.Typ[types.Float64])
	aliasType(scope, pkg, "uint", types.Typ[types.Uint32])
	aliasType(scope, pkg, ctypes.MangledName("struct", "ConstantString"), tyConstantString)
	aliasType(scope, pkg, ctypes.MangledName("union", "arg"), tyArg)
	aliasType(scope, pkg, "va_list", ctypes.Valist)

	scope.Insert(nameInt128)
}

func aliasType(scope *types.Scope, pkg *types.Package, name string, typ types.Type) {
	o := types.NewTypeName(token.NoPos, pkg, name, typ)
	scope.Insert(o)
}

var (
	tnameConstantString = types.NewTypeName(token.NoPos, pkg, "ConstantString", nil)
	tnameArg            = types.NewTypeName(token.NoPos, pkg, "UnionArg", nil)
)

var (
	tyChar           = types.Typ[types.Int8]
	tyUchar          = types.Typ[types.Uint8]
	tyInt16          = types.Typ[types.Int16]
	tyUint16         = types.Typ[types.Uint16]
	tyInt32          = types.Typ[types.Int32]
	tyUint32         = types.Typ[types.Uint32]
	tyInt64          = types.Typ[types.Int64]
	tyUint64         = types.Typ[types.Uint64]
	tyInt            = ctypes.Int
	tyInt100         = types.NewArray(tyInt, 100)
	tyInt3           = types.NewArray(tyInt, 3)
	tyInt3_100       = types.NewArray(tyInt3, 100)
	tyPInt3_100      = types.NewPointer(tyInt3_100)
	tyPInt100        = types.NewPointer(tyInt100)
	tyUint           = ctypes.Uint
	tyString         = types.Typ[types.String]
	tyCharPtr        = types.NewPointer(tyChar)
	tyCharPtrPtr     = types.NewPointer(tyCharPtr)
	tyConstantString = types.NewNamed(tnameConstantString, tyString, nil)
	tyArg            = types.NewNamed(tnameArg, tyString, nil)
	tyEmptyInterface = types.NewInterfaceType(nil, nil)
)

var (
	paramInt        = types.NewParam(token.NoPos, pkg, "", tyInt)
	paramVoidPtr    = types.NewParam(token.NoPos, pkg, "", ctypes.UnsafePointer)
	paramCharPtrPtr = types.NewParam(token.NoPos, pkg, "", tyCharPtrPtr)
	paramAnySlice   = types.NewParam(token.NoPos, pkg, "", types.NewSlice(tyEmptyInterface))
	paramPAnySlice  = types.NewParam(token.NoPos, pkg, "", types.NewPointer(types.NewSlice(tyEmptyInterface)))
)

var (
	typesInt     = types.NewTuple(paramInt)
	typesIntVA   = types.NewTuple(paramInt, paramAnySlice)
	typesIntPVA  = types.NewTuple(paramInt, paramPAnySlice)
	typesVoidPtr = types.NewTuple(paramVoidPtr)
	typesPICC    = types.NewTuple(paramVoidPtr, paramInt, paramCharPtrPtr, paramCharPtrPtr)
)

func newFn(in, out *types.Tuple) types.Type {
	return types.NewSignature(nil, in, out, false)
}

func newFnv(in, out *types.Tuple) types.Type {
	return types.NewSignature(nil, in, out, true)
}

func newFnProto(in, out *types.Tuple, variadic bool) types.Type {
	return ctypes.NewFunc(in, out, variadic)
}

var (
	tyFnHandle    = newFn(typesInt, nil)
	paramFnHandle = types.NewParam(token.NoPos, pkg, "", tyFnHandle)
	typesIF       = types.NewTuple(paramInt, paramFnHandle)
	typesF        = types.NewTuple(paramFnHandle)
)

// -----------------------------------------------------------------------------

type testCase struct {
	qualType string
	flags    int
	anonym   types.Type
	typ      types.Type
	err      string
}

var cases = []testCase{
	{qualType: "int", typ: tyInt},
	{qualType: "unsigned int", typ: tyUint},
	{qualType: "struct ConstantString", typ: tyConstantString},
	{qualType: "union arg", typ: tyArg},
	{qualType: "volatile signed int", typ: tyInt},
	{qualType: "__int128", typ: tyInt128},
	{qualType: "signed", typ: tyInt},
	{qualType: "signed short", typ: tyInt16},
	{qualType: "signed long", typ: ctypes.Long},
	{qualType: "unsigned", typ: tyUint},
	{qualType: "uint", typ: tyUint32},
	{qualType: "unsigned char", typ: tyUchar},
	{qualType: "unsigned __int128", typ: tyUint128},
	{qualType: "unsigned long", typ: ctypes.Ulong},
	{qualType: "unsigned long long", typ: tyUint64},
	{qualType: "long double", typ: ctypes.LongDouble},
	{qualType: "_Complex float", typ: types.Typ[types.Complex64]},
	{qualType: "_Complex double", typ: types.Typ[types.Complex128]},
	{qualType: "_Complex long double", typ: types.Typ[types.Complex128]},
	{qualType: "int (*)(void)", typ: newFn(nil, typesInt)},
	{qualType: "int (void)", typ: newFnProto(nil, typesInt, false)},
	{qualType: "void (void) __attribute__((noreturn))", typ: newFnProto(nil, nil, false)},
	{qualType: "void (*)(void *)", typ: newFn(typesVoidPtr, nil)},
	{qualType: "void (^ _Nonnull)(void)", typ: newFn(nil, nil)},
	{qualType: "void (int, ...)", typ: newFnProto(typesIntVA, nil, true)},
	{qualType: "void (int, va_list*)", typ: newFn(typesIntPVA, nil)},
	{qualType: "va_list *", typ: types.NewPointer(types.NewSlice(tyEmptyInterface))},
	{qualType: "int (*)()", typ: newFn(nil, typesInt)},
	{qualType: "int (*)(int, ...)", typ: newFnv(typesIntVA, typesInt)},
	{qualType: "int (*)(int, struct __va_list_tag*)", typ: newFn(typesIntVA, typesInt)},
	{qualType: "int (*volatile)(int, struct __va_list_tag* restrict)", typ: newFn(typesIntVA, typesInt)},
	{qualType: "int (const char *, const char *, unsigned int)", flags: FlagGetRetType, typ: tyInt},
	{qualType: "const char *restrict", typ: tyCharPtr},
	{qualType: "const char [7]", typ: types.NewArray(tyChar, 7)},
	{qualType: "const char [7]", flags: FlagIsParam, typ: tyCharPtr},
	{qualType: "char []", flags: FlagIsStructField, typ: types.NewArray(tyChar, 0)},
	{qualType: "char []", flags: FlagIsExtern, typ: types.NewArray(tyChar, -1)},
	{qualType: "char []", flags: 0, err: emsgDefArrWithoutLen},
	{qualType: "char []", flags: FlagIsTypedef, typ: types.NewArray(tyChar, -1)},
	{qualType: "char []", flags: FlagIsParam, typ: tyCharPtr},
	{qualType: "int [100][3]", typ: tyInt3_100},
	{qualType: "int (*)[100][3]", typ: tyPInt3_100},
	{qualType: "int (*)[100]", typ: tyPInt100},
	{qualType: "int (*const [2])(void *)", typ: types.NewArray(newFn(typesVoidPtr, typesInt), 2)},
	{qualType: "char *", typ: tyCharPtr},
	{qualType: "void", typ: ctypes.Void},
	{qualType: "void *", typ: ctypes.UnsafePointer},
	{qualType: "int (*_Nullable)(void *, int, char **, char **)", typ: newFn(typesPICC, typesInt)},
	{qualType: "void (*(*)(int, void (*)(int)))(int)", typ: newFn(typesIF, typesF)},
	{qualType: "void (*(int, void (*)(int)))(int)", typ: newFnProto(typesIF, typesF, false)},
	{qualType: "void (*(int, void (*)(int)))(int)", flags: FlagGetRetType, typ: tyFnHandle},
	{qualType: "int (*)(void *, int, const char *, void (**)(void *, int, void **), void **)"},
	{qualType: "struct (anonymous) [2]", anonym: tyInt, typ: types.NewArray(tyInt, 2)},
	{qualType: "enum a", typ: ctypes.Int},
}

type baseEnv struct {
	pkg       *types.Package
	tyInt128  types.Type
	tyUint128 types.Type
}

func (p *baseEnv) Pkg() *types.Package {
	return p.pkg
}

func (p *baseEnv) Int128() types.Type {
	return p.tyInt128
}

func (p *baseEnv) Uint128() types.Type {
	return p.tyUint128
}

func TestCases(t *testing.T) {
	sel := ""
	for _, c := range cases {
		if sel != "" && c.qualType != sel {
			continue
		}
		t.Run(c.qualType, func(t *testing.T) {
			conf := &Config{
				Scope: scope, Flags: c.flags, Anonym: c.anonym,
				ParseEnv: &baseEnv{pkg: pkg, tyInt128: tyInt128, tyUint128: tyUint128},
			}
			typ, _, err := ParseType(c.qualType, conf)
			if err != nil {
				if errMsgOf(err) != c.err {
					t.Fatal("ParseType:", err, ", expected:", c.err)
				}
			} else if c.typ != nil && !ctypes.Identical(typ, c.typ) {
				t.Fatal("ParseType:", typ, ", expected:", c.typ)
			}
		})
	}
}

func errMsgOf(err error) string {
	if e, ok := err.(*ParseTypeError); ok {
		return e.ErrMsg
	}
	return err.Error()
}

// -----------------------------------------------------------------------------

func TestIsArrayWithoutLen(t *testing.T) {
	_, _, err := ParseType("byte[]", &Config{Scope: types.Universe})
	if !IsArrayWithoutLen(err) {
		t.Fatal("ParseType:", err)
	}
	_, _, err = ParseType("byte[]", &Config{Scope: types.Universe, Flags: FlagIsExtern})
	if IsArrayWithoutLen(err) {
		t.Fatal("ParseType:", err)
	}
}

// -----------------------------------------------------------------------------
