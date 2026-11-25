package cl

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/xgo-dev/llgo/cmd/compiler/llvmgen"
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
	simdMaskedUnary
	simdFunnelVector
	simdFunnelScalar
	simdTernaryTable
	simdMaskedSplat
	simdTwoSourcePermute
	simdMaskFromBits
	simdMaskToBits
	simdScalarShift
	simdUnsignedVector
	simdSignedVector
	simdConvert
	simdWidenBinary
	simdDotMixed
	simdSad
	simdCarrylessSelector
	simdStoreMasked
	simdIntegerResize
	simdSplat
	simdHalf
	simdConvertLo
	simdPairSelector
	simdPairHalfSelectors
	simdScalarSelector
	simdAESRound
	simdGaloisAffine
	simdByteImmediate
	simdBool
	simdConcatShift
)

type simdOperation struct {
	op        llvmgen.SIMDOp
	signature simdSignature
	elements  types.BasicInfo
}

// The default entry gives not-yet-implemented intrinsic declarations a defined,
// recoverable failure. Adding an implementation replaces this fallback for that
// operation; functions with Go bodies continue through normal compilation.
var simdOperations = map[simdKey]simdOperation{
	{"*", "*"}:                                  {llvmgen.SIMDUnimplemented, simdUnsupported, 0},
	{"numeric", "IsNaN"}:                        {llvmgen.SIMDIsNaN, simdToMask, types.IsFloat},
	{"numeric", "IsZero"}:                       {llvmgen.SIMDIsZero, simdBool, types.IsInteger},
	{"numeric", "ConcatShiftBytesRight"}:        {llvmgen.SIMDConcatShiftBytes, simdConcatShift, types.IsUnsigned},
	{"numeric", "ConcatShiftBytesRightGrouped"}: {llvmgen.SIMDConcatShiftBytesGrouped, simdConcatShift, types.IsUnsigned},
	{"numeric", "OnesCount"}:                    {llvmgen.SIMDOnesCount, simdUnary, types.IsInteger},
	{"numeric", "ShiftSaturated"}:               {llvmgen.SIMDShiftSaturated, simdSignedVector, types.IsInteger},
	{"numeric", "reduceSum"}:                    {llvmgen.SIMDReduceSum, simdUnary, types.IsInteger},
	{"numeric", "reduceMin"}:                    {llvmgen.SIMDReduceMin, simdUnary, 0},
	{"numeric", "reduceMax"}:                    {llvmgen.SIMDReduceMax, simdUnary, 0},
	{"numeric", "Add"}:                          {llvmgen.SIMDAdd, simdBinary, 0},
	{"numeric", "Sub"}:                          {llvmgen.SIMDSub, simdBinary, 0},
	{"numeric", "And"}:                          {llvmgen.SIMDAnd, simdBinary, types.IsInteger},
	{"numeric", "Or"}:                           {llvmgen.SIMDOr, simdBinary, types.IsInteger},
	{"numeric", "Xor"}:                          {llvmgen.SIMDXor, simdBinary, types.IsInteger},
	{"numeric", "GetElem"}:                      {llvmgen.SIMDExtractLane, simdExtract, 0},
	{"numeric", "SetElem"}:                      {llvmgen.SIMDInsertLane, simdInsert, 0},
	{"numeric", "StoreArray"}:                   {llvmgen.SIMDStore, simdStore, 0},
	{"numeric", "Mul"}:                          {llvmgen.SIMDMul, simdBinary, 0},
	{"numeric", "Div"}:                          {llvmgen.SIMDDiv, simdBinary, types.IsFloat},
	{"numeric", "AndNot"}:                       {llvmgen.SIMDAndNot, simdBinary, types.IsInteger},
	{"numeric", "OrNot"}:                        {llvmgen.SIMDOrNot, simdBinary, types.IsInteger},
	{"numeric", "Not"}:                          {llvmgen.SIMDNot, simdUnary, types.IsInteger},
	{"numeric", "Neg"}:                          {llvmgen.SIMDNeg, simdUnary, 0},
	{"numeric", "Abs"}:                          {llvmgen.SIMDAbs, simdUnary, 0},
	{"numeric", "Sqrt"}:                         {llvmgen.SIMDSqrt, simdUnary, types.IsFloat},
	{"numeric", "Ceil"}:                         {llvmgen.SIMDCeil, simdUnary, types.IsFloat},
	{"numeric", "Floor"}:                        {llvmgen.SIMDFloor, simdUnary, types.IsFloat},
	{"numeric", "Trunc"}:                        {llvmgen.SIMDTrunc, simdUnary, types.IsFloat},
	{"numeric", "Round"}:                        {llvmgen.SIMDRound, simdUnary, types.IsFloat},
	{"numeric", "Equal"}:                        {llvmgen.SIMDEqual, simdCompare, 0},
	{"numeric", "NotEqual"}:                     {llvmgen.SIMDNotEqual, simdCompare, 0},
	{"numeric", "Less"}:                         {llvmgen.SIMDLess, simdCompare, 0},
	{"numeric", "LessEqual"}:                    {llvmgen.SIMDLessEqual, simdCompare, 0},
	{"numeric", "Greater"}:                      {llvmgen.SIMDGreater, simdCompare, 0},
	{"numeric", "GreaterEqual"}:                 {llvmgen.SIMDGreaterEqual, simdCompare, 0},
	{"numeric", "ToMask"}:                       {llvmgen.SIMDToMask, simdToMask, types.IsInteger},
	{"numeric", "asMask"}:                       {llvmgen.SIMDBitcast, simdToMask, types.IsInteger},
	{"numeric", "bitSelect"}:                    {llvmgen.SIMDBitSelect, simdTernary, types.IsInteger},
	{"numeric", "BitSelect"}:                    {llvmgen.SIMDBitSelect, simdTernary, types.IsInteger},
	{"numeric", "bitSelectNot"}:                 {llvmgen.SIMDBitSelectNot, simdTernary, types.IsInteger},
	{"numeric", "blendMasked"}:                  {llvmgen.SIMDBlend, simdMaskedBlend, types.IsInteger},
	{"numeric", "blend"}:                        {llvmgen.SIMDBlend, simdTernary, types.IsInteger},
	{"mask", "And"}:                             {llvmgen.SIMDAnd, simdBinary, 0},
	{"mask", "Or"}:                              {llvmgen.SIMDOr, simdBinary, 0},
	{"mask", "Xor"}:                             {llvmgen.SIMDXor, simdBinary, 0},
	{"mask", "AndNot"}:                          {llvmgen.SIMDAndNot, simdBinary, 0},
	{"mask", "Not"}:                             {llvmgen.SIMDNot, simdUnary, 0},
	{"numeric", "ShiftAllLeft"}:                 {llvmgen.SIMDShiftAllLeft, simdScalarShift, types.IsInteger},
	{"numeric", "ShiftAllRight"}:                {llvmgen.SIMDShiftAllRight, simdScalarShift, types.IsInteger},
	{"numeric", "ShiftLeft"}:                    {llvmgen.SIMDShiftLeft, simdUnsignedVector, types.IsInteger},
	{"numeric", "ShiftRight"}:                   {llvmgen.SIMDShiftRight, simdUnsignedVector, types.IsInteger},
	{"numeric", "Shift"}:                        {llvmgen.SIMDShift, simdSignedVector, types.IsInteger},
	{"numeric", "AddSaturated"}:                 {llvmgen.SIMDAddSaturated, simdBinary, types.IsInteger},
	{"numeric", "SubSaturated"}:                 {llvmgen.SIMDSubSaturated, simdBinary, types.IsInteger},
	{"numeric", "Min"}:                          {llvmgen.SIMDMin, simdBinary, 0},
	{"numeric", "LookupOrZero"}:                 {llvmgen.SIMDLookupOrZero, simdBinary, types.IsInteger},
	{"numeric", "PermuteOrZeroGrouped"}:         {llvmgen.SIMDPermuteOrZeroGrouped, simdSignedVector, types.IsInteger},
	{"numeric", "ConcatPermute"}:                {llvmgen.SIMDConcatPermute, simdTwoSourcePermute, 0},
	{"numeric", "Compress"}:                     {llvmgen.SIMDCompress, simdMaskedUnary, 0},
	{"numeric", "Expand"}:                       {llvmgen.SIMDExpand, simdMaskedUnary, 0},
	{"numeric", "PermuteOrZero"}:                {llvmgen.SIMDPermuteOrZero, simdSignedVector, types.IsInteger},
	{"numeric", "Permute"}:                      {llvmgen.SIMDPermute, simdUnsignedVector, 0},
	{"numeric", "Max"}:                          {llvmgen.SIMDMax, simdBinary, 0},
	{"numeric", "GetLo"}:                        {llvmgen.SIMDGetLo, simdHalf, 0},
	{"numeric", "GetHi"}:                        {llvmgen.SIMDGetHi, simdHalf, 0},
	{"numeric", "SetLo"}:                        {llvmgen.SIMDSetLo, simdHalf, 0},
	{"numeric", "SetHi"}:                        {llvmgen.SIMDSetHi, simdHalf, 0},

	{"numeric", "InterleaveLo"}:                  {llvmgen.SIMDInterleaveLo, simdBinary, types.IsInteger},
	{"numeric", "InterleaveHi"}:                  {llvmgen.SIMDInterleaveHi, simdBinary, types.IsInteger},
	{"numeric", "InterleaveLoGrouped"}:           {llvmgen.SIMDInterleaveLoGrouped, simdBinary, types.IsInteger},
	{"numeric", "InterleaveHiGrouped"}:           {llvmgen.SIMDInterleaveHiGrouped, simdBinary, types.IsInteger},
	{"numeric", "ConcatEven"}:                    {llvmgen.SIMDConcatEven, simdBinary, types.IsInteger},
	{"numeric", "ConcatOdd"}:                     {llvmgen.SIMDConcatOdd, simdBinary, types.IsInteger},
	{"numeric", "InterleaveEven"}:                {llvmgen.SIMDInterleaveEven, simdBinary, types.IsInteger},
	{"numeric", "InterleaveOdd"}:                 {llvmgen.SIMDInterleaveOdd, simdBinary, types.IsInteger},
	{"numeric", "concatSelectedConstant"}:        {llvmgen.SIMDConcatSelected, simdPairSelector, 0},
	{"numeric", "concatSelectedConstantGrouped"}: {llvmgen.SIMDConcatSelectedGrouped, simdPairSelector, 0},
	{"numeric", "ConcatPermute128Scalars"}:       {llvmgen.SIMDConcatPermute128, simdPairHalfSelectors, 0},
	{"numeric", "permuteScalars"}:                {llvmgen.SIMDPermuteScalars, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsGrouped"}:         {llvmgen.SIMDPermuteScalarsGrouped, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsHi"}:              {llvmgen.SIMDPermuteScalarsHi, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsLo"}:              {llvmgen.SIMDPermuteScalarsLo, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsHiGrouped"}:       {llvmgen.SIMDPermuteScalarsHiGrouped, simdScalarSelector, types.IsInteger},
	{"numeric", "permuteScalarsLoGrouped"}:       {llvmgen.SIMDPermuteScalarsLoGrouped, simdScalarSelector, types.IsInteger},

	{"numeric", "ConvertLo2ToFloat64"}: {llvmgen.SIMDConvertLo, simdConvertLo, 0},

	{"numeric", "ConcatAddPairs"}:                    {llvmgen.SIMDConcatAddPairs, simdBinary, 0},
	{"numeric", "ConcatAddPairsGrouped"}:             {llvmgen.SIMDConcatAddPairsGrouped, simdBinary, 0},
	{"numeric", "ConcatSubPairs"}:                    {llvmgen.SIMDConcatSubPairs, simdBinary, 0},
	{"numeric", "ConcatSubPairsGrouped"}:             {llvmgen.SIMDConcatSubPairsGrouped, simdBinary, 0},
	{"numeric", "Average"}:                           {llvmgen.SIMDAverage, simdBinary, types.IsInteger},
	{"numeric", "MulSign"}:                           {llvmgen.SIMDMulSign, simdBinary, types.IsInteger},
	{"numeric", "MulHigh"}:                           {llvmgen.SIMDMulHigh, simdBinary, types.IsInteger},
	{"numeric", "MulWidenEven"}:                      {llvmgen.SIMDMulWidenEven, simdWidenBinary, types.IsInteger},
	{"numeric", "DotProductPairs"}:                   {llvmgen.SIMDDotProductPairs, simdWidenBinary, types.IsInteger},
	{"numeric", "DotProductPairsSaturated"}:          {llvmgen.SIMDDotProductPairsSaturated, simdDotMixed, types.IsUnsigned},
	{"numeric", "SumOf8AbsDiff"}:                     {llvmgen.SIMDSumOf8AbsDiff, simdSad, types.IsUnsigned},
	{"numeric", "AddOddSubEven"}:                     {llvmgen.SIMDAddOddSubEven, simdBinary, types.IsFloat},
	{"numeric", "MulAddEvenSubOdd"}:                  {llvmgen.SIMDMulAddEvenSubOdd, simdTernary, types.IsFloat},
	{"numeric", "MulAddOddSubEven"}:                  {llvmgen.SIMDMulAddOddSubEven, simdTernary, types.IsFloat},
	{"numeric", "ConcatAddPairsSaturated"}:           {llvmgen.SIMDConcatAddPairsSaturated, simdBinary, types.IsInteger},
	{"numeric", "ConcatAddPairsSaturatedGrouped"}:    {llvmgen.SIMDConcatAddPairsSaturatedGrouped, simdBinary, types.IsInteger},
	{"numeric", "ConcatSubPairsSaturated"}:           {llvmgen.SIMDConcatSubPairsSaturated, simdBinary, types.IsInteger},
	{"numeric", "ConcatSubPairsSaturatedGrouped"}:    {llvmgen.SIMDConcatSubPairsSaturatedGrouped, simdBinary, types.IsInteger},
	{"numeric", "Reciprocal"}:                        {llvmgen.SIMDReciprocal, simdUnary, types.IsFloat},
	{"numeric", "ReciprocalSqrt"}:                    {llvmgen.SIMDReciprocalSqrt, simdUnary, types.IsFloat},
	{"numeric", "Scale"}:                             {llvmgen.SIMDScale, simdBinary, types.IsFloat},
	{"numeric", "CeilScaled"}:                        {llvmgen.SIMDCeilScaled, simdByteImmediate, types.IsFloat},
	{"numeric", "CeilScaledResidue"}:                 {llvmgen.SIMDCeilScaledResidue, simdByteImmediate, types.IsFloat},
	{"numeric", "FloorScaled"}:                       {llvmgen.SIMDFloorScaled, simdByteImmediate, types.IsFloat},
	{"numeric", "FloorScaledResidue"}:                {llvmgen.SIMDFloorScaledResidue, simdByteImmediate, types.IsFloat},
	{"numeric", "RoundScaled"}:                       {llvmgen.SIMDRoundScaled, simdByteImmediate, types.IsFloat},
	{"numeric", "RoundScaledResidue"}:                {llvmgen.SIMDRoundScaledResidue, simdByteImmediate, types.IsFloat},
	{"numeric", "TruncScaled"}:                       {llvmgen.SIMDTruncScaled, simdByteImmediate, types.IsFloat},
	{"numeric", "TruncScaledResidue"}:                {llvmgen.SIMDTruncScaledResidue, simdByteImmediate, types.IsFloat},
	{"numeric", "AESEncryptLastRound"}:               {llvmgen.SIMDAESEncryptLastRound, simdAESRound, types.IsUnsigned},
	{"numeric", "AESEncryptOneRound"}:                {llvmgen.SIMDAESEncryptOneRound, simdAESRound, types.IsUnsigned},
	{"numeric", "AESDecryptLastRound"}:               {llvmgen.SIMDAESDecryptLastRound, simdAESRound, types.IsUnsigned},
	{"numeric", "AESDecryptOneRound"}:                {llvmgen.SIMDAESDecryptOneRound, simdAESRound, types.IsUnsigned},
	{"numeric", "AESInvMixColumns"}:                  {llvmgen.SIMDAESInvMixColumns, simdUnary, types.IsUnsigned},
	{"numeric", "AESRoundKeyGenAssist"}:              {llvmgen.SIMDAESRoundKeyGenAssist, simdByteImmediate, types.IsUnsigned},
	{"numeric", "SHA1FourRounds"}:                    {llvmgen.SIMDSHA1FourRounds, simdCarrylessSelector, types.IsUnsigned},
	{"numeric", "SHA1Message1"}:                      {llvmgen.SIMDSHA1Message1, simdBinary, types.IsUnsigned},
	{"numeric", "SHA1Message2"}:                      {llvmgen.SIMDSHA1Message2, simdBinary, types.IsUnsigned},
	{"numeric", "SHA1NextE"}:                         {llvmgen.SIMDSHA1NextE, simdBinary, types.IsUnsigned},
	{"numeric", "SHA256Message1"}:                    {llvmgen.SIMDSHA256Message1, simdBinary, types.IsUnsigned},
	{"numeric", "SHA256Message2"}:                    {llvmgen.SIMDSHA256Message2, simdBinary, types.IsUnsigned},
	{"numeric", "SHA256TwoRounds"}:                   {llvmgen.SIMDSHA256TwoRounds, simdTernary, types.IsUnsigned},
	{"numeric", "GaloisFieldMul"}:                    {llvmgen.SIMDGaloisFieldMul, simdBinary, types.IsUnsigned},
	{"numeric", "GaloisFieldAffineTransform"}:        {llvmgen.SIMDGaloisFieldAffineTransform, simdGaloisAffine, types.IsUnsigned},
	{"numeric", "GaloisFieldAffineTransformInverse"}: {llvmgen.SIMDGaloisFieldAffineTransformInverse, simdGaloisAffine, types.IsUnsigned},
	{"numeric", "MulAdd"}:                            {llvmgen.SIMDFMA, simdTernary, types.IsFloat},
	{"numeric", "LeadingZeros"}:                      {llvmgen.SIMDLeadingZeros, simdUnary, types.IsInteger},
	{"numeric", "LeadingSignBits"}:                   {llvmgen.SIMDLeadingSignBits, simdUnary, types.IsInteger},
	{"numeric", "MulWidenHi"}:                        {llvmgen.SIMDMulWidenHi, simdWidenBinary, types.IsInteger},
	{"numeric", "MulWidenLo"}:                        {llvmgen.SIMDMulWidenLo, simdWidenBinary, types.IsInteger},
	{"numeric", "RotateLeft"}:                        {llvmgen.SIMDRotateLeft, simdBinary, types.IsInteger},
	{"numeric", "RotateRight"}:                       {llvmgen.SIMDRotateRight, simdBinary, types.IsInteger},
	{"numeric", "tern"}:                              {llvmgen.SIMDTernaryLogic, simdTernaryTable, types.IsInteger},
	{"numeric", "RotateAllLeft"}:                     {llvmgen.SIMDRotateAllLeft, simdScalarShift, types.IsInteger},
	{"numeric", "RotateAllRight"}:                    {llvmgen.SIMDRotateAllRight, simdScalarShift, types.IsInteger},
	{"numeric", "StoreArrayMasked"}:                  {llvmgen.SIMDStoreMasked, simdStoreMasked, 0},
	{"numeric", "carrylessMultiply"}:                 {llvmgen.SIMDCarrylessMultiply, simdCarrylessSelector, types.IsUnsigned},
	{"numeric", "carrylessMultiplyWidenLo"}:          {llvmgen.SIMDCarrylessMultiplyLo, simdBinary, types.IsUnsigned},
}

// These registrations share lowering but retain exact declaration names and
// signature checks. Source Go slice helpers keep their own bounds checks.
func init() {
	for _, name := range []string{"ConvertToInt8", "ConvertToUint8", "ConvertToInt16", "ConvertToUint16", "ConvertToInt32", "ConvertToUint32", "ConvertToInt64", "ConvertToUint64", "ConvertToFloat32", "ConvertToFloat64"} {
		simdOperations[simdKey{"numeric", name}] = simdOperation{llvmgen.SIMDConvert, simdConvert, 0}
	}
	for _, elem := range []string{"Int16", "Uint16", "Int32", "Uint32", "Int64", "Uint64"} {
		simdOperations[simdKey{"numeric", "ExtendTo" + elem}] = simdOperation{llvmgen.SIMDExtend, simdIntegerResize, types.IsInteger}
		for _, lanes := range []int{2, 4, 8, 16} {
			for _, part := range []struct {
				name string
				op   llvmgen.SIMDOp
			}{{"Lo", llvmgen.SIMDExtendLo}, {"Hi", llvmgen.SIMDExtendHi}} {
				name := fmt.Sprintf("Extend%s%dTo%s", part.name, lanes, elem)
				simdOperations[simdKey{"numeric", name}] = simdOperation{part.op, simdIntegerResize, types.IsInteger}
			}
		}
	}
	for _, elem := range []string{"Int8", "Uint8", "Int16", "Uint16", "Int32", "Uint32"} {
		simdOperations[simdKey{"numeric", "SaturateTo" + elem}] = simdOperation{llvmgen.SIMDSaturate, simdIntegerResize, types.IsInteger}
		simdOperations[simdKey{"numeric", "TruncTo" + elem}] = simdOperation{llvmgen.SIMDTruncate, simdIntegerResize, types.IsInteger}
	}
	for _, name := range []string{"SaturateToInt16Concat", "SaturateToUint16Concat"} {
		simdOperations[simdKey{"numeric", name}] = simdOperation{llvmgen.SIMDSaturateConcat, simdIntegerResize, types.IsInteger}
		simdOperations[simdKey{"numeric", name + "Grouped"}] = simdOperation{llvmgen.SIMDSaturateConcatGrouped, simdIntegerResize, types.IsInteger}
	}
	for _, name := range []string{"Mask8x16", "Mask16x8", "Mask32x4", "Mask64x2"} {
		simdOperations[simdKey{"", name + "FromBits"}] = simdOperation{llvmgen.SIMDMaskFromBits, simdMaskFromBits, 0}
	}
	simdOperations[simdKey{"mask", "ToBits"}] = simdOperation{llvmgen.SIMDMaskToBits, simdMaskToBits, 0}
	for _, name := range []string{"ToInt8x16", "ToInt16x8", "ToInt32x4", "ToInt64x2"} {
		simdOperations[simdKey{"mask", name}] = simdOperation{llvmgen.SIMDBitcast, simdBitcast, 0}
	}
	for _, name := range []string{"ToBits", "BitsToInt8", "BitsToInt16", "BitsToInt32", "BitsToInt64", "BitsToFloat32", "BitsToFloat64", "ReshapeToUint8s", "ReshapeToUint16s", "ReshapeToUint32s", "ReshapeToUint64s"} {
		simdOperations[simdKey{"numeric", name}] = simdOperation{llvmgen.SIMDBitcast, simdBitcast, 0}
	}
	for _, name := range []string{"Int8x16", "Uint8x16", "Int16x8", "Uint16x8", "Int32x4", "Uint32x4", "Int64x2", "Uint64x2", "Float32x4", "Float64x2"} {
		simdOperations[simdKey{"", "Load" + name + "Array"}] = simdOperation{llvmgen.SIMDLoad, simdLoad, 0}
		simdOperations[simdKey{"", "Broadcast" + name}] = simdOperation{llvmgen.SIMDBroadcast, simdBroadcast, 0}
	}
	for _, bits := range []int{16, 32, 64} {
		for _, direction := range []struct {
			name string
			op   llvmgen.SIMDOp
		}{{"Left", llvmgen.SIMDFunnelLeft}, {"Right", llvmgen.SIMDFunnelRight}} {
			name := direction.name + "ConcatMod" + fmt.Sprint(bits)
			simdOperations[simdKey{"numeric", "Shift" + name}] = simdOperation{direction.op, simdFunnelVector, types.IsInteger}
			simdOperations[simdKey{"numeric", "ShiftAll" + name}] = simdOperation{direction.op, simdFunnelScalar, types.IsInteger}
		}
	}
	for _, lanes := range []int{2, 4, 8, 16, 32, 64} {
		simdOperations[simdKey{"numeric", fmt.Sprintf("broadcast1To%dMasked", lanes)}] = simdOperation{llvmgen.SIMDSplatMasked, simdMaskedSplat, 0}
		simdOperations[simdKey{"numeric", fmt.Sprintf("broadcast1To%d", lanes)}] = simdOperation{llvmgen.SIMDSplatLane0, simdSplat, 0}
	}
	for _, name := range []string{"Int8x32", "Uint8x32", "Int16x16", "Uint16x16", "Int32x8", "Uint32x8", "Int64x4", "Uint64x4", "Float32x8", "Float64x4",
		"Int8x64", "Uint8x64", "Int16x32", "Uint16x32", "Int32x16", "Uint32x16", "Int64x8", "Uint64x8", "Float32x16", "Float64x8"} {
		simdOperations[simdKey{"", "Load" + name + "Array"}] = simdOperation{llvmgen.SIMDLoad, simdLoad, 0}
		simdOperations[simdKey{"", "Broadcast" + name}] = simdOperation{llvmgen.SIMDBroadcast, simdBroadcast, 0}
	}
	for _, name := range []string{"Mask8x32", "Mask16x16", "Mask32x8", "Mask64x4", "Mask8x64", "Mask16x32", "Mask32x16", "Mask64x8"} {
		simdOperations[simdKey{"", name + "FromBits"}] = simdOperation{llvmgen.SIMDMaskFromBits, simdMaskFromBits, 0}
	}
	for _, name := range []string{"ToInt8x32", "ToInt16x16", "ToInt32x8", "ToInt64x4", "ToInt8x64", "ToInt16x32", "ToInt32x16", "ToInt64x8"} {
		simdOperations[simdKey{"mask", name}] = simdOperation{llvmgen.SIMDBitcast, simdBitcast, 0}
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
	lanes, ok := llvmgen.SIMDNumericShape(result)
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
			return simdOperation{op: llvmgen.SIMDClearAVXUpperBits}, true
		}
		return fallback()
	}
	key := simdKey{name: obj.Name()}
	var vector types.Type
	if recv := sig.Recv(); recv != nil {
		vector = recv.Type()
		if _, ok := llvmgen.SIMDNumericShape(vector); ok {
			key.receiver = "numeric"
		} else if _, ok := llvmgen.SIMDMaskShape(vector); ok {
			key.receiver = "mask"
		} else {
			return fallback()
		}
	} else if sig.Results().Len() == 1 {
		vector = sig.Results().At(0).Type()
	}
	desc, ok := simdOperations[key]
	if arch == "arm64" && key == (simdKey{"numeric", "MulAdd"}) {
		lanes, _ := llvmgen.SIMDNumericShape(vector)
		if lanes.Elem().Underlying().(*types.Basic).Info()&types.IsInteger != 0 && lanes.Len() >= 4 {
			desc = simdOperation{llvmgen.SIMDIntegerMulAdd, simdTernary, types.IsInteger}
		}
	}
	if llvmgen.SIMDX86Special(desc.op) && arch != "amd64" {
		return fallback()
	}
	switch desc.op {
	case llvmgen.SIMDShiftSaturated, llvmgen.SIMDReduceSum, llvmgen.SIMDReduceMin, llvmgen.SIMDReduceMax:
		if arch != "arm64" {
			return fallback()
		}
	}
	if !ok || !desc.matches(sig, vector) {
		return fallback()
	}
	if desc.op == llvmgen.SIMDLookupOrZero && arch != "arm64" && arch != "wasm" {
		return fallback()
	}
	if (desc.signature == simdPairSelector || desc.signature == simdPairHalfSelectors || desc.signature == simdScalarSelector) && arch != "amd64" {
		return fallback()
	}
	if desc.signature == simdConvert {
		shape, _ := llvmgen.SIMDNumericShape(sig.Results().At(0).Type())
		elem := shape.Elem().(*types.Basic).Name()
		if key.name != "ConvertTo"+strings.ToUpper(elem[:1])+elem[1:] {
			return fallback()
		}
	}
	if desc.signature == simdFunnelVector || desc.signature == simdFunnelScalar {
		lanes, _ := llvmgen.SIMDNumericShape(vector)
		if !strings.HasSuffix(key.name, fmt.Sprint(simdVectorBits(vector)/lanes.Len())) {
			return fallback()
		}
	}
	if desc.signature == simdSplat || desc.signature == simdMaskedSplat {
		shape, _ := llvmgen.SIMDNumericShape(sig.Results().At(0).Type())
		if strings.TrimSuffix(key.name, "Masked") != fmt.Sprintf("broadcast1To%d", shape.Len()) {
			return fallback()
		}
	}
	if desc.signature == simdIntegerResize {
		// The spelling describes the result shape, not just the operation.
		shape, _ := llvmgen.SIMDNumericShape(sig.Results().At(0).Type())
		elem := shape.Elem().(*types.Basic).Name()
		elem = strings.ToUpper(elem[:1]) + elem[1:]
		name := "SaturateTo" + elem
		switch desc.op {
		case llvmgen.SIMDExtend:
			name = "ExtendTo" + elem
		case llvmgen.SIMDTruncate:
			name = "TruncTo" + elem
		case llvmgen.SIMDSaturateConcatGrouped:
			name += "ConcatGrouped"
		case llvmgen.SIMDExtendLo:
			name = fmt.Sprintf("ExtendLo%dTo%s", shape.Len(), elem)
		case llvmgen.SIMDExtendHi:
			name = fmt.Sprintf("ExtendHi%dTo%s", shape.Len(), elem)
		case llvmgen.SIMDSaturateConcat:
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
	lanes, ok := llvmgen.SIMDVectorShape(vector)
	if !ok {
		return false
	}
	switch d.op {
	case llvmgen.SIMDAESRoundKeyGenAssist, llvmgen.SIMDAESInvMixColumns, llvmgen.SIMDSHA1FourRounds, llvmgen.SIMDSHA1Message1, llvmgen.SIMDSHA1Message2, llvmgen.SIMDSHA1NextE, llvmgen.SIMDSHA256Message1, llvmgen.SIMDSHA256Message2, llvmgen.SIMDSHA256TwoRounds:
		if lanes.Len() != 4 || !types.Identical(lanes.Elem(), types.Typ[types.Uint32]) {
			return false
		}
	case llvmgen.SIMDGaloisFieldMul:
		if !types.Identical(lanes.Elem(), types.Typ[types.Uint8]) {
			return false
		}
	}
	// Check operation-specific widths before the general wide-operation gate,
	// so malformed plain and grouped declarations use the same shape validation.
	switch d.op {
	case llvmgen.SIMDInterleaveLoGrouped, llvmgen.SIMDInterleaveHiGrouped:
		if simdVectorBits(vector) < 256 {
			return false
		}
	case llvmgen.SIMDInterleaveLo, llvmgen.SIMDInterleaveHi,
		llvmgen.SIMDConcatEven, llvmgen.SIMDConcatOdd, llvmgen.SIMDInterleaveEven, llvmgen.SIMDInterleaveOdd:
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
	if (d.op == llvmgen.SIMDLookupOrZero || d.op == llvmgen.SIMDPermuteOrZero) && lanes.Len() != 16 {
		return false
	}
	if (d.op == llvmgen.SIMDCarrylessMultiply || d.op == llvmgen.SIMDCarrylessMultiplyLo) &&
		!types.Identical(lanes.Elem(), types.Typ[types.Uint64]) {
		return false
	}
	if (d.op == llvmgen.SIMDConcatAddPairs || d.op == llvmgen.SIMDConcatSubPairs) && simdVectorBits(vector) != 128 ||
		(d.op == llvmgen.SIMDConcatAddPairsGrouped || d.op == llvmgen.SIMDConcatSubPairsGrouped) && simdVectorBits(vector) != 256 {
		return false
	}
	var params []types.Type
	result := vector
	switch d.signature {
	case simdAESRound, simdGaloisAffine:
		n := 1
		elem := types.Uint32
		if d.signature == simdGaloisAffine {
			n = 2
			elem = types.Uint64
		}
		if sig.Params().Len() != n || !types.Identical(lanes.Elem(), types.Typ[types.Uint8]) {
			return false
		}
		t := sig.Params().At(0).Type()
		shape, ok := llvmgen.SIMDNumericShape(t)
		if !ok || simdVectorBits(t) != width || !types.Identical(shape.Elem(), types.Typ[elem]) {
			return false
		}
		params = []types.Type{t}
		if n == 2 {
			params = append(params, types.Typ[types.Uint8])
		}
	case simdByteImmediate:
		params = []types.Type{types.Typ[types.Uint8]}
	case simdConcatShift:
		n := int64(16)
		if d.op == llvmgen.SIMDConcatShiftBytesGrouped {
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
		case llvmgen.SIMDPermuteScalarsGrouped, llvmgen.SIMDPermuteScalarsHiGrouped, llvmgen.SIMDPermuteScalarsLoGrouped:
			bits = max(256, width)
		}
		if d.op == llvmgen.SIMDPermuteScalars || d.op == llvmgen.SIMDPermuteScalarsGrouped {
			elemBits = 32
		}
		if simdVectorBits(vector) != bits || bits/lanes.Len() != elemBits {
			return false
		}
		params = []types.Type{types.Typ[types.Uint8]}
	case simdPairSelector:
		bits := int64(128)
		if d.op == llvmgen.SIMDConcatSelectedGrouped {
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
	case simdSplat, simdMaskedSplat:
		if d.signature == simdMaskedSplat {
			if sig.Params().Len() != 1 {
				return false
			}
			typ := sig.Params().At(0).Type()
			shape, ok := llvmgen.SIMDMaskShape(typ)
			if !ok || shape.Len() != lanes.Len() || simdVectorBits(typ) != width {
				return false
			}
			params = []types.Type{typ}
		}
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		shape, ok := llvmgen.SIMDNumericShape(result)
		if !ok || !types.Identical(shape.Elem(), lanes.Elem()) || shape.Len() < lanes.Len() {
			return false
		}
	case simdHalf:
		if width < 256 || sig.Results().Len() != 1 {
			return false
		}
		var half types.Type
		if d.op == llvmgen.SIMDGetLo || d.op == llvmgen.SIMDGetHi {
			result = sig.Results().At(0).Type()
			half = result
		} else {
			if sig.Params().Len() != 1 {
				return false
			}
			half = sig.Params().At(0).Type()
			params = []types.Type{half}
		}
		shape, ok := llvmgen.SIMDNumericShape(half)
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
		shape, ok := llvmgen.SIMDNumericShape(result)
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
		case llvmgen.SIMDExtend, llvmgen.SIMDExtendLo, llvmgen.SIMDExtendHi:
			if toBits <= fromBits || shape.Len() > lanes.Len() || from&types.IsUnsigned != to&types.IsUnsigned {
				return false
			}
			if (d.op == llvmgen.SIMDExtend) != (shape.Len() == lanes.Len()) {
				return false
			}
			if d.op == llvmgen.SIMDExtendHi && shape.Len()*2 != lanes.Len() {
				return false
			}
		case llvmgen.SIMDSaturateConcat, llvmgen.SIMDSaturateConcatGrouped:
			bits := int64(128)
			if d.op == llvmgen.SIMDSaturateConcatGrouped {
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
			if d.op == llvmgen.SIMDTruncate {
				if from&types.IsUnsigned != to&types.IsUnsigned {
					return false
				}
			} else if from&types.IsUnsigned != 0 && to&types.IsUnsigned == 0 {
				return false
			}
		}
	case simdSad, simdDotMixed:
		if sig.Results().Len() != 1 || sig.Params().Len() != 1 || !types.Identical(lanes.Elem(), types.Typ[types.Uint8]) {
			return false
		}
		result = sig.Results().At(0).Type()
		out, ok := llvmgen.SIMDNumericShape(result)
		if !ok || simdVectorBits(result) != width {
			return false
		}
		if d.signature == simdSad {
			if !types.Identical(out.Elem(), types.Typ[types.Uint64]) {
				return false
			}
			params = []types.Type{vector}
		} else {
			y := sig.Params().At(0).Type()
			ys, ok := llvmgen.SIMDNumericShape(y)
			if !ok || ys.Len() != lanes.Len() || !types.Identical(ys.Elem(), types.Typ[types.Int8]) || !types.Identical(out.Elem(), types.Typ[types.Int16]) {
				return false
			}
			params = []types.Type{y}
		}
	case simdWidenBinary:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		widened, ok := llvmgen.SIMDNumericShape(result)
		if !ok || simdVectorBits(result) != width || widened.Len()*2 != lanes.Len() {
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
		mask, ok := llvmgen.SIMDMaskShape(maskType)
		if !ok || mask.Len() != lanes.Len() {
			return false
		}
		params, result = []types.Type{types.NewPointer(lanes), maskType}, nil
	case simdTernaryTable:
		params = []types.Type{types.Typ[types.Uint8], vector, vector}
	case simdFunnelScalar:
		params = []types.Type{vector, types.Typ[types.Uint64]}
	case simdFunnelVector, simdTwoSourcePermute, simdUnsignedVector, simdSignedVector:
		if d.signature == simdTwoSourcePermute || d.signature == simdFunnelVector {
			params = []types.Type{vector}
		}
		if sig.Params().Len() != len(params)+1 {
			return false
		}
		counts := sig.Params().At(len(params)).Type()
		shape, ok := llvmgen.SIMDNumericShape(counts)
		if !ok || shape.Len() != lanes.Len() || simdVectorBits(counts) != width {
			return false
		}
		info := shape.Elem().Underlying().(*types.Basic).Info()
		if info&types.IsInteger == 0 || (info&types.IsUnsigned != 0) != (d.signature != simdSignedVector) {
			return false
		}
		params = append(params, counts)
	case simdMaskFromBits, simdMaskToBits:
		if _, ok := llvmgen.SIMDMaskShape(vector); !ok {
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
		mask, ok := llvmgen.SIMDMaskShape(result)
		if !ok || mask.Len() != lanes.Len() || simdVectorBits(result) != simdVectorBits(vector) {
			return false
		}
		if d.signature == simdCompare {
			params = []types.Type{vector}
		}
	case simdMaskedBlend, simdMaskedUnary:
		if d.signature == simdMaskedBlend {
			params = []types.Type{vector}
		}
		if sig.Params().Len() != len(params)+1 {
			return false
		}
		maskType := sig.Params().At(len(params)).Type()
		mask, ok := llvmgen.SIMDMaskShape(maskType)
		if !ok || mask.Len() != lanes.Len() || simdVectorBits(maskType) != width {
			return false
		}
		params = append(params, maskType)
	case simdTernary:
		params = []types.Type{vector, vector}
	case simdBitcast, simdConvert, simdConvertLo:
		if sig.Results().Len() != 1 {
			return false
		}
		result = sig.Results().At(0).Type()
		shape, ok := llvmgen.SIMDNumericShape(result)
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
	if shape, ok := llvmgen.SIMDVectorShape(typ); ok {
		return (&types.StdSizes{WordSize: 8, MaxAlign: 8}).Sizeof(shape) * 8
	}
	return 0
}

// Wide operations include the portable256 bridge's intrinsic dependencies.
// Width-changing operations validate their lane counts and element sizes
// explicitly rather than inheriting SIMD128 support by name.
func (d simdOperation) supportsWide() bool {
	if llvmgen.SIMDX86Special(d.op) {
		return true
	}
	switch d.op {
	case llvmgen.SIMDMulSign, llvmgen.SIMDMulHigh, llvmgen.SIMDMulWidenEven, llvmgen.SIMDDotProductPairs, llvmgen.SIMDDotProductPairsSaturated, llvmgen.SIMDSumOf8AbsDiff, llvmgen.SIMDAddOddSubEven, llvmgen.SIMDMulAddEvenSubOdd, llvmgen.SIMDMulAddOddSubEven, llvmgen.SIMDConcatAddPairsSaturated, llvmgen.SIMDConcatAddPairsSaturatedGrouped, llvmgen.SIMDConcatSubPairsSaturated, llvmgen.SIMDConcatSubPairsSaturatedGrouped, llvmgen.SIMDFunnelLeft, llvmgen.SIMDFunnelRight, llvmgen.SIMDRotateLeft, llvmgen.SIMDRotateRight, llvmgen.SIMDTernaryLogic, llvmgen.SIMDSplatMasked, llvmgen.SIMDCompress, llvmgen.SIMDExpand, llvmgen.SIMDConcatPermute, llvmgen.SIMDPermute, llvmgen.SIMDPermuteOrZeroGrouped, llvmgen.SIMDShiftRight, llvmgen.SIMDShiftLeft, llvmgen.SIMDLeadingZeros, llvmgen.SIMDIsNaN, llvmgen.SIMDIsZero, llvmgen.SIMDConcatShiftBytesGrouped, llvmgen.SIMDOnesCount, llvmgen.SIMDAdd, llvmgen.SIMDSub, llvmgen.SIMDMul, llvmgen.SIMDDiv,
		llvmgen.SIMDInterleaveLoGrouped, llvmgen.SIMDInterleaveHiGrouped,
		llvmgen.SIMDConcatSelectedGrouped, llvmgen.SIMDConcatPermute128,
		llvmgen.SIMDPermuteScalarsGrouped, llvmgen.SIMDPermuteScalarsHiGrouped, llvmgen.SIMDPermuteScalarsLoGrouped,
		llvmgen.SIMDConcatAddPairsGrouped, llvmgen.SIMDConcatSubPairsGrouped,
		llvmgen.SIMDShiftAllLeft, llvmgen.SIMDShiftAllRight,
		llvmgen.SIMDAnd, llvmgen.SIMDOr, llvmgen.SIMDXor, llvmgen.SIMDAndNot, llvmgen.SIMDOrNot,
		llvmgen.SIMDNeg, llvmgen.SIMDNot, llvmgen.SIMDAbs,
		llvmgen.SIMDAddSaturated, llvmgen.SIMDSubSaturated, llvmgen.SIMDAverage,
		llvmgen.SIMDMin, llvmgen.SIMDMax, llvmgen.SIMDSqrt, llvmgen.SIMDFMA,
		llvmgen.SIMDConvert, llvmgen.SIMDCarrylessMultiply,
		llvmgen.SIMDCeil, llvmgen.SIMDFloor, llvmgen.SIMDTrunc, llvmgen.SIMDRound,
		llvmgen.SIMDExtend, llvmgen.SIMDExtendLo, llvmgen.SIMDExtendHi,
		llvmgen.SIMDSaturate, llvmgen.SIMDTruncate, llvmgen.SIMDSaturateConcatGrouped,
		llvmgen.SIMDLoad, llvmgen.SIMDStore, llvmgen.SIMDStoreMasked, llvmgen.SIMDBroadcast,
		llvmgen.SIMDSplatLane0, llvmgen.SIMDExtractLane, llvmgen.SIMDInsertLane,
		llvmgen.SIMDGetLo, llvmgen.SIMDGetHi, llvmgen.SIMDSetLo, llvmgen.SIMDSetHi,
		llvmgen.SIMDMaskFromBits, llvmgen.SIMDMaskToBits,
		llvmgen.SIMDEqual, llvmgen.SIMDNotEqual, llvmgen.SIMDLess, llvmgen.SIMDLessEqual,
		llvmgen.SIMDGreater, llvmgen.SIMDGreaterEqual, llvmgen.SIMDToMask,
		llvmgen.SIMDBitcast, llvmgen.SIMDBitSelect, llvmgen.SIMDBitSelectNot, llvmgen.SIMDBlend:
		return true
	}
	return false
}

func (p *context) simdOperation(fn *ssa.Function) (simdOperation, bool) {
	desc, ok := lookupSIMD(fn, p.prog.Target().GOARCH)
	return desc, ok && desc.op != llvmgen.SIMDUnimplemented
}

func (p *context) simdCall(b llvmgen.Builder, fn *ssa.Function, args []ssa.Value) llvmgen.Expr {
	desc, ok := p.simdOperation(fn)
	if !ok {
		panic("invalid SIMD intrinsic")
	}
	return b.SIMD(desc.op, p.simdResultType(fn.Signature), p.compileValues(b, args, fnNormal)...)
}

func (p *context) simdResultType(sig *types.Signature) llvmgen.Type {
	if sig.Results().Len() == 0 {
		return p.prog.Void()
	}
	return p.prog.Type(sig.Results().At(0).Type(), llvmgen.GoDecl)
}
