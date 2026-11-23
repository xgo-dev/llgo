package build

import (
	"slices"
	"strings"

	llssa "github.com/xgo-dev/llgo/ssa"
	"github.com/xgo-dev/llvm"
)

// The Darwin linker creates a new target machine for LTO. Carry the selected
// Go ISA in the bitcode so it reaches that machine as function attributes.
func applyDarwinARM64LTOFeatures(ltoEnabled bool, target *llssa.Target, mod llvm.Module) {
	if !ltoEnabled || target.GOOS != "darwin" || target.GOARCH != "arm64" ||
		target.Target != "" || target.GOARM64 == "" || target.GOARM64 == "v8.0" {
		return
	}

	const attrName = "target-features"
	features := target.Spec().Features
	defaultFeatures := strings.Split(features, ",")
	lc := mod.Context()
	defaultAttr := lc.CreateStringAttribute(attrName, features)
	for fn := mod.FirstFunction(); !fn.IsNil(); fn = llvm.NextFunction(fn) {
		if fn.IsDeclaration() {
			continue
		}
		attr := fn.GetStringAttributeAtIndex(-1, attrName)
		if attr.IsNil() || attr.GetStringValue() == "" {
			fn.AddFunctionAttr(defaultAttr)
			continue
		}
		explicit := strings.Split(attr.GetStringValue(), ",")
		merged := make([]string, 0, len(defaultFeatures)+len(explicit))
		for _, feature := range defaultFeatures {
			name := feature[1:]
			if !slices.Contains(explicit, "+"+name) && !slices.Contains(explicit, "-"+name) {
				merged = append(merged, feature)
			}
		}
		merged = append(merged, explicit...)
		fn.AddFunctionAttr(lc.CreateStringAttribute(attrName, strings.Join(merged, ",")))
	}
}
