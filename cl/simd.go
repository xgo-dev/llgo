package cl

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

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
	simdMaskedBlend
	simdMaskFromBits
	simdMaskToBits
	simdScalarShift
	simdUnsignedVector
	simdSignedVector
	simdConvert
	simdWidenBinary
	simdCarrylessSelector
	simdStoreMasked
	simdIntegerResize
	simdSplat
	simdHalf
	simdConvertLo
	simdPairSelector
	simdPairHalfSelectors
	simdScalarSelector
	simdBool
	simdConcatShift
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
	{"*", "*"}:                                  {llssa.SIMDUnimplemented, simdUnsupported, 0},
	{"numeric", "IsNaN"}:                        {llssa.SIMDIsNaN, simdToMask, types.IsFloat},
	{"numeric", "IsZero"}:                       {llssa.SIMDIsZero, simdBool, types.IsInteger},
	{"numeric", "ConcatShiftBytesRight"}:        {llssa.SIMDConcatShiftBytes, simdConcatShift, types.IsUnsigned},
	{"numeric", "ConcatShiftBytesRightGrouped"}: {llssa.SIMDConcatShiftBytesGrouped, simdConcatShift, types.IsUnsigned},
	{"numeric", "OnesCount"}:                    {llssa.SIMDOnesCount, simdUnary, types.IsInteger},
	{"numeric", "ShiftSaturated"}:               {llssa.SIMDShiftSaturated, simdSignedVector, types.IsInteger},
	{"numeric", "reduceSum"}:                    {llssa.SIMDReduceSum, simdUnary, types.IsInteger},
	{"numeric", "reduceMin"}:                    {llssa.SIMDReduceMin, simdUnary, 0},
	{"numeric", "reduceMax"}:                    {llssa.SIMDReduceMax, simdUnary, 0},
	{"numeric", "Add"}:                          {llssa.SIMDAdd, simdBinary, 0},
	{"numeric", "Sub"}:                          {llssa.SIMDSub, simdBinary, 0},
	{"numeric", "And"}:                          {llssa.SIMDAnd, simdBinary, types.IsInteger},
	{"numeric", "Or"}:                           {llssa.SIMDOr, simdBinary, types.IsInteger},
	{"numeric", "Xor"}:                          {llssa.SIMDXor, simdBinary, types.IsInteger},
	{"numeric", "GetElem"}:                      {llssa.SIMDExtractLane, simdExtract, 0},
	{"numeric", "SetElem"}:                      {llssa.SIMDInsertLane, simdInsert, 0},
	{"numeric", "StoreArray"}:                   {llssa.SIMDStore, simdStore, 0},
	{"numeric", "Mul"}:                          {llssa.SIMDMul, simdBinary, 0},
	{"numeric", "Div"}:                          {llssa.SIMDDiv, simdBinary, types.IsFloat},
	{"numeric", "AndNot"}:                       {llssa.SIMDAndNot, simdBinary, types.IsInteger},
	{"numeric", "OrNot"}:                        {llssa.SIMDOrNot, simdBinary, types.IsInteger},
	{"numeric", "Not"}:                          {llssa.SIMDNot, simdUnary, types.IsInteger},
	{"numeric", "Neg"}:                          {llssa.SIMDNeg, simdUnary, 0},
	{"numeric", "Abs"}:                          {llssa.SIMDAbs, simdUnary, 0},
	{"numeric", "Sqrt"}:                         {llssa.SIMDSqrt, simdUnary, types.IsFloat},
	{"numeric", "Ceil"}:                         {llssa.SIMDCeil, simdUnary, types.IsFloat},
	{"numeric", "Floor"}:                        {llssa.SIMDFloor, simdUnary, types.IsFloat},
	{"numeric", "Trunc"}:                        {llssa.SIMDTrunc, simdUnary, types.IsFloat},
	{"numeric", "Round"}:                        {llssa.SIMDRound, simdUnary, types.IsFloat},
	{"numeric", "Equal"}:                        {llssa.SIMDEqual, simdCompare, 0},
	{"numeric", "NotEqual"}:                     {llssa.SIMDNotEqual, simdCompare, 0},
	{"numeric", "Less"}:                         {llssa.SIMDLess, simdCompare, 0},
	{"numeric", "LessEqual"}:                    {llssa.SIMDLessEqual, simdCompare, 0},
	{"numeric", "Greater"}:                      {llssa.SIMDGreater, simdCompare, 0},
	{"numeric", "GreaterEqual"}:                 {llssa.SIMDGreaterEqual, simdCompare, 0},
	{"numeric", "ToMask"}:                       {llssa.SIMDToMask, simdToMask, types.IsInteger},
	{"numeric", "asMask"}:                       {llssa.SIMDBitcast, simdToMask, types.IsInteger},
	{"numeric", "bitSelect"}:                    {llssa.SIMDBitSelect, simdTernary, types.IsInteger},
	{"numeric", "BitSelect"}:                    {llssa.SIMDBitSelect, simdTernary, types.IsInteger},
	{"numeric", "bitSelectNot"}:                 {llssa.SIMDBitSelectNot, simdTernary, types.IsInteger},
	{"numeric", "blendMasked"}:                  {llssa.SIMDBlend, simdMaskedBlend, types.IsInteger},
	{"numeric", "blend"}:                        {llssa.SIMDBlend, simdTernary, types.IsInteger},
	{"mask", "And"}:                             {llssa.SIMDAnd, simdBinary, 0},
	{"mask", "Or"}:                              {llssa.SIMDOr, simdBinary, 0},
	{"mask", "Xor"}:                             {llssa.SIMDXor, simdBinary, 0},
	{"mask", "AndNot"}:                          {llssa.SIMDAndNot, simdBinary, 0},
	{"mask", "Not"}:                             {llssa.SIMDNot, simdUnary, 0},
	{"numeric", "ShiftAllLeft"}:                 {llssa.SIMDShiftAllLeft, simdScalarShift, types.IsInteger},
	{"numeric", "ShiftAllRight"}:                {llssa.SIMDShiftAllRight, simdScalarShift, types.IsInteger},
	{"numeric", "ShiftLeft"}:                    {llssa.SIMDShiftLeft, simdUnsignedVector, types.IsInteger},
	{"numeric", "ShiftRight"}:                   {llssa.SIMDShiftRight, simdUnsignedVector, types.IsInteger},
	{"numeric", "Shift"}:                        {llssa.SIMDShift, simdSignedVector, types.IsInteger},
	{"numeric", "AddSaturated"}:                 {llssa.SIMDAddSaturated, simdBinary, types.IsInteger},
	{"numeric", "SubSaturated"}:                 {llssa.SIMDSubSaturated, simdBinary, types.IsInteger},
	{"numeric", "Min"}:                          {llssa.SIMDMin, simdBinary, 0},
	{"numeric", "LookupOrZero"}:                 {llssa.SIMDLookupOrZero, simdBinary, types.IsInteger},
	{"numeric", "PermuteOrZero"}:                {llssa.SIMDPermuteOrZero, simdSignedVector, types.IsInteger},
	{"numeric", "Permute"}:                      {llssa.SIMDPermute, simdUnsignedVector, types.IsInteger},
	{"numeric", "Max"}:                          {llssa.SIMDMax, simdBinary, 0},
	{"numeric", "GetLo"}:                        {llssa.SIMDGetLo, simdHalf, 0},
	{"numeric", "GetHi"}:                        {llssa.SIMDGetHi, simdHalf, 0},
	{"numeric", "SetLo"}:                        {llssa.SIMDSetLo, simdHalf, 0},
	{"numeric", "SetHi"}:                        {llssa.SIMDSetHi, simdHalf, 0},

	{"numeric", "InterleaveLo"}:                  {llssa.SIMDInterleaveLo, simdBinary, types.IsInteger},
	{"numeric", "InterleaveHi"}:                  {llssa.SIMDInterleaveHi, simdBinary, types.IsInteger},
	{"numeric", "InterleaveLoGrouped"}:           {llssa.SIMDInterleaveLoGrouped, simdBinary, types.IsInteger},
	{"numeric", "InterleaveHiGrouped"}:           {llssa.SIMDInterleaveHiGrouped, simdBinary, types.IsInteger},
	{"numeric", "ConcatEven"}:                    {llssa.SIMDConcatEven, simdBinary, types.IsInteger},
	{"numeric", "ConcatOdd"}:                     {llssa.SIMDConcatOdd, simdBinary, types.IsInteger},
	{"numeric", "InterleaveEven"}:                {llssa.SIMDInterleaveEven, simdBinary, types.IsInteger},
	{"numeric", "InterleaveOdd"}:                 {llssa.SIMDInterleaveOdd, simdBinary, types.IsInteger},
	{"numeric", "concatSelectedConstant"}:        {llssa.SIMDConcatSelected, simdPairSelector, 0},
	{"numeric", "concatSelectedConstantGrouped"}: {llssa.SIMDConcatSelectedGrouped, simdPairSelector, 0},
	{"numeric", "ConcatPermute128Scalars"}:       {llssa.SIMDConcatPermute128, simdPairHalfSelectors, 0},
	{"numeric", "permuteScalars"}:                {llssa.SIMDPermuteScalars, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsGrouped"}:         {llssa.SIMDPermuteScalarsGrouped, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsHi"}:              {llssa.SIMDPermuteScalarsHi, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsLo"}:              {llssa.SIMDPermuteScalarsLo, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsHiGrouped"}:       {llssa.SIMDPermuteScalarsHiGrouped, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsLoGrouped"}:       {llssa.SIMDPermuteScalarsLoGrouped, simdScalarSelector, types.IsInteger},

	{"numeric", "ConvertLo2ToFloat64"}: {llssa.SIMDConvertLo, simdConvertLo, 0},

	{"numeric", "ConcatAddPairs"}:           {llssa.SIMDConcatAddPairs, simdBinary, 0},
	{"numeric", "ConcatAddPairsGrouped"}:    {llssa.SIMDConcatAddPairsGrouped, simdBinary, 0},
	{"numeric", "ConcatSubPairs"}:           {llssa.SIMDConcatSubPairs, simdBinary, 0},
	{"numeric", "ConcatSubPairsGrouped"}:    {llssa.SIMDConcatSubPairsGrouped, simdBinary, 0},
	{"numeric", "Average"}:                  {llssa.SIMDAverage, simdBinary, types.IsInteger},
	{"numeric", "MulAdd"}:                   {llssa.SIMDFMA, simdTernary, types.IsFloat},
	{"numeric", "LeadingZeros"}:             {llssa.SIMDLeadingZeros, simdUnary, types.IsInteger},
	{"numeric", "LeadingSignBits"}:          {llssa.SIMDLeadingSignBits, simdUnary, types.IsInteger},
	{"numeric", "MulWidenHi"}:               {llssa.SIMDMulWidenHi, simdWidenBinary, types.IsInteger},
	{"numeric", "MulWidenLo"}:               {llssa.SIMDMulWidenLo, simdWidenBinary, types.IsInteger},
	{"numeric", "RotateAllLeft"}:            {llssa.SIMDRotateAllLeft, simdScalarShift, types.IsInteger},
	{"numeric", "RotateAllRight"}:           {llssa.SIMDRotateAllRight, simdScalarShift, types.IsInteger},
	{"numeric", "StoreArrayMasked"}:         {llssa.SIMDStoreMasked, simdStoreMasked, 0},
	{"numeric", "carrylessMultiply"}:        {llssa.SIMDCarrylessMultiply, simdCarrylessSelector, types.IsUnsigned},
	{"numeric", "carrylessMultiplyWidenLo"}: {llssa.SIMDCarrylessMultiplyLo, simdBinary, types.IsUnsigned},
}

// These registrations share lowering but retain exact declaration names and
// signature checks. Source Go slice helpers keep their own bounds checks.
func init() {
	for _, name := range []string{"ConvertToInt8", "ConvertToUint8", "ConvertToInt16", "ConvertToUint16", "ConvertToInt32", "ConvertToUint32", "ConvertToInt64", "ConvertToUint64", "ConvertToFloat32", "ConvertToFloat64"} {
		simdOperations[simdKey{"numeric", name}] = simdOperation{llssa.SIMDConvert, simdConvert, 0}
	}
	for _, elem := range []string{"Int16", "Uint16", "Int32", "Uint32", "Int64", "Uint64"} {
		simdOperations[simdKey{"numeric", "ExtendTo" + elem}] = simdOperation{llssa.SIMDExtend, simdIntegerResize, types.IsInteger}
		for _, lanes := range []int{2, 4, 8, 16} {
			for _, part := range []struct {
				name string
				op   llssa.SIMDOp
			}{{"Lo", llssa.SIMDExtendLo}, {"Hi", llssa.SIMDExtendHi}} {
				name := fmt.Sprintf("Extend%s%dTo%s", part.name, lanes, elem)
				simdOperations[simdKey{"numeric", name}] = simdOperation{part.op, simdIntegerResize, types.IsInteger}
			}
		}
	}
	for _, elem := range []string{"Int8", "Uint8", "Int16", "Uint16", "Int32", "Uint32"} {
		simdOperations[simdKey{"numeric", "SaturateTo" + elem}] = simdOperation{llssa.SIMDSaturate, simdIntegerResize, types.IsInteger}
		simdOperations[simdKey{"numeric", "TruncTo" + elem}] = simdOperation{llssa.SIMDTruncate, simdIntegerResize, types.IsInteger}
	}
	for _, name := range []string{"SaturateToInt16Concat", "SaturateToUint16Concat"} {
		simdOperations[simdKey{"numeric", name}] = simdOperation{llssa.SIMDSaturateConcat, simdIntegerResize, types.IsInteger}
		simdOperations[simdKey{"numeric", name + "Grouped"}] = simdOperation{llssa.SIMDSaturateConcatGrouped, simdIntegerResize, types.IsInteger}
	}
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
	for _, lanes := range []int{2, 4, 8, 16, 32, 64} {
		simdOperations[simdKey{"numeric", fmt.Sprintf("broadcast1To%d", lanes)}] = simdOperation{llssa.SIMDSplatLane0, simdSplat, 0}
	}
	for _, name := range []string{"Int8x32", "Uint8x32", "Int16x16", "Uint16x16", "Int32x8", "Uint32x8", "Int64x4", "Uint64x4", "Float32x8", "Float64x4",
		"Int8x64", "Uint8x64", "Int16x32", "Uint16x32", "Int32x16", "Uint32x16", "Int64x8", "Uint64x8", "Float32x16", "Float64x8"} {
		simdOperations[simdKey{"", "Load" + name + "Array"}] = simdOperation{llssa.SIMDLoad, simdLoad, 0}
		simdOperations[simdKey{"", "Broadcast" + name}] = simdOperation{llssa.SIMDBroadcast, simdBroadcast, 0}
	}
	for _, name := range []string{"Mask8x32", "Mask16x16", "Mask32x8", "Mask64x4", "Mask8x64", "Mask16x32", "Mask32x16", "Mask64x8"} {
		simdOperations[simdKey{"", name + "FromBits"}] = simdOperation{llssa.SIMDMaskFromBits, simdMaskFromBits, 0}
	}
	for _, name := range []string{"ToInt8x32", "ToInt16x16", "ToInt32x8", "ToInt64x4", "ToInt8x64", "ToInt16x32", "ToInt32x16", "ToInt64x8"} {
		simdOperations[simdKey{"mask", name}] = simdOperation{llssa.SIMDBitcast, simdBitcast, 0}
	}
}

// Go's amd64 partial-load helpers mask an ordinary full-array load. LLVM
// needs the mask on the load itself so inactive lanes never access memory.
func simdPartialLoadType(f *ssa.Function, arch string) types.Type {
	obj, ok := f.Object().(*types.Func)
	if arch != "amd64" || !ok || obj.Pkg() == nil || obj.Pkg().Path() != "simd/archsimd" ||
		!strings.HasSuffix(obj.Name(), "Part") || !types.Identical(obj.Type(), f.Signature) {
		return nil
	}
	sig := f.Signature
	if sig.Recv() != nil || sig.Params().Len() != 1 || sig.Results().Len() != 2 ||
		!types.Identical(sig.Results().At(1).Type(), types.Typ[types.Int]) {
		return nil
	}
	result := sig.Results().At(0).Type()
	lanes, ok := llssa.SIMDNumericShape(result)
	if !ok || !types.Identical(sig.Params().At(0).Type(), types.NewSlice(lanes.Elem())) ||
		obj.Name() != "Load"+types.Unalias(result).(*types.Named).Obj().Name()+"Part" {
		return nil
	}
	return result
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
	if obj.Name() == "ClearAVXUpperBits" && sig.Recv() == nil && sig.Params().Len() == 0 && sig.Results().Len() == 0 && !sig.Variadic() {
		if arch == "amd64" {
			return simdOperation{op: llssa.SIMDClearAVXUpperBits}, true
		}
		return fallback()
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
	if arch == "arm64" && key == (simdKey{"numeric", "MulAdd"}) {
		lanes, _ := llssa.SIMDNumericShape(vector)
		if lanes.Elem().Underlying().(*types.Basic).Info()&types.IsInteger != 0 && lanes.Len() >= 4 {
			desc = simdOperation{llssa.SIMDIntegerMulAdd, simdTernary, types.IsInteger}
		}
	}
	switch desc.op {
	case llssa.SIMDShiftSaturated, llssa.SIMDReduceSum, llssa.SIMDReduceMin, llssa.SIMDReduceMax:
		if arch != "arm64" {
			return fallback()
		}
	}
	if !ok || !desc.matches(sig, vector) {
		return fallback()
	}
	if desc.op == llssa.SIMDLookupOrZero && arch != "arm64" && arch != "wasm" {
		return fallback()
	}
	if (desc.signature == simdPairSelector || desc.signature == simdPairHalfSelectors || desc.signature == simdScalarSelector) && arch != "amd64" {
		return fallback()
	}
	if desc.signature == simdConvert {
		shape, _ := llssa.SIMDNumericShape(sig.Results().At(0).Type())
		elem := shape.Elem().(*types.Basic).Name()
		if key.name != "ConvertTo"+strings.ToUpper(elem[:1])+elem[1:] {
			return fallback()
		}
	}
	if desc.signature == simdSplat {
		shape, _ := llssa.SIMDNumericShape(sig.Results().At(0).Type())
		if key.name != fmt.Sprintf("broadcast1To%d", shape.Len()) {
			return fallback()
		}
	}
	if desc.signature == simdIntegerResize {
		// The spelling describes the result shape, not just the operation.
		shape, _ := llssa.SIMDNumericShape(sig.Results().At(0).Type())
		elem := shape.Elem().(*types.Basic).Name()
		elem = strings.ToUpper(elem[:1]) + elem[1:]
		name := "SaturateTo" + elem
		switch desc.op {
		case llssa.SIMDExtend:
			name = "ExtendTo" + elem
		case llssa.SIMDTruncate:
			name = "TruncTo" + elem
		case llssa.SIMDSaturateConcatGrouped:
			name += "ConcatGrouped"
		case llssa.SIMDExtendLo:
			name = fmt.Sprintf("ExtendLo%dTo%s", shape.Len(), elem)
		case llssa.SIMDExtendHi:
			name = fmt.Sprintf("ExtendHi%dTo%s", shape.Len(), elem)
		case llssa.SIMDSaturateConcat:
			name += "Concat"
		}
		if key.name != name {
			return fallback()
		}
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
	// Check operation-specific widths before the general wide-operation gate,
	// so malformed plain and grouped declarations use the same shape validation.
	switch d.op {
	case llssa.SIMDInterleaveLoGrouped, llssa.SIMDInterleaveHiGrouped:
		if simdVectorBits(vector) < 256 {
			return false
		}
	case llssa.SIMDInterleaveLo, llssa.SIMDInterleaveHi,
		llssa.SIMDConcatEven, llssa.SIMDConcatOdd, llssa.SIMDInterleaveEven, llssa.SIMDInterleaveOdd:
		if simdVectorBits(vector) != 128 {
			return false
		}
	}
	// Extending type recognition must not silently enable width-specific
	// handlers, or cross-width conversions with different lane semantics.
	width := simdVectorBits(vector)
	wide := width > 128
	for _, tuple := range []*types.Tuple{sig.Params(), sig.Results()} {
		for i := 0; i < tuple.Len(); i++ {
			wide = wide || simdVectorBits(tuple.At(i).Type()) > 128
		}
	}
	if wide && !d.supportsWide() {
		return false
	}
	if d.elements != 0 && lanes.Elem().Underlying().(*types.Basic).Info()&d.elements == 0 {
		return false
	}
	if (d.op == llssa.SIMDLookupOrZero || d.op == llssa.SIMDPermuteOrZero) && lanes.Len() != 16 {
		return false
	}
	if (d.op == llssa.SIMDCarrylessMultiply || d.op == llssa.SIMDCarrylessMultiplyLo) &&
		!types.Identical(lanes.Elem(), types.Typ[types.Uint64]) {
		return false
	}
	if (d.op == llssa.SIMDConcatAddPairs || d.op == llssa.SIMDConcatSubPairs) && simdVectorBits(vector) != 128 ||
		(d.op == llssa.SIMDConcatAddPairsGrouped || d.op == llssa.SIMDConcatSubPairsGrouped) && simdVectorBits(vector) != 256 {
		return false
	}
	var params []types.Type
	result := vector
	switch d.signature {
	case simdConcatShift:
		n := int64(16)
		if d.op == llssa.SIMDConcatShiftBytesGrouped {
			n = max(32, width/8)
		}
		if lanes.Len() != n || !types.Identical(lanes.Elem(), types.Typ[types.Uint8]) {
			return false
		}
		params = []types.Type{vector, types.Typ[types.Uint64]}
	case simdBool:
		result = types.Typ[types.Bool]
	case simdScalarSelector:
		bits, elemBits := int64(128), int64(16)
		switch d.op {
		case llssa.SIMDPermuteScalarsGrouped, llssa.SIMDPermuteScalarsHiGrouped, llssa.SIMDPermuteScalarsLoGrouped:
			bits = max(256, width)
		}
		if d.op == llssa.SIMDPermuteScalars || d.op == llssa.SIMDPermuteScalarsGrouped {
			elemBits = 32
		}
		if simdVectorBits(vector) != bits || bits/lanes.Len() != elemBits {
			return false
		}
		params = []types.Type{types.Typ[types.Uint8]}
	case simdPairSelector:
		bits := int64(128)
		if d.op == llssa.SIMDConcatSelectedGrouped {
			bits = max(256, width)
		}
		elemBits := simdVectorBits(vector) / lanes.Len()
		if simdVectorBits(vector) != bits || elemBits != 32 && elemBits != 64 {
			return false
		}
		params = []types.Type{types.Typ[types.Uint8], vector}
	case simdPairHalfSelectors:
		if simdVectorBits(vector) != 256 {
			return false
		}
		params = []types.Type{types.Typ[types.Uint8], types.Typ[types.Uint8], vector}
	case simdSplat:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		shape, ok := llssa.SIMDNumericShape(result)
		if !ok || !types.Identical(shape.Elem(), lanes.Elem()) || shape.Len() < lanes.Len() {
			return false
		}
	case simdHalf:
		if width < 256 || sig.Results().Len() != 1 {
			return false
		}
		var half types.Type
		if d.op == llssa.SIMDGetLo || d.op == llssa.SIMDGetHi {
			result = sig.Results().At(0).Type()
			half = result
		} else {
			if sig.Params().Len() != 1 {
				return false
			}
			half = sig.Params().At(0).Type()
			params = []types.Type{half}
		}
		shape, ok := llssa.SIMDNumericShape(half)
		if !ok || 2*simdVectorBits(half) != width || !types.Identical(shape.Elem(), lanes.Elem()) {
			return false
		}
	case simdScalarShift:
		params = []types.Type{types.Typ[types.Uint64]}
	case simdCarrylessSelector:
		params = []types.Type{types.Typ[types.Uint8], vector}
	case simdIntegerResize:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		shape, ok := llssa.SIMDNumericShape(result)
		if !ok {
			return false
		}
		from := lanes.Elem().Underlying().(*types.Basic).Info()
		to := shape.Elem().Underlying().(*types.Basic).Info()
		if to&types.IsInteger == 0 {
			return false
		}
		fromBits := simdVectorBits(vector) / lanes.Len()
		toBits := simdVectorBits(result) / shape.Len()
		switch d.op {
		case llssa.SIMDExtend, llssa.SIMDExtendLo, llssa.SIMDExtendHi:
			if toBits <= fromBits || shape.Len() > lanes.Len() || from&types.IsUnsigned != to&types.IsUnsigned {
				return false
			}
			if (d.op == llssa.SIMDExtend) != (shape.Len() == lanes.Len()) {
				return false
			}
			if d.op == llssa.SIMDExtendHi && shape.Len()*2 != lanes.Len() {
				return false
			}
		case llssa.SIMDSaturateConcat, llssa.SIMDSaturateConcatGrouped:
			bits := int64(128)
			if d.op == llssa.SIMDSaturateConcatGrouped {
				bits = max(256, width)
			}
			if simdVectorBits(vector) != bits || simdVectorBits(result) != bits || shape.Len() != lanes.Len()*2 || from&types.IsUnsigned != 0 {
				return false
			}
			params = []types.Type{vector}
		default:
			// Unary narrowing keeps all source lanes, then pads a short result
			// to the minimum 128-bit vector with zeroes.
			if toBits >= fromBits || shape.Len() != max(lanes.Len(), 128/toBits) {
				return false
			}
			if d.op == llssa.SIMDTruncate {
				if from&types.IsUnsigned != to&types.IsUnsigned {
					return false
				}
			} else if from&types.IsUnsigned != 0 && to&types.IsUnsigned == 0 {
				return false
			}
		}
	case simdWidenBinary:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		widened, ok := llssa.SIMDNumericShape(result)
		if !ok || widened.Len()*2 != lanes.Len() {
			return false
		}
		from := lanes.Elem().Underlying().(*types.Basic).Info()
		to := widened.Elem().Underlying().(*types.Basic).Info()
		if to&types.IsInteger == 0 || from&types.IsUnsigned != to&types.IsUnsigned {
			return false
		}
		params = []types.Type{vector}
	case simdStoreMasked:
		if sig.Params().Len() != 2 {
			return false
		}
		maskType := sig.Params().At(1).Type()
		mask, ok := llssa.SIMDMaskShape(maskType)
		if !ok || mask.Len() != lanes.Len() {
			return false
		}
		params, result = []types.Type{types.NewPointer(lanes), maskType}, nil
	case simdUnsignedVector, simdSignedVector:
		if sig.Params().Len() != 1 {
			return false
		}
		counts := sig.Params().At(0).Type()
		shape, ok := llssa.SIMDNumericShape(counts)
		if !ok || shape.Len() != lanes.Len() {
			return false
		}
		info := shape.Elem().Underlying().(*types.Basic).Info()
		if info&types.IsInteger == 0 || (info&types.IsUnsigned != 0) != (d.signature == simdUnsignedVector) {
			return false
		}
		params = []types.Type{counts}
	case simdMaskFromBits, simdMaskToBits:
		if _, ok := llssa.SIMDMaskShape(vector); !ok {
			return false
		}
		bits := types.Typ[types.Uint8]
		if lanes.Len() == 16 {
			bits = types.Typ[types.Uint16]
		} else if lanes.Len() == 32 {
			bits = types.Typ[types.Uint32]
		} else if lanes.Len() == 64 {
			bits = types.Typ[types.Uint64]
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
		if !ok || mask.Len() != lanes.Len() || simdVectorBits(result) != simdVectorBits(vector) {
			return false
		}
		if d.signature == simdCompare {
			params = []types.Type{vector}
		}
	case simdMaskedBlend:
		if sig.Params().Len() != 2 {
			return false
		}
		maskType := sig.Params().At(1).Type()
		mask, ok := llssa.SIMDMaskShape(maskType)
		if !ok || mask.Len() != lanes.Len() || simdVectorBits(maskType) != width {
			return false
		}
		params = []types.Type{vector, maskType}
	case simdTernary:
		params = []types.Type{vector, vector}
	case simdBitcast, simdConvert, simdConvertLo:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		shape, ok := llssa.SIMDNumericShape(result)
		if !ok {
			return false
		}
		if d.signature == simdBitcast {
			if simdVectorBits(result) != simdVectorBits(vector) {
				return false
			}
			break
		}
		fromBits, toBits := simdVectorBits(vector)/lanes.Len(), simdVectorBits(result)/shape.Len()
		from, to := lanes.Elem().Underlying().(*types.Basic).Info(), shape.Elem().Underlying().(*types.Basic).Info()
		if from&types.IsFloat != 0 && to&types.IsFloat != 0 && fromBits == toBits {
			return false
		}
		if d.signature == simdConvertLo {
			if to&types.IsFloat == 0 || fromBits != 32 || toBits != 64 || lanes.Len() != 4 || shape.Len() != 2 {
				return false
			}
		} else if from&types.IsFloat == 0 && to&types.IsFloat == 0 {
			// Same-width integer ConvertTo methods reinterpret signedness;
			// integer extension/truncation uses the explicit resize operations.
			if fromBits != toBits || shape.Len() != lanes.Len() {
				return false
			}
		} else if shape.Len() != max(lanes.Len(), 128/toBits) {
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

func simdVectorBits(typ types.Type) int64 {
	if shape, ok := llssa.SIMDVectorShape(typ); ok {
		return (&types.StdSizes{WordSize: 8, MaxAlign: 8}).Sizeof(shape) * 8
	}
	return 0
}

// Wide operations include the portable256 bridge's intrinsic dependencies.
// Width-changing operations validate their lane counts and element sizes
// explicitly rather than inheriting SIMD128 support by name.
func (d simdOperation) supportsWide() bool {
	switch d.op {
	case llssa.SIMDIsNaN, llssa.SIMDIsZero, llssa.SIMDConcatShiftBytesGrouped, llssa.SIMDOnesCount, llssa.SIMDAdd, llssa.SIMDSub, llssa.SIMDMul, llssa.SIMDDiv,
		llssa.SIMDInterleaveLoGrouped, llssa.SIMDInterleaveHiGrouped,
		llssa.SIMDConcatSelectedGrouped, llssa.SIMDConcatPermute128,
		llssa.SIMDPermuteScalarsGrouped, llssa.SIMDPermuteScalarsHiGrouped, llssa.SIMDPermuteScalarsLoGrouped,
		llssa.SIMDConcatAddPairsGrouped, llssa.SIMDConcatSubPairsGrouped,
		llssa.SIMDShiftAllLeft, llssa.SIMDShiftAllRight,
		llssa.SIMDAnd, llssa.SIMDOr, llssa.SIMDXor, llssa.SIMDAndNot, llssa.SIMDOrNot,
		llssa.SIMDNeg, llssa.SIMDNot, llssa.SIMDAbs,
		llssa.SIMDAddSaturated, llssa.SIMDSubSaturated, llssa.SIMDAverage,
		llssa.SIMDMin, llssa.SIMDMax, llssa.SIMDSqrt, llssa.SIMDFMA,
		llssa.SIMDConvert, llssa.SIMDCarrylessMultiply,
		llssa.SIMDCeil, llssa.SIMDFloor, llssa.SIMDTrunc, llssa.SIMDRound,
		llssa.SIMDExtend, llssa.SIMDExtendLo, llssa.SIMDExtendHi,
		llssa.SIMDSaturate, llssa.SIMDTruncate, llssa.SIMDSaturateConcatGrouped,
		llssa.SIMDLoad, llssa.SIMDStore, llssa.SIMDStoreMasked, llssa.SIMDBroadcast,
		llssa.SIMDSplatLane0, llssa.SIMDExtractLane, llssa.SIMDInsertLane,
		llssa.SIMDGetLo, llssa.SIMDGetHi, llssa.SIMDSetLo, llssa.SIMDSetHi,
		llssa.SIMDMaskFromBits, llssa.SIMDMaskToBits,
		llssa.SIMDEqual, llssa.SIMDNotEqual, llssa.SIMDLess, llssa.SIMDLessEqual,
		llssa.SIMDGreater, llssa.SIMDGreaterEqual, llssa.SIMDToMask,
		llssa.SIMDBitcast, llssa.SIMDBitSelect, llssa.SIMDBitSelectNot, llssa.SIMDBlend:
		return true
	}
	return false
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
	return p.prog.Type(sig.Results().At(0).Type(), llssa.GoDecl)
}
