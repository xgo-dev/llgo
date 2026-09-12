//go:build !llgo_noffi

package reflect

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/abi"
	"github.com/xgo-dev/llgo/runtime/internal/ffi"
	"github.com/xgo-dev/llgo/runtime/internal/runtime"
)

func (v Value) call(op string, in []Value) (out []Value) {
	var (
		ft   *abi.FuncType
		tin  []*abi.Type
		args []unsafe.Pointer
		fn   unsafe.Pointer
		env  unsafe.Pointer
		ret  unsafe.Pointer
		ioff int
	)
	if v.typ_.IsClosure() && v.flag&flagMethod == 0 {
		ft = v.typ_.StructType().Fields[0].Typ.FuncType()
		c := (*struct {
			fn  unsafe.Pointer
			env unsafe.Pointer
		})(v.ptr)
		fn = c.fn
		env = c.env
		tin = ft.In
		if methodvalueNoFFIRepresentation && env != nil {
			if direct, ok := directMethodValueEnvAt(env); ok {
				// A direct method value stores its receiver in the closure
				// context. It is an ordinary method argument, not a hidden
				// closure environment, on either ABI.
				tin = append([]*abi.Type{direct.receiverType}, tin...)
				args = append(args, direct.receiver)
				env = nil
			}
		}
		if env != nil && ffi.ClosureEnvExplicit {
			tin = append([]*abi.Type{rtypeOf(unsafe.Pointer(nil))}, tin...)
			ioff = 1
			args = append(args, unsafe.Pointer(&env))
		}
	} else {
		if v.flag&flagMethod != 0 {
			var (
				rcvrtype *abi.Type
			)
			rcvrtype, ft, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
			tin = append([]*abi.Type{rcvrtype}, ft.In...)
			ioff = 1
			var ptr unsafe.Pointer
			storeRcvr(v, unsafe.Pointer(&ptr))
			args = append(args, unsafe.Pointer(&ptr))
		} else {
			if v.flag&flagIndir != 0 {
				fn = *(*unsafe.Pointer)(v.ptr)
			} else {
				fn = v.ptr
			}
			ft = v.typ_.FuncType()
			tin = ft.In
		}
	}

	if fn == nil {
		panic("reflect.Value.Call: call of nil function")
	}

	isSlice := op == "CallSlice"
	n := len(ft.In)
	isVariadic := ft.Variadic()
	if isSlice {
		if !isVariadic {
			panic("reflect: CallSlice of non-variadic function")
		}
		if len(in) < n {
			panic("reflect: CallSlice with too few input arguments")
		}
		if len(in) > n {
			panic("reflect: CallSlice with too many input arguments")
		}
	} else {
		if isVariadic {
			n--
		}
		if len(in) < n {
			panic("reflect: Call with too few input arguments")
		}
		if !isVariadic && len(in) > n {
			panic("reflect: Call with too many input arguments")
		}
	}
	for _, x := range in {
		if x.Kind() == Invalid {
			panic("reflect: " + op + " using zero Value argument")
		}
	}
	for i := 0; i < n; i++ {
		if xt, targ := in[i].Type(), ft.In[i]; !xt.AssignableTo(toPublicType(targ)) {
			panic("reflect: " + op + " using " + xt.String() + " as type " + stringFor(targ))
		}
	}
	if !isSlice && isVariadic {
		// prepare slice for remaining values
		m := len(in) - n
		slice := MakeSlice(toRType(ft.In[n]), m, m)
		elem := toPublicType(ft.In[n].Elem()) // FIXME cast to slice type and Elem()
		for i := 0; i < m; i++ {
			x := in[n+i]
			if xt := x.Type(); !xt.AssignableTo(elem) {
				panic("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op)
			}
			slice.Index(i).Set(x)
		}
		origIn := in
		in = make([]Value, n+1)
		copy(in[:n], origIn)
		in[n] = slice
	}

	nin := len(in)
	if nin != len(ft.In) {
		panic("reflect.Value.Call: wrong argument count")
	}

	ffiArgs := make([]*ffi.Type, 0, len(tin)+4)
	for i := 0; i < ioff; i++ {
		ffiArgs = append(ffiArgs, toFFIType(tin[i]))
	}
	for i, arg := range in {
		typ := tin[ioff+i]
		if ffiCallSliceAsTriple && typ.Kind() == abi.Slice {
			h := (*unsafeheaderSlice)(arg.ptr)
			ffiArgs = append(ffiArgs, ffi.TypePointer, ffi.TypeInt, ffi.TypeInt)
			args = append(args, unsafe.Pointer(&h.Data), unsafe.Pointer(&h.Len), unsafe.Pointer(&h.Cap))
			continue
		}
		ffiArgs = append(ffiArgs, toFFIType(typ))
		args = append(args, toFFIArg(arg, typ))
	}

	sig, err := ffi.NewSignature(toFFIRetType(ft.Out), ffiArgs...)
	if err != nil {
		panic(err)
	}
	if sig.RType != ffi.TypeVoid {
		v := runtime.AllocZ(sig.RType.Size)
		// libffi expects rvalue to point to the return value storage.
		ret = v
	}

	ffi.CallWithEnv(sig, fn, env, ret, args...)
	tout := toRuntimeTypes(ft.Out)
	switch n := len(tout); n {
	case 0:
	case 1:
		out := NewAt(toType(tout[0]), ret).Elem()
		resolveIndirectValue(&out, tout[0])
		return []Value{out}
	default:
		out = make([]Value, n)
		alignment := uintptr(sig.RType.Alignment)
		var off uintptr
		for i, tout := range tout {
			out[i] = NewAt(toType(tout), add(ret, off, "")).Elem()
			resolveIndirectValue(&out[i], tout)
			off += (tout.Size_ + alignment - 1) &^ (alignment - 1)
		}
	}
	return
}

func toFFIWordArg(v Value) unsafe.Pointer {
	if v.flag&flagIndir != 0 {
		return v.ptr
	}
	return unsafe.Pointer(&v.ptr)
}

func toFFIArg(v Value, typ *abi.Type) unsafe.Pointer {
	kind := typ.Kind()
	switch kind {
	case abi.Bool, abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64,
		abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr,
		abi.Float32, abi.Float64:
		if v.flag&flagIndir != 0 {
			return v.ptr
		} else {
			return unsafe.Pointer(&v.ptr)
		}
	case abi.Complex64, abi.Complex128:
		return unsafe.Pointer(v.ptr)
	case abi.Array:
		if v.flag&flagIndir != 0 {
			return v.ptr
		}
		return unsafe.Pointer(&v.ptr)
	case abi.Chan:
		return toFFIWordArg(v)
	case abi.Func:
		if v.flag&flagIndir != 0 {
			return v.ptr
		}
		return unsafe.Pointer(&v.ptr)
	case abi.Interface:
		i := v.Interface()
		iface := typ.InterfaceType()
		if len(iface.Methods) == 0 {
			return unsafe.Pointer(&i)
		}
		// Interface-valued reflect arguments carry the dynamic concrete type
		// in i. Using v.typ() here instead constructs an itab for the source
		// interface type itself, leaving method entries invalid when the
		// reflected callee invokes them.
		itab := runtime.NewItab(iface, rtypeOf(i))
		data := struct {
			itab *runtime.Itab
			data unsafe.Pointer
		}{itab, (*emptyInterface)(unsafe.Pointer(&i)).word}
		return unsafe.Pointer(&data)
	case abi.Map:
		return toFFIWordArg(v)
	case abi.Pointer:
		return toFFIWordArg(v)
	case abi.Slice:
		return v.ptr
	case abi.String:
		return v.ptr
	case abi.Struct:
		if v.flag&flagIndir != 0 {
			return v.ptr
		}
		return unsafe.Pointer(&v.ptr)
	case abi.UnsafePointer:
		return toFFIWordArg(v)
	}
	panic("reflect.toFFIArg unsupport type " + v.typ().String())
}

var (
	ffiTypeClosure = ffi.StructOf(ffi.TypePointer, ffi.TypePointer)
)

func toFFIType(typ *abi.Type) *ffi.Type {
	kind := typ.Kind()
	switch kind {
	case abi.Bool, abi.Int, abi.Int8, abi.Int16, abi.Int32, abi.Int64,
		abi.Uint, abi.Uint8, abi.Uint16, abi.Uint32, abi.Uint64, abi.Uintptr,
		abi.Float32, abi.Float64, abi.Complex64, abi.Complex128:
		return ffi.Typ[kind]
	case abi.Array:
		st := typ.ArrayType()
		return ffi.ArrayOf(toFFIType(st.Elem), int(st.Len))
	case abi.Chan:
		return ffi.TypePointer
	case abi.Func:
		return ffiTypeClosure
	case abi.Interface:
		return ffi.TypeInterface
	case abi.Map:
		return ffi.TypePointer
	case abi.Pointer:
		return ffi.TypePointer
	case abi.Slice:
		return ffi.TypeSlice
	case abi.String:
		return ffi.TypeString
	case abi.Struct:
		if typ.IsClosure() {
			return ffiTypeClosure
		}
		return toFFIStructType(typ)
	case abi.UnsafePointer:
		return ffi.TypePointer
	}
	panic("reflect.toFFIType unsupport type " + typ.String())
}

func toFFIStructType(typ *abi.Type) *ffi.Type {
	st := typ.StructType()
	fields := make([]*ffi.Type, 0, len(st.Fields))
	var off uintptr
	for _, fs := range st.Fields {
		if fs.Offset > off {
			fields, off = appendFFIPadding(fields, off, fs.Offset-off)
		}
		if fs.Typ.Size_ == 0 {
			continue
		}
		fields = append(fields, toFFIType(fs.Typ))
		off = fs.Offset + fs.Typ.Size_
	}
	// Do not pad to typ.Size_: trailing zero-sized fields can enlarge the
	// Go-visible size without consuming registers in llgo's callable ABI.
	return ffi.StructOf(fields...)
}

func appendFFIPadding(fields []*ffi.Type, off, size uintptr) ([]*ffi.Type, uintptr) {
	for size > 0 {
		switch {
		case off%8 == 0 && size >= 8:
			fields = append(fields, ffi.TypeUint64)
			off += 8
			size -= 8
		case off%4 == 0 && size >= 4:
			fields = append(fields, ffi.TypeUint32)
			off += 4
			size -= 4
		case off%2 == 0 && size >= 2:
			fields = append(fields, ffi.TypeUint16)
			off += 2
			size -= 2
		default:
			fields = append(fields, ffi.TypeUint8)
			off++
			size--
		}
	}
	return fields, off
}

func toFFISig(tin, tout []*abi.Type) (*ffi.Signature, error) {
	args := make([]*ffi.Type, len(tin))
	for i, in := range tin {
		args[i] = toFFIType(in)
	}
	return ffi.NewSignature(toFFIRetType(tout), args...)
}

func toFFIRetType(tout []*abi.Type) *ffi.Type {
	switch n := len(tout); n {
	case 0:
		return ffi.TypeVoid
	case 1:
		return toFFIType(tout[0])
	default:
		fields := make([]*ffi.Type, n)
		for i, out := range tout {
			fields[i] = toFFIType(out)
		}
		return ffi.StructOf(fields...)
	}
}
