// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

// Runtime type representation.

package runtime

import (
	"github.com/xgo-dev/llgo/runtime/abi"
)

type _type = abi.Type
type ptrtype = abi.PtrType
type functype = abi.FuncType
type namedclosuretype = abi.NamedClosureType
type closurefunctype = abi.ClosureFuncType
type qualifiedFuncType = abi.QualifiedFuncType
type slicetype = abi.SliceType
type chantype = abi.ChanType
type structfield = abi.StructField
type uncommonType = abi.UncommonType

/*
type maptype = abi.MapType

type arraytype = abi.ArrayType

type chantype = abi.ChanType

type slicetype = abi.SliceType

type functype = abi.FuncType

type ptrtype = abi.PtrType

type name = abi.Name

type structtype = abi.StructType
*/
