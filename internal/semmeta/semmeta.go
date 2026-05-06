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

package semmeta

import (
	"fmt"
	"strings"

	"github.com/goplus/llvm"
)

const (
	UseIfaceMetadata       = "llgo.useiface"
	UseIfaceMethodMetadata = "llgo.useifacemethod"
	InterfaceInfoMetadata  = "llgo.interfaceinfo"
	MethodInfoMetadata     = "llgo.methodinfo"
	UseNamedMethodMetadata = "llgo.usenamedmethod"
	ReflectMethodMetadata  = "llgo.reflectmethod"
)

// Symbol is the metadata-level identity used by link-time semantic method pruning.
// In current LLGo this is the symbol spelling emitted by ssa/cl into LLVM IR.
type Symbol string

// MethodSig identifies one interface-visible method slot by normalized method name
// plus method type symbol.
//
// Name follows the same rule used by current LLGo metadata emission:
// exported methods keep their declared name, while unexported methods are
// package-qualified so whole-program matching can distinguish private methods
// from different packages.
type MethodSig struct {
	Name  string
	MType Symbol
}

// IfaceMethodUse records one interface method demand produced by a reachable
// owner. Target is the interface type symbol, and Sig identifies the demanded
// interface method slot on that interface.
type IfaceMethodUse struct {
	Target Symbol
	Sig    MethodSig
}

// MethodSlot describes one candidate method-table slot on a concrete type.
//
// Index must match the final abi.Method slot index. If a slot is later judged
// live by whole-program analysis, its MType/IFn/TFn are pulled back into the
// ordinary reachability graph.
type MethodSlot struct {
	Index int
	Sig   MethodSig
	IFn   Symbol
	TFn   Symbol
}

// ModuleInfo is the per-module semantic metadata view used by link-time
// semantic method pruning.
//
// Current ssa/cl emission writes those semantic facts one row at a time into
// llgo.xxx named metadata; Read folds the rows of one llvm.Module back into
// this grouped form so later stages can merge them under a whole-program view.
type ModuleInfo struct {
	// InterfaceInfo records the complete method set of an interface:
	// interface symbol -> []MethodSig.
	InterfaceInfo map[Symbol][]MethodSig
	// UseIface records concrete types that have entered the interface dispatch
	// semantic domain:
	// owner symbol -> []concrete type symbol.
	UseIface map[Symbol][]Symbol
	// UseIfaceMethod records interface method demands:
	// owner symbol -> []IfaceMethodUse.
	UseIfaceMethod map[Symbol][]IfaceMethodUse
	// MethodInfo records the candidate method-table slots of a concrete type:
	// concrete type symbol -> []MethodSlot.
	MethodInfo map[Symbol][]MethodSlot
	// UseNamedMethod records exact MethodByName-like constant-name lookups:
	// owner symbol -> []normalized method name.
	UseNamedMethod map[Symbol][]string
	// ReflectMethod marks owners that force conservative reflection handling.
	ReflectMethod map[Symbol]struct{}
}

// Emitter owns the current llgo.xxx metadata wire format.
//
// The writer is intentionally row-oriented because LLVM named metadata is
// appended one operand at a time. Read performs the inverse step and folds
// those rows back into ModuleInfo.
type Emitter struct {
	mod  llvm.Module
	ctx  llvm.Context
	seen map[string]struct{}
}

func NewEmitter(mod llvm.Module) *Emitter {
	return &Emitter{
		mod:  mod,
		ctx:  mod.Context(),
		seen: make(map[string]struct{}),
	}
}

func (e *Emitter) AddUseIface(owner, target Symbol) {
	e.add(
		UseIfaceMetadata,
		metadataKey(string(owner), string(target)),
		e.mdString(string(owner)),
		e.mdString(string(target)),
	)
}

func (e *Emitter) AddUseIfaceMethod(owner Symbol, use IfaceMethodUse) {
	e.add(
		UseIfaceMethodMetadata,
		metadataKey(string(owner), string(use.Target), use.Sig.Name, string(use.Sig.MType)),
		e.mdString(string(owner)),
		e.mdString(string(use.Target)),
		e.mdString(use.Sig.Name),
		e.mdString(string(use.Sig.MType)),
	)
}

func (e *Emitter) AddInterfaceInfo(target Symbol, methods []MethodSig) {
	if len(methods) == 0 {
		return
	}
	for _, method := range methods {
		e.add(
			InterfaceInfoMetadata,
			metadataKey(string(target), method.Name, string(method.MType)),
			e.mdString(string(target)),
			e.mdString(method.Name),
			e.mdString(string(method.MType)),
		)
	}
}

func (e *Emitter) AddMethodInfo(typeSym Symbol, slots []MethodSlot) {
	if len(slots) == 0 {
		return
	}
	for _, slot := range slots {
		e.add(
			MethodInfoMetadata,
			metadataKey(string(typeSym), fmt.Sprint(slot.Index), slot.Sig.Name, string(slot.Sig.MType), string(slot.IFn), string(slot.TFn)),
			e.mdString(string(typeSym)),
			e.mdInt32(slot.Index),
			e.mdString(slot.Sig.Name),
			e.mdString(string(slot.Sig.MType)),
			e.mdString(string(slot.IFn)),
			e.mdString(string(slot.TFn)),
		)
	}
}

func (e *Emitter) AddUseNamedMethod(owner Symbol, name string) {
	e.add(
		UseNamedMethodMetadata,
		metadataKey(string(owner), name),
		e.mdString(string(owner)),
		e.mdString(name),
	)
}

func (e *Emitter) AddReflectMethod(owner Symbol) {
	e.add(
		ReflectMethodMetadata,
		string(owner),
		e.mdString(string(owner)),
	)
}

// Read reconstructs the semantic metadata facts carried by one llvm.Module.
func Read(mod llvm.Module) ModuleInfo {
	info := ModuleInfo{
		InterfaceInfo:  make(map[Symbol][]MethodSig),
		UseIface:       make(map[Symbol][]Symbol),
		UseIfaceMethod: make(map[Symbol][]IfaceMethodUse),
		MethodInfo:     make(map[Symbol][]MethodSlot),
		UseNamedMethod: make(map[Symbol][]string),
		ReflectMethod:  make(map[Symbol]struct{}),
	}
	readUseIface(mod, &info)
	readUseIfaceMethod(mod, &info)
	readInterfaceInfo(mod, &info)
	readMethodInfo(mod, &info)
	readUseNamedMethod(mod, &info)
	readReflectMethod(mod, &info)
	return info
}

func readUseIface(mod llvm.Module, info *ModuleInfo) {
	for _, row := range mod.NamedMetadataOperands(UseIfaceMetadata) {
		fields := row.MDNodeOperands()
		owner := fieldString(fields[0])
		target := fieldString(fields[1])
		info.UseIface[Symbol(owner)] = append(info.UseIface[Symbol(owner)], Symbol(target))
	}
}

func readUseIfaceMethod(mod llvm.Module, info *ModuleInfo) {
	for _, row := range mod.NamedMetadataOperands(UseIfaceMethodMetadata) {
		fields := row.MDNodeOperands()
		owner := fieldString(fields[0])
		target := fieldString(fields[1])
		name := fieldString(fields[2])
		mtyp := fieldString(fields[3])
		info.UseIfaceMethod[Symbol(owner)] = append(info.UseIfaceMethod[Symbol(owner)], IfaceMethodUse{
			Target: Symbol(target),
			Sig: MethodSig{
				Name:  name,
				MType: Symbol(mtyp),
			},
		})
	}
}

func readInterfaceInfo(mod llvm.Module, info *ModuleInfo) {
	for _, row := range mod.NamedMetadataOperands(InterfaceInfoMetadata) {
		fields := row.MDNodeOperands()
		target := fieldString(fields[0])
		name := fieldString(fields[1])
		mtyp := fieldString(fields[2])
		info.InterfaceInfo[Symbol(target)] = append(info.InterfaceInfo[Symbol(target)], MethodSig{Name: name, MType: Symbol(mtyp)})
	}
}

func readMethodInfo(mod llvm.Module, info *ModuleInfo) {
	for _, row := range mod.NamedMetadataOperands(MethodInfoMetadata) {
		fields := row.MDNodeOperands()
		target := Symbol(fieldString(fields[0]))
		info.MethodInfo[target] = append(info.MethodInfo[target], MethodSlot{
			Index: fieldInt(fields[1]),
			Sig: MethodSig{
				Name:  fieldString(fields[2]),
				MType: Symbol(fieldString(fields[3])),
			},
			IFn: Symbol(fieldString(fields[4])),
			TFn: Symbol(fieldString(fields[5])),
		})
	}
}

func readUseNamedMethod(mod llvm.Module, info *ModuleInfo) {
	for _, row := range mod.NamedMetadataOperands(UseNamedMethodMetadata) {
		fields := row.MDNodeOperands()
		owner := fieldString(fields[0])
		name := fieldString(fields[1])
		info.UseNamedMethod[Symbol(owner)] = append(info.UseNamedMethod[Symbol(owner)], name)
	}
}

func readReflectMethod(mod llvm.Module, info *ModuleInfo) {
	for _, row := range mod.NamedMetadataOperands(ReflectMethodMetadata) {
		fields := row.MDNodeOperands()
		owner := fieldString(fields[0])
		info.ReflectMethod[Symbol(owner)] = struct{}{}
	}
}

func fieldString(field llvm.Value) string {
	return field.MDString()
}

func fieldInt(field llvm.Value) int {
	return int(field.ZExtValue())
}

func (e *Emitter) add(table, key string, fields ...llvm.Metadata) {
	fullKey := table + ":" + key
	if _, ok := e.seen[fullKey]; ok {
		return
	}
	e.seen[fullKey] = struct{}{}
	e.mod.AddNamedMetadataOperand(table, e.ctx.MDNode(fields))
}

func metadataKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func (e *Emitter) mdString(s string) llvm.Metadata {
	return e.ctx.MDString(s)
}

func (e *Emitter) mdInt32(i int) llvm.Metadata {
	return llvm.ConstInt(e.ctx.Int32Type(), uint64(i), false).ConstantAsMetadata()
}
