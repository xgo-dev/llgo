package abi

import (
	"go/token"
	"go/types"

	runtimeabi "github.com/goplus/llgo/runtime/abi"
)

func SwissMapGroupType(t *types.Map, sizes types.Sizes) types.Type {
	key := t.Key()
	elem := t.Elem()
	if sizes.Sizeof(key) > runtimeabi.MapMaxKeyBytes {
		key = types.NewPointer(key)
	}
	if sizes.Sizeof(elem) > runtimeabi.MapMaxElemBytes {
		elem = types.NewPointer(elem)
	}
	slot := types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "key", key, false),
		types.NewField(token.NoPos, nil, "elem", elem, false),
	}, nil)
	slots := types.NewArray(slot, runtimeabi.MapGroupSlots)
	return types.NewStruct([]*types.Var{
		types.NewField(token.NoPos, nil, "ctrl", types.Typ[types.Uint64], false),
		types.NewField(token.NoPos, nil, "slots", slots, false),
	}, nil)
}

func SwissMapTypeFlags(t *types.Map, sizes types.Sizes) (flags int) {
	if needkeyupdate(t.Key()) {
		flags |= runtimeabi.MapNeedKeyUpdate
	}
	if hashMightPanic(t.Key()) {
		flags |= runtimeabi.MapHashMightPanic
	}
	if sizes.Sizeof(t.Key()) > runtimeabi.MapMaxKeyBytes {
		flags |= runtimeabi.MapIndirectKey
	}
	if sizes.Sizeof(t.Elem()) > runtimeabi.MapMaxElemBytes {
		flags |= runtimeabi.MapIndirectElem
	}
	return
}

func SwissMapSlotLayout(group types.Type, sizes types.Sizes) (slotSize, elemOff uintptr) {
	groupStruct := group.(*types.Struct)
	slots := groupStruct.Field(1).Type().(*types.Array)
	slot := slots.Elem().(*types.Struct)
	offsets := sizes.Offsetsof([]*types.Var{slot.Field(0), slot.Field(1)})
	return uintptr(sizes.Sizeof(slot)), uintptr(offsets[1])
}
