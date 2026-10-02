package cl

import (
	"fmt"
	"go/ast"
	"go/types"

	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

// An empty receiver identifies a package function. "numeric" groups official
// numeric vector methods; concrete shapes come from the resolved Go types.
type simdKey struct{ receiver, name string }
type simdSignature uint8

const (
	simdBinary simdSignature = iota
	simdExtract
	simdInsert
	simdUnsupported
	simdUnary
	simdLoad
	simdStore
	simdBroadcast
	simdBitcast
	simdCompare
	simdToMask
	simdTernary
	simdMaskFromBits
	simdMaskToBits
)

type simdOperation struct {
	op        llssa.SIMDOp
	signature simdSignature
	elements  types.BasicInfo
}

// The default entry gives not-yet-implemented intrinsic declarations a defined,
// recoverable failure. Adding an implementation replaces this fallback for that
// operation; functions with Go bodies continue through normal compilation.
var simdOperations = map[simdKey]simdOperation{
	{"*", "*"}:                  {llssa.SIMDUnimplemented, simdUnsupported, 0},
	{"numeric", "Add"}:          {llssa.SIMDAdd, simdBinary, 0},
	{"numeric", "Sub"}:          {llssa.SIMDSub, simdBinary, 0},
	{"numeric", "And"}:          {llssa.SIMDAnd, simdBinary, types.IsInteger},
	{"numeric", "Or"}:           {llssa.SIMDOr, simdBinary, types.IsInteger},
	{"numeric", "Xor"}:          {llssa.SIMDXor, simdBinary, types.IsInteger},
	{"numeric", "GetElem"}:      {llssa.SIMDExtractLane, simdExtract, 0},
	{"numeric", "SetElem"}:      {llssa.SIMDInsertLane, simdInsert, 0},
	{"numeric", "StoreArray"}:   {llssa.SIMDStore, simdStore, 0},
	{"numeric", "Mul"}:          {llssa.SIMDMul, simdBinary, 0},
	{"numeric", "Div"}:          {llssa.SIMDDiv, simdBinary, types.IsFloat},
	{"numeric", "AndNot"}:       {llssa.SIMDAndNot, simdBinary, types.IsInteger},
	{"numeric", "OrNot"}:        {llssa.SIMDOrNot, simdBinary, types.IsInteger},
	{"numeric", "Not"}:          {llssa.SIMDNot, simdUnary, types.IsInteger},
	{"numeric", "Neg"}:          {llssa.SIMDNeg, simdUnary, 0},
	{"numeric", "Abs"}:          {llssa.SIMDAbs, simdUnary, 0},
	{"numeric", "Sqrt"}:         {llssa.SIMDSqrt, simdUnary, types.IsFloat},
	{"numeric", "Ceil"}:         {llssa.SIMDCeil, simdUnary, types.IsFloat},
	{"numeric", "Floor"}:        {llssa.SIMDFloor, simdUnary, types.IsFloat},
	{"numeric", "Trunc"}:        {llssa.SIMDTrunc, simdUnary, types.IsFloat},
	{"numeric", "Round"}:        {llssa.SIMDRound, simdUnary, types.IsFloat},
	{"numeric", "Equal"}:        {llssa.SIMDEqual, simdCompare, 0},
	{"numeric", "NotEqual"}:     {llssa.SIMDNotEqual, simdCompare, 0},
	{"numeric", "Less"}:         {llssa.SIMDLess, simdCompare, 0},
	{"numeric", "LessEqual"}:    {llssa.SIMDLessEqual, simdCompare, 0},
	{"numeric", "Greater"}:      {llssa.SIMDGreater, simdCompare, 0},
	{"numeric", "GreaterEqual"}: {llssa.SIMDGreaterEqual, simdCompare, 0},
	{"numeric", "ToMask"}:       {llssa.SIMDToMask, simdToMask, types.IsInteger},
	{"numeric", "asMask"}:       {llssa.SIMDBitcast, simdToMask, types.IsInteger},
	{"numeric", "bitSelect"}:    {llssa.SIMDBitSelect, simdTernary, types.IsInteger},
	{"numeric", "BitSelect"}:    {llssa.SIMDBitSelect, simdTernary, types.IsInteger},
	{"numeric", "bitSelectNot"}: {llssa.SIMDBitSelectNot, simdTernary, types.IsInteger},
	{"numeric", "blend"}:        {llssa.SIMDBlend, simdTernary, types.IsInteger},
	{"mask", "And"}:             {llssa.SIMDAnd, simdBinary, 0},
	{"mask", "Or"}:              {llssa.SIMDOr, simdBinary, 0},
	{"mask", "Xor"}:             {llssa.SIMDXor, simdBinary, 0},
	{"mask", "AndNot"}:          {llssa.SIMDAndNot, simdBinary, 0},
	{"mask", "Not"}:             {llssa.SIMDNot, simdUnary, 0},
}

// These registrations share lowering but retain exact declaration names and
// signature checks. Source Go slice helpers keep their own bounds checks.
func init() {
	for _, name := range []string{"Mask8x16", "Mask16x8", "Mask32x4", "Mask64x2"} {
		simdOperations[simdKey{"", name + "FromBits"}] = simdOperation{llssa.SIMDMaskFromBits, simdMaskFromBits, 0}
	}
	simdOperations[simdKey{"mask", "ToBits"}] = simdOperation{llssa.SIMDMaskToBits, simdMaskToBits, 0}
	for _, name := range []string{"ToInt8x16", "ToInt16x8", "ToInt32x4", "ToInt64x2"} {
		simdOperations[simdKey{"mask", name}] = simdOperation{llssa.SIMDBitcast, simdBitcast, 0}
	}
	for _, name := range []string{"ToBits", "BitsToInt8", "BitsToInt16", "BitsToInt32", "BitsToInt64", "BitsToFloat32", "BitsToFloat64", "ReshapeToUint8s", "ReshapeToUint16s", "ReshapeToUint32s", "ReshapeToUint64s"} {
		simdOperations[simdKey{"numeric", name}] = simdOperation{llssa.SIMDBitcast, simdBitcast, 0}
	}
	for _, name := range []string{"Int8x16", "Uint8x16", "Int16x8", "Uint16x8", "Int32x4", "Uint32x4", "Int64x2", "Uint64x2", "Float32x4", "Float64x2"} {
		simdOperations[simdKey{"", "Load" + name + "Array"}] = simdOperation{llssa.SIMDLoad, simdLoad, 0}
		simdOperations[simdKey{"", "Broadcast" + name}] = simdOperation{llssa.SIMDBroadcast, simdBroadcast, 0}
	}
	for _, lanes := range []int{2, 4, 8, 16} {
		simdOperations[simdKey{"numeric", fmt.Sprintf("broadcast1To%d", lanes)}] = simdOperation{llssa.SIMDSplatLane0, simdUnary, 0}
	}
}

// Resolve only declared official operations, never synthetic wrapper names.
// Direct calls and calls inside the existing SSA wrappers use this same path.
func lookupSIMD(fn *ssa.Function, arch string) (simdOperation, bool) {
	switch arch {
	case "amd64", "arm64", "wasm":
	default:
		return simdOperation{}, false
	}
	obj, ok := fn.Object().(*types.Func)
	if !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" {
		return simdOperation{}, false
	}
	sig := fn.Signature
	if !types.Identical(sig, obj.Type()) {
		return simdOperation{}, false
	}
	decl, isDecl := fn.Syntax().(*ast.FuncDecl)
	fallback := func() (simdOperation, bool) {
		// Imported declarations and synthetic wrappers are not definitions. Emit
		// the fallback only for a source intrinsic declaration in archsimd.
		if isDecl && decl.Body == nil {
			return simdOperations[simdKey{"*", "*"}], true
		}
		return simdOperation{}, false
	}
	if isDecl && decl.Body != nil {
		return simdOperation{}, false
	}
	key := simdKey{name: obj.Name()}
	var vector types.Type
	if recv := sig.Recv(); recv != nil {
		vector = recv.Type()
		if _, ok := llssa.SIMDNumericShape(vector); ok {
			key.receiver = "numeric"
		} else if _, ok := llssa.SIMDMaskShape(vector); ok {
			key.receiver = "mask"
		} else {
			return fallback()
		}
	} else if sig.Results().Len() == 1 {
		vector = sig.Results().At(0).Type()
	}
	desc, ok := simdOperations[key]
	if !ok || !desc.matches(sig, vector) {
		return fallback()
	}
	return desc, true
}

func (d simdOperation) matches(sig *types.Signature, vector types.Type) bool {
	if vector == nil || sig.Variadic() {
		return false
	}
	lanes, ok := llssa.SIMDVectorShape(vector)
	if !ok {
		return false
	}
	if d.elements != 0 && lanes.Elem().Underlying().(*types.Basic).Info()&d.elements == 0 {
		return false
	}
	var params []types.Type
	result := vector
	switch d.signature {
	case simdMaskFromBits, simdMaskToBits:
		if _, ok := llssa.SIMDMaskShape(vector); !ok {
			return false
		}
		bits := types.Typ[types.Uint8]
		if lanes.Len() == 16 {
			bits = types.Typ[types.Uint16]
		}
		if d.signature == simdMaskFromBits {
			params = []types.Type{bits}
		} else {
			result = bits
		}
	case simdCompare, simdToMask:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		mask, ok := llssa.SIMDMaskShape(result)
		if !ok || mask.Len() != lanes.Len() {
			return false
		}
		if d.signature == simdCompare {
			params = []types.Type{vector}
		}
	case simdTernary:
		params = []types.Type{vector, vector}
	case simdBitcast:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		if _, ok := llssa.SIMDNumericShape(result); !ok {
			return false
		}
	case simdUnary:
		// Receiver only.
	case simdLoad, simdStore:
		params = []types.Type{types.NewPointer(lanes)}
		if d.signature == simdStore {
			result = nil
		}
	case simdBroadcast:
		params = []types.Type{lanes.Elem()}
	case simdBinary:
		params = []types.Type{vector}
	case simdExtract:
		params, result = []types.Type{types.Typ[types.Uint8]}, lanes.Elem()
	case simdInsert:
		params = []types.Type{types.Typ[types.Uint8], lanes.Elem()}
	default:
		return false
	}
	if sig.Recv() == nil && d.signature != simdLoad && d.signature != simdBroadcast && d.signature != simdMaskFromBits {
		params = append([]types.Type{vector}, params...)
	}
	if sig.Params().Len() != len(params) {
		return false
	}
	if result == nil {
		if sig.Results().Len() != 0 {
			return false
		}
	} else if sig.Results().Len() != 1 || !types.Identical(sig.Results().At(0).Type(), result) {
		return false
	}
	for i, typ := range params {
		if !types.Identical(sig.Params().At(i).Type(), typ) {
			return false
		}
	}
	return true
}

func (p *context) simdOperation(fn *ssa.Function) (simdOperation, bool) {
	desc, ok := lookupSIMD(fn, p.prog.Target().GOARCH)
	return desc, ok && desc.op != llssa.SIMDUnimplemented
}

func (p *context) simdCall(b llssa.Builder, fn *ssa.Function, args []ssa.Value) llssa.Expr {
	desc, ok := p.simdOperation(fn)
	if !ok {
		panic("invalid SIMD intrinsic")
	}
	return b.SIMD(desc.op, p.simdResultType(fn.Signature), p.compileValues(b, args, fnNormal)...)
}

func (p *context) simdResultType(sig *types.Signature) llssa.Type {
	if sig.Results().Len() == 0 {
		return p.prog.Void()
	}
	return p.prog.Type(sig.Results().At(0).Type(), llssa.InGo)
}
