package abi

import (
	"go/token"
	"go/types"
	"log"

	"github.com/xgo-dev/llgo/runtime/abi"
)

// Builds a type representing a Bucket structure for
// the given map type. This type is not visible to users -
// we include only enough information to generate a correct GC
// program for it.
// Make sure this stays in sync with runtime/map.go.
//
//	A "bucket" is a "struct" {
//	      tophash [BUCKETSIZE]uint8
//	      keys [BUCKETSIZE]keyType
//	      elems [BUCKETSIZE]elemType
//	      overflow *bucket
//	    }
const (
	BUCKETSIZE  = abi.MapBucketCount
	MAXKEYSIZE  = abi.MapMaxKeyBytes
	MAXELEMSIZE = abi.MapMaxElemBytes
)

func makefield(name string, t types.Type) *types.Var {
	return types.NewField(token.NoPos, nil, name, t, false)
}

func (b *Builder) mapBucketType(t *types.Map) types.Type {
	sizes := b.Sizes
	keytype := t.Key()
	elemtype := t.Elem()
	if b.Size(keytype) > MAXKEYSIZE {
		keytype = types.NewPointer(keytype)
	}
	if b.Size(elemtype) > MAXELEMSIZE {
		elemtype = types.NewPointer(elemtype)
	}

	field := make([]*types.Var, 0, 5)

	// The first field is: uint8 topbits[BUCKETSIZE].
	arr := types.NewArray(types.Typ[types.Uint8], BUCKETSIZE)
	field = append(field, makefield("topbits", arr))

	arr = types.NewArray(keytype, BUCKETSIZE)
	//arr.SetNoalg(true)
	keys := makefield("keys", arr)
	field = append(field, keys)

	arr = types.NewArray(elemtype, BUCKETSIZE)
	//arr.SetNoalg(true)
	elems := makefield("elems", arr)
	field = append(field, elems)

	// If keys and elems have no pointers, the map implementation
	// can keep a list of overflow pointers on the side so that
	// buckets can be marked as having no pointers.
	// Arrange for the bucket to have no pointers by changing
	// the type of the overflow field to uintptr in this case.
	// See comment on hmap.overflow in runtime/map.go.
	otyp := types.Typ[types.UnsafePointer]
	if !HasPtrData(elemtype) && !HasPtrData(keytype) {
		otyp = types.Typ[types.Uintptr]
	}
	overflow := makefield("overflow", otyp)
	field = append(field, overflow)

	// link up fields
	bucket := types.NewStruct(field[:], nil)

	// Check invariants that map code depends on.
	if !types.Comparable(t.Key()) {
		log.Fatalf("unsupported map key type for %v", t)
	}
	if BUCKETSIZE < 8 {
		log.Fatalf("bucket size %d too small for proper alignment %d", BUCKETSIZE, 8)
	}
	if uint8(sizes.Alignof(keytype)) > BUCKETSIZE {
		log.Fatalf("key align too big for %v", t)
	}
	if uint8(sizes.Alignof(elemtype)) > BUCKETSIZE {
		log.Fatalf("elem align %d too big for %v, BUCKETSIZE=%d", sizes.Alignof(elemtype), t, BUCKETSIZE)
	}
	if sizes.Alignof(keytype) > MAXKEYSIZE {
		log.Fatalf("key size too large for %v", t)
	}
	if sizes.Alignof(elemtype) > MAXELEMSIZE {
		log.Fatalf("elem size too large for %v", t)
	}
	if sizes.Alignof(t.Key()) > MAXKEYSIZE && !isPointer(keytype) {
		log.Fatalf("key indirect incorrect for %v", t)
	}
	if sizes.Alignof(t.Elem()) > MAXELEMSIZE && !isPointer(elemtype) {
		log.Fatalf("elem indirect incorrect for %v", t)
	}
	if sizes.Sizeof(keytype)%sizes.Alignof(keytype) != 0 {
		log.Fatalf("key size not a multiple of key align for %v", t)
	}
	if sizes.Sizeof(elemtype)%sizes.Alignof(elemtype) != 0 {
		log.Fatalf("elem size not a multiple of elem align for %v", t)
	}
	if uint8(sizes.Alignof(bucket))%uint8(sizes.Alignof(keytype)) != 0 {
		log.Fatalf("bucket align not multiple of key align %v", t)
	}
	if uint8(sizes.Alignof(bucket))%uint8(sizes.Alignof(elemtype)) != 0 {
		log.Fatalf("bucket align not multiple of elem align %v", t)
	}
	offs := sizes.Offsetsof(field)
	if offs[1]%sizes.Alignof(keytype) != 0 {
		log.Fatalf("bad alignment of keys in bmap for %v", t)
	}
	if offs[2]%sizes.Alignof(elemtype) != 0 {
		log.Fatalf("bad alignment of elems in bmap for %v", t)
	}

	// // Double-check that overflow field is final memory in struct,
	// // with no padding at end.
	// if overflow.Offset != bucket.Size()-int64(types.PtrSize) {
	// 	log.Fatalf("bad offset of overflow in bmap for %v, overflow.Offset=%d, bucket.Size()-int64(types.PtrSize)=%d",
	// 		t, overflow.Offset, bucket.Size()-int64(types.PtrSize))
	// }
	return bucket
}

func isPointer(t types.Type) (ok bool) {
	_, ok = t.Underlying().(*types.Pointer)
	return
}

func MapTypeFlags(t *types.Map, sizes types.Sizes) (flags int) {
	if sizes.Sizeof(t.Key()) > MAXKEYSIZE {
		flags |= 1 // indirect key
	}
	if sizes.Sizeof(t.Elem()) > MAXELEMSIZE {
		flags |= 2 // indirect value
	}
	if IsReflexive(t.Key()) {
		flags |= 4 // reflexive key
	}
	if needkeyupdate(t.Key()) {
		flags |= 8 // need key update
	}
	if hashMightPanic(t.Key()) {
		flags |= 16 // hash might panic
	}
	return
}

// $GOROOT/src/cmd/compile/internal/reflectdata/reflect.go
// func MapBucketType(t *types.Type) *types.Type {
// 	if t.MapType().Bucket != nil {
// 		return t.MapType().Bucket
// 	}

// 	keytype := t.Key()
// 	elemtype := t.Elem()
// 	types.CalcSize(keytype)
// 	types.CalcSize(elemtype)
// 	if keytype.Size() > MAXKEYSIZE {
// 		keytype = types.NewPtr(keytype)
// 	}
// 	if elemtype.Size() > MAXELEMSIZE {
// 		elemtype = types.NewPtr(elemtype)
// 	}

// 	field := make([]*types.Field, 0, 5)

// 	// The first field is: uint8 topbits[BUCKETSIZE].
// 	arr := types.NewArray(types.Types[types.TUINT8], BUCKETSIZE)
// 	field = append(field, makefield("topbits", arr))

// 	arr = types.NewArray(keytype, BUCKETSIZE)
// 	arr.SetNoalg(true)
// 	keys := makefield("keys", arr)
// 	field = append(field, keys)

// 	arr = types.NewArray(elemtype, BUCKETSIZE)
// 	arr.SetNoalg(true)
// 	elems := makefield("elems", arr)
// 	field = append(field, elems)

// 	// If keys and elems have no pointers, the map implementation
// 	// can keep a list of overflow pointers on the side so that
// 	// buckets can be marked as having no pointers.
// 	// Arrange for the bucket to have no pointers by changing
// 	// the type of the overflow field to uintptr in this case.
// 	// See comment on hmap.overflow in runtime/map.go.
// 	otyp := types.Types[types.TUNSAFEPTR]
// 	if !elemtype.HasPointers() && !keytype.HasPointers() {
// 		otyp = types.Types[types.TUINTPTR]
// 	}
// 	overflow := makefield("overflow", otyp)
// 	field = append(field, overflow)

// 	// link up fields
// 	bucket := types.NewStruct(field[:])
// 	bucket.SetNoalg(true)
// 	types.CalcSize(bucket)

// 	// Check invariants that map code depends on.
// 	if !types.IsComparable(t.Key()) {
// 		base.Fatalf("unsupported map key type for %v", t)
// 	}
// 	if BUCKETSIZE < 8 {
// 		base.Fatalf("bucket size %d too small for proper alignment %d", BUCKETSIZE, 8)
// 	}
// 	if uint8(keytype.Alignment()) > BUCKETSIZE {
// 		base.Fatalf("key align too big for %v", t)
// 	}
// 	if uint8(elemtype.Alignment()) > BUCKETSIZE {
// 		base.Fatalf("elem align %d too big for %v, BUCKETSIZE=%d", elemtype.Alignment(), t, BUCKETSIZE)
// 	}
// 	if keytype.Size() > MAXKEYSIZE {
// 		base.Fatalf("key size too large for %v", t)
// 	}
// 	if elemtype.Size() > MAXELEMSIZE {
// 		base.Fatalf("elem size too large for %v", t)
// 	}
// 	if t.Key().Size() > MAXKEYSIZE && !keytype.IsPtr() {
// 		base.Fatalf("key indirect incorrect for %v", t)
// 	}
// 	if t.Elem().Size() > MAXELEMSIZE && !elemtype.IsPtr() {
// 		base.Fatalf("elem indirect incorrect for %v", t)
// 	}
// 	if keytype.Size()%keytype.Alignment() != 0 {
// 		base.Fatalf("key size not a multiple of key align for %v", t)
// 	}
// 	if elemtype.Size()%elemtype.Alignment() != 0 {
// 		base.Fatalf("elem size not a multiple of elem align for %v", t)
// 	}
// 	if uint8(bucket.Alignment())%uint8(keytype.Alignment()) != 0 {
// 		base.Fatalf("bucket align not multiple of key align %v", t)
// 	}
// 	if uint8(bucket.Alignment())%uint8(elemtype.Alignment()) != 0 {
// 		base.Fatalf("bucket align not multiple of elem align %v", t)
// 	}
// 	if keys.Offset%keytype.Alignment() != 0 {
// 		base.Fatalf("bad alignment of keys in bmap for %v", t)
// 	}
// 	if elems.Offset%elemtype.Alignment() != 0 {
// 		base.Fatalf("bad alignment of elems in bmap for %v", t)
// 	}

// 	// Double-check that overflow field is final memory in struct,
// 	// with no padding at end.
// 	if overflow.Offset != bucket.Size()-int64(types.PtrSize) {
// 		base.Fatalf("bad offset of overflow in bmap for %v, overflow.Offset=%d, bucket.Size()-int64(types.PtrSize)=%d",
// 			t, overflow.Offset, bucket.Size()-int64(types.PtrSize))
// 	}

// 	t.MapType().Bucket = bucket

// 	bucket.StructType().Map = t
// 	return bucket
// }

// PtrDataSize returns the length in bytes of the prefix of t
// containing pointer data. Anything after this offset is scalar data.
//
// PtrDataSize is only defined for actual Go types. It's an error to
// use it on compiler-internal types (e.g., TSSA, TRESULTS).
func HasPtrData(t types.Type) bool {
	switch t := t.Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.String:
			return true
		case types.UnsafePointer:
			return true
		default:
			return false
		}
	case *types.Pointer:
		return true
	case *types.Signature, *types.Chan, *types.Map:
		return true
	case *types.Interface:
		return true
	case *types.Slice:
		return true
	case *types.Array:
		if t.Len() == 0 {
			return false
		}
		return HasPtrData(t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if HasPtrData(t.Field(i).Type()) {
				return true
			}
		}
		return false
	default:
		log.Fatalf("PtrDataSize: unexpected type, %v", t)
		return false
	}
}

// $GOROOT/src/cmd/compile/internal/types/type.go
// func PtrDataSize(t *Type) int64 {
// 	switch t.Kind() {
// 	case TBOOL, TINT8, TUINT8, TINT16, TUINT16, TINT32,
// 		TUINT32, TINT64, TUINT64, TINT, TUINT,
// 		TUINTPTR, TCOMPLEX64, TCOMPLEX128, TFLOAT32, TFLOAT64:
// 		return 0

// 	case TPTR:
// 		if t.Elem().NotInHeap() {
// 			return 0
// 		}
// 		return int64(PtrSize)

// 	case TUNSAFEPTR, TFUNC, TCHAN, TMAP:
// 		return int64(PtrSize)

// 	case TSTRING:
// 		// struct { byte *str; intgo len; }
// 		return int64(PtrSize)

// 	case TINTER:
// 		// struct { Itab *tab;	void *data; } or
// 		// struct { Type *type; void *data; }
// 		// Note: see comment in typebits.Set
// 		return 2 * int64(PtrSize)

// 	case TSLICE:
// 		if t.Elem().NotInHeap() {
// 			return 0
// 		}
// 		// struct { byte *array; uintgo len; uintgo cap; }
// 		return int64(PtrSize)

// 	case TARRAY:
// 		if t.NumElem() == 0 {
// 			return 0
// 		}
// 		// t.NumElem() > 0
// 		size := PtrDataSize(t.Elem())
// 		if size == 0 {
// 			return 0
// 		}
// 		return (t.NumElem()-1)*t.Elem().Size() + size

// 	case TSTRUCT:
// 		// Find the last field that has pointers, if any.
// 		fs := t.Fields().Slice()
// 		for i := len(fs) - 1; i >= 0; i-- {
// 			if size := PtrDataSize(fs[i].Type); size > 0 {
// 				return fs[i].Offset + size
// 			}
// 		}
// 		return 0

// 	case TSSA:
// 		if t != TypeInt128 {
// 			base.Fatalf("PtrDataSize: unexpected ssa type %v", t)
// 		}
// 		return 0

// 	default:
// 		base.Fatalf("PtrDataSize: unexpected type, %v", t)
// 		return 0
// 	}
// }

// IsReflexive reports whether t has a reflexive equality operator.
// That is, if x==x for all x of type t.
func IsReflexive(t types.Type) bool {
	switch t := t.Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Float32, types.Float64, types.Complex64, types.Complex128:
			return false
		default:
			return true
		}
	case *types.Pointer, *types.Chan:
		return true
	case *types.Interface:
		return false
	case *types.Array:
		return IsReflexive(t.Elem())
	case *types.Struct:
		for i, n := 0, t.NumFields(); i < n; i++ {
			if !IsReflexive(t.Field(i).Type()) {
				return false
			}
		}
		return true
	default:
		log.Fatalf("bad type for map key: %v", t)
		return false
	}
}

// $GOROOT/src/cmd/compile/internal/types/type.go
// func IsReflexive(t *Type) bool {
// 	switch t.Kind() {
// 	case TBOOL,
// 		TINT,
// 		TUINT,
// 		TINT8,
// 		TUINT8,
// 		TINT16,
// 		TUINT16,
// 		TINT32,
// 		TUINT32,
// 		TINT64,
// 		TUINT64,
// 		TUINTPTR,
// 		TPTR,
// 		TUNSAFEPTR,
// 		TSTRING,
// 		TCHAN:
// 		return true

// 	case TFLOAT32,
// 		TFLOAT64,
// 		TCOMPLEX64,
// 		TCOMPLEX128,
// 		TINTER:
// 		return false

// 	case TARRAY:
// 		return IsReflexive(t.Elem())

// 	case TSTRUCT:
// 		for _, t1 := range t.Fields().Slice() {
// 			if !IsReflexive(t1.Type) {
// 				return false
// 			}
// 		}
// 		return true

// 	default:
// 		base.Fatalf("bad type for map key: %v", t)
// 		return false
// 	}
// }

// $GOROOT/src/cmd/compile/internal/types/type.go
// needkeyupdate reports whether map updates with t as a key
// need the key to be updated.
func needkeyupdate(t types.Type) bool {
	switch t := t.Underlying().(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Float32, types.Float64, types.Complex64, types.Complex128:
			return true // floats and complex can be +0/-0
		case types.String:
			return true // strings might have smaller backing stores
		default:
			return false
		}
	case *types.Interface:
		return true
	case *types.Pointer, *types.Chan:
		return false
	case *types.Array:
		return needkeyupdate(t.Elem())
	case *types.Struct:
		for i, n := 0, t.NumFields(); i < n; i++ {
			if needkeyupdate(t.Field(i).Type()) {
				return true
			}
		}
		return false
	default:
		log.Fatalf("bad type for map key: %v", t)
		return true
	}
}

// $GOROOT/src/cmd/compile/internal/reflectdata/reflect.go
// func needkeyupdate(t *types.Type) bool {
// 	switch t.Kind() {
// 	case types.TBOOL, types.TINT, types.TUINT, types.TINT8, types.TUINT8, types.TINT16, types.TUINT16, types.TINT32, types.TUINT32,
// 		types.TINT64, types.TUINT64, types.TUINTPTR, types.TPTR, types.TUNSAFEPTR, types.TCHAN:
// 		return false

// 	case types.TFLOAT32, types.TFLOAT64, types.TCOMPLEX64, types.TCOMPLEX128, // floats and complex can be +0/-0
// 		types.TINTER,
// 		types.TSTRING: // strings might have smaller backing stores
// 		return true

// 	case types.TARRAY:
// 		return needkeyupdate(t.Elem())

// 	case types.TSTRUCT:
// 		for _, t1 := range t.Fields().Slice() {
// 			if needkeyupdate(t1.Type) {
// 				return true
// 			}
// 		}
// 		return false

// 	default:
// 		base.Fatalf("bad type for map key: %v", t)
// 		return true
// 	}
// }

// hashMightPanic reports whether the hash of a map key of type t might panic.
func hashMightPanic(t types.Type) bool {
	switch t := t.Underlying().(type) {
	case *types.Interface:
		return true
	case *types.Array:
		return hashMightPanic(t.Elem())
	case *types.Struct:
		for i, n := 0, t.NumFields(); i < n; i++ {
			if hashMightPanic(t.Field(i).Type()) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// $GOROOT/src/cmd/compile/internal/reflectdata/reflect.go
// func hashMightPanic(t *types.Type) bool {
// 	switch t.Kind() {
// 	case types.TINTER:
// 		return true

// 	case types.TARRAY:
// 		return hashMightPanic(t.Elem())

// 	case types.TSTRUCT:
// 		for _, t1 := range t.Fields().Slice() {
// 			if hashMightPanic(t1.Type) {
// 				return true
// 			}
// 		}
// 		return false

// 	default:
// 		return false
// 	}
// }
