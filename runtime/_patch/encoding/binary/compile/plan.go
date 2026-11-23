package compile

import (
	"go/ast"
	"go/types"
	"math"

	"github.com/xgo-dev/llgo/internal/build/calltemplate"
	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
)

type rewrite struct {
	call      *ast.CallExpr
	op        string
	order     types.Type
	input     types.Type
	wire      *layout
	ptr       bool
	slice     bool
	unchecked bool
	helper    string
}

var Generator = calltemplate.Generator{
	Package:         "encoding/binary",
	FallbackEntries: []string{"Read", "Write", "Append"},
	Apply:           rewriteBinary,
}

func rewriteBinary(input calltemplate.Generation) (bool, error) {
	entry, library := input.Entry, input.Library
	if shadowsGeneratedBuiltins(entry.Types) {
		return false, nil
	}
	maxWidth := int64(math.MaxInt64)
	if entry.TypesSizes.Sizeof(types.Typ[types.Int]) == 4 {
		maxWidth = math.MaxInt32
	}
	shortBuffer := shortBufferHelper(library.Types)
	out := packageRewrites{pkg: entry}
	for _, c := range input.Calls {
		var op string
		switch c.Candidate.Name() {
		case "llgoSize":
			op = "size"
		case "llgoEncode":
			op = "encode"
		case "llgoDecode":
			op = "decode"
		default:
			return false, nil
		}
		plan, ok := planBinaryCall(entry, c.Call, op, library.Types, maxWidth)
		if !ok || op != "size" && !plan.unchecked && shortBuffer == "" {
			return false, nil
		}
		out.calls = append(out.calls, plan)
	}
	return true, out.apply(library, shortBuffer)
}

type packageRewrites struct {
	pkg   *packages.Package
	calls []rewrite
}

func planBinaryCall(pkg *packages.Package, call *ast.CallExpr, op string, library *types.Package, maxWidth int64) (rewrite, bool) {
	r := rewrite{call: call, op: op}
	switch {
	case op == "size" && len(call.Args) == 1:
	case (op == "encode" || op == "decode") && len(call.Args) == 3:
		r.order = pkg.TypesInfo.TypeOf(call.Args[1])
		if !standardByteOrder(r.order, library) {
			return r, false
		}
	default:
		return r, false
	}
	operand := call.Args[len(call.Args)-1]
	r.input = pkg.TypesInfo.TypeOf(operand)
	if r.input == nil || llssa.HasTypeParam(r.input) {
		return r, false
	}
	value := r.input
	if ptr, ok := value.Underlying().(*types.Pointer); ok {
		if op != "size" && !nonNilPointer(pkg.TypesInfo, operand) {
			return r, false
		}
		r.ptr = true
		value = ptr.Elem()
	}
	if slice, ok := value.Underlying().(*types.Slice); ok {
		r.slice = true
		value = slice.Elem()
	} else if op == "decode" && !r.ptr {
		return r, false
	}
	// Basic operands already have fast paths with distinct nil, overlap and
	// floating-point behavior. Start at the fixed-size composite fallback.
	switch value.Underlying().(type) {
	case *types.Array, *types.Struct:
	default:
		return r, false
	}
	r.wire = fixedLayout(value, maxWidth)
	if r.wire == nil || op != "size" && !canAccess(r.wire, pkg.Types, op == "decode") {
		return r, false
	}
	if op != "size" && !r.slice {
		r.unchecked = r.wire.width == 0 || arraySliceLength(pkg.TypesInfo, call.Args[0]) >= r.wire.width
	}
	return r, true
}

func standardByteOrder(typ types.Type, library *types.Package) bool {
	if typ == nil {
		return false
	}
	named, ok := types.Unalias(typ).(*types.Named)
	if !ok || named.Obj().Pkg() != library {
		return false
	}
	switch named.Obj().Name() {
	case "littleEndian", "bigEndian", "nativeEndian":
		return true
	}
	return false
}

func canAccess(wire *layout, owner *types.Package, decode bool) bool {
	for _, f := range wire.fields {
		if f.field.Name() == "_" {
			continue
		}
		if !f.field.Exported() && (decode || f.field.Pkg() != owner) || !canAccess(f.value, owner, decode) {
			return false
		}
	}
	return wire.elem == nil || canAccess(wire.elem, owner, decode)
}

func shadowsGeneratedBuiltins(pkg *types.Package) bool {
	for _, name := range []string{"bool", "byte", "complex", "error", "false", "float32",
		"float64", "imag", "int", "len", "nil", "real", "true", "uint16", "uint32", "uint64"} {
		if pkg.Scope().Lookup(name) != nil {
			return true
		}
	}
	return false
}
