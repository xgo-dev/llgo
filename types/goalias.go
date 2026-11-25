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

import gotypes "go/types"

// Leaf constants and enums aliased directly from go/types. None of these types
// expose go/types.Type or go/types.Object in their methods, so downstream
// packages stay purely in LLGo's type system with zero dual-type leakage.
type (
	BasicKind     = gotypes.BasicKind
	BasicInfo     = gotypes.BasicInfo
	ChanDir       = gotypes.ChanDir
	SelectionKind = gotypes.SelectionKind
	ImportMode    = gotypes.ImportMode
)

const (
	Invalid BasicKind = gotypes.Invalid

	// Predeclared types
	Bool          BasicKind = gotypes.Bool
	Int           BasicKind = gotypes.Int
	Int8          BasicKind = gotypes.Int8
	Int16         BasicKind = gotypes.Int16
	Int32         BasicKind = gotypes.Int32
	Int64         BasicKind = gotypes.Int64
	Uint          BasicKind = gotypes.Uint
	Uint8         BasicKind = gotypes.Uint8
	Uint16        BasicKind = gotypes.Uint16
	Uint32        BasicKind = gotypes.Uint32
	Uint64        BasicKind = gotypes.Uint64
	Uintptr       BasicKind = gotypes.Uintptr
	Float32       BasicKind = gotypes.Float32
	Float64       BasicKind = gotypes.Float64
	Complex64     BasicKind = gotypes.Complex64
	Complex128    BasicKind = gotypes.Complex128
	String        BasicKind = gotypes.String
	UnsafePointer BasicKind = gotypes.UnsafePointer

	// Untyped basic types
	UntypedBool    BasicKind = gotypes.UntypedBool
	UntypedInt     BasicKind = gotypes.UntypedInt
	UntypedRune    BasicKind = gotypes.UntypedRune
	UntypedFloat   BasicKind = gotypes.UntypedFloat
	UntypedComplex BasicKind = gotypes.UntypedComplex
	UntypedString  BasicKind = gotypes.UntypedString
	UntypedNil     BasicKind = gotypes.UntypedNil

	// Aliases
	Byte BasicKind = gotypes.Byte
	Rune BasicKind = gotypes.Rune
)

const (
	IsBoolean  BasicInfo = gotypes.IsBoolean
	IsInteger  BasicInfo = gotypes.IsInteger
	IsUnsigned BasicInfo = gotypes.IsUnsigned
	IsFloat    BasicInfo = gotypes.IsFloat
	IsComplex  BasicInfo = gotypes.IsComplex
	IsString   BasicInfo = gotypes.IsString
	IsUntyped  BasicInfo = gotypes.IsUntyped

	IsOrdered   BasicInfo = gotypes.IsOrdered
	IsConstType BasicInfo = gotypes.IsConstType
)

const (
	SendRecv ChanDir = gotypes.SendRecv
	SendOnly ChanDir = gotypes.SendOnly
	RecvOnly ChanDir = gotypes.RecvOnly
)

const (
	FieldVal   SelectionKind = gotypes.FieldVal
	MethodVal  SelectionKind = gotypes.MethodVal
	MethodExpr SelectionKind = gotypes.MethodExpr
)
