package ssa

import (
	"encoding/json"
	"go/types"
	"sort"
	"strings"

	"github.com/xgo-dev/llgo/internal/funcattrs"
	"github.com/xgo-dev/llvm"
)

func (p Program) SetFunctionAttributes(name string, attrs []funcattrs.Attribute) error {
	if len(attrs) == 0 {
		return nil
	}
	p.packageSyntax.mu.Lock()
	defer p.packageSyntax.mu.Unlock()
	merged, err := funcattrs.Merge(p.packageSyntax.functionAttributes[name], attrs)
	if err == nil {
		p.packageSyntax.functionAttributes[name] = merged
	}
	return err
}

// CheckAttributeInstrumentation rejects known compiler-inserted effects that
// v1 cannot reconcile with a source contract. No package receives an exemption.
func (f Function) CheckAttributeInstrumentation(reason string, names ...string) {
	metadata := f.impl.GetStringAttributeAtIndex(-1, funcattrs.Metadata)
	if metadata.IsNil() {
		return
	}
	var attrs []funcattrs.Attribute
	if err := json.Unmarshal([]byte(metadata.GetStringValue()), &attrs); err != nil {
		panic(err)
	}
	for _, a := range attrs {
		for _, name := range names {
			if a.Name == name {
				panic(a.Error("%s with %s is not supported in v1", name, reason))
			}
		}
	}
}

// SetFunctionAttributeOrigin binds a concrete generic symbol to its declaration.
// Logical indices are preserved, and Apply validates the instantiated types.
func (p Program) SetFunctionAttributeOrigin(instance, origin string) {
	if instance == origin {
		return
	}
	p.packageSyntax.mu.Lock()
	p.packageSyntax.attributeOrigins[instance] = origin
	p.packageSyntax.mu.Unlock()
}

func (p Program) functionAttributes(name string) ([]funcattrs.Attribute, error) {
	p.packageSyntax.mu.RLock()
	defer p.packageSyntax.mu.RUnlock()
	data := p.packageSyntax
	if origin, ok := data.attributeOrigins[name]; ok {
		name = origin
	}
	resolve := func(s string) string {
		// Linknames are symbol mappings, not an instruction to follow arbitrary
		// alias chains. This mirrors the frontend's one-step resolution.
		if link, ok := data.linknames[s]; ok {
			return strings.TrimPrefix(strings.TrimPrefix(link, "C."), "stdcall.")
		}
		return s
	}
	resolved := resolve(name)
	var owners []string
	for source := range data.functionAttributes {
		if source == name || resolve(source) == resolved {
			owners = append(owners, source)
		}
	}
	sort.Strings(owners)
	var sets [][]funcattrs.Attribute
	for _, owner := range owners {
		sets = append(sets, data.functionAttributes[owner])
	}
	return funcattrs.Merge(sets...)
}

func (p Program) applyFunctionAttributes(fn llvm.Value, name string, sig *types.Signature, hasEnvironment bool) {
	attrs, err := p.functionAttributes(name)
	if err != nil {
		panic(err)
	}
	offset := 0
	if hasEnvironment {
		offset = 1
	}
	if err = funcattrs.Apply(p.ctx, fn, sig, attrs, offset, p.Int().ll.IntTypeWidth()); err != nil {
		panic(err)
	}
}
