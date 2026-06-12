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

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package reflect

import (
	"strconv"
	"sync"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/goplus/llgo/runtime/abi"
	clite "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/runtime"
	"github.com/goplus/llgo/runtime/internal/runtime/goarch"
)

// A Kind represents the specific kind of type that a Type represents.
// The zero Kind is not a valid kind.
type Kind uint

const (
	Invalid Kind = iota
	Bool
	Int
	Int8
	Int16
	Int32
	Int64
	Uint
	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Float32
	Float64
	Complex64
	Complex128
	Array
	Chan
	Func
	Interface
	Map
	Pointer
	Slice
	String
	Struct
	UnsafePointer
)

// Ptr is the old name for the Pointer kind.
const Ptr = Pointer

// uncommonType is present only for defined types or types with methods
// (if T is a defined type, the uncommonTypes for T and *T have methods).
// Using a pointer to this struct reduces the overall size required
// to describe a non-defined type with no methods.
type uncommonType = abi.UncommonType

// Embed this type to get common/uncommon
type common struct {
	abi.Type
}

// rtype is the common implementation of most values.
// It is embedded in other struct types.
type rtype struct {
	t abi.Type
}

func (t *rtype) common() *abi.Type {
	return &t.t
}

func (t *rtype) uncommon() *abi.UncommonType {
	return t.t.Uncommon()
}

// ChanDir represents a channel type's direction.
type ChanDir int

const (
	RecvDir ChanDir             = 1 << iota // <-chan
	SendDir                                 // chan<-
	BothDir = RecvDir | SendDir             // chan
)

func (d ChanDir) String() string {
	switch d {
	case SendDir:
		return "chan<-"
	case RecvDir:
		return "<-chan"
	case BothDir:
		return "chan"
	}
	return "ChanDir" + strconv.Itoa(int(d))
}

// arrayType represents a fixed array type.
type arrayType = abi.ArrayType

// chanType represents a channel type.
type chanType = abi.ChanType

// funcType represents a function type.
//
// A *rtype for each in and out parameter is stored in an array that
// directly follows the funcType (and possibly its uncommonType). So
// a function type with one method, one input, and one output is:
//
//	struct {
//		funcType
//		uncommonType
//		[2]*rtype    // [0] is in, [1] is out
//	}
type funcType = abi.FuncType

// interfaceType represents an interface type.
type interfaceType struct {
	abi.InterfaceType // can embed directly because not a public type.
}

func (t *interfaceType) common() *abi.Type {
	return &t.Type
}

func (t *interfaceType) uncommon() *abi.UncommonType {
	return t.Uncommon()
}

// mapType represents a map type.
type mapType struct {
	abi.MapType
}

// ptrType represents a pointer type.
type ptrType struct {
	abi.PtrType
}

// sliceType represents a slice type.
type sliceType struct {
	abi.SliceType
}

// Struct field
type structField = abi.StructField

// structType represents a struct type.
type structType struct {
	abi.StructType
}

/*
 * The compiler knows the exact layout of all the data structures above.
 * The compiler does not know about the data structures and methods below.
 */

// Method represents a single method.
type Method struct {
	// Name is the method name.
	Name string

	// PkgPath is the package path that qualifies a lower case (unexported)
	// method name. It is empty for upper case (exported) method names.
	// The combination of PkgPath and Name uniquely identifies a method
	// in a method set.
	// See https://golang.org/ref/spec#Uniqueness_of_identifiers
	PkgPath string

	Type  Type  // method type
	Func  Value // func with receiver as first argument
	Index int   // index for Type.Method
}

// IsExported reports whether the method is exported.
func (m Method) IsExported() bool {
	return m.PkgPath == ""
}

const (
	kindDirectIface = 1 << 5
	kindGCProg      = 1 << 6 // Type.gc points to GC program
	kindMask        = (1 << 5) - 1
)

// String returns the name of k.
func (k Kind) String() string {
	if uint(k) < uint(len(kindNames)) {
		return kindNames[uint(k)]
	}
	return "kind" + strconv.Itoa(int(k))
}

var kindNames = []string{
	Invalid:       "invalid",
	Bool:          "bool",
	Int:           "int",
	Int8:          "int8",
	Int16:         "int16",
	Int32:         "int32",
	Int64:         "int64",
	Uint:          "uint",
	Uint8:         "uint8",
	Uint16:        "uint16",
	Uint32:        "uint32",
	Uint64:        "uint64",
	Uintptr:       "uintptr",
	Float32:       "float32",
	Float64:       "float64",
	Complex64:     "complex64",
	Complex128:    "complex128",
	Array:         "array",
	Chan:          "chan",
	Func:          "func",
	Interface:     "interface",
	Map:           "map",
	Pointer:       "ptr",
	Slice:         "slice",
	String:        "string",
	Struct:        "struct",
	UnsafePointer: "unsafe.Pointer",
}

func (t *rtype) String() string {
	return t.t.String()
}

func (t *rtype) Size() uintptr { return t.t.Size() }

func (t *rtype) Bits() int {
	if t == nil {
		panic("reflect: Bits of nil Type")
	}
	k := t.Kind()
	if k < Int || k > Complex128 {
		panic("reflect: Bits of non-arithmetic Type " + t.String())
	}
	return int(t.t.Size_) * 8
}

func (t *rtype) Align() int { return t.t.Align() }

func (t *rtype) FieldAlign() int { return t.t.FieldAlign() }

func (t *rtype) Kind() Kind { return Kind(t.t.Kind()) }

func (t *rtype) exportedMethods() []abi.Method {
	ut := t.uncommon()
	if ut == nil {
		return nil
	}
	return ut.ExportedMethods()
}

func (t *rtype) NumMethod() int {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.NumMethod()
	}
	return len(t.exportedMethods())
}

func (t *rtype) Method(i int) (m Method) {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.Method(i)
	}
	methods := t.exportedMethods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	m.Name = p.Name()
	fl := flag(Func)
	ft := p.Mtyp_
	in := make([]Type, 0, 1+len(ft.In))
	in = append(in, t)
	for _, arg := range ft.In {
		in = append(in, toRType(arg))
	}
	out := make([]Type, 0, len(ft.Out))
	for _, ret := range ft.Out {
		out = append(out, toRType(ret))
	}
	mt := FuncOf(in, out, ft.Variadic())
	m.Type = mt
	mtfn := (*funcType)(unsafe.Pointer(&mt.(*rtype).t))
	fv := &struct {
		fn  unsafe.Pointer
		env unsafe.Pointer
	}{p.Tfn_, nil}
	m.Func = Value{closureOf(mtfn), unsafe.Pointer(fv), fl | flagIndir}
	m.Index = i
	return m
}

func (t *rtype) MethodByName(name string) (m Method, ok bool) {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
		return tt.MethodByName(name)
	}
	ut := t.uncommon()
	if ut == nil {
		return Method{}, false
	}

	methods := ut.ExportedMethods()

	// We are looking for the first index i where the string becomes >= s.
	// This is a copy of sort.Search, with f(h) replaced by (t.nameOff(methods[h].name).name() >= name).
	i, j := 0, len(methods)
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if !(methods[h].Name() >= name) {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	if i < len(methods) && name == methods[i].Name() {
		return t.Method(i), true
	}

	return Method{}, false
}

func (t *rtype) PkgPath() string {
	if t.t.TFlag&abi.TFlagNamed == 0 {
		return ""
	}
	ut := t.uncommon()
	if ut == nil {
		if t.t.Kind() == abi.UnsafePointer {
			return "unsafe"
		}
		return ""
	}
	return ut.PkgPath_
}

func pkgPathFor(t *abi.Type) string {
	return toRType(t).PkgPath()
}

func (t *rtype) Name() string {
	if !t.t.HasName() {
		return ""
	}
	s := t.String()
	i := len(s) - 1
	sqBrackets := 0
	for i >= 0 && (s[i] != '.' || sqBrackets != 0) {
		switch s[i] {
		case ']':
			sqBrackets++
		case '[':
			sqBrackets--
		}
		i--
	}
	return s[i+1:]
}

func nameFor(t *abi.Type) string {
	return toRType(t).Name()
}

func (t *rtype) ChanDir() ChanDir {
	if t.Kind() != Chan {
		panic("reflect: ChanDir of non-chan type " + t.String())
	}
	tt := (*abi.ChanType)(unsafe.Pointer(t))
	return ChanDir(tt.Dir)
}

func toRType(t *abi.Type) *rtype {
	return (*rtype)(unsafe.Pointer(t))
}

func elem(t *abi.Type) *abi.Type {
	et := t.Elem()
	if et != nil {
		if et.IsClosure() {
			ft := toFuncType(et.StructType())
			return &ft.Type
		}
		return et
	}
	panic("reflect: Elem of invalid type " + stringFor(t))
}

func (t *rtype) Elem() Type {
	return toType(elem(t.common()))
}

func (t *rtype) Field(i int) StructField {
	if t.Kind() != Struct {
		panic("reflect: Field of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.Field(i)
}

func (t *rtype) FieldByIndex(index []int) StructField {
	if t.Kind() != Struct {
		panic("reflect: FieldByIndex of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.FieldByIndex(index)
}

func (t *rtype) FieldByName(name string) (StructField, bool) {
	if t.Kind() != Struct {
		panic("reflect: FieldByName of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.FieldByName(name)
}

func (t *rtype) FieldByNameFunc(match func(string) bool) (StructField, bool) {
	if t.Kind() != Struct {
		panic("reflect: FieldByNameFunc of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	return tt.FieldByNameFunc(match)
}

func (t *rtype) Key() Type {
	if t.Kind() != Map {
		panic("reflect: Key of non-map type " + t.String())
	}
	tt := (*mapType)(unsafe.Pointer(t))
	return toType(tt.Key)
}

func (t *rtype) Len() int {
	if t.Kind() != Array {
		panic("reflect: Len of non-array type " + t.String())
	}
	tt := (*arrayType)(unsafe.Pointer(t))
	return int(tt.Len)
}

func (t *rtype) NumField() int {
	if t.Kind() != Struct {
		panic("reflect: NumField of non-struct type " + t.String())
	}
	tt := (*structType)(unsafe.Pointer(t))
	return len(tt.Fields)
}

func (t *rtype) In(i int) Type {
	if t.Kind() != Func {
		panic("reflect: In of non-func type " + t.String())
	}
	tt := (*abi.FuncType)(unsafe.Pointer(t))
	return toType(tt.In[i])
}

func (t *rtype) NumIn() int {
	if t.Kind() != Func {
		panic("reflect: NumIn of non-func type " + t.String())
	}
	tt := (*abi.FuncType)(unsafe.Pointer(t))
	return len(tt.In)
}

func (t *rtype) NumOut() int {
	if t.Kind() != Func {
		panic("reflect: NumOut of non-func type " + t.String())
	}
	tt := (*abi.FuncType)(unsafe.Pointer(t))
	return len(tt.Out)
}

func (t *rtype) Out(i int) Type {
	if t.Kind() != Func {
		panic("reflect: Out of non-func type " + t.String())
	}
	tt := (*abi.FuncType)(unsafe.Pointer(t))
	return toType(tt.Out[i])
}

func (t *rtype) IsVariadic() bool {
	if t.Kind() != Func {
		panic("reflect: IsVariadic of non-func type " + t.String())
	}
	tt := (*abi.FuncType)(unsafe.Pointer(t))
	return tt.Variadic()
}

// add returns p+x.
//
// The whySafe string is ignored, so that the function still inlines
// as efficiently as p+x, but all call sites should use the string to
// record why the addition is safe, which is to say why the addition
// does not cause x to advance to the very end of p's allocation
// and therefore point incorrectly at the next block in memory.
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}

// Method returns the i'th method in the type's method set.
func (t *interfaceType) Method(i int) (m Method) {
	if i < 0 || i >= len(t.Methods) {
		return
	}
	p := &t.Methods[i]
	m.Name = p.Name()
	m.PkgPath = p.PkgPath()
	m.Type = toType(&p.Typ_.Type)
	m.Index = i
	return
}

// NumMethod returns the number of interface methods in the type's method set.
func (t *interfaceType) NumMethod() int { return len(t.Methods) }

// MethodByName method with the given name in the type's method set.
func (t *interfaceType) MethodByName(name string) (m Method, ok bool) {
	if t == nil {
		return
	}
	var p *abi.Imethod
	for i := range t.Methods {
		p = &t.Methods[i]
		if p.Name() == name {
			return t.Method(i), true
		}
	}
	return
}

// A StructField describes a single field in a struct.
type StructField struct {
	// Name is the field name.
	Name string

	// PkgPath is the package path that qualifies a lower case (unexported)
	// field name. It is empty for upper case (exported) field names.
	// See https://golang.org/ref/spec#Uniqueness_of_identifiers
	PkgPath string

	Type      Type      // field type
	Tag       StructTag // field tag string
	Offset    uintptr   // offset within struct, in bytes
	Index     []int     // index sequence for Type.FieldByIndex
	Anonymous bool      // is an embedded field
}

// IsExported reports whether the field is exported.
func (f StructField) IsExported() bool {
	return f.PkgPath == ""
}

// A StructTag is the tag string in a struct field.
//
// By convention, tag strings are a concatenation of
// optionally space-separated key:"value" pairs.
// Each key is a non-empty string consisting of non-control
// characters other than space (U+0020 ' '), quote (U+0022 '"'),
// and colon (U+003A ':').  Each value is quoted using U+0022 '"'
// characters and Go string literal syntax.
type StructTag string

// Get returns the value associated with key in the tag string.
// If there is no such key in the tag, Get returns the empty string.
// If the tag does not have the conventional format, the value
// returned by Get is unspecified. To determine whether a tag is
// explicitly set to the empty string, use Lookup.
func (tag StructTag) Get(key string) string {
	v, _ := tag.Lookup(key)
	return v
}

// Lookup returns the value associated with key in the tag string.
// If the key is present in the tag the value (which may be empty)
// is returned. Otherwise the returned value will be the empty string.
// The ok return value reports whether the value was explicitly set in
// the tag string. If the tag does not have the conventional format,
// the value returned by Lookup is unspecified.
func (tag StructTag) Lookup(key string) (value string, ok bool) {
	// When modifying this code, also update the validateStructTag code
	// in cmd/vet/structtag.go.

	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

// Field returns the i'th struct field.
func (t *structType) Field(i int) (f StructField) {
	if i < 0 || i >= len(t.Fields) {
		panic("reflect: Field index out of bounds")
	}
	p := &t.Fields[i]
	f.Type = toType(p.Typ)
	f.Name = p.Name_
	f.Anonymous = p.Embedded()
	if !abi.IsExported(p.Name_) {
		f.PkgPath = t.PkgPath_
	}
	if tag := p.Tag_; tag != "" {
		f.Tag = StructTag(tag)
	}
	f.Offset = p.Offset

	// NOTE(rsc): This is the only allocation in the interface
	// presented by a reflect.Type. It would be nice to avoid,
	// at least in the common cases, but we need to make sure
	// that misbehaving clients of reflect cannot affect other
	// uses of reflect. One possibility is CL 5371098, but we
	// postponed that ugliness until there is a demonstrated
	// need for the performance. This is issue 2320.
	f.Index = []int{i}
	return
}

// FieldByIndex returns the nested field corresponding to index.
func (t *structType) FieldByIndex(index []int) (f StructField) {
	f.Type = toType(&t.Type)
	for i, x := range index {
		if i > 0 {
			ft := f.Type
			if ft.Kind() == Pointer && ft.Elem().Kind() == Struct {
				ft = ft.Elem()
			}
			f.Type = ft
		}
		f = f.Type.Field(x)
	}
	return
}

// A fieldScan represents an item on the fieldByNameFunc scan work list.
type fieldScan struct {
	typ   *structType
	index []int
}

// FieldByNameFunc returns the struct field with a name that satisfies the
// match function and a boolean to indicate if the field was found.
func (t *structType) FieldByNameFunc(match func(string) bool) (result StructField, ok bool) {
	// This uses the same condition that the Go language does: there must be a unique instance
	// of the match at a given depth level. If there are multiple instances of a match at the
	// same depth, they annihilate each other and inhibit any possible match at a lower level.
	// The algorithm is breadth first search, one depth level at a time.

	// The current and next slices are work queues:
	// current lists the fields to visit on this depth level,
	// and next lists the fields on the next lower level.
	current := []fieldScan{}
	next := []fieldScan{{typ: t}}

	// nextCount records the number of times an embedded type has been
	// encountered and considered for queueing in the 'next' slice.
	// We only queue the first one, but we increment the count on each.
	// If a struct type T can be reached more than once at a given depth level,
	// then it annihilates itself and need not be considered at all when we
	// process that next depth level.
	var nextCount map[*structType]int

	// visited records the structs that have been considered already.
	// Embedded pointer fields can create cycles in the graph of
	// reachable embedded types; visited avoids following those cycles.
	// It also avoids duplicated effort: if we didn't find the field in an
	// embedded type T at level 2, we won't find it in one at level 4 either.
	visited := map[*structType]bool{}

	for len(next) > 0 {
		current, next = next, current[:0]
		count := nextCount
		nextCount = nil

		// Process all the fields at this depth, now listed in 'current'.
		// The loop queues embedded fields found in 'next', for processing during the next
		// iteration. The multiplicity of the 'current' field counts is recorded
		// in 'count'; the multiplicity of the 'next' field counts is recorded in 'nextCount'.
		for _, scan := range current {
			t := scan.typ
			if visited[t] {
				// We've looked through this type before, at a higher level.
				// That higher level would shadow the lower level we're now at,
				// so this one can't be useful to us. Ignore it.
				continue
			}
			visited[t] = true
			for i := range t.Fields {
				f := &t.Fields[i]
				// Find name and (for embedded field) type for field f.
				fname := f.Name_
				var ntyp *abi.Type
				if f.Embedded() {
					// Embedded field of type T or *T.
					ntyp = f.Typ
					if ntyp.Kind() == abi.Pointer {
						ntyp = ntyp.Elem()
					}
				}

				// Does it match?
				if match(fname) {
					// Potential match
					if count[t] > 1 || ok {
						// Name appeared multiple times at this level: annihilate.
						return StructField{}, false
					}
					result = t.Field(i)
					result.Index = nil
					result.Index = append(result.Index, scan.index...)
					result.Index = append(result.Index, i)
					ok = true
					continue
				}

				// Queue embedded struct fields for processing with next level,
				// but only if we haven't seen a match yet at this level and only
				// if the embedded types haven't already been queued.
				if ok || ntyp == nil || ntyp.Kind() != abi.Struct {
					continue
				}
				styp := (*structType)(unsafe.Pointer(ntyp))
				if nextCount[styp] > 0 {
					nextCount[styp] = 2 // exact multiple doesn't matter
					continue
				}
				if nextCount == nil {
					nextCount = map[*structType]int{}
				}
				nextCount[styp] = 1
				if count[t] > 1 {
					nextCount[styp] = 2 // exact multiple doesn't matter
				}
				var index []int
				index = append(index, scan.index...)
				index = append(index, i)
				next = append(next, fieldScan{styp, index})
			}
		}
		if ok {
			break
		}
	}
	return
}

// FieldByName returns the struct field with the given name
// and a boolean to indicate if the field was found.
func (t *structType) FieldByName(name string) (f StructField, present bool) {
	// Quick check for top-level name, or struct without embedded fields.
	hasEmbeds := false
	if name != "" {
		for i := range t.Fields {
			tf := &t.Fields[i]
			if tf.Name_ == name {
				return t.Field(i), true
			}
			if tf.Embedded() {
				hasEmbeds = true
			}
		}
	}
	if !hasEmbeds {
		return
	}
	return t.FieldByNameFunc(func(s string) bool { return s == name })
}

// TypeOf returns the reflection Type that represents the dynamic type of i.
// If i is a nil interface value, TypeOf returns nil.
func TypeOf(i any) Type {
	eface := *(*emptyInterface)(unsafe.Pointer(&i))
	// check nil
	if eface.typ == nil {
		return nil
	}
	// check closure type
	if eface.typ.IsClosure() {
		ft := toFuncType(eface.typ.StructType())
		return toType(&ft.Type)
	}
	// Noescape so this doesn't make i to escape. See the comment
	// at Value.typ for why this is safe.
	return toType((*abi.Type)(unsafe.Pointer(eface.typ)))
}

var namedFuncMap sync.Map // map[*abi.StructType]*abi.FuncType

func toFuncType(typ *abi.StructType) *abi.FuncType {
	if !typ.HasName() {
		return typ.Fields[0].Typ.FuncType()
	}
	if i, ok := namedFuncMap.Load(typ); ok {
		return i.(*abi.FuncType)
	}
	size := unsafe.Sizeof(abi.FuncType{})
	u := typ.Uncommon()
	if u != nil {
		size += unsafe.Sizeof(abi.UncommonType{}) + uintptr(u.Mcount)*unsafe.Sizeof(abi.Method{})
	}
	ftyp := typ.Fields[0].Typ.FuncType()
	ptr := runtime.AllocU(size)
	t := (*abi.FuncType)(ptr)
	t.Size_ = ftyp.Size_
	t.PtrBytes = ftyp.PtrBytes
	t.Hash = fnv1(typ.Hash, []byte("-funcType")...)
	t.TFlag = ftyp.TFlag | abi.TFlagNamed
	t.Align_ = ftyp.Align_
	t.FieldAlign_ = ftyp.FieldAlign_
	t.Kind_ = ftyp.Kind_
	t.Equal = ftyp.Equal
	t.GCData = ftyp.GCData
	t.Str_ = typ.Str_
	t.In = ftyp.In
	t.Out = ftyp.Out
	if typ.TFlag&abi.TFlagExtraStar != 0 {
		t.TFlag |= abi.TFlagExtraStar
	}
	if u != nil {
		t.TFlag |= abi.TFlagUncommon
		tu := t.Uncommon()
		tu.PkgPath_ = u.PkgPath_
		tu.Mcount = u.Mcount
		tu.Xcount = u.Xcount
		tu.Moff = u.Moff
		uptr := add(unsafe.Pointer(u), uintptr(u.Moff), "t.mcount > 0")
		tptr := add(unsafe.Pointer(tu), uintptr(tu.Moff), "t.mcount > 0")

		for i := 0; i < int(u.Mcount); i++ {
			uelemPtr := add(uptr, uintptr(i)*unsafe.Sizeof(Method{}), "accessing method element")
			telemPtr := add(tptr, uintptr(i)*unsafe.Sizeof(Method{}), "accessing method element")
			*(*Method)(telemPtr) = *(*Method)(uelemPtr)
		}
	}
	tt, _ := namedFuncMap.LoadOrStore(typ, t)
	return tt.(*abi.FuncType)
}

func toClosureType(typ *abi.FuncType) *abi.StructType {
	size := unsafe.Sizeof(abi.StructType{})
	u := typ.Uncommon()
	if u != nil {
		size += unsafe.Sizeof(abi.UncommonType{}) + uintptr(u.Mcount)*unsafe.Sizeof(abi.Method{})
	}
	ftyp := typ
	fnstr := funcStr(ftyp)
	for _, t := range typelist {
		if t.Kind() == abi.Func && t.String() == fnstr {
			ftyp = t.FuncType()
			break
		}
	}
	ptr := runtime.AllocU(size)
	t := (*abi.StructType)(ptr)
	t.Size_ = ftyp.Size_
	t.PtrBytes = ftyp.PtrBytes
	t.Hash = fnv1(typ.Hash, []byte("-structType")...)
	t.TFlag = ftyp.TFlag | abi.TFlagNamed | abi.TFlagClosure
	t.Align_ = ftyp.Align_
	t.FieldAlign_ = ftyp.FieldAlign_
	t.Kind_ = uint8(abi.Struct)
	t.Equal = ftyp.Equal
	t.GCData = ftyp.GCData
	t.Str_ = typ.Str_
	t.Fields = []abi.StructField{
		abi.StructField{
			Name_: "$f",
			Typ:   &ftyp.Type,
		}, abi.StructField{
			Name_:  "$data",
			Typ:    unsafePointerType,
			Offset: pointerSize,
		},
	}
	t.PkgPath_ = ""
	if typ.TFlag&abi.TFlagExtraStar != 0 {
		t.TFlag |= abi.TFlagExtraStar
	}
	if u != nil {
		t.TFlag |= abi.TFlagUncommon
		tu := t.Uncommon()
		tu.PkgPath_ = u.PkgPath_
		tu.Mcount = u.Mcount
		tu.Xcount = u.Xcount
		tu.Moff = u.Moff
		uptr := add(unsafe.Pointer(u), uintptr(u.Moff), "t.mcount > 0")
		tptr := add(unsafe.Pointer(tu), uintptr(tu.Moff), "t.mcount > 0")

		for i := 0; i < int(u.Mcount); i++ {
			uelemPtr := add(uptr, uintptr(i)*unsafe.Sizeof(Method{}), "accessing method element")
			telemPtr := add(tptr, uintptr(i)*unsafe.Sizeof(Method{}), "accessing method element")
			*(*Method)(telemPtr) = *(*Method)(uelemPtr)
		}
	}
	namedFuncMap.Store(t, typ)
	return t
}

// rtypeOf directly extracts the *rtype of the provided value.
func rtypeOf(i any) *abi.Type {
	eface := *(*emptyInterface)(unsafe.Pointer(&i))
	return eface.typ
}

// ptrMap is the cache for PointerTo.
var ptrMap sync.Map // map[*rtype]*ptrType

// PtrTo returns the pointer type with element t.
// For example, if t represents type Foo, PtrTo(t) represents *Foo.
//
// PtrTo is the old spelling of PointerTo.
// The two functions behave identically.
func PtrTo(t Type) Type { return PointerTo(t) }

// PointerTo returns the pointer type with element t.
// For example, if t represents type Foo, PointerTo(t) represents *Foo.
func PointerTo(t Type) Type {
	return toRType(t.(*rtype).ptrTo())
}

func (t *rtype) ptrTo() *abi.Type {
	at := &t.t
	if at.PtrToThis_ != nil {
		return at.PtrToThis_
	}
	// Check the cache.
	if pi, ok := ptrMap.Load(t); ok {
		return &pi.(*ptrType).Type
	}

	// Look in known types.
	s := "*" + t.String()
	for _, tt := range typesByString(s) {
		p := (*ptrType)(unsafe.Pointer(tt))
		if p.Elem != &t.t {
			continue
		}
		pi, _ := ptrMap.LoadOrStore(t, p)
		return &pi.(*ptrType).Type
	}

	// Create a new ptrType starting with the description
	// of an *unsafe.Pointer.
	var iptr any = (*unsafe.Pointer)(nil)
	prototype := *(**ptrType)(unsafe.Pointer(&iptr))
	pp := *prototype

	pp.Str_ = s
	pp.PtrToThis_ = nil

	// For the type structures linked into the binary, the
	// compiler provides a good hash of the string.
	// Create a good hash for the new string by using
	// the FNV-1 hash's mixing function to combine the
	// old hash and the new "*".
	pp.Hash = fnv1(t.t.Hash, '*')

	pp.Elem = at

	pi, _ := ptrMap.LoadOrStore(t, &pp)
	return &pi.(*ptrType).Type
}

func ptrTo(t *abi.Type) *abi.Type {
	return toRType(t).ptrTo()
}

// TODO(xsw): check if this is needed
// fnv1 incorporates the list of bytes into the hash x using the FNV-1 hash function.
func fnv1(x uint32, list ...byte) uint32 {
	for _, b := range list {
		x = x*16777619 ^ uint32(b)
	}
	return x
}

func (t *rtype) Implements(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.Implements")
	}
	if u.Kind() != Interface {
		panic("reflect: non-interface type passed to Type.Implements")
	}
	return implements(u.common(), t.common())
}

func (t *rtype) AssignableTo(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.AssignableTo")
	}
	uu := u.common()
	return directlyAssignable(uu, t.common()) || implements(uu, t.common())
}

func (t *rtype) ConvertibleTo(u Type) bool {
	if u == nil {
		panic("reflect: nil type passed to Type.ConvertibleTo")
	}
	return convertOp(u.common(), t.common()) != nil
}

func (t *rtype) Comparable() bool {
	return t.t.Equal != nil
}

// implements reports whether the type V implements the interface type T.
//
//go:linkname implements github.com/goplus/llgo/runtime/internal/runtime.Implements
func implements(T, V *abi.Type) bool

// specialChannelAssignability reports whether a value x of channel type V
// can be directly assigned (using memmove) to another channel type T.
// https://golang.org/doc/go_spec.html#Assignability
// T and V must be both of Chan kind.
func specialChannelAssignability(T, V *abi.Type) bool {
	// Special case:
	// x is a bidirectional channel value, T is a channel type,
	// x's type V and T have identical element types,
	// and at least one of V or T is not a defined type.
	return V.ChanDir() == abi.BothDir && (nameFor(T) == "" || nameFor(V) == "") && haveIdenticalType(T.Elem(), V.Elem(), true)
}

// directlyAssignable reports whether a value x of type V can be directly
// assigned (using memmove) to a value of type T.
// https://golang.org/doc/go_spec.html#Assignability
// Ignoring the interface rules (implemented elsewhere)
// and the ideal constant rules (no ideal constants at run time).
func directlyAssignable(T, V *abi.Type) bool {
	// x's type V is identical to T?
	if T == V {
		return true
	}

	// Otherwise at least one of T and V must not be defined
	// and they must have the same kind.
	if T.HasName() && V.HasName() || T.Kind() != V.Kind() {
		return false
	}

	if T.Kind() == abi.Chan && specialChannelAssignability(T, V) {
		return true
	}

	// x's type T and V must have identical underlying types.
	return haveIdenticalUnderlyingType(T, V, true)
}

func haveIdenticalType(T, V *abi.Type, cmpTags bool) bool {
	if cmpTags {
		return T == V
	}
	if nameFor(T) != nameFor(V) || T.Kind() != V.Kind() || pkgPathFor(T) != pkgPathFor(V) {
		return false
	}

	return haveIdenticalUnderlyingType(T, V, false)
}

func haveIdenticalUnderlyingType(T, V *abi.Type, cmpTags bool) bool {
	if T == V {
		return true
	}

	kind := Kind(T.Kind())
	if kind != Kind(V.Kind()) {
		return false
	}

	// Non-composite types of equal kind have same underlying type
	// (the predefined instance of the type).
	if Bool <= kind && kind <= Complex128 || kind == String || kind == UnsafePointer {
		return true
	}

	// Composite types.
	switch kind {
	case Array:
		return T.Len() == V.Len() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Chan:
		return V.ChanDir() == T.ChanDir() && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Func:
		t := (*funcType)(unsafe.Pointer(T))
		v := (*funcType)(unsafe.Pointer(V))
		if len(t.Out) != len(v.Out) || len(t.In) != len(v.In) {
			return false
		}
		for i := 0; i < len(t.In); i++ {
			if !haveIdenticalType(t.In[i], v.In[i], cmpTags) {
				return false
			}
		}
		for i := 0; i < len(t.Out); i++ {
			if !haveIdenticalType(t.Out[i], v.Out[i], cmpTags) {
				return false
			}
		}
		return true

	case Interface:
		t := (*interfaceType)(unsafe.Pointer(T))
		v := (*interfaceType)(unsafe.Pointer(V))
		if len(t.Methods) == 0 && len(v.Methods) == 0 {
			return true
		}
		// Might have the same methods but still
		// need a run time conversion.
		return false

	case Map:
		return haveIdenticalType(T.Key(), V.Key(), cmpTags) && haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Pointer, Slice:
		return haveIdenticalType(T.Elem(), V.Elem(), cmpTags)

	case Struct:
		t := (*structType)(unsafe.Pointer(T))
		v := (*structType)(unsafe.Pointer(V))
		if len(t.Fields) != len(v.Fields) {
			return false
		}
		if t.PkgPath_ != v.PkgPath_ {
			return false
		}
		for i := range t.Fields {
			tf := &t.Fields[i]
			vf := &v.Fields[i]
			if tf.Name_ != vf.Name_ {
				return false
			}
			if !haveIdenticalType(tf.Typ, vf.Typ, cmpTags) {
				return false
			}
			if cmpTags && tf.Tag_ != vf.Tag_ {
				return false
			}
			if tf.Offset != vf.Offset {
				return false
			}
			if tf.Embedded() != vf.Embedded() {
				return false
			}
		}
		return true
	}

	return false
}

// SliceOf returns the slice type with element type t.
// For example, if t represents int, SliceOf(t) represents []int.
func SliceOf(t Type) Type {
	typ := t.common()

	// Look in cache.
	ckey := cacheKey{Slice, typ, nil, 0}
	if slice, ok := lookupCache.Load(ckey); ok {
		return slice.(Type)
	}

	// Look in known types.
	s := "[]" + stringFor(typ)
	for _, tt := range typesByString(s) {
		slice := (*sliceType)(unsafe.Pointer(tt))
		if slice.Elem == typ {
			ti, _ := lookupCache.LoadOrStore(ckey, toRType(tt))
			return ti.(Type)
		}
	}

	// Make a slice type.
	var islice any = ([]unsafe.Pointer)(nil)
	prototype := *(**sliceType)(unsafe.Pointer(&islice))
	slice := *prototype
	slice.TFlag = 0
	slice.Str_ = s
	slice.Hash = fnv1(typ.Hash, '[')
	slice.Elem = typ
	slice.PtrToThis_ = nil

	ti, _ := lookupCache.LoadOrStore(ckey, toRType(&slice.Type))
	return ti.(Type)
}

// ChanOf returns the channel type with the given direction and element type.
// For example, if t represents int, ChanOf(RecvDir, t) represents <-chan int.
//
// The gc runtime imposes a limit of 64 kB on channel element types.
// If t's size is equal to or exceeds this limit, ChanOf panics.
func ChanOf(dir ChanDir, t Type) Type {
	typ := t.common()

	// Look in cache.
	ckey := cacheKey{Chan, typ, nil, uintptr(dir)}
	if ch, ok := lookupCache.Load(ckey); ok {
		return ch.(*rtype)
	}

	// This restriction is imposed by the gc compiler and the runtime.
	if typ.Size_ >= 1<<16 {
		panic("reflect.ChanOf: element size too large")
	}

	// Look in known types.
	var s string
	switch dir {
	default:
		panic("reflect.ChanOf: invalid dir")
	case SendDir:
		s = "chan<- " + stringFor(typ)
	case RecvDir:
		s = "<-chan " + stringFor(typ)
	case BothDir:
		typeStr := stringFor(typ)
		if typeStr[0] == '<' {
			// typ is recv chan, need parentheses as "<-" associates with leftmost
			// chan possible, see:
			// * https://golang.org/ref/spec#Channel_types
			// * https://github.com/golang/go/issues/39897
			s = "chan (" + typeStr + ")"
		} else {
			s = "chan " + typeStr
		}
	}
	for _, tt := range typesByString(s) {
		ch := (*chanType)(unsafe.Pointer(tt))
		if ch.Elem == typ && ch.Dir == abi.ChanDir(dir) {
			ti, _ := lookupCache.LoadOrStore(ckey, toRType(tt))
			return ti.(Type)
		}
	}

	// Make a channel type.
	var ichan any = (chan unsafe.Pointer)(nil)
	prototype := *(**chanType)(unsafe.Pointer(&ichan))
	ch := *prototype
	ch.TFlag = abi.TFlagRegularMemory
	ch.Dir = abi.ChanDir(dir)
	ch.Str_ = s
	ch.Hash = fnv1(typ.Hash, 'c', byte(dir))
	ch.Elem = typ

	ti, _ := lookupCache.LoadOrStore(ckey, toRType(&ch.Type))
	return ti.(Type)
}

// toType converts from a *rtype to a Type that can be returned
// to the client of package reflect. In gc, the only concern is that
// a nil *rtype must be replaced by a nil Type, but in gccgo this
// function takes care of ensuring that multiple *rtype for the same
// type are coalesced into a single Type.
func toType(t *abi.Type) Type {
	if t == nil {
		return nil
	}
	return toRType(t)
}

// The lookupCache caches ArrayOf, ChanOf, MapOf and SliceOf lookups.
var lookupCache sync.Map // map[cacheKey]*rtype

// A cacheKey is the key for use in the lookupCache.
// Four values describe any of the types we are looking for:
// type kind, one or two subtypes, and an extra integer.
type cacheKey struct {
	kind  Kind
	t1    *abi.Type
	t2    *abi.Type
	extra uintptr
}

// The funcLookupCache caches FuncOf lookups.
// FuncOf does not share the common lookupCache since cacheKey is not
// sufficient to represent functions unambiguously.
var funcLookupCache struct {
	sync.Mutex // Guards stores (but not loads) on m.

	// m is a map[uint32][]*rtype keyed by the hash calculated in FuncOf.
	// Elements of m are append-only and thus safe for concurrent reading.
	m sync.Map
}

// FuncOf returns the function type with the given argument and result types.
// For example if k represents int and e represents string,
// FuncOf([]Type{k}, []Type{e}, false) represents func(int) string.
//
// The variadic argument controls whether the function is variadic. FuncOf
// panics if the in[len(in)-1] does not represent a slice and variadic is
// true.
func FuncOf(in, out []Type, variadic bool) Type {
	if variadic && (len(in) == 0 || in[len(in)-1].Kind() != Slice) {
		panic("reflect.FuncOf: last arg of variadic func must be slice")
	}

	// Make a func type.
	var ifunc any = (func())(nil)
	// Closures are always struct{fn *funcType, data unsafe.Pointer}
	// This is guaranteed by closureOf() construction
	closureType := *(**abi.StructType)(unsafe.Pointer(&ifunc))
	prototype := closureType.Fields[0].Typ.FuncType()
	ft := &funcType{}
	*ft = *prototype
	ft.In = make([]*abi.Type, len(in))
	ft.Out = make([]*abi.Type, len(out))

	// Build a hash and minimally populate ft.
	var hash uint32
	for i, in := range in {
		t := in.(*rtype)
		ft.In[i] = &t.t
		hash = fnv1(hash, byte(t.t.Hash>>24), byte(t.t.Hash>>16), byte(t.t.Hash>>8), byte(t.t.Hash))
	}
	if variadic {
		hash = fnv1(hash, 'v')
	}
	hash = fnv1(hash, '.')
	for i, out := range out {
		t := out.(*rtype)
		ft.Out[i] = &t.t
		hash = fnv1(hash, byte(t.t.Hash>>24), byte(t.t.Hash>>16), byte(t.t.Hash>>8), byte(t.t.Hash))
	}

	ft.TFlag = 0
	ft.Hash = hash
	ft.Kind_ = uint8(abi.Func) | abi.KindDirectIface
	if variadic {
		ft.TFlag |= abi.TFlagVariadic
	}

	funcLookupCache.Lock()
	defer funcLookupCache.Unlock()

	// Look in cache.
	if ts, ok := funcLookupCache.m.Load(hash); ok {
		for _, t := range ts.([]*abi.Type) {
			if haveIdenticalUnderlyingType(&ft.Type, t, true) {
				return toRType(t)
			}
		}
	}

	addToCache := func(tt *abi.Type) Type {
		var rts []*abi.Type
		if rti, ok := funcLookupCache.m.Load(hash); ok {
			rts = rti.([]*abi.Type)
		}
		funcLookupCache.m.Store(hash, append(rts, tt))
		return toType(tt)
	}

	str := funcStr(ft)
	for _, tt := range typesByString(str) {
		if haveIdenticalUnderlyingType(&ft.Type, tt, true) {
			return addToCache(tt)
		}
	}

	// Populate the remaining fields of ft and store in cache.
	ft.Str_ = str
	ft.PtrToThis_ = nil
	return addToCache(&ft.Type)
}

func stringFor(t *abi.Type) string {
	return toRType(t).String()
}

// funcStr builds a string representation of a funcType.
func funcStr(ft *funcType) string {
	repr := make([]byte, 0, 64)
	repr = append(repr, "func("...)
	for i, t := range ft.In {
		if i > 0 {
			repr = append(repr, ", "...)
		}
		if ft.Variadic() && i == len(ft.In)-1 {
			repr = append(repr, "..."...)
			repr = append(repr, stringFor((*sliceType)(unsafe.Pointer(t)).Elem)...)
		} else {
			repr = append(repr, stringFor(t)...)
		}
	}
	repr = append(repr, ')')
	out := ft.Out
	if len(out) == 1 {
		repr = append(repr, ' ')
	} else if len(out) > 1 {
		repr = append(repr, " ("...)
	}
	for i, t := range out {
		if i > 0 {
			repr = append(repr, ", "...)
		}
		repr = append(repr, stringFor(t)...)
	}
	if len(out) > 1 {
		repr = append(repr, ')')
	}
	return string(repr)
}

// MapOf returns the map type with the given key and element types.
// For example, if k represents int and e represents string,
// MapOf(k, e) represents map[int]string.
//
// If the key type is not a valid map key type (that is, if it does
// not implement Go's == operator), MapOf panics.
func MapOf(key, elem Type) Type {
	ktyp := key.common()
	etyp := elem.common()

	if ktyp.Equal == nil {
		panic("reflect.MapOf: invalid key type " + stringFor(ktyp))
	}

	// Look in cache.
	ckey := cacheKey{Map, ktyp, etyp, 0}
	if mt, ok := lookupCache.Load(ckey); ok {
		return mt.(Type)
	}

	// Look in known types.
	s := "map[" + stringFor(ktyp) + "]" + stringFor(etyp)
	for _, tt := range typesByString(s) {
		mt := (*mapType)(unsafe.Pointer(tt))
		if mt.Key == ktyp && mt.Elem == etyp {
			ti, _ := lookupCache.LoadOrStore(ckey, toRType(tt))
			return ti.(Type)
		}
	}

	// Make a map type.
	// Note: flag values must match those used in the TMAP case
	// in ../cmd/compile/internal/reflectdata/reflect.go:writeType.
	var imap any = (map[unsafe.Pointer]unsafe.Pointer)(nil)
	mt := **(**mapType)(unsafe.Pointer(&imap))
	mt.Str_ = s
	mt.TFlag = 0
	mt.Hash = fnv1(etyp.Hash, 'm', byte(ktyp.Hash>>24), byte(ktyp.Hash>>16), byte(ktyp.Hash>>8), byte(ktyp.Hash))
	mt.Key = ktyp
	mt.Elem = etyp
	mt.Bucket = bucketOf(ktyp, etyp)
	mt.Hasher = func(p unsafe.Pointer, seed uintptr) uintptr {
		return typehash(ktyp, p, seed)
	}
	mt.Flags = 0
	if ktyp.Size_ > maxKeySize {
		mt.KeySize = uint8(goarch.PtrSize)
		mt.Flags |= 1 // indirect key
	} else {
		mt.KeySize = uint8(ktyp.Size_)
	}
	if etyp.Size_ > maxValSize {
		mt.ValueSize = uint8(goarch.PtrSize)
		mt.Flags |= 2 // indirect value
	} else {
		mt.MapType.ValueSize = uint8(etyp.Size_)
	}
	mt.MapType.BucketSize = uint16(mt.Bucket.Size_)
	if isReflexive(ktyp) {
		mt.Flags |= 4
	}
	if needKeyUpdate(ktyp) {
		mt.Flags |= 8
	}
	if hashMightPanic(ktyp) {
		mt.Flags |= 16
	}
	mt.PtrToThis_ = nil

	ti, _ := lookupCache.LoadOrStore(ckey, toRType(&mt.Type))
	return ti.(Type)
}

// isReflexive reports whether the == operation on the type is reflexive.
// That is, x == x for all values x of type t.
func isReflexive(t *abi.Type) bool {
	switch Kind(t.Kind()) {
	case Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr, Chan, Pointer, String, UnsafePointer:
		return true
	case Float32, Float64, Complex64, Complex128, Interface:
		return false
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return isReflexive(tt.Elem)
	case Struct:
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.Fields {
			if !isReflexive(f.Typ) {
				return false
			}
		}
		return true
	default:
		// Func, Map, Slice, Invalid
		panic("isReflexive called on non-key type " + stringFor(t))
	}
}

// needKeyUpdate reports whether map overwrites require the key to be copied.
func needKeyUpdate(t *abi.Type) bool {
	switch Kind(t.Kind()) {
	case Bool, Int, Int8, Int16, Int32, Int64, Uint, Uint8, Uint16, Uint32, Uint64, Uintptr, Chan, Pointer, UnsafePointer:
		return false
	case Float32, Float64, Complex64, Complex128, Interface, String:
		// Float keys can be updated from +0 to -0.
		// String keys can be updated to use a smaller backing store.
		// Interfaces might have floats of strings in them.
		return true
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return needKeyUpdate(tt.Elem)
	case Struct:
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.Fields {
			if needKeyUpdate(f.Typ) {
				return true
			}
		}
		return false
	default:
		// Func, Map, Slice, Invalid
		panic("needKeyUpdate called on non-key type " + stringFor(t))
	}
}

// hashMightPanic reports whether the hash of a map key of type t might panic.
func hashMightPanic(t *abi.Type) bool {
	switch Kind(t.Kind()) {
	case Interface:
		return true
	case Array:
		tt := (*arrayType)(unsafe.Pointer(t))
		return hashMightPanic(tt.Elem)
	case Struct:
		tt := (*structType)(unsafe.Pointer(t))
		for _, f := range tt.Fields {
			if hashMightPanic(f.Typ) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

// Make sure these routines stay in sync with ../runtime/map.go!
// These types exist only for GC, so we only fill out GC relevant info.
// Currently, that's just size and the GC program. We also fill in string
// for possible debugging use.
const (
	bucketSize uintptr = abi.MapBucketCount
	maxKeySize uintptr = abi.MapMaxKeyBytes
	maxValSize uintptr = abi.MapMaxElemBytes
)

func bucketOf(ktyp, etyp *abi.Type) *abi.Type {
	if ktyp.Size_ > maxKeySize {
		ktyp = ptrTo(ktyp)
	}
	if etyp.Size_ > maxValSize {
		etyp = ptrTo(etyp)
	}

	// Prepare GC data if any.
	// A bucket is at most bucketSize*(1+maxKeySize+maxValSize)+ptrSize bytes,
	// or 2064 bytes, or 258 pointer-size words, or 33 bytes of pointer bitmap.
	// Note that since the key and value are known to be <= 128 bytes,
	// they're guaranteed to have bitmaps instead of GC programs.
	var gcdata *byte
	var ptrdata uintptr

	size := bucketSize*(1+ktyp.Size_+etyp.Size_) + goarch.PtrSize
	if size&uintptr(ktyp.Align_-1) != 0 || size&uintptr(etyp.Align_-1) != 0 {
		panic("reflect: bad size computation in MapOf")
	}

	if ktyp.PtrBytes != 0 || etyp.PtrBytes != 0 {
		nptr := (bucketSize*(1+ktyp.Size_+etyp.Size_) + goarch.PtrSize) / goarch.PtrSize
		n := (nptr + 7) / 8

		// Runtime needs pointer masks to be a multiple of uintptr in size.
		n = (n + goarch.PtrSize - 1) &^ (goarch.PtrSize - 1)
		mask := make([]byte, n)
		base := bucketSize / goarch.PtrSize

		if ktyp.PtrBytes != 0 {
			emitGCMask(mask, base, ktyp, bucketSize)
		}
		base += bucketSize * ktyp.Size_ / goarch.PtrSize

		if etyp.PtrBytes != 0 {
			emitGCMask(mask, base, etyp, bucketSize)
		}
		base += bucketSize * etyp.Size_ / goarch.PtrSize

		word := base
		mask[word/8] |= 1 << (word % 8)
		gcdata = &mask[0]
		ptrdata = (word + 1) * goarch.PtrSize

		// overflow word must be last
		if ptrdata != size {
			panic("reflect: bad layout computation in MapOf")
		}
	}

	b := &abi.Type{
		Align_:   goarch.PtrSize,
		Size_:    size,
		Kind_:    uint8(Struct),
		PtrBytes: ptrdata,
		GCData:   gcdata,
	}
	b.Str_ = "bucket(" + stringFor(ktyp) + "," + stringFor(etyp) + ")"
	return b
}

func (t *rtype) gcSlice(begin, end uintptr) []byte {
	return (*[1 << 30]byte)(unsafe.Pointer(t.t.GCData))[begin:end:end]
}

// emitGCMask writes the GC mask for [n]typ into out, starting at bit
// offset base.
func emitGCMask(out []byte, base uintptr, typ *abi.Type, n uintptr) {
	// if typ.Kind_&kindGCProg != 0 {
	// 	panic("reflect: unexpected GC program")
	// }
	// ptrs := typ.PtrBytes / goarch.PtrSize
	// words := typ.Size_ / goarch.PtrSize
	// mask := typ.GcSlice(0, (ptrs+7)/8)
	// for j := uintptr(0); j < ptrs; j++ {
	// 	if (mask[j/8]>>(j%8))&1 != 0 {
	// 		for i := uintptr(0); i < n; i++ {
	// 			k := base + i*words + j
	// 			out[k/8] |= 1 << (k % 8)
	// 		}
	// 	}
	// }
}

// ArrayOf returns the array type with the given length and element type.
// For example, if t represents int, ArrayOf(5, t) represents [5]int.
//
// If the resulting type would be larger than the available address space,
// ArrayOf panics.
func ArrayOf(length int, elem Type) Type {
	if length < 0 {
		panic("reflect: negative length passed to ArrayOf")
	}

	typ := elem.common()

	// Look in cache.
	ckey := cacheKey{Array, typ, nil, uintptr(length)}
	if array, ok := lookupCache.Load(ckey); ok {
		return array.(Type)
	}

	// Look in known types.
	s := "[" + strconv.Itoa(length) + "]" + stringFor(typ)
	for _, tt := range typesByString(s) {
		array := (*arrayType)(unsafe.Pointer(tt))
		if array.Elem == typ {
			ti, _ := lookupCache.LoadOrStore(ckey, toRType(tt))
			return ti.(Type)
		}
	}

	// Make an array type.
	var iarray any = [1]unsafe.Pointer{}
	prototype := *(**arrayType)(unsafe.Pointer(&iarray))
	array := *prototype
	array.TFlag = typ.TFlag & abi.TFlagRegularMemory
	array.Str_ = s
	array.Hash = fnv1(typ.Hash, '[')
	for n := uint32(length); n > 0; n >>= 8 {
		array.Hash = fnv1(array.Hash, byte(n))
	}
	array.Hash = fnv1(array.Hash, ']')
	array.Elem = typ
	array.PtrToThis_ = nil
	if typ.Size_ > 0 {
		max := ^uintptr(0) / typ.Size_
		if uintptr(length) > max {
			panic("reflect.ArrayOf: array size would exceed virtual address space")
		}
	}
	array.Size_ = typ.Size_ * uintptr(length)
	if length > 0 && typ.Pointers() {
		array.PtrBytes = typ.Size_*uintptr(length-1) + typ.PtrBytes
	} else {
		array.PtrBytes = 0
	}
	array.Align_ = typ.Align_
	array.FieldAlign_ = typ.FieldAlign_
	array.Len = uintptr(length)
	array.Slice = &(SliceOf(elem).(*rtype).t)

	switch {
	case array.PtrBytes == 0:
		// No pointers.
		array.GCData = nil

	case length == 1:
		// In memory, 1-element array looks just like the element.
		// We share the bitmask with the element type.
		array.TFlag |= typ.TFlag & abi.TFlagGCMaskOnDemand
		array.GCData = typ.GCData

	case array.PtrBytes <= abi.MaxPtrmaskBytes*8*goarch.PtrSize:
		// Create pointer mask by repeating the element bitmask Len times.
		n := (array.PtrBytes/goarch.PtrSize + 7) / 8
		// Runtime needs pointer masks to be a multiple of uintptr in size.
		n = (n + goarch.PtrSize - 1) &^ (goarch.PtrSize - 1)
		mask := make([]byte, n)
		emitGCMask(mask, 0, typ, array.Len)
		array.GCData = &mask[0]

	default:
		// Runtime will build the mask if needed. We just need to allocate
		// space to store it.
		array.TFlag |= abi.TFlagGCMaskOnDemand
		array.GCData = (*byte)(unsafe.Pointer(new(uintptr)))
	}

	etyp := typ
	esize := etyp.Size()

	array.Equal = nil
	if eequal := etyp.Equal; eequal != nil {
		array.Equal = func(p, q unsafe.Pointer) bool {
			for i := 0; i < length; i++ {
				pi := arrayAt(p, i, esize, "i < length")
				qi := arrayAt(q, i, esize, "i < length")
				if !eequal(pi, qi) {
					return false
				}

			}
			return true
		}
	}

	switch {
	case length == 1 && !typ.IfaceIndir():
		// array of 1 direct iface type can be direct
		array.Kind_ |= abi.KindDirectIface
	default:
		array.Kind_ &^= abi.KindDirectIface
	}

	ti, _ := lookupCache.LoadOrStore(ckey, toRType(&array.Type))
	return ti.(Type)
}

// The structLookupCache caches StructOf lookups.
// StructOf does not share the common lookupCache since we need to pin
// the memory associated with *structTypeFixedN.
var structLookupCache struct {
	sync.Mutex // Guards stores (but not loads) on m.

	// m is a map[uint32][]Type keyed by the hash calculated in StructOf.
	// Elements in m are append-only and thus safe for concurrent reading.
	m sync.Map
}

type structTypeUncommon struct {
	structType
	u uncommonType
}

// isLetter reports whether a given 'rune' is classified as a Letter.
func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

// isValidFieldName checks if a string is a valid (struct) field name or not.
//
// According to the language spec, a field name should be an identifier.
//
// identifier = letter { letter | unicode_digit } .
// letter = unicode_letter | "_" .
func isValidFieldName(fieldName string) bool {
	for i, c := range fieldName {
		if i == 0 && !isLetter(c) {
			return false
		}

		if !(isLetter(c) || unicode.IsDigit(c)) {
			return false
		}
	}

	return len(fieldName) > 0
}

// This must match cmd/compile/internal/compare.IsRegularMemory
func isRegularMemory(t Type) bool {
	switch t.Kind() {
	case Array:
		elem := t.Elem()
		if isRegularMemory(elem) {
			return true
		}
		return elem.Comparable() && t.Len() == 0
	case Int8, Int16, Int32, Int64, Int, Uint8, Uint16, Uint32, Uint64, Uint, Uintptr, Chan, Pointer, Bool, UnsafePointer:
		return true
	case Struct:
		num := t.NumField()
		switch num {
		case 0:
			return true
		case 1:
			field := t.Field(0)
			if field.Name == "_" {
				return false
			}
			return isRegularMemory(field.Type)
		default:
			for i := 0; i < num; i++ {
				field := t.Field(i)
				if field.Name == "_" || !isRegularMemory(field.Type) || isPaddedField(t, i) {
					return false
				}
			}
			return true
		}
	}
	return false
}

// isPaddedField reports whether the i'th field of struct type t is followed
// by padding.
func isPaddedField(t Type, i int) bool {
	field := t.Field(i)
	if i+1 < t.NumField() {
		return field.Offset+field.Type.Size() != t.Field(i+1).Offset
	}
	return field.Offset+field.Type.Size() != t.Size()
}

// StructOf returns the struct type containing fields.
// The Offset and Index fields are ignored and computed as they would be
// by the compiler.
//
// StructOf currently does not support promoted methods of embedded fields
// and panics if passed unexported StructFields.
func StructOf(fields []StructField) Type {
	var (
		hash       = fnv1(0, []byte("struct {")...)
		size       uintptr
		typalign   uint8
		comparable = true
		methods    []abi.Method

		fs   = make([]structField, len(fields))
		repr = make([]byte, 0, 64)
		fset = map[string]struct{}{} // fields' names
	)

	lastzero := uintptr(0)
	repr = append(repr, "struct {"...)
	pkgpath := ""
	for i, field := range fields {
		if field.Name == "" {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no name")
		}
		if !isValidFieldName(field.Name) {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has invalid name")
		}
		if field.Type == nil {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no type")
		}
		f, fpkgpath := runtimeStructField(field)
		ft := f.Typ
		if fpkgpath != "" {
			if pkgpath == "" {
				pkgpath = fpkgpath
			} else if pkgpath != fpkgpath {
				panic("reflect.Struct: fields with different PkgPath " + pkgpath + " and " + fpkgpath)
			}
		}

		// Update string and hash
		name := f.Name_
		hash = fnv1(hash, []byte(name)...)
		if !f.Embedded() {
			repr = append(repr, (" " + name)...)
		} else {
			// Embedded field
			if f.Typ.Kind() == abi.Pointer {
				// Embedded ** and *interface{} are illegal
				elem := ft.Elem()
				if k := elem.Kind(); k == abi.Pointer || k == abi.Interface {
					panic("reflect.StructOf: illegal embedded field type " + stringFor(ft))
				}
			}

			switch Kind(f.Typ.Kind()) {
			case Interface:
				ift := (*interfaceType)(unsafe.Pointer(ft))
				for _, m := range ift.Methods {
					if m.PkgPath() != "" {
						// TODO(sbinet).  Issue 15924.
						panic("reflect: embedded interface with unexported method(s) not implemented")
					}

					fnStub := clite.Func(embeddedIfaceMethStub)
					methods = append(methods, abi.Method{
						Name_: m.Name(),
						Mtyp_: m.Typ_,
						Ifn_:  fnStub,
						Tfn_:  fnStub,
					})
				}
			case Pointer:
				ptr := (*ptrType)(unsafe.Pointer(ft))
				if unt := ptr.Uncommon(); unt != nil {
					if i > 0 && unt.Mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 {
						panic("reflect: embedded type with methods not implemented if there is more than one field")
					}
					for _, m := range unt.Methods() {
						if m.PkgPath() != "" {
							// TODO(sbinet).
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
						methods = append(methods, abi.Method{
							Name_: m.Name(),
							Mtyp_: m.Mtyp_,
							Ifn_:  m.Ifn_,
							Tfn_:  m.Tfn_,
						})
					}
				}
				if unt := ptr.Elem.Uncommon(); unt != nil {
					for _, m := range unt.Methods() {
						if m.PkgPath() != "" {
							// TODO(sbinet)
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
						methods = append(methods, abi.Method{
							Name_: m.Name(),
							Mtyp_: m.Mtyp_,
							Ifn_:  m.Ifn_,
							Tfn_:  m.Tfn_,
						})
					}
				}
			default:
				if unt := ft.Uncommon(); unt != nil {
					if i > 0 && unt.Mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 && ft.Kind_&abi.KindDirectIface != 0 {
						panic("reflect: embedded type with methods not implemented for non-pointer type")
					}
					for _, m := range unt.Methods() {
						if m.PkgPath() != "" {
							// TODO(sbinet)
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
						methods = append(methods, abi.Method{
							Name_: m.Name(),
							Mtyp_: m.Mtyp_,
							Ifn_:  m.Ifn_,
							Tfn_:  m.Tfn_,
						})

					}
				}
			}
		}
		if _, dup := fset[name]; dup && name != "_" {
			panic("reflect.StructOf: duplicate field " + name)
		}
		fset[name] = struct{}{}

		hash = fnv1(hash, byte(ft.Hash>>24), byte(ft.Hash>>16), byte(ft.Hash>>8), byte(ft.Hash))

		repr = append(repr, (" " + stringFor(ft))...)
		if f.Tag_ != "" {
			hash = fnv1(hash, []byte(f.Tag_)...)
			repr = append(repr, (" " + strconv.Quote(f.Tag_))...)
		}
		if i < len(fields)-1 {
			repr = append(repr, ';')
		}

		comparable = comparable && (ft.Equal != nil)

		offset := align(size, uintptr(ft.Align_))
		if offset < size {
			panic("reflect.StructOf: struct size would exceed virtual address space")
		}
		if ft.Align_ > typalign {
			typalign = ft.Align_
		}
		size = offset + ft.Size_
		if size < offset {
			panic("reflect.StructOf: struct size would exceed virtual address space")
		}
		f.Offset = offset

		if ft.Size_ == 0 {
			lastzero = size
		}

		fs[i] = f
	}

	if size > 0 && lastzero == size {
		// This is a non-zero sized struct that ends in a
		// zero-sized field. We add an extra byte of padding,
		// to ensure that taking the address of the final
		// zero-sized field can't manufacture a pointer to the
		// next object in the heap. See issue 9401.
		size++
		if size == 0 {
			panic("reflect.StructOf: struct size would exceed virtual address space")
		}
	}

	var typ *structType
	var ut *uncommonType

	if len(methods) == 0 {
		t := new(structTypeUncommon)
		typ = &t.structType
		ut = &t.u
	} else {
		// A *rtype representing a struct is followed directly in memory by an
		// array of method objects representing the methods attached to the
		// struct. To get the same layout for a run time generated type, we
		// need an array directly following the uncommonType memory.
		// A similar strategy is used for funcTypeFixed4, ...funcTypeFixedN.
		tt := New(StructOf([]StructField{
			{Name: "S", Type: TypeOf(structType{})},
			{Name: "U", Type: TypeOf(uncommonType{})},
			{Name: "M", Type: ArrayOf(len(methods), TypeOf(methods[0]))},
		}))

		typ = (*structType)(tt.Elem().Field(0).Addr().UnsafePointer())
		ut = (*uncommonType)(tt.Elem().Field(1).Addr().UnsafePointer())

		copy(tt.Elem().Field(2).Slice(0, len(methods)).Interface().([]abi.Method), methods)
	}
	// TODO(sbinet): Once we allow embedding multiple types,
	// methods will need to be sorted like the compiler does.
	// TODO(sbinet): Once we allow non-exported methods, we will
	// need to compute xcount as the number of exported methods.
	ut.Mcount = uint16(len(methods))
	ut.Xcount = ut.Mcount
	ut.Moff = uint32(unsafe.Sizeof(uncommonType{}))

	if len(fs) > 0 {
		repr = append(repr, ' ')
	}
	repr = append(repr, '}')
	hash = fnv1(hash, '}')
	str := string(repr)

	// Round the size up to be a multiple of the alignment.
	s := align(size, uintptr(typalign))
	if s < size {
		panic("reflect.StructOf: struct size would exceed virtual address space")
	}
	size = s

	// Make the struct type.
	var istruct any = struct{}{}
	prototype := *(**structType)(unsafe.Pointer(&istruct))
	*typ = *prototype
	typ.Fields = fs
	if pkgpath != "" {
		typ.PkgPath_ = pkgpath
	}

	// Look in cache.
	if ts, ok := structLookupCache.m.Load(hash); ok {
		for _, st := range ts.([]Type) {
			t := st.common()
			if haveIdenticalUnderlyingType(&typ.Type, t, true) {
				return toType(t)
			}
		}
	}

	// Not in cache, lock and retry.
	structLookupCache.Lock()
	defer structLookupCache.Unlock()
	if ts, ok := structLookupCache.m.Load(hash); ok {
		for _, st := range ts.([]Type) {
			t := st.common()
			if haveIdenticalUnderlyingType(&typ.Type, t, true) {
				return toType(t)
			}
		}
	}

	addToCache := func(t Type) Type {
		var ts []Type
		if ti, ok := structLookupCache.m.Load(hash); ok {
			ts = ti.([]Type)
		}
		structLookupCache.m.Store(hash, append(ts, t))
		return t
	}

	// Look in known types.
	for _, t := range typesByString(str) {
		if haveIdenticalUnderlyingType(&typ.Type, t, true) {
			// even if 't' wasn't a structType with methods, we should be ok
			// as the 'u uncommonType' field won't be accessed except when
			// tflag&abi.TFlagUncommon is set.
			return addToCache(toType(t))
		}
	}

	typ.Str_ = str
	if isRegularMemory(toType(&typ.Type)) {
		typ.TFlag = abi.TFlagRegularMemory
	} else {
		typ.TFlag = 0
	}
	typ.Hash = hash
	typ.Size_ = size
	typ.PtrBytes = typeptrdata(&typ.Type)
	typ.Align_ = typalign
	typ.FieldAlign_ = typalign
	typ.PtrToThis_ = nil
	if len(methods) > 0 {
		typ.TFlag |= abi.TFlagUncommon
	}

	if typ.PtrBytes == 0 {
		typ.GCData = nil
	} else if typ.PtrBytes <= abi.MaxPtrmaskBytes*8*goarch.PtrSize {
		bv := new(bitVector)
		addTypeBits(bv, 0, &typ.Type)
		typ.GCData = &bv.data[0]
	} else {
		// Runtime will build the mask if needed. We just need to allocate
		// space to store it.
		typ.TFlag |= abi.TFlagGCMaskOnDemand
		typ.GCData = (*byte)(unsafe.Pointer(new(uintptr)))
	}

	typ.Equal = nil
	if comparable {
		typ.Equal = func(p, q unsafe.Pointer) bool {
			for _, ft := range typ.Fields {
				pi := add(p, ft.Offset, "&x.field safe")
				qi := add(q, ft.Offset, "&x.field safe")
				if !ft.Typ.Equal(pi, qi) {
					return false
				}
			}
			return true
		}
	}

	switch {
	case len(fs) == 1 && !fs[0].Typ.IfaceIndir():
		// structs of 1 direct iface type can be direct
		typ.Kind_ |= abi.KindDirectIface
	default:
		typ.Kind_ &^= abi.KindDirectIface
	}

	return addToCache(toType(&typ.Type))
}

const (
	pointerSize  = unsafe.Sizeof(uintptr(0))
	pointerAlign = uint8(unsafe.Alignof(uintptr(0)))
)

var closureLookupCache struct {
	sync.Mutex // Guards stores (but not loads) on m.

	// m is a map[uint32][]*rtype keyed by the hash calculated in FuncOf.
	// Elements of m are append-only and thus safe for concurrent reading.
	m sync.Map
}

// Struct returns a struct type.
func closureOf(ftyp *abi.FuncType) *abi.Type {
	if ftyp.HasName() {
		var t *abi.StructType
		namedFuncMap.Range(func(key interface{}, value interface{}) bool {
			if value == ftyp {
				t = key.(*abi.StructType)
				return false
			}
			return true
		})
		if t != nil {
			return &t.Type
		}
		return &toClosureType(ftyp).Type
	}

	fields := []abi.StructField{
		abi.StructField{
			Name_: "$f",
			Typ:   &ftyp.Type,
		}, abi.StructField{
			Name_:  "$data",
			Typ:    unsafePointerType,
			Offset: pointerSize,
		},
	}
	size := 2 * pointerSize

	repr := make([]byte, 0, 64)
	hash := fnv1(0, []byte("struct {")...)
	repr = append(repr, "struct {"...)
	for i, f := range fields {
		if i > 0 {
			repr = append(repr, ';')
		}
		ft := f.Typ
		repr = append(repr, ' ')
		repr = append(repr, f.Name_...)
		repr = append(repr, ' ')
		repr = append(repr, ft.String()...)
		hash = fnv1(hash, []byte(f.Name_)...)
		hash = fnv1(hash, byte(ft.Hash>>24), byte(ft.Hash>>16), byte(ft.Hash>>8), byte(ft.Hash))
	}
	repr = append(repr, ' ')
	repr = append(repr, '}')
	hash = fnv1(hash, '}')
	str := string(repr)

	st := &abi.StructType{
		Type: abi.Type{
			Size_:       size,
			Kind_:       uint8(abi.Struct),
			Str_:        str,
			TFlag:       abi.TFlagClosure,
			PtrBytes:    2 * pointerSize,
			Align_:      pointerAlign,
			FieldAlign_: pointerAlign,
			Hash:        hash,
		},
		PkgPath_: "",
		Fields:   fields,
	}

	closureLookupCache.Lock()
	defer closureLookupCache.Unlock()

	// Look in cache.
	if ts, ok := closureLookupCache.m.Load(hash); ok {
		for _, t := range ts.([]*abi.Type) {
			if haveIdenticalUnderlyingType(&st.Type, t, true) {
				return t
			}
		}
	}

	addToCache := func(tt *abi.Type) *abi.Type {
		var rts []*abi.Type
		if rti, ok := closureLookupCache.m.Load(hash); ok {
			rts = rti.([]*abi.Type)
		}
		closureLookupCache.m.Store(hash, append(rts, tt))
		return tt
	}

	for _, tt := range typesByString(str) {
		if haveIdenticalUnderlyingType(&st.Type, tt, true) {
			return addToCache(tt)
		}
	}
	return addToCache(&st.Type)
}

func structStr(fields []abi.StructField) string {
	repr := make([]byte, 0, 64)
	repr = append(repr, "struct {"...)
	for i, st := range fields {
		if i > 0 {
			repr = append(repr, ';')
		}
		repr = append(repr, ' ')
		if !st.Embedded_ {
			repr = append(repr, st.Name_...)
			repr = append(repr, ' ')
		}
		repr = append(repr, st.Typ.String()...)
	}
	if len(fields) > 0 {
		repr = append(repr, ' ')
	}
	repr = append(repr, '}')
	return string(repr)
}

// Note: this type must agree with runtime.bitvector.
type bitVector struct {
	n    uint32 // number of bits
	data []byte
}

// append a bit to the bitmap.
func (bv *bitVector) append(bit uint8) {
	if bv.n%(8*goarch.PtrSize) == 0 {
		// Runtime needs pointer masks to be a multiple of uintptr in size.
		// Since reflect passes bv.data directly to the runtime as a pointer mask,
		// we append a full uintptr of zeros at a time.
		for i := 0; i < goarch.PtrSize; i++ {
			bv.data = append(bv.data, 0)
		}
	}
	bv.data[bv.n/8] |= bit << (bv.n % 8)
	bv.n++
}

func addTypeBits(bv *bitVector, offset uintptr, t *abi.Type) {
	if !t.Pointers() {
		return
	}

	switch Kind(t.Kind_ & abi.KindMask) {
	case Chan, Func, Map, Pointer, Slice, String, UnsafePointer:
		// 1 pointer at start of representation
		for bv.n < uint32(offset/goarch.PtrSize) {
			bv.append(0)
		}
		bv.append(1)

	case Interface:
		// 2 pointers
		for bv.n < uint32(offset/goarch.PtrSize) {
			bv.append(0)
		}
		bv.append(1)
		bv.append(1)

	case Array:
		// repeat inner type
		tt := (*arrayType)(unsafe.Pointer(t))
		for i := 0; i < int(tt.Len); i++ {
			addTypeBits(bv, offset+uintptr(i)*tt.Elem.Size_, tt.Elem)
		}

	case Struct:
		// apply fields
		tt := (*structType)(unsafe.Pointer(t))
		for i := range tt.Fields {
			f := &tt.Fields[i]
			addTypeBits(bv, offset+f.Offset, f.Typ)
		}
	}
}

// typeptrdata returns the length in bytes of the prefix of t
// containing pointer data. Anything after this offset is scalar data.
// keep in sync with ../cmd/compile/internal/reflectdata/reflect.go
func typeptrdata(t *abi.Type) uintptr {
	switch t.Kind() {
	case abi.Struct:
		st := (*structType)(unsafe.Pointer(t))
		// find the last field that has pointers.
		field := -1
		for i := range st.Fields {
			ft := st.Fields[i].Typ
			if ft.Pointers() {
				field = i
			}
		}
		if field == -1 {
			return 0
		}
		f := st.Fields[field]
		return f.Offset + f.Typ.PtrBytes

	default:
		panic("reflect.typeptrdata: unexpected type, " + stringFor(t))
	}
}

// align returns the result of rounding x up to a multiple of n.
// n must be a power of two.
func align(x, n uintptr) uintptr {
	return (x + n - 1) &^ (n - 1)
}

func embeddedIfaceMethStub() {
	panic("reflect: StructOf does not support methods of embedded interfaces")
}

// runtimeStructField takes a StructField value passed to StructOf and
// returns both the corresponding internal representation, of type
// structField, and the pkgpath value to use for this field.
func runtimeStructField(field StructField) (structField, string) {
	if field.Anonymous && field.PkgPath != "" {
		panic("reflect.StructOf: field \"" + field.Name + "\" is anonymous but has PkgPath set")
	}

	if field.IsExported() {
		// Best-effort check for misuse.
		// Since this field will be treated as exported, not much harm done if Unicode lowercase slips through.
		c := field.Name[0]
		if 'a' <= c && c <= 'z' || c == '_' {
			panic("reflect.StructOf: field \"" + field.Name + "\" is unexported but missing PkgPath")
		}
	}

	//	resolveReflectType(field.Type.common()) // install in runtime
	f := structField{
		Name_:     field.Name,
		Typ:       field.Type.common(),
		Tag_:      string(field.Tag),
		Offset:    0,
		Embedded_: field.Anonymous,
	}
	return f, field.PkgPath
}

//go:linkname typelist github.com/goplus/llgo/runtime/internal/runtime.typelist
var typelist []*abi.Type

func typesByString(s string) (typs []*abi.Type) {
	for _, t := range typelist {
		if t.String() == s {
			typs = append(typs, t)
		}
	}
	return
}

func findNamed(pkgPath string, name string) *abi.Type {
	for _, typ := range typelist {
		if typ.TFlag&(abi.TFlagNamed|abi.TFlagUncommon) != 0 &&
			typ.Str_ == name && typ.Uncommon().PkgPath_ == pkgPath {
			return typ
		}
	}
	return nil
}

func findElem(kind abi.Kind, elem *abi.Type, extra uintptr) *abi.Type {
	for _, typ := range typelist {
		if typ.Kind() == kind && typ.Elem() == elem {
			switch kind {
			case abi.Chan:
				if uintptr(typ.ChanDir()) == extra {
					return typ
				}
			case abi.Array:
				if uintptr(typ.Len()) == extra {
					return typ
				}
			default:
				return typ
			}
		}
	}
	return nil
}

func findMap(key, elem *abi.Type) *abi.Type {
	for _, typ := range typelist {
		if typ.Kind() == abi.Map {
			if mt := typ.MapType(); mt.Key == key && mt.Elem == elem {
				return typ
			}
		}
	}
	return nil
}

func eqFields(s1, s2 []abi.StructField) bool {
	n := len(s1)
	if n != len(s2) {
		return false
	}
	for i := 0; i < n; i++ {
		f1, f2 := s1[i], s2[i]
		if f1.Name_ != f2.Name_ || f1.Embedded_ != f2.Embedded_ || f1.Typ != f2.Typ {
			return false
		}
	}
	return true
}

func findStruct(pkgPath string, size uintptr, fields []abi.StructField) *abi.Type {
	for _, typ := range typelist {
		if typ.Kind() == abi.Struct && typ.Size() == size {
			if st := typ.StructType(); (st.PkgPath_ == pkgPath) && eqFields(st.Fields, fields) {
				return typ
			}
		}
	}
	return nil
}

func eqImethods(s1, s2 []abi.Imethod) bool {
	n := len(s1)
	if n != len(s2) {
		return false
	}
	for i := 0; i < n; i++ {
		f1, f2 := s1[i], s2[i]
		if f1.Name_ != f2.Name_ || f1.Typ_ != f2.Typ_ {
			return false
		}
	}
	return true
}

func findInterface(pkgPath string, methods []abi.Imethod) *abi.InterfaceType {
	for _, typ := range typelist {
		if typ.Kind() == abi.Interface {
			if it := typ.InterfaceType(); (it.PkgPath_ == pkgPath) && eqImethods(it.Methods, methods) {
				return it
			}
		}
	}
	return nil
}

func eqTypes(s1, s2 []*abi.Type) bool {
	n := len(s1)
	if n != len(s2) {
		return false
	}
	for i := 0; i < n; i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func findFunc(in, out []*abi.Type, variadic bool) *abi.FuncType {
	for _, typ := range typelist {
		if typ.Kind() == abi.Func {
			if ft := typ.FuncType(); ft.Variadic() == variadic && eqTypes(ft.In, in) && eqTypes(ft.Out, out) {
				return ft
			}
		}
	}
	return nil
}

func findClosure(fields []abi.StructField) *abi.Type {
	for _, typ := range typelist {
		if typ.Kind() == abi.Struct && typ.Size() == pointerSize {
			if st := typ.StructType(); st.IsClosure() && eqFields(st.Fields, fields) {
				return typ
			}
		}
	}
	return nil
}
