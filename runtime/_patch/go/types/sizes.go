package types

import "runtime"

// llgoSizes implements go/types.Sizes for LLGo's standard-Go layout rules:
//   - 2-word Go function/closure values ({codePtr, envPtr} = 2*WordSize)
//   - structs.HostLayout 64-bit alignment on 32-bit architectures where HostAlign64 is true
//   - sync/atomic.align64 8-byte alignment
//   - trailing zero-sized struct field padding (1 byte) and struct tail alignment padding
//   - target word/alignment rules matching github.com/xgo-dev/llgo/types.Context.GoSizes()
type llgoSizes struct {
	WordSize    int64
	MaxAlign    int64
	hostAlign64 bool
}

var llgoArchSizes = map[string]*llgoSizes{
	"386":      {WordSize: 4, MaxAlign: 4, hostAlign64: false},
	"amd64":    {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"amd64p32": {WordSize: 4, MaxAlign: 8, hostAlign64: true},
	"arm":      {WordSize: 4, MaxAlign: 4, hostAlign64: true},
	"arm64":    {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"loong64":  {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"mips":     {WordSize: 4, MaxAlign: 4, hostAlign64: true},
	"mipsle":   {WordSize: 4, MaxAlign: 4, hostAlign64: true},
	"mips64":   {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"mips64le": {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"ppc64":    {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"ppc64le":  {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"riscv64":  {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"s390x":    {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"sparc64":  {WordSize: 8, MaxAlign: 8, hostAlign64: true},
	"wasm":     {WordSize: 4, MaxAlign: 4, hostAlign64: true},
}

func init() {
	if runtime.GOARCH == "wasm" {
		ws := int64(4 << (^uintptr(0) >> 63))
		llgoArchSizes["wasm"] = &llgoSizes{WordSize: ws, MaxAlign: ws, hostAlign64: true}
	}
}

func llgoIsStructsPkg(path string) bool {
	return path == "structs" || path == "github.com/xgo-dev/llgo/runtime/internal/structs"
}

func llgoIsHostLayoutMarker(T Type) bool {
	if T == nil {
		return false
	}
	if n, ok := Unalias(T).(*Named); ok {
		if obj := n.Obj(); obj != nil && obj.Pkg() != nil && llgoIsStructsPkg(obj.Pkg().Path()) &&
			(obj.Name() == "HostLayout" || obj.Name() == "hostLayout") {
			return true
		}
		if st, ok := n.Underlying().(*Struct); ok && st.NumFields() == 1 {
			if f0, ok := Unalias(st.Field(0).Type()).(*Named); ok {
				if obj := f0.Obj(); obj != nil && obj.Pkg() != nil && llgoIsStructsPkg(obj.Pkg().Path()) && obj.Name() == "hostLayout" {
					return true
				}
			}
		}
	}
	return false
}

func llgoStructHasHostLayout(st *Struct) bool {
	if st == nil {
		return false
	}
	for i := 0; i < st.NumFields(); i++ {
		if llgoIsHostLayoutMarker(st.Field(i).Type()) {
			return true
		}
	}
	return false
}

func (s *llgoSizes) fieldAlignof(T Type, hostLayout bool) int64 {
	if hostLayout && s.hostAlign64 {
		switch t := T.Underlying().(type) {
		case *Basic:
			switch t.kind {
			case Int64, Uint64, Float64, Complex128:
				return 8
			}
		case *Array:
			return s.fieldAlignof(t.elem, true)
		case *Struct:
			return s.alignofStruct(T, t, true)
		}
	}
	return s.Alignof(T)
}

func (s *llgoSizes) alignofStruct(T Type, t *Struct, inheritedHost bool) int64 {
	if len(t.fields) == 0 && _IsSyncAtomicAlign64(T) {
		return 8
	}
	host := inheritedHost || llgoStructHasHostLayout(t)
	max := int64(1)
	for _, f := range t.fields {
		if a := s.fieldAlignof(f.typ, host); a > max {
			max = a
		}
	}
	return max
}

func (s *llgoSizes) Alignof(T Type) (result int64) {
	defer func() {
		assert(result >= 1)
	}()
	switch t := T.Underlying().(type) {
	case *Array:
		return s.Alignof(t.elem)
	case *Struct:
		return s.alignofStruct(T, t, false)
	case *Slice, *Interface, *Signature:
		assert(!isTypeParam(T))
		return s.WordSize
	case *Basic:
		if t.Info()&IsString != 0 {
			return s.WordSize
		}
	case *TypeParam, *Union:
		panic("unreachable")
	}
	a := s.Sizeof(T)
	if a < 1 {
		return 1
	}
	if isComplex(T) {
		a /= 2
	}
	if a > s.MaxAlign {
		return s.MaxAlign
	}
	return a
}

func (s *llgoSizes) offsetsofHost(fields []*Var, host bool) []int64 {
	offsets := make([]int64, len(fields))
	var offs int64
	for i, f := range fields {
		if offs < 0 {
			offsets[i] = -1
			continue
		}
		a := s.fieldAlignof(f.typ, host)
		offs = align(offs, a)
		offsets[i] = offs
		if d := s.sizeofHost(f.typ, host); d >= 0 && offs >= 0 {
			offs += d
		} else {
			offs = -1
		}
	}
	return offsets
}

func (s *llgoSizes) Offsetsof(fields []*Var) []int64 {
	host := false
	for _, f := range fields {
		if llgoIsHostLayoutMarker(f.typ) {
			host = true
			break
		}
	}
	return s.offsetsofHost(fields, host)
}

func (s *llgoSizes) sizeofHost(T Type, inheritedHost bool) int64 {
	switch t := T.Underlying().(type) {
	case *Basic:
		assert(isTyped(T))
		k := t.kind
		if int(k) < len(basicSizes) {
			if sz := basicSizes[k]; sz > 0 {
				return int64(sz)
			}
		}
		if k == String {
			return s.WordSize * 2
		}
	case *Array:
		n := t.len
		if n <= 0 {
			return 0
		}
		esize := s.sizeofHost(t.elem, inheritedHost)
		if esize < 0 {
			return -1
		}
		if esize == 0 {
			return 0
		}
		const maxInt64 = 1<<63 - 1
		if esize > maxInt64/n {
			return -1
		}
		return esize * n
	case *Slice:
		return s.WordSize * 3
	case *Struct:
		n := t.NumFields()
		if n == 0 {
			return 0
		}
		host := inheritedHost || llgoStructHasHostLayout(t)
		offsets := s.offsetsofHost(t.fields, host)
		offs := offsets[n-1]
		size := s.sizeofHost(t.fields[n-1].typ, host)
		if offs < 0 || size < 0 {
			return -1
		}
		if offs > 0 && size == 0 {
			size = 1
		}
		return align(offs+size, s.alignofStruct(T, t, host))
	case *Interface, *Signature:
		assert(!isTypeParam(T))
		return s.WordSize * 2
	case *TypeParam, *Union:
		panic("unreachable")
	}
	return s.WordSize
}

func (s *llgoSizes) Sizeof(T Type) int64 {
	return s.sizeofHost(T, false)
}

// SizesFor returns the Sizes used by a compiler for an architecture, adding
// LLGo's standard-Go layout rules for compiler "llgo".
func SizesFor(compiler, arch string) Sizes {
	switch compiler {
	case "llgo":
		if s, ok := llgoArchSizes[arch]; ok {
			return s
		}
	case "gc":
		if s := gcSizesFor(compiler, arch); s != nil {
			return Sizes(s)
		}
	case "gccgo":
		if s, ok := gccgoArchSizes[arch]; ok {
			return Sizes(s)
		}
	}
	return nil
}

// stdSizes is used when Config.Sizes == nil under LLGo.
var stdSizes = func() Sizes {
	if s := SizesFor("llgo", runtime.GOARCH); s != nil {
		return s
	}
	return SizesFor("llgo", "amd64")
}()
