package llvmfmv

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xgo-dev/llvm"
)

const RegisterOwnerAttribute = attributePrefix + "register-owner"

// A fresh provider emits eligible Go definitions in this invocation. Cached
// providers advertise the actual companion symbols in their archive manifest.
type RegisterProvider struct {
	Fresh   bool
	Symbols []string
}

func (p RegisterProvider) provides(name string) bool {
	if p.Fresh {
		return true
	}
	_, ok := slices.BinarySearch(p.Symbols, name)
	return ok
}

func exportName(original string, sig wideSignature) string {
	return original + sig.suffix() + "_go"
}

func setRegisterABI(entry llvm.Value, sig wideSignature, features string) {
	ctx := entry.GlobalParent().Context()
	features = strings.TrimPrefix(features+",+avx,+avx2", ",")
	if sig.width == 512 {
		features += ",+avx512f"
	}
	entry.AddFunctionAttr(ctx.CreateStringAttribute("target-features", features))
	if strings.Contains(entry.GlobalParent().Target(), "windows") {
		entry.SetFunctionCallConv(llvm.CallConv(80))
	}
}

// Export the original typed body before public Go calls acquire their pointer
// carriers. Internal calls in the copy still go through normal ABI lowering.
func (s WideCalls) Export(m llvm.Module, owner string) ([]string, error) {
	var exported []string
	var source *sourceInfo
	for f := m.FirstFunction(); !f.IsNil(); f = llvm.NextFunction(f) {
		sig, ok := s[f.Name()]
		if !ok || f.IsDeclaration() || stringAttribute(f, RegisterOwnerAttribute) != owner || f.Linkage() != llvm.ExternalLinkage {
			continue
		}
		name := exportName(f.Name(), sig)
		if !m.NamedValue(name).IsNil() {
			return nil, fmt.Errorf("SIMD register export collision: %s", name)
		}
		entry := f.CloneFunction()
		entry.SetName(name)
		for _, attr := range []string{EntryAttribute, InlineEntryAttribute, wideOrigin, RegisterOwnerAttribute} {
			entry.RemoveStringAttributeAtIndex(-1, attr)
		}
		setRegisterABI(entry, sig, stringAttribute(f, "target-features"))
		if source == nil {
			source = newSourceInfo(m)
		}
		source.clone(f, entry)
		exported = append(exported, name)
	}
	slices.Sort(exported)
	return exported, nil
}

func importRegisterEntry(m llvm.Module, target llvm.Value, sig wideSignature) llvm.Value {
	name := exportName(stringAttribute(target, wideOrigin), sig)
	entry := m.NamedFunction(name)
	if entry.IsNil() {
		entry = llvm.AddFunction(m, name, sig.typ)
		setRegisterABI(entry, sig, "")
		offset := 0
		if sig.typ.ReturnType().TypeKind() == llvm.VectorTypeKind {
			offset = 1
		}
		for i, t := range sig.typ.ParamTypes() {
			if t.TypeKind() != llvm.VectorTypeKind {
				for _, attr := range target.GetAttributesAtIndex(i + offset + 1) {
					entry.AddAttributeAtIndex(i+1, attr)
				}
			}
		}
		if offset == 0 {
			for _, attr := range target.GetAttributesAtIndex(0) {
				entry.AddAttributeAtIndex(0, attr)
			}
		}
	}
	return entry
}
