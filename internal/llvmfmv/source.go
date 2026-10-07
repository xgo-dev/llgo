package llvmfmv

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/xgo-dev/llvm"
)

type sourceInfo struct {
	mod       llvm.Module
	functions map[string][][]llvm.Value
	lines     map[string][][]llvm.Value
	usedIDs   map[uint64]bool
}

// Runtime tables intentionally use strings, so LLVM's generic function clone
// cannot update them. Keep these layouts in sync with ssa/funcinfo.go:
// funcinfo = {version, symbol, Go name, file, line, column[, flags]}
// pcline   = {version, id, symbol, file, line, column}
func newSourceInfo(mod llvm.Module) *sourceInfo {
	s := &sourceInfo{mod: mod, functions: make(map[string][][]llvm.Value),
		lines: make(map[string][][]llvm.Value), usedIDs: make(map[uint64]bool)}
	for _, row := range mod.NamedMetadataOperands("llgo.funcinfo") {
		fields := row.MDNodeOperands()
		if len(fields) >= 6 && !fields[1].IsNil() && fields[1].IsAMDString() {
			name := fields[1].MDString()
			s.functions[name] = append(s.functions[name], fields)
		}
	}
	for _, row := range mod.NamedMetadataOperands("llgo.pcline") {
		fields := row.MDNodeOperands()
		if len(fields) != 6 || fields[1].IsNil() || fields[1].IsAConstantInt().IsNil() {
			continue
		}
		s.usedIDs[fields[1].ZExtValue()] = true
		if !fields[2].IsNil() && fields[2].IsAMDString() {
			name := fields[2].MDString()
			s.lines[name] = append(s.lines[name], fields)
		}
	}
	return s
}

func metadataFields(fields []llvm.Value) []llvm.Metadata {
	mds := make([]llvm.Metadata, len(fields))
	for i, value := range fields {
		if !value.IsNil() {
			mds[i] = value.AsMetadata()
		}
	}
	return mds
}

func (s *sourceInfo) clone(fn, variant llvm.Value) {
	ctx := s.mod.Context()
	name := variant.Name()
	// LLVM's CloneFunction gives the implementation its own subprogram and
	// remaps local debug scopes. Preserve the Go display name and source paths.
	if sp := variant.Subprogram(); !sp.IsNil() {
		sp.SetSubprogramLinkageName(name)
	}
	for _, fields := range s.functions[fn.Name()] {
		mds := metadataFields(fields)
		mds[1] = ctx.MDString(name)
		s.mod.AddNamedMetadataOperand("llgo.funcinfo", ctx.MDNode(mds))
	}
	var replacements []string
	for _, fields := range s.lines[fn.Name()] {
		old := fields[1].ZExtValue()
		oldHex := fmt.Sprintf("%016x", old)
		hash := fnv.New64a()
		hash.Write([]byte(name + ":" + oldHex))
		id := hash.Sum64()
		for id == 0 || s.usedIDs[id] {
			id++
		}
		s.usedIDs[id] = true
		mds := metadataFields(fields)
		mds[1] = llvm.ConstInt(fields[1].Type(), id, false).AsMetadata()
		mds[2] = ctx.MDString(name)
		s.mod.AddNamedMetadataOperand("llgo.pcline", ctx.MDNode(mds))
		replacements = append(replacements, oldHex, fmt.Sprintf("%016x", id))
	}
	if len(replacements) == 0 {
		return
	}
	rewrite := strings.NewReplacer(replacements...)
	for bb := variant.FirstBasicBlock(); !bb.IsNil(); bb = llvm.NextBasicBlock(bb) {
		for inst := bb.FirstInstruction(); !inst.IsNil(); inst = llvm.NextInstruction(inst) {
			if inst.IsACallInst().IsNil() && inst.IsAInvokeInst().IsNil() {
				continue
			}
			callee := inst.CalledValue()
			if callee.IsAInlineAsm().IsNil() {
				continue
			}
			asm := callee.InlineAsmInfo()
			if !strings.Contains(asm.Assembly, "__llgo_pcsite_") {
				continue
			}
			cloned := llvm.InlineAsm(asm.Type, rewrite.Replace(asm.Assembly), asm.Constraints,
				asm.HasSideEffects, asm.IsAlignStack, asm.Dialect, asm.CanThrow)
			inst.SetOperand(inst.OperandsCount()-1, cloned)
		}
	}
}
