package funcattrs

import (
	"encoding/json"
	"go/types"

	"github.com/xgo-dev/llvm"
)

// Metadata carries source contracts through bitcode. Backend decisions and
// physical parameter numbers are stored separately in a short-lived value plan.
const Metadata = "llgo.source.attributes.v1"

// ResultPathResolver locates one Go result in LLVM's multi-result value,
// including target-specific padding wrappers. It is not source field syntax.
type ResultPathResolver func(index int) []int

func Apply(ctx llvm.Context, fn llvm.Value, sig *types.Signature, attrs []Attribute, environment, intBits int, resolver ...ResultPathResolver) error {
	if len(attrs) == 0 {
		return nil
	}
	if err := Validate(attrs, sig, intBits, false); err != nil {
		return err
	}
	var resolve ResultPathResolver
	if len(resolver) != 0 {
		resolve = resolver[0]
	}
	if err := prepareValueContracts(ctx, fn, sig, attrs, environment, intBits, resolve); err != nil {
		return err
	}
	if err := ApplyEffects(ctx, fn, sig, attrs, environment, intBits); err != nil {
		return err
	}
	data, err := json.Marshal(attrs)
	if err != nil {
		return err
	}
	fn.AddFunctionAttr(ctx.CreateStringAttribute(Metadata, string(data)))
	return nil
}

// Representation records what happens to a previous LLVM value. Source value
// facts already refer to logical SSA values and follow their ABI replacements;
// this map is only needed to retain sound native attributes and effect summaries.
type Representation uint8

const (
	Direct Representation = iota
	Omitted
	Fragments
	Indirect
)

type ABIValue struct {
	Kind    Representation
	Indices []int // LLVM attribute indices: 0 = return, 1.. = arguments
}

type ABIMapping struct {
	Result ABIValue
	Params []ABIValue // indexed by the previous LLVM parameter position
}

func DirectValue(index int) ABIValue { return ABIValue{Kind: Direct, Indices: []int{index}} }

var valueAttributes = []string{"nonnull", "align", "returned", "range"}

// RemapFunction runs after blanket function-attribute copying, so generated
// transport effects can widen the copied summary. A native value attribute is
// preserved only on an unchanged direct value; logical facts survive in the body.
func RemapFunction(from, to llvm.Value, m ABIMapping) error {
	remapValues(m, from.GetEnumAttributeAtIndex, to.AddAttributeAtIndex, to.RemoveEnumAttributeAtIndex)
	RemapFunctionEffects(from, to, m)
	return nil
}

func RemapCall(from, to llvm.Value, m ABIMapping) error {
	// ABI call replacements are new instructions and do not blanket-copy
	// returned attributes, so dropping an incompatible one needs no removal.
	remapValues(m, from.GetCallSiteEnumAttribute, to.AddCallSiteAttribute, nil)
	RemapCallEffects(from, to, m)
	return nil
}

func remapValues(m ABIMapping, get func(int, uint) llvm.Attribute, add func(int, llvm.Attribute), remove func(int, uint)) {
	values := append([]ABIValue{m.Result}, m.Params...)
	for old, v := range values {
		if v.Kind != Direct || len(v.Indices) != 1 {
			continue
		}
		for _, name := range valueAttributes {
			if name == "returned" && m.Result.Kind != Direct {
				if remove != nil {
					remove(v.Indices[0], llvm.AttributeKindID(name))
				}
				continue
			}
			if attr := get(old, llvm.AttributeKindID(name)); !attr.IsNil() {
				add(v.Indices[0], attr)
			}
		}
	}
}
