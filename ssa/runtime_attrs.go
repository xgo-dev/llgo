package ssa

import "github.com/xgo-dev/llvm"

// runtimeContracts names the fixed runtime entries whose implementations obey
// these contracts, including declarations imported into a different module.
// Match resolved symbols so go:linkname callers get the same attributes.
var runtimeContracts = map[string]string{
	PkgRuntime + ".AssertNilDerefPtr": "checked",
	PkgRuntime + ".CStrCopy":          "cstrcopy",
	PkgRuntime + ".memequal":          "compare",
	PkgRuntime + ".StringEqual":       "read",
	PkgRuntime + ".StringLess":        "read",
	PkgRuntime + ".Typedmemmove":      "move",
	"reflect.typedmemmove":            "move",
	PkgRuntime + ".Typedmemclr":       "clear",
	PkgRuntime + ".MapLen":            "length",
	PkgRuntime + ".ChanCap":           "length",
	PkgRuntime + ".Memhash":           "hash",
	PkgRuntime + ".Memhash32":         "hash",
	PkgRuntime + ".Memhash64":         "hash",

	PkgRuntime + ".Panic":                   "panic",
	PkgRuntime + ".PanicErrorString":        "panic",
	PkgRuntime + ".PanicIndex":              "panic",
	PkgRuntime + ".PanicIndexU":             "panic",
	PkgRuntime + ".PanicSliceConvert":       "panic",
	PkgRuntime + ".PanicTypeAssert":         "panic",
	PkgRuntime + ".PanicTypeAssertionError": "panic",
	PkgRuntime + ".PanicExtendIndex":        "panic",
	PkgRuntime + ".PanicExtendIndexU":       "panic",
}

// LLVM 22 MemoryEffects uses two ModRef bits per location, with argument
// memory first. Ref=1, ModRef=3; there are six locations (including target
// memory). CaptureInfo stores return components in the low four bits.
const (
	runtimeArgRead       = 1
	runtimeArgReadWrite  = 3
	runtimeMemoryRead    = 0x555
	runtimeCaptureReturn = 0xf
)

func (p Program) addRuntimeAttributes(fn llvm.Value, name string) {
	contract, ok := runtimeContracts[name]
	if !ok {
		return
	}
	add := func(index int, name string, value uint64) {
		fn.AddAttributeAtIndex(index, p.ctx.CreateEnumAttribute(llvm.AttributeKindID(name), value))
	}
	pointer := func(index int, access string) {
		add(index, access, 0)
		add(index, "captures", 0)
	}
	switch contract {
	case "checked":
		// nil is a valid input: the runtime must still raise a recoverable
		// panic. Only a normal return guarantees non-nullness.
		add(0, "nonnull", 0)
		add(1, "returned", 0)
		return
	case "panic":
		add(-1, "cold", 0)
		add(-1, "noreturn", 0)
		return
	}
	for _, attr := range []string{"nofree", "nosync", "nounwind", "willreturn"} {
		add(-1, attr, 0)
	}
	switch contract {
	case "cstrcopy":
		add(1, "returned", 0)
		add(1, "writeonly", 0)
		add(1, "captures", runtimeCaptureReturn)
		// The source string is an aggregate, not a pointer argument.
		add(-1, "memory", runtimeMemoryRead|runtimeArgReadWrite)
	case "compare":
		pointer(1, "readonly")
		pointer(2, "readonly")
		add(-1, "memory", runtimeArgRead)
	case "read":
		// String data is carried in aggregate parameters.
		add(-1, "memory", runtimeMemoryRead)
	case "move", "clear":
		pointer(1, "readonly")
		pointer(2, "writeonly")
		if contract == "move" {
			pointer(3, "readonly")
		}
		// No noalias or nonnull: overlap and zero-byte copies are valid.
		add(-1, "memory", runtimeArgReadWrite)
	case "length":
		pointer(1, "readonly")
		add(-1, "memory", runtimeArgRead)
		addNonNegativeReturnRange(p.ctx, fn, p.Int().ll.IntTypeWidth())
	case "hash":
		pointer(1, "readonly")
		// Hashing also reads the process-global hashkey seed.
		add(-1, "memory", runtimeMemoryRead)
	}
}

func addNonNegativeReturnRange(ctx llvm.Context, fn llvm.Value, bits int) {
	if bits != 32 && bits != 64 {
		panic("ssa: unsupported runtime integer width")
	}
	attr := ctx.CreateConstantRangeAttribute(llvm.AttributeKindID("range"), bits,
		[]uint64{0}, []uint64{uint64(1) << (bits - 1)})
	fn.AddAttributeAtIndex(0, attr)
}
