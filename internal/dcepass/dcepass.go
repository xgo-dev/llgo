// Package dcepass rewrites ABI metadata so link-time dead code elimination can
// drop method bodies that are no longer referenced by live method slots.
package dcepass

import (
	"fmt"
	"strings"

	"github.com/xgo-dev/llvm"
)

// EmitStrongTypeOverrides emits method-pruned strong ABI type symbols into dst.
//
// srcMods contain the original package modules. For each constant ABI type
// global with a method array, this function creates a same-name strong global in
// dst and clears IFn/TFn for method slots not listed in liveSlots[typeName].
func EmitStrongTypeOverrides(dst llvm.Module, srcMods []llvm.Module, liveSlots map[string][]int) error {
	if dst.IsNil() {
		return fmt.Errorf("destination module is nil")
	}
	emitted := make(map[string]bool)
	emitter := newOverrideEmitter(dst)
	for _, src := range srcMods {
		if src.IsNil() {
			continue
		}
		for g := src.FirstGlobal(); !g.IsNil(); g = llvm.NextGlobal(g) {
			name := g.Name()
			if name == "" || emitted[name] || !g.IsGlobalConstant() {
				continue
			}
			init := g.Initializer()
			if init.IsNil() {
				continue
			}
			methodsVal, elemTy, ok := methodArray(init)
			if !ok || methodsVal.OperandsCount() == 0 {
				continue
			}
			if err := emitter.emitTypeOverride(g, methodsVal, elemTy, liveSlotSet(liveSlots[name])); err != nil {
				return fmt.Errorf("emit override %q: %w", name, err)
			}
			emitted[name] = true
		}
	}
	return nil
}

type overrideEmitter struct {
	dst    llvm.Module
	values map[llvm.Value]llvm.Value
}

func newOverrideEmitter(dst llvm.Module) *overrideEmitter {
	return &overrideEmitter{
		dst:    dst,
		values: make(map[llvm.Value]llvm.Value),
	}
}

func (e *overrideEmitter) emitTypeOverride(srcType, methodsVal llvm.Value, elemTy llvm.Type, keepIdx map[int]bool) error {
	init := srcType.Initializer()
	dstType, err := e.ensureOverrideGlobal(srcType)
	if err != nil {
		return err
	}
	e.values[srcType] = dstType

	fieldCount := init.OperandsCount()
	fields := make([]llvm.Value, fieldCount)
	for i := 0; i < fieldCount-1; i++ {
		clone, err := e.cloneConst(init.Operand(i))
		if err != nil {
			return err
		}
		fields[i] = clone
	}

	elemFields := elemTy.StructElementTypes()
	if len(elemFields) < 4 {
		return fmt.Errorf("method element type has %d fields", len(elemFields))
	}
	zeroPtr := llvm.ConstPointerNull(elemFields[2])
	methodCount := methodsVal.OperandsCount()
	methods := make([]llvm.Value, methodCount)
	for i := 0; i < methodCount; i++ {
		orig := methodsVal.Operand(i)
		if keepIdx[i] {
			clone, err := e.cloneConst(orig)
			if err != nil {
				return err
			}
			methods[i] = clone
			continue
		}
		nameField, err := e.cloneConst(orig.Operand(0))
		if err != nil {
			return err
		}
		mtypField, err := e.cloneConst(orig.Operand(1))
		if err != nil {
			return err
		}
		methods[i] = llvm.ConstNamedStruct(elemTy, []llvm.Value{nameField, mtypField, zeroPtr, zeroPtr})
	}
	fields[fieldCount-1] = llvm.ConstArray(elemTy, methods)

	dstType.SetInitializer(constStructOfType(init.Type(), fields))
	dstType.SetGlobalConstant(true)
	dstType.SetLinkage(llvm.ExternalLinkage)
	copyGlobalAttrs(dstType, srcType)
	return nil
}

func (e *overrideEmitter) ensureOverrideGlobal(src llvm.Value) (llvm.Value, error) {
	name := src.Name()
	if name == "" {
		return llvm.Value{}, fmt.Errorf("type global has empty name")
	}
	dst := e.dst.NamedGlobal(name)
	if dst.IsNil() {
		dst = llvm.AddGlobal(e.dst, src.GlobalValueType(), name)
	}
	e.values[src] = dst
	return dst, nil
}

func (e *overrideEmitter) cloneConst(v llvm.Value) (llvm.Value, error) {
	if v.IsNil() {
		return llvm.Value{}, nil
	}
	if mapped, ok := e.values[v]; ok {
		return mapped, nil
	}
	if gv := v.IsAGlobalValue(); !gv.IsNil() {
		clone, err := e.cloneGlobalValue(gv)
		if err != nil {
			return llvm.Value{}, err
		}
		e.values[v] = clone
		return clone, nil
	}
	if !v.IsAConstantStruct().IsNil() {
		ops, err := e.cloneOperands(v)
		if err != nil {
			return llvm.Value{}, err
		}
		clone := constStructOfType(v.Type(), ops)
		e.values[v] = clone
		return clone, nil
	}
	return v, nil
}

func (e *overrideEmitter) cloneOperands(v llvm.Value) ([]llvm.Value, error) {
	n := v.OperandsCount()
	ops := make([]llvm.Value, n)
	for i := 0; i < n; i++ {
		clone, err := e.cloneConst(v.Operand(i))
		if err != nil {
			return nil, err
		}
		ops[i] = clone
	}
	return ops, nil
}

func (e *overrideEmitter) cloneGlobalValue(v llvm.Value) (llvm.Value, error) {
	if mapped, ok := e.values[v]; ok {
		return mapped, nil
	}
	if fn := v.IsAFunction(); !fn.IsNil() {
		name := fn.Name()
		if name == "" {
			return llvm.Value{}, fmt.Errorf("function ref has empty name")
		}
		dstFn := e.dst.NamedFunction(name)
		if dstFn.IsNil() {
			dstFn = llvm.AddFunction(e.dst, name, fn.GlobalValueType())
		}
		e.values[v] = dstFn
		return dstFn, nil
	}
	if gv := v.IsAGlobalVariable(); !gv.IsNil() {
		clone, err := e.cloneGlobalVariable(gv)
		if err != nil {
			return llvm.Value{}, err
		}
		e.values[v] = clone
		return clone, nil
	}
	name := v.Name()
	if name == "" {
		return llvm.Value{}, fmt.Errorf("unsupported unnamed global ref")
	}
	dstG := e.dst.NamedGlobal(name)
	if dstG.IsNil() {
		dstG = llvm.AddGlobal(e.dst, v.GlobalValueType(), name)
		dstG.SetLinkage(llvm.ExternalLinkage)
	}
	e.values[v] = dstG
	return dstG, nil
}

func (e *overrideEmitter) cloneGlobalVariable(src llvm.Value) (llvm.Value, error) {
	if mapped, ok := e.values[src]; ok {
		return mapped, nil
	}
	name := src.Name()
	local := name == "" || isLocalLinkage(src.Linkage())
	if !local {
		dst := e.dst.NamedGlobal(name)
		if dst.IsNil() {
			dst = llvm.AddGlobal(e.dst, src.GlobalValueType(), name)
			dst.SetLinkage(llvm.ExternalLinkage)
		}
		e.values[src] = dst
		return dst, nil
	}

	dst := llvm.AddGlobal(e.dst, src.GlobalValueType(), "")
	e.values[src] = dst
	copyGlobalAttrs(dst, src)
	dst.SetLinkage(src.Linkage())
	dst.SetGlobalConstant(src.IsGlobalConstant())
	if init := src.Initializer(); !init.IsNil() {
		cloneInit, err := e.cloneConst(init)
		if err != nil {
			return llvm.Value{}, err
		}
		dst.SetInitializer(cloneInit)
	}
	return dst, nil
}

func methodArray(init llvm.Value) (llvm.Value, llvm.Type, bool) {
	fieldCount := init.OperandsCount()
	if fieldCount == 0 {
		return llvm.Value{}, llvm.Type{}, false
	}
	methodsVal := init.Operand(fieldCount - 1)
	if methodsVal.Type().TypeKind() != llvm.ArrayTypeKind {
		return llvm.Value{}, llvm.Type{}, false
	}
	elemTy := methodsVal.Type().ElementType()
	if elemTy.TypeKind() != llvm.StructTypeKind {
		return llvm.Value{}, llvm.Type{}, false
	}
	if elemTy.StructElementTypesCount() != 4 {
		return llvm.Value{}, llvm.Type{}, false
	}
	if !strings.Contains(elemTy.StructName(), "runtime/abi.Method") {
		return llvm.Value{}, llvm.Type{}, false
	}
	return methodsVal, elemTy, true
}

func liveSlotSet(slots []int) map[int]bool {
	out := make(map[int]bool, len(slots))
	for _, slot := range slots {
		out[slot] = true
	}
	return out
}

func copyGlobalAttrs(dst, src llvm.Value) {
	dst.SetVisibility(src.Visibility())
	if sec := src.Section(); sec != "" {
		dst.SetSection(sec)
	}
	dst.SetThreadLocal(src.IsThreadLocal())
	if align := src.Alignment(); align > 0 {
		dst.SetAlignment(align)
	}
}

func isLocalLinkage(linkage llvm.Linkage) bool {
	return linkage == llvm.PrivateLinkage || linkage == llvm.InternalLinkage
}

func constStructOfType(typ llvm.Type, fields []llvm.Value) llvm.Value {
	if typ.StructName() != "" {
		return llvm.ConstNamedStruct(typ, fields)
	}
	return llvm.ConstStruct(fields, typ.IsStructPacked())
}
