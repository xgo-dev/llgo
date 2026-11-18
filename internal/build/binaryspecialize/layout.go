// Package binaryspecialize models fixed-size encoding/binary operations using
// the concrete operand types known to the compiler.
package binaryspecialize

import "go/types"

// layout describes the encoding/binary wire representation. Its width is
// independent of the target's Go struct padding and alignment.
type layout struct {
	typ    types.Type
	width  int64
	elem   *layout
	fields []fieldLayout
}

type fieldLayout struct {
	field  *types.Var
	offset int64
	value  *layout
}

func fixedLayout(typ types.Type, maxWidth int64) *layout {
	wire := &layout{typ: typ}
	switch base := typ.Underlying().(type) {
	case *types.Basic:
		switch base.Kind() {
		case types.Bool, types.Int8, types.Uint8:
			wire.width = 1
		case types.Int16, types.Uint16:
			wire.width = 2
		case types.Int32, types.Uint32, types.Float32:
			wire.width = 4
		case types.Int64, types.Uint64, types.Float64, types.Complex64:
			wire.width = 8
		case types.Complex128:
			wire.width = 16
		default:
			return nil
		}
		if wire.width > maxWidth {
			return nil
		}
	case *types.Array:
		item := fixedLayout(base.Elem(), maxWidth)
		if item == nil || base.Len() < 0 || item.width != 0 && base.Len() > maxWidth/item.width {
			return nil
		}
		wire.width = base.Len() * item.width
		wire.elem = item
	case *types.Struct:
		for i := 0; i < base.NumFields(); i++ {
			field := base.Field(i)
			value := fixedLayout(field.Type(), maxWidth)
			if value == nil || value.width > maxWidth-wire.width {
				return nil
			}
			wire.fields = append(wire.fields, fieldLayout{field, wire.width, value})
			wire.width += value.width
		}
	default:
		return nil
	}
	return wire
}
