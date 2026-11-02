/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
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

package ssa

import (
	"crypto/sha256"
	"encoding/binary"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strconv"

	"github.com/xgo-dev/llgo/internal/genmethod"
	"github.com/xgo-dev/llgo/ssa/abi"
	"github.com/xgo-dev/llvm"
)

// -----------------------------------------------------------------------------

/*
type Type struct {
	Size_       uintptr
	PtrBytes    uintptr // number of (prefix) bytes in the type that can contain pointers
	Hash        uint32  // hash of type; avoids computation in hash tables
	TFlag       TFlag   // extra type information flags
	Align_      uint8   // alignment of variable with this type
	FieldAlign_ uint8   // alignment of struct field with this type
	Kind_       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	Equal func(unsafe.Pointer, unsafe.Pointer) bool
	// GCData stores the GC type data for the garbage collector.
	// If the KindGCProg bit is set in kind, GCData is a GC program.
	// Otherwise it is a ptrmask bitmap. See mbitmap.go for details.
	GCData     *byte
	Str_       string // string form
	PtrToThis_ *Type  // type for pointer to this type, may be nil
}
*/

var (
	// func(unsafe.Pointer, unsafe.Pointer) bool
	equalFunc = types.NewSignature(nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.UnsafePointer]),
		types.NewVar(token.NoPos, nil, "", types.Typ[types.UnsafePointer])),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Bool])), false)
	// func(p unsafe.Pointer, h uintptr) uintptr
	hashFunc = types.NewSignature(nil, types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.UnsafePointer]),
		types.NewVar(token.NoPos, nil, "", types.Typ[types.Uintptr])),
		types.NewTuple(types.NewVar(token.NoPos, nil, "", types.Typ[types.Uintptr])), false)
)

func directIfaceType(t types.Type) bool {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		return directIfaceType(t.Underlying())
	case *types.Pointer:
		return true
	case *types.Chan, *types.Map, *types.Signature:
		return true
	case *types.Basic:
		return t.Kind() == types.UnsafePointer
	case *types.Array:
		return t.Len() == 1 && directIfaceType(t.Elem())
	case *types.Struct:
		return t.NumFields() == 1 && directIfaceType(t.Field(0).Type())
	}
	return false
}

// staticPtrToThis reports whether *T must have a statically generated type
// descriptor. Runtime reflection can synthesize descriptors for methodless
// pointer types, but it cannot synthesize their method metadata.
func staticPtrToThis(t types.Type) (types.Type, bool) {
	if _, ok := types.Unalias(t).(*types.Pointer); ok {
		return nil, false
	}
	ptr := types.NewPointer(t)
	return ptr, types.NewMethodSet(ptr).Len() != 0
}

func (b Builder) abiCommonFields(t types.Type, name string, hasUncommon bool, global llvm.Value) (fields []llvm.Value) {
	prog := b.Prog
	ab := prog.abi
	// Size uintptr
	fields = append(fields, prog.IntVal(uint64(ab.Size(t)), prog.Uintptr()).impl)
	// PtrBytes uintptr
	fields = append(fields, prog.IntVal(uint64(ab.PtrBytes(t)), prog.Uintptr()).impl)
	// Hash uint32
	h := sha256.Sum256([]byte(name))
	hash := binary.LittleEndian.Uint32(h[:4])
	fields = append(fields, prog.IntVal(uint64(hash), prog.Uint32()).impl)
	// TFlag uint8
	tflag := ab.TFlag(t)
	if hasUncommon {
		tflag |= abi.TFlagUncommon
	}
	fields = append(fields, prog.IntVal(uint64(tflag), prog.Byte()).impl)
	// Align uint8
	align := prog.IntVal(uint64(ab.Align(t)), prog.Byte()).impl
	fields = append(fields, align)
	// FieldAlign uint8
	fieldAlign := prog.IntVal(uint64(ab.FieldAlign(t)), prog.Byte()).impl
	fields = append(fields, fieldAlign)
	// Kind uint8
	kind := uint8(ab.Kind(t))
	if directIfaceType(t) {
		kind |= uint8(abi.KindDirectIface)
	}
	fields = append(fields, prog.IntVal(uint64(kind), prog.Byte()).impl)
	// Equal func(unsafe.Pointer, unsafe.Pointer) bool
	var equal Expr
	switch name := ab.EqualName(t); name {
	case "":
		equal = prog.Nil(prog.Type(equalFunc, GoDecl))
	case "structequal", "arrayequal":
		// Hidden regular-memory structs may lose their field metadata during
		// dev Wasm DCE. Compare bytes without consulting that metadata.
		if prog.enableDeadcodeDrop && prog.target.effectiveGOARCH() == "wasm" && ab.IsRegularMemory(t) {
			equal = b.Pkg.regularMemoryEqual(uint64(ab.Size(t)))
			b.Pkg.recordAbiTypeFakeUse(global, equal.impl)
			typ := prog.Type(equal.raw.Type, GoDecl)
			equal = checkExpr(equal, typ.raw.Type, b)
		} else {
			equal = b.Pkg.rtEnvFunc(RuntimeSymbol(name, "structequal", "arrayequal"))
			b.Pkg.recordAbiTypeFakeUse(global, equal.impl)
			env := b.abiType(t)
			equal = b.aggregateValue(prog.Type(equalFunc, GoDecl), equal.impl, env.impl)
		}
	default:
		equal = b.Pkg.rtFunc(RuntimeSymbol(name,
			"memequal0", "memequal8", "memequal16", "memequal32", "memequal64", "memequalptr", "f32equal",
			"f64equal", "c64equal", "c128equal", "strequal", "nilinterequal", "interequal",
		))
		b.Pkg.recordAbiTypeFakeUse(global, equal.impl)
		typ := b.Prog.Type(equal.raw.Type, GoDecl)
		equal = checkExpr(equal, typ.raw.Type, b)
	}
	fields = append(fields, equal.impl)
	// GCData     *byte
	fields = append(fields, prog.Nil(prog.Pointer(prog.Byte())).impl)
	// Str_       string
	fields = append(fields, b.Str(ab.Str(t)).impl)
	// PtrToThis_ *Type
	ptr, static := staticPtrToThis(t)
	if !static {
		fields = append(fields, prog.Nil(prog.AbiTypePtr()).impl)
	} else {
		fields = append(fields, b.abiType(ptr).impl)
	}
	return
}

// Share one metadata-independent equality function per size, including arrays.
func (p Package) regularMemoryEqual(size uint64) Expr {
	switch size {
	case 0, 1, 2, 4, 8, 16:
		return p.rtFunc(RuntimeSymbol("memequal"+strconv.FormatUint(size*8, 10),
			"memequal0", "memequal8", "memequal16", "memequal32", "memequal64", "memequal128"))
	}
	name := "_llgo_memequal." + strconv.FormatUint(size, 10)
	fn := p.FuncOf(name)
	if fn == nil {
		fn = p.NewFunc(name, equalFunc, GoDecl)
		fn.impl.SetLinkage(llvm.WeakODRLinkage)
		builder := fn.MakeBody(1)
		defer builder.Dispose()
		builder.Return(builder.Call(p.rtFunc("memequal"), fn.Param(0), fn.Param(1), p.Prog.IntVal(size, p.Prog.Uintptr())))
		builder.EndBuild()
	}
	return fn.Expr
}

// Match typehashImpl exactly, including its distinct 4/8-byte algorithms.
func (p Package) regularMemoryHash(size uint64) Expr {
	// Omit size 0: typehashImpl calls memhash(p, seed, 0), whereas memhash0
	// returns the seed unchanged. Zero-sized keys need the wrapper below.
	switch size {
	case 1, 2, 4, 8, 16:
		return p.rtFunc(RuntimeSymbol("memhash"+strconv.FormatUint(size*8, 10),
			"memhash8", "memhash16", "memhash32", "memhash64", "memhash128"))
	}
	name := "_llgo_memhash." + strconv.FormatUint(size, 10)
	fn := p.FuncOf(name)
	if fn == nil {
		fn = p.NewFunc(name, hashFunc, GoDecl)
		fn.impl.SetLinkage(llvm.WeakODRLinkage)
		builder := fn.MakeBody(1)
		defer builder.Dispose()
		builder.Return(builder.Call(p.rtFunc("memhash"), fn.Param(0), fn.Param(1), p.Prog.IntVal(size, p.Prog.Uintptr())))
		builder.EndBuild()
	}
	return fn.Expr
}

func (b Builder) abiMapHasher(key types.Type, global llvm.Value) Expr {
	prog, pkg := b.Prog, b.Pkg
	var hash Expr
	if prog.enableDeadcodeDrop && prog.target.effectiveGOARCH() == "wasm" {
		if prog.abi.IsRegularMemory(key) {
			hash = pkg.regularMemoryHash(uint64(prog.abi.Size(key)))
		} else if basic, ok := key.Underlying().(*types.Basic); ok {
			var name string
			switch basic.Kind() {
			case types.String:
				name = "strhash"
			case types.Float32:
				name = "f32hash"
			case types.Float64:
				name = "f64hash"
			case types.Complex64:
				name = "c64hash"
			case types.Complex128:
				name = "c128hash"
			}
			if name != "" {
				hash = pkg.rtFunc(RuntimeSymbol(name, "strhash", "f32hash", "f64hash", "c64hash", "c128hash"))
			}
		}
	}
	if !hash.IsNil() {
		pkg.recordAbiTypeFakeUse(global, hash.impl)
		return checkExpr(hash, prog.Type(hashFunc, GoDecl).raw.Type, b)
	}
	hash = pkg.rtEnvFunc("typehash")
	pkg.recordAbiTypeFakeUse(global, hash.impl)
	return b.aggregateValue(prog.Type(hashFunc, GoDecl), hash.impl, b.abiType(key).impl)
}

func (b Builder) abiMapBucket(t *types.Map, bucket types.Type) Expr {
	prog, pkg := b.Prog, b.Pkg
	if !prog.enableDeadcodeDrop || prog.target.effectiveGOARCH() != "wasm" {
		return b.abiType(bucket)
	}
	// Buckets are allocator records, not source Go values. Like reflect.bucketOf,
	// retain their physical layout without comparison algorithms or fields.
	// A separate symbol prevents sharing the stripped descriptor with any Go
	// struct of the same layout, and keeps it out of the reflection type list.
	name, _ := prog.abi.TypeName(t)
	name += "$bucket"
	g := pkg.VarOf(name)
	if g == nil {
		ab := prog.abi
		typ := prog.AbiType()
		g = pkg.doNewVar(name, prog.AbiTypePtr())
		fields := []llvm.Value{
			prog.IntVal(uint64(ab.Size(bucket)), prog.Uintptr()).impl,
			prog.IntVal(uint64(ab.PtrBytes(bucket)), prog.Uintptr()).impl,
			prog.IntVal(0, prog.Uint32()).impl, // Hash
			prog.IntVal(0, prog.Byte()).impl,   // TFlag
			prog.IntVal(uint64(ab.Align(bucket)), prog.Byte()).impl,
			prog.IntVal(uint64(ab.FieldAlign(bucket)), prog.Byte()).impl,
			prog.IntVal(uint64(ab.Kind(bucket)), prog.Byte()).impl,
			prog.Nil(prog.Type(equalFunc, GoDecl)).impl,
			prog.Nil(prog.Pointer(prog.Byte())).impl, // GCData, as for other static types
			b.Str("bucket").impl,
			prog.Nil(prog.AbiTypePtr()).impl,
		}
		g.impl.SetInitializer(prog.constStructValue(typ, fields))
		g.impl.SetGlobalConstant(true)
		pkg.setODRLinkage(g.impl, llvm.WeakODRLinkage)
	}
	return Expr{g.impl, prog.AbiTypePtr()}
}

/*
type StructField struct {
	Name_  string  // name is always non-empty
	Typ    *Type   // type of field
	Offset uintptr // byte offset of field

	Tag_      string
	Embedded_ bool
}
*/

func (b Builder) abiStructFields(t *types.Struct, name string) llvm.Value {
	prog := b.Prog
	n := t.NumFields()
	if n == 0 {
		return prog.Nil(prog.rtType("Slice")).impl
	}
	g := b.Pkg.VarOf(name)
	if g == nil {
		ft := prog.rtType("structfield")
		offsets := b.abiStructFieldOffsets(t)
		fields := make([]llvm.Value, n)
		for i := 0; i < n; i++ {
			f := t.Field(i)
			var values []llvm.Value
			values = append(values, b.Str(f.Name()).impl)
			values = append(values, b.abiType(abi.PublicType(f.Type())).impl)
			values = append(values, prog.IntVal(uint64(offsets[i]), prog.Uintptr()).impl)
			values = append(values, b.Str(t.Tag(i)).impl)
			values = append(values, prog.BoolVal(f.Embedded()).impl)
			fields[i] = prog.constStructValue(ft, values)
		}
		atyp := prog.rawType(types.NewArray(ft.RawType(), int64(n)))
		data := Expr{llvm.ConstArray(ft.ll, fields), atyp}
		g = b.Pkg.doNewVar(name, prog.Pointer(atyp))
		g.Init(data)
		g.impl.SetGlobalConstant(true)
		b.Pkg.setODRLinkage(g.impl, llvm.WeakODRLinkage)
	}
	size := uint64(n)
	return prog.constStructValue(prog.rtType("Slice"), []llvm.Value{
		g.impl,
		prog.IntVal(size, prog.Int()).impl,
		prog.IntVal(size, prog.Int()).impl,
	})
}

func (b Builder) abiStructFieldOffsets(t *types.Struct) []int64 {
	fields := make([]*types.Var, t.NumFields())
	for i := range fields {
		fields[i] = t.Field(i)
	}
	return b.Prog.abi.Sizes.Offsetsof(fields)
}

/*
type InterfaceType struct {
	Type
	PkgPath_ string    // import path
	Methods  []Imethod // sorted by hash
}

type Imethod struct {
	Name_ string    // name of method
	Typ_  *FuncType // .(*FuncType) underneath
}
*/

func (b Builder) abiInterfaceImethods(t *types.Interface, name string) llvm.Value {
	prog := b.Prog
	n := t.NumMethods()
	if n == 0 {
		return prog.Nil(prog.rtType("Slice")).impl
	}
	g := b.Pkg.VarOf(name)
	if g == nil {
		ft := prog.rtType("Imethod")
		fields := make([]llvm.Value, n)
		for i := 0; i < n; i++ {
			f := t.Method(i)
			var values []llvm.Value
			name := f.Name()
			if !token.IsExported(name) {
				name = abi.FullName(f.Pkg(), name)
			}
			values = append(values, b.Str(name).impl)
			ftyp := funcType(prog, f.Type())
			values = append(values, b.abiType(ftyp).impl)
			fields[i] = prog.constStructValue(ft, values)
		}
		atyp := prog.rawType(types.NewArray(ft.RawType(), int64(n)))
		data := Expr{llvm.ConstArray(ft.ll, fields), atyp}
		g = b.Pkg.doNewVar(name, prog.Pointer(atyp))
		g.Init(data)
		g.impl.SetGlobalConstant(true)
		b.Pkg.setODRLinkage(g.impl, llvm.WeakODRLinkage)
	}
	size := uint64(n)
	return prog.constStructValue(prog.rtType("Slice"), []llvm.Value{
		g.impl,
		prog.IntVal(size, prog.Int()).impl,
		prog.IntVal(size, prog.Int()).impl,
	})
}

func (b Builder) abiTuples(t *types.Tuple, name string) llvm.Value {
	prog := b.Prog
	n := t.Len()
	if n == 0 {
		return prog.Nil(prog.rtType("Slice")).impl
	}
	g := b.Pkg.VarOf(name)
	if g == nil {
		fields := make([]llvm.Value, n)
		for i := 0; i < n; i++ {
			fields[i] = b.abiType(abi.PublicType(t.At(i).Type())).impl
		}
		ft := prog.AbiTypePtr()
		atyp := prog.rawType(types.NewArray(ft.RawType(), int64(n)))
		for i, field := range fields {
			fields[i] = prog.toStorageConstant(ft, field)
		}
		data := Expr{llvm.ConstArray(prog.storageType(ft), fields), atyp}
		g = b.Pkg.doNewVar(name, prog.Pointer(atyp))
		g.Init(data)
		g.impl.SetGlobalConstant(true)
		b.Pkg.setODRLinkage(g.impl, llvm.WeakODRLinkage)
	}
	size := uint64(n)
	return prog.constStructValue(prog.rtType("Slice"), []llvm.Value{
		g.impl,
		prog.IntVal(size, prog.Int()).impl,
		prog.IntVal(size, prog.Int()).impl,
	})
}

func (b Builder) abiExtendedFields(t types.Type, name string, global llvm.Value) (fields []llvm.Value) {
	prog := b.Prog
	pkg := b.Pkg
	switch t := types.Unalias(t).(type) {
	case *types.Basic:
	case *types.Pointer:
		fields = []llvm.Value{
			b.abiType(abi.PublicType(t.Elem())).impl,
		}
	case *types.Chan:
		dir, _ := abi.ChanDir(t.Dir())
		fields = []llvm.Value{
			b.abiType(abi.PublicType(t.Elem())).impl,
			prog.IntVal(uint64(dir), prog.Int()).impl,
		}
	case *types.Slice:
		fields = []llvm.Value{
			b.abiType(abi.PublicType(t.Elem())).impl,
		}
	case *types.Array:
		elem := abi.PublicType(t.Elem())
		fields = []llvm.Value{
			b.abiType(elem).impl,
			b.abiType(types.NewSlice(elem)).impl,
			prog.IntVal(uint64(t.Len()), prog.Uintptr()).impl,
		}
	case *types.Map:
		bucket := prog.abi.MapBucket(t)
		flags := prog.abi.MapFlags(t)
		keySize := prog.abi.Size(t.Key())
		if flags&1 != 0 {
			keySize = prog.abi.PtrSize
		}
		elemSize := prog.abi.Size(t.Elem())
		if flags&2 != 0 {
			elemSize = prog.abi.PtrSize
		}
		hasher := b.abiMapHasher(t.Key(), global)
		fields = []llvm.Value{
			b.abiType(abi.PublicType(t.Key())).impl,
			// Map operations use Elem for physical copies during bucket
			// evacuation. Keep LLGo's two-word closure representation here;
			// reflect converts it back to the public function type in Elem.
			b.abiType(t.Elem()).impl,
			b.abiMapBucket(t, bucket).impl,
			hasher.impl,
			prog.IntVal(uint64(keySize), prog.Byte()).impl,
			prog.IntVal(uint64(elemSize), prog.Byte()).impl,
			prog.IntVal(uint64(prog.abi.Size(bucket)), prog.Uint16()).impl,
			prog.IntVal(uint64(flags), prog.Uint32()).impl,
		}
	case *types.Signature:
		name, _ := prog.abi.TypeName(t)
		fields = []llvm.Value{
			b.abiTuples(t.Params(), name+"$in"),
			b.abiTuples(t.Results(), name+"$out"),
		}
		if prog.target.usesWasmReflectBridges() {
			bridges := pkg.wasmReflectBridge(t)
			fields = append(fields, bridges.call.impl, bridges.make.impl)
		}
	case *types.Struct:
		name, _ = prog.abi.TypeName(t)
		var pkgPath string
		n := t.NumFields()
		for i := 0; i < n; i++ {
			if f := t.Field(i); !f.Exported() {
				if pkg := f.Pkg(); pkg != nil {
					pkgPath = reflectPkgPath(pkg)
					break
				}
			}
		}
		fields = []llvm.Value{
			b.Str(pkgPath).impl,
			b.abiStructFields(t, name+"$fields"),
		}
	case *types.Interface:
		name, _ = prog.abi.TypeName(t)
		fields = []llvm.Value{
			b.Str(pkg.Path()).impl,
			b.abiInterfaceImethods(t, name+"$imethods"),
		}
	case *types.Named:
		return b.abiExtendedFields(t.Underlying(), name, global)
	}
	return
}

func (b Builder) recordTypeChildren(parentName string, t types.Type) {
	mb := b.Pkg.metaBuilder
	if mb == nil {
		return
	}
	parent := mb.Sym(parentName)
	for _, child := range b.directTypeChildren(t) {
		childName, _ := b.Prog.abi.TypeName(child)
		mb.AddTypeChild(parent, mb.Sym(childName))
	}
}

func (b Builder) directTypeChildren(t types.Type) []types.Type {
	children := b.structuralTypeChildren(t)
	// Runtime reflection can lazily derive methodless *T descriptors. Keep the
	// type-child edge only when the static descriptor carries method metadata.
	if ptr, static := staticPtrToThis(t); static {
		children = append(children, ptr)
	}
	return children
}

func (b Builder) structuralTypeChildren(t types.Type) []types.Type {
	switch t := types.Unalias(t).(type) {
	case *types.Basic:
		return nil
	case *types.Pointer:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Chan:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Slice:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Array:
		return []types.Type{abi.PublicType(t.Elem())}
	case *types.Map:
		return []types.Type{
			abi.PublicType(t.Key()),
			abi.PublicType(t.Elem()),
		}
	case *types.Signature:
		var children []types.Type
		children = appendTupleTypeChildren(children, t.Params())
		children = appendTupleTypeChildren(children, t.Results())
		return children
	case *types.Struct:
		children := make([]types.Type, 0, t.NumFields())
		for i := 0; i < t.NumFields(); i++ {
			children = append(children, abi.PublicType(t.Field(i).Type()))
		}
		return children
	case *types.Named:
		if basic, ok := t.Underlying().(*types.Basic); ok && basic.Info()&(types.IsFloat|types.IsComplex) != 0 {
			// Panic printing can inspect a named floating-point kind without
			// converting it to the corresponding basic interface type.
			return []types.Type{basic}
		}
		return b.structuralTypeChildren(t.Underlying())
	}
	return nil
}

func appendTupleTypeChildren(children []types.Type, tuple *types.Tuple) []types.Type {
	if tuple == nil {
		return children
	}
	for i := 0; i < tuple.Len(); i++ {
		children = append(children, abi.PublicType(tuple.At(i).Type()))
	}
	return children
}

func (b Builder) abiUncommonPkg(t types.Type) (*types.Package, string) {
retry:
	switch typ := types.Unalias(t).(type) {
	case *types.Pointer:
		t = typ.Elem()
		goto retry
	case *types.Named:
		pkg := typ.Obj().Pkg()
		return pkg, reflectPkgPath(pkg)
	}
	return nil, b.Pkg.Path()
}

func reflectPkgPath(pkg *types.Package) string {
	if pkg == nil {
		return ""
	}
	if pkg.Path() == "command-line-arguments" && pkg.Name() != "" {
		return pkg.Name()
	}
	return abi.PathOf(pkg)
}

func (b Builder) abiUncommonMethodSet(t types.Type) (mset *types.MethodSet, ok bool) {
	prog := b.Prog
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if _, b := t.Underlying().(*types.Interface); b {
			return &types.MethodSet{}, true
		}
		mset := types.NewMethodSet(t)
		if mset.Len() != 0 {
			if prog.compileMethods != nil {
				prog.compileMethods(b.Pkg, t)
			}
		}
		return mset, true
	case *types.Struct, *types.Pointer:
		if mset := types.NewMethodSet(t); mset.Len() != 0 {
			if prog.compileMethods != nil {
				prog.compileMethods(b.Pkg, t)
			}
			return mset, true
		}
	}
	return
}

func abiUncommonMethodSetForDeclaration(t types.Type) (mset *types.MethodSet, ok bool) {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if _, isInterface := t.Underlying().(*types.Interface); isInterface {
			return &types.MethodSet{}, true
		}
		return types.NewMethodSet(t), true
	case *types.Struct, *types.Pointer:
		if mset := types.NewMethodSet(t); mset.Len() != 0 {
			return mset, true
		}
	}
	return
}

/*
type UncommonType struct {
	PkgPath_ string // import path; empty for built-in types like int, string
	Mcount   uint16 // number of methods
	Xcount   uint16 // number of exported methods
	Moff     uint32 // offset from this uncommontype to [mcount]Method
}
*/

func (b Builder) abiUncommonType(t types.Type, methods []*types.Selection) llvm.Value {
	prog := b.Prog
	ft := prog.rtType("uncommonType")
	var fields []llvm.Value
	_, pkgPath := b.abiUncommonPkg(t)
	fields = append(fields, b.Str(pkgPath).impl)
	mcount := len(methods)
	var xcount int
	for i := 0; i < mcount; i++ {
		if ast.IsExported(methods[i].Obj().Name()) {
			xcount++
		}
	}
	moff := prog.SizeOf(ft)
	fields = append(fields, prog.IntVal(uint64(mcount), prog.Uint16()).impl)
	fields = append(fields, prog.IntVal(uint64(xcount), prog.Uint16()).impl)
	fields = append(fields, prog.IntVal(moff, prog.Uint32()).impl)
	return prog.constStructValue(ft, fields)
}

/*
type Method struct {
	Name_ string    // name of method
	Mtyp_ *FuncType // method type (without receiver)
	Ifn_  Text      // fn used in interface call (one-word receiver)
	Tfn_  Text      // fn used for normal method call
}
*/

func (b Builder) abiUncommonMethods(t types.Type, methods []*types.Selection) llvm.Value {
	prog := b.Prog
	ft := prog.rtType("Method")
	n := len(methods)
	fields := make([]llvm.Value, n)
	typeName, _ := prog.abi.TypeName(t)
	for i := 0; i < n; i++ {
		m := methods[i]
		obj := m.Obj().(*types.Func)
		fullName := abiMethodName(obj)
		name := b.Str(fullName).impl
		ifn, tfn := b.abiMethodFuncs(t, m)
		var values []llvm.Value
		values = append(values, name)
		ftyp := funcType(prog, m.Type())
		values = append(values, b.abiType(ftyp).impl)
		values = append(values, ifn)
		values = append(values, tfn)
		fields[i] = prog.constStructValue(ft, values)
		if prog.target.usesWasmReflectBridges() {
			// Type.Method constructs a method-expression signature at runtime.
			// Retain that descriptor with the method wrapper only when reflection
			// can expose it.
			expression := methodExprSignature(m.Type().(*types.Signature))
			b.abiType(expression)
			if mb := b.Pkg.metaBuilder; mb != nil {
				expressionName, _ := prog.abi.TypeName(expression)
				mb.AddOrdinaryEdge(mb.Sym(tfn.Name()), mb.Sym(expressionName))
			}
		}
		if mb := b.Pkg.metaBuilder; mb != nil {
			mtypeName, _ := prog.abi.TypeName(ftyp)
			mb.AddMethodSlot(mb.Sym(typeName), fullName, mb.Sym(mtypeName), mb.Sym(ifn.Name()), mb.Sym(tfn.Name()))
		}
	}
	return llvm.ConstArray(ft.ll, fields)
}

// abiMethodFuncs returns the interface-call and method-expression entry points
// stored in a concrete type's method metadata. Keep this shared with static
// itab construction so both representations select exactly the same wrapper.
func (b Builder) abiMethodFuncs(t types.Type, m *types.Selection) (ifn, tfn llvm.Value) {
	pkg, _ := b.abiUncommonPkg(t)
	anonymous := pkg == nil
	if anonymous {
		pkg = types.NewPackage(b.Pkg.Path(), "")
	}
	obj := m.Obj().(*types.Func)
	mSig := m.Type().(*types.Signature)
	// funcName uses this same receiver package for the emitted wrapper
	// definition; MethodSymbolName must see identical inputs here.
	mSymbolName := MethodSymbolName(pkg, obj, obj.Name())
	tfnFn := b.abiMethodFunc(anonymous, pkg, mSymbolName, mSig)
	// Tfn is used as a method-expression funcval. Its explicit receiver is
	// already part of that semantic signature, so it is a no-env entry.
	tfn = tfnFn.impl
	ifn = tfn
	if _, ok := m.Recv().Underlying().(*types.Pointer); !ok {
		pRecv := types.NewVar(token.NoPos, pkg, "", types.NewPointer(mSig.Recv().Type()))
		pSig := types.NewSignature(pRecv, mSig.Params(), mSig.Results(), mSig.Variadic())
		ifn = b.abiMethodFunc(anonymous, pkg, mSymbolName, pSig).impl
	}
	return
}

func (b Builder) abiInterfaceMethods(mset *types.MethodSet) []*types.Selection {
	n := mset.Len()
	methods := make([]*types.Selection, 0, n)
	for i := 0; i < n; i++ {
		m := mset.At(i)
		if genmethod.SupportsGenericMethods && genmethod.IsGenericMethod(m.Type()) {
			continue
		}
		fn, _ := m.Obj().(*types.Func)
		if b.Prog.isNoInterfaceMethod(fn) {
			continue
		}
		methods = append(methods, m)
	}
	return methods
}

// closure func type
func funcType(prog Program, typ types.Type) types.Type {
	ftyp := prog.Type(typ, GoDecl)
	return ftyp.raw.Type.(*types.Struct).Field(0).Type()
}

func abiMethodName(obj types.Object) string {
	name := obj.Name()
	if token.IsExported(name) {
		return name
	}
	return abi.FullName(obj.Pkg(), name)
}

func methodExprSignature(sig *types.Signature) *types.Signature {
	recv := sig.Recv()
	if recv == nil {
		return sig
	}
	params := sig.Params()
	n := params.Len()
	vars := make([]*types.Var, n+1)
	vars[0] = types.NewVar(recv.Pos(), recv.Pkg(), recv.Name(), recv.Type())
	for i := 0; i < n; i++ {
		vars[i+1] = params.At(i)
	}
	return types.NewSignatureType(nil, nil, nil, types.NewTuple(vars...), sig.Results(), sig.Variadic())
}

func (b Builder) abiMethodFunc(anonymous bool, mPkg *types.Package, mName string, mSig *types.Signature) Function {
	var fullName string
	if anonymous {
		fullName = b.Pkg.Path() + "." + types.TypeString(mSig.Recv().Type(), abi.PathOf) + "." + mName
	} else {
		fullName = FuncName(mPkg, mName, mSig.Recv(), false)
	}
	if b.Pkg.fnlink != nil {
		fullName = b.Pkg.fnlink(fullName)
	}
	return b.Pkg.NewFunc(fullName, mSig, GoDecl) // TODO(xsw): use rawType to speed up
}

/*
	struct Type {
		CommonType (_type)
		Extended
	}

	struct {
		Type
		UncommonType
		[N]Method
	}
*/
func (b Builder) abiType(t types.Type) Expr {
	prog := b.Prog
	pkg := b.Pkg
	name, _ := prog.abi.TypeName(t)
	g := pkg.VarOf(name)
	if g == nil {
		if prog.patchType != nil {
			t = prog.patchType(t)
		}
		b.recordTypeChildren(name, t)
		mset, hasUncommon := b.abiUncommonMethodSet(t)
		var methods []*types.Selection
		if hasUncommon {
			methods = b.abiInterfaceMethods(mset)
		}
		methodCount := len(methods)
		rt := prog.rtNamedABI(t)
		var typ types.Type = rt
		if hasUncommon {
			ut := prog.rtNamed("uncommonType")
			mt := prog.rtNamed("Method")
			structFields := []*types.Var{
				types.NewVar(token.NoPos, nil, "T", rt),
				types.NewVar(token.NoPos, nil, "U", ut),
				types.NewVar(token.NoPos, nil, "M", types.NewArray(mt, int64(methodCount))),
			}
			typ = types.NewStruct(structFields, nil)
		}
		g = pkg.doNewVar(name, prog.Type(types.NewPointer(typ), GoDecl))
		commonFields := b.abiCommonFields(t, name, hasUncommon, g.impl)
		extendedFields := b.abiExtendedFields(t, name, g.impl)
		fields := commonFields
		if len(extendedFields) != 0 {
			fields = append([]llvm.Value{
				prog.constStructValue(prog.AbiType(), fields),
			}, extendedFields...)
		}
		if hasUncommon {
			fields = []llvm.Value{
				prog.constStructValue(prog.Type(rt, GoDecl), fields),
				b.abiUncommonType(t, methods),
				b.abiUncommonMethods(t, methods),
			}
			if pkg.metaBuilder != nil {
				pkg.abiTypeWithUncommon[g.impl] = struct{}{}
			}
		}
		g.impl.SetInitializer(prog.constStructValueAs(prog.Type(typ, GoDecl), g.impl.GlobalValueType(), fields))
		g.impl.SetGlobalConstant(true)
		b.Pkg.setODRLinkage(g.impl, llvm.WeakODRLinkage)
		if prog.enableGoGlobalDCE {
			prog.addMethodTypeMetadata(g.impl, prog.Type(typ, GoDecl), methods)
		}
		prog.abiSymbol[name] = &AbiSymbol{Name: name, PkgPath: pkg.Path(), Raw: t, Typ: g.Type, MSet: mset}
	}
	ret := Expr{llvm.ConstGEP(g.impl.GlobalValueType(), g.impl, []llvm.Value{
		llvm.ConstInt(prog.Int32().ll, 0, false),
		llvm.ConstInt(prog.Int32().ll, 0, false),
	}), prog.AbiTypePtr()}
	if prog.enableGoGlobalDCE {
		b.recordAbiTypeFakeUses(g.impl)
	}
	return ret
}

// AbiTypes snapshots the runtime type descriptors materialized by this
// Program without retaining any LLVM-owned values.
func (p Program) AbiTypes() []AbiTypeInfo {
	if len(p.abiSymbol) == 0 {
		return nil
	}
	names := make([]string, 0, len(p.abiSymbol))
	for name := range p.abiSymbol {
		names = append(names, name)
	}
	sort.Strings(names)
	ret := make([]AbiTypeInfo, 0, len(names))
	for _, name := range names {
		sym := p.abiSymbol[name]
		ret = append(ret, AbiTypeInfo{Name: name, Raw: sym.Raw})
	}
	return ret
}

// RegisterAbiTypes declares runtime type descriptors owned by other Programs.
// It records only Go type identity and target-local LLVM types; descriptor
// definitions remain in the package modules that materialized them.
func (p Package) RegisterAbiTypes(infos []AbiTypeInfo) {
	builder := &aBuilder{Prog: p.Prog, Pkg: p}
	for _, info := range infos {
		if info.Raw == nil {
			continue
		}
		if _, ok := p.Prog.abiSymbol[info.Name]; ok {
			continue
		}

		t := info.Raw
		mset, hasUncommon := abiUncommonMethodSetForDeclaration(t)
		var methods []*types.Selection
		if hasUncommon {
			methods = builder.abiInterfaceMethods(mset)
		}
		rt := p.Prog.rtNamedABI(t)
		var typ types.Type = rt
		if hasUncommon {
			ut := p.Prog.rtNamed("uncommonType")
			mt := p.Prog.rtNamed("Method")
			typ = types.NewStruct([]*types.Var{
				types.NewVar(token.NoPos, nil, "T", rt),
				types.NewVar(token.NoPos, nil, "U", ut),
				types.NewVar(token.NoPos, nil, "M", types.NewArray(mt, int64(len(methods)))),
			}, nil)
		}
		p.Prog.abiSymbol[info.Name] = &AbiSymbol{
			Name:    info.Name,
			PkgPath: p.Path(),
			Raw:     t,
			Typ:     p.Prog.Type(types.NewPointer(typ), GoDecl),
			MSet:    mset,
		}
	}
}

func (p Package) getAbiTypesFor(name string, filter func(sym *AbiSymbol) bool) Expr {
	prog := p.Prog
	var names []string
	if filter == nil {
		names = make([]string, 0, len(prog.abiSymbol))
		for k := range prog.abiSymbol {
			names = append(names, k)
		}
	} else {
		names = make([]string, 0, len(prog.abiSymbol))
		for k, sym := range prog.abiSymbol {
			if filter(sym) {
				names = append(names, k)
			}
		}
	}
	sort.Strings(names)
	fields := make([]llvm.Value, len(names))
	for i, name := range names {
		g := p.doNewVar(name, prog.abiSymbol[name].Typ)
		g.impl.SetLinkage(llvm.ExternalLinkage)
		g.impl.SetGlobalConstant(true)
		ptr := Expr{llvm.ConstGEP(g.impl.GlobalValueType(), g.impl, []llvm.Value{
			llvm.ConstInt(prog.Int32().ll, 0, false),
			llvm.ConstInt(prog.Int32().ll, 0, false),
		}), prog.AbiTypePtr()}
		fields[i] = ptr.impl
	}
	ft := prog.AbiTypePtr()
	atyp := prog.rawType(types.NewArray(ft.RawType(), int64(len(names))))
	for i, field := range fields {
		fields[i] = prog.toStorageConstant(ft, field)
	}
	data := Expr{llvm.ConstArray(prog.storageType(ft), fields), atyp}
	array := p.doNewVar(name+"$array", prog.Pointer(atyp))
	array.Init(data)
	array.impl.SetGlobalConstant(true)
	size := uint64(len(names))
	typ := prog.Slice(prog.AbiTypePtr())
	g := p.doNewVar(name+"$slice", prog.Pointer(typ))
	g.impl.SetInitializer(prog.constStructValueAs(typ, g.impl.GlobalValueType(), []llvm.Value{
		array.impl,
		prog.IntVal(size, prog.Int()).impl,
		prog.IntVal(size, prog.Int()).impl,
	}))
	g.impl.SetGlobalConstant(true)
	return g.Expr
}

func (p Package) getAbiTypes(name string) Expr {
	return p.getAbiTypesFor(name, nil)
}

func (p Package) InitAbiTypesFor(fname string, filter func(sym *AbiSymbol) bool) Function {
	if len(p.Prog.abiSymbol) == 0 {
		return nil
	}
	prog := p.Prog
	initFn := p.NewFunc(fname, NoArgsNoRet, CDecl)
	b := initFn.MakeBody(1)
	g := p.NewVarEx(PkgRuntime+".typelist", prog.Pointer(prog.Slice(prog.AbiTypePtr())))
	b.Store(g.Expr, b.Load(p.getAbiTypesFor(fname, filter)))
	b.Return()
	return initFn
}

func (p Package) InitAbiTypes(fname string) Function {
	return p.InitAbiTypesFor(fname, nil)
}

// -----------------------------------------------------------------------------

// Keep the descriptor selector and its audited literal domain in one place.
func (p Program) rtNamedABI(t types.Type) *types.Named {
	return p.rtNamed(RuntimeSymbol(p.abi.RuntimeName(t),
		"_type", "ptrtype", "slicetype", "functype", "interfacetype", "structtype",
		"maptype", "arraytype", "chantype",
	))
}
