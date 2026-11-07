package llvmintrinsic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Build representative overloads in Go. The binding exposes LLVM metadata and
// canonical type construction, not a test-specific signature generator.
func sampleSignature(t *testing.T, ctx llvm.Context, id, variant int) llvm.Type {
	t.Helper()
	entries := llvm.IntrinsicDescriptors(id)
	lanes, size := 4, 0
	for _, d := range entries {
		switch d.Kind {
		case "OneNthEltsVecArgument":
			divisor := int(d.Value) * 4
			a, b := lanes, divisor
			for b != 0 {
				a, b = b, a%b
			}
			lanes = lanes / a * divisor
		case "Argument":
			if d.ArgumentKind != 7 {
				size = max(size, int(d.Argument)+1)
			}
		case "VecOfAnyPtrsToElt":
			size = max(size, int(d.Argument)+1, int(d.Reference)+1)
		}
	}
	overloads := make([]llvm.Type, size)
	i32, f32 := ctx.Int32Type(), ctx.FloatType()
	vector := func(element llvm.Type) llvm.Type {
		if variant == 3 {
			return llvm.ScalableVectorType(element, lanes)
		}
		return llvm.VectorType(element, lanes)
	}
	for _, d := range entries {
		if d.Kind != "Argument" || d.ArgumentKind == 7 {
			continue
		}
		var typ llvm.Type
		switch d.ArgumentKind {
		case 0:
			typ = []llvm.Type{i32, llvm.ArrayType(ctx.Int8Type(), 3), ctx.StructType([]llvm.Type{i32, ctx.DoubleType()}, false), llvm.ArrayType(ctx.Int8Type(), 0)}[variant]
		case 1, 2:
			typ = i32
			if d.ArgumentKind == 2 {
				typ = f32
			}
			if variant != 0 {
				typ = vector(typ)
			}
		case 3:
			typ = i32
			if variant%2 != 0 {
				typ = f32
			}
			typ = vector(typ)
		case 4:
			typ = llvm.PointerType(ctx.Int8Type(), variant%2)
		default:
			t.Fatalf("unknown overload domain %d", d.ArgumentKind)
		}
		overloads[d.Argument] = typ
	}
	for _, d := range entries {
		if d.Kind == "VecOfAnyPtrsToElt" {
			n, scale := overloads[d.Reference].VectorElementCount()
			ptr := llvm.PointerType(ctx.Int8Type(), variant%2)
			if scale {
				overloads[d.Argument] = llvm.ScalableVectorType(ptr, n)
			} else {
				overloads[d.Argument] = llvm.VectorType(ptr, n)
			}
		}
	}
	for _, typ := range overloads {
		if typ.IsNil() {
			t.Fatalf("unresolved overload for %s", llvm.IntrinsicName(id))
		}
	}
	return ctx.IntrinsicType(id, overloads)
}

// Decode the linked IIT metadata only in the coverage test. Production's nil
// description delegates directly to LLVM; it does not duplicate this decoder.
func metadataPattern(t *testing.T, entries []llvm.IITDescriptor) Type {
	t.Helper()
	pos := 0
	var read func() Type
	ref := func(index int) Type { return Ref(fmt.Sprintf("T%d", index)) }
	read = func() Type {
		if pos == len(entries) {
			t.Fatal("truncated IIT metadata")
		}
		d := entries[pos]
		pos++
		switch d.Kind {
		case "Void":
			return Void
		case "Token":
			return kindType(llvm.TokenTypeKind)
		case "Metadata":
			return kindType(llvm.MetadataTypeKind)
		case "Double":
			return F64
		case "Integer":
			return Int(uint(d.Value))
		case "Pointer":
			return Ptr(d.Value)
		case "VarArg":
			return VarArgs(Any)
		case "Struct":
			fields := make([]Type, int(d.Value))
			for i := range fields {
				fields[i] = read()
			}
			return Struct(fields...)
		case "Argument":
			if d.ArgumentKind == 7 {
				return ref(int(d.Argument))
			}
			domains := []Type{Any, Integers, Floats, AnyVector, AnyPointer}
			if int(d.ArgumentKind) >= len(domains) {
				t.Fatalf("unknown LLVM overload domain %d", d.ArgumentKind)
			}
			return Bind(fmt.Sprintf("T%d", int(d.Argument)), domains[int(d.ArgumentKind)])
		case "VecElementArgument":
			return Elem(ref(int(d.Argument)))
		case "SameVecWidthArgument":
			return Shape(ref(int(d.Argument)), read())
		case "OneNthEltsVecArgument":
			return Vector(Divide(Lanes(ref(int(d.Reference))), uint64(d.Value)), Elem(ref(int(d.Reference))))
		case "VecOfAnyPtrsToElt":
			return Bind(fmt.Sprintf("T%d", int(d.Argument)), Vector(Lanes(ref(int(d.Reference))), AnyPointer))
		default:
			t.Fatalf("unhandled generic IIT descriptor %q (%d)", d.Kind, int(d.Value))
		}
		return nil
	}
	result := read()
	var params []Type
	for pos < len(entries) {
		params = append(params, read())
	}
	return Func(result, params...)
}

func TestAllGenericIntrinsicSignatures(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	families, positives, negatives := 0, 0, 0
	seen := make(map[string]bool)
	for id := 1; id < llvm.IntrinsicCount(); id++ {
		if llvm.IntrinsicIsTarget(id) {
			continue
		}
		name := llvm.IntrinsicName(id)
		families++
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			pattern := metadataPattern(t, llvm.IntrinsicDescriptors(id))
			for variant := 0; variant < 4; variant++ {
				sig := sampleSignature(t, ctx, id, variant)
				if sig.IsNil() {
					t.Fatalf("no signature sample for variant %d", variant)
				}
				if _, err := Match(pattern, sig); err != nil {
					t.Fatalf("variant %d: %v", variant, err)
				}
				// Both the metadata-derived description and the nil/default path
				// must agree with LLVM's canonical signature, without fake calls.
				for _, spec := range []Type{nil, pattern} {
					binding, err := (Intrinsic{Name: name, Type: spec}).Resolve(sig)
					if err != nil {
						t.Fatalf("variant %d: %v", variant, err)
					}
					if binding.ID() != id || binding.Signature() != sig {
						t.Fatal("lost canonical signature")
					}
				}
				positives++
				wrong := llvm.FunctionType(sig.ReturnType(), sig.ParamTypes(), !sig.IsFunctionVarArg())
				if _, err := Match(pattern, wrong); err == nil {
					t.Fatal("accepted a mismatched vararg flag")
				}
				if _, err := (Intrinsic{Name: name}).Resolve(wrong); err == nil {
					t.Fatal("LLVM accepted a mismatched vararg flag")
				}
				negatives++
			}
		})
	}
	if families == 0 || !seen["llvm.memcpy"] || !seen["llvm.vector.deinterleave3"] || !seen["llvm.fake.use"] {
		t.Fatal("incomplete linked intrinsic enumeration")
	}
	t.Logf("linked generic families=%d, metadata signature samples=%d, vararg rejections=%d", families, positives, negatives)
}

func TestMetadataRelationsRejectMutations(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	i8, i16, i32 := ctx.Int8Type(), ctx.Int16Type(), ctx.Int32Type()
	for _, tc := range []struct {
		name   string
		mutate func(llvm.Type) llvm.Type
	}{
		{"llvm.ctpop", func(sig llvm.Type) llvm.Type { return llvm.FunctionType(sig.ReturnType(), []llvm.Type{i16}, false) }},
		{"llvm.vector.reduce.add", func(sig llvm.Type) llvm.Type { return llvm.FunctionType(i16, sig.ParamTypes(), false) }},
		{"llvm.masked.load", func(sig llvm.Type) llvm.Type {
			p := sig.ParamTypes()
			p[1] = llvm.VectorType(ctx.Int1Type(), 2)
			return llvm.FunctionType(sig.ReturnType(), p, false)
		}},
		{"llvm.masked.gather", func(sig llvm.Type) llvm.Type {
			p := sig.ParamTypes()
			p[0] = llvm.VectorType(llvm.PointerType(i8, 0), 2)
			return llvm.FunctionType(sig.ReturnType(), p, false)
		}},
		{"llvm.masked.gather", func(sig llvm.Type) llvm.Type {
			p := sig.ParamTypes()
			p[0] = llvm.VectorType(i32, 4)
			return llvm.FunctionType(sig.ReturnType(), p, false)
		}},
		{"llvm.vector.interleave3", func(sig llvm.Type) llvm.Type {
			p := sig.ParamTypes()
			p[0] = llvm.VectorType(i32, 3)
			return llvm.FunctionType(sig.ReturnType(), p, false)
		}},
		{"llvm.vector.deinterleave3", func(sig llvm.Type) llvm.Type {
			fields := sig.ReturnType().StructElementTypes()
			fields[1] = llvm.VectorType(i32, 3)
			return llvm.FunctionType(ctx.StructType(fields, false), sig.ParamTypes(), false)
		}},
		{"llvm.memcpy", func(sig llvm.Type) llvm.Type {
			p := sig.ParamTypes()
			p[0] = ctx.Int64Type()
			return llvm.FunctionType(sig.ReturnType(), p, false)
		}},
		{"llvm.ctpop", func(sig llvm.Type) llvm.Type {
			return llvm.FunctionType(sig.ReturnType(), []llvm.Type{llvm.ArrayType(i32, 4)}, false)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := llvm.LookupIntrinsicID(tc.name)
			sig := tc.mutate(sampleSignature(t, ctx, id, 0))
			if _, err := Match(metadataPattern(t, llvm.IntrinsicDescriptors(id)), sig); err == nil {
				t.Fatal("description accepted mutation")
			}
			if _, err := (Intrinsic{Name: tc.name}).Resolve(sig); err == nil {
				t.Fatal("LLVM accepted mutation")
			}
		})
	}
}

func TestInterleaveExactDivisionBeforeLLVM(t *testing.T) {
	ctx := llvm.NewContext()
	defer ctx.Dispose()
	for _, name := range []string{"llvm.vector.interleave3", "llvm.vector.deinterleave3"} {
		t.Run(name, func(t *testing.T) {
			id := llvm.LookupIntrinsicID(name)
			sub, full := llvm.VectorType(ctx.Int32Type(), 3), llvm.VectorType(ctx.Int32Type(), 10)
			var sig llvm.Type
			if strings.Contains(name, "deinterleave") {
				sig = llvm.FunctionType(ctx.StructType([]llvm.Type{sub, sub, sub}, false), []llvm.Type{full}, false)
			} else {
				sig = llvm.FunctionType(full, []llvm.Type{sub, sub, sub}, false)
			}
			// LLVM 22's matcher truncates this division in release builds, and
			// can assert in assertion builds. Reject it in the refinement first.
			_, err := (Intrinsic{Name: name, Type: metadataPattern(t, llvm.IntrinsicDescriptors(id))}).Resolve(sig)
			if err == nil || !strings.Contains(err.Error(), "not divisible") {
				t.Fatalf("got %v", err)
			}
		})
	}
}
