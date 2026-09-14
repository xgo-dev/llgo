package funcattrs

import (
	"go/types"
	"math/big"

	"github.com/xgo-dev/llvm"
)

// Apply attaches value attributes while parameters still follow the Go signature.
func Apply(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment, intBits int) error {
	if err := Validate(attrs, sig, intBits, false); err != nil {
		return err
	}
	for _, attr := range attrs {
		index := 1 + environment
		if attr.Target.Scope == Result {
			index = 0
		} else if attr.Target.Scope == Parameter {
			index += attr.Target.Index
			if sig.Recv() != nil {
				index++
			}
		}
		var typ llvm.Type
		if index == 0 {
			typ = fn.GlobalValueType().ReturnType()
		} else if index <= fn.ParamsCount() {
			typ = fn.Param(index - 1).Type()
		} else {
			return attr.Error("selected parameter has no LLVM value")
		}
		if attr.Name == "nonnull" {
			if typ.TypeKind() != llvm.PointerTypeKind {
				return attr.Error("selected value is not an LLVM pointer")
			}
			fn.AddAttributeAtIndex(index, ctx.CreateEnumAttribute(llvm.AttributeKindID("nonnull"), 0))
			continue
		}
		selected, _ := ResolveTarget(sig, attr.Target)
		// LLVM has one range per value. Intersect the two source spellings first.
		if attr.Name == "nonnegative" {
			hasRange := false
			for _, other := range attrs {
				if other.Target.Equal(attr.Target) && other.Name == "range" {
					hasRange = true
				}
			}
			if hasRange {
				continue
			}
		} else if attr.Range.Lower.Sign() < 0 {
			for _, other := range attrs {
				if other.Target.Equal(attr.Target) && other.Name == "nonnegative" {
					attr.Range = &RangeBounds{Lower: new(big.Int), Upper: attr.Range.Upper}
				}
			}
		}
		bits, bounds, full, err := IntegerRange(attr, selected, intBits)
		if err != nil {
			return err
		}
		if typ.TypeKind() != llvm.IntegerTypeKind || typ.IntTypeWidth() != bits {
			return attr.Error("selected integer does not have its source width")
		}
		if !full {
			fn.AddAttributeAtIndex(index, ctx.CreateConstantRangeAttribute(llvm.AttributeKindID("range"), bits, bounds[:1], bounds[1:]))
		}
	}
	return nil
}

// CopyValueAttributes is used only when ABI conversion preserves the whole value.
func CopyValueAttributes(from, to llvm.Value, old, new int) {
	for _, name := range []string{"nonnull", "range"} {
		if attr := from.GetEnumAttributeAtIndex(old, llvm.AttributeKindID(name)); !attr.IsNil() {
			to.AddAttributeAtIndex(new, attr)
		}
	}
}
