package llvmintrinsic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/xgo-dev/llvm"
)

// Decode the linked IIT metadata only in the coverage test. Production's nil
// description delegates directly to LLVM; it does not duplicate this decoder.
func metadataPattern(t *testing.T, entries []descriptor) Type {
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
		switch d.kind {
		case "void":
			return Void
		case "token":
			return kindType(llvm.TokenTypeKind)
		case "metadata":
			return kindType(llvm.MetadataTypeKind)
		case "double":
			return F64
		case "integer":
			return Int(uint(d.n))
		case "pointer":
			return Ptr(uint32(d.n))
		case "vararg":
			return VarArgs(Any)
		case "struct":
			fields := make([]Type, d.n)
			for i := range fields {
				fields[i] = read()
			}
			return Struct(fields...)
		case "argument":
			if d.domain == 7 {
				return ref(d.argument)
			}
			domains := []Type{Any, Integers, Floats, AnyVector, AnyPointer}
			if d.domain >= len(domains) {
				t.Fatalf("unknown LLVM overload domain %d", d.domain)
			}
			return Bind(fmt.Sprintf("T%d", d.argument), domains[d.domain])
		case "element":
			return Elem(ref(d.argument))
		case "shape":
			return Shape(ref(d.argument), read())
		case "divide_lanes":
			return Vector(Divide(Lanes(ref(d.argument)), uint64(d.n)), Elem(ref(d.argument)))
		case "pointer_vector":
			return Bind(fmt.Sprintf("T%d", d.argument), Vector(Lanes(ref(d.n)), AnyPointer))
		default:
			t.Fatalf("unhandled generic IIT descriptor %q (%d)", d.kind, d.n)
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
	for id := 1; id < intrinsicCount(); id++ {
		if targetIntrinsic(id) {
			continue
		}
		name := baseName(id)
		families++
		seen[name] = true
		t.Run(name, func(t *testing.T) {
			pattern := metadataPattern(t, descriptors(id))
			for variant := 0; variant < 4; variant++ {
				sig := sampleSignature(ctx, id, variant)
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
			sig := tc.mutate(sampleSignature(ctx, id, 0))
			if _, err := Match(metadataPattern(t, descriptors(id)), sig); err == nil {
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
			_, err := (Intrinsic{Name: name, Type: metadataPattern(t, descriptors(id))}).Resolve(sig)
			if err == nil || !strings.Contains(err.Error(), "not divisible") {
				t.Fatalf("got %v", err)
			}
		})
	}
}
