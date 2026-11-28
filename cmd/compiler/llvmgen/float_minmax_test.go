package llvmgen

import (
	"go/token"
	"math"
	"testing"

	"github.com/xgo-dev/llvm"
)

func TestFloatMinMaxBits(t *testing.T) {
	prog := NewProgram(nil)
	defer prog.Dispose()
	pkg := prog.NewPackage("minmaxbits", "minmaxbits")
	fn := pkg.NewFunc("test", NoArgsNoRet, CDecl)
	b := fn.MakeBody(1)
	for _, width := range []int{32, 64} {
		intType, floatType := prog.tyInt64(), prog.Float64().ll
		values := []uint64{0, 0x8000000000000000, 0x7ff8000000000001, 0xfff8000000000042, 0x7ff0000000000001, 0x7ff0000000000000, 0xfff0000000000000, 1, 0x8000000000000001, 0x0010000000000000, 0x8010000000000000, 0x3ff0000000000000, 0xbff0000000000000, 0x7fefffffffffffff, 0xffefffffffffffff}
		if width == 32 {
			intType, floatType = prog.tyInt32(), prog.Float32().ll
			values = []uint64{0, 0x80000000, 0x7fc00001, 0xffc00042, 0x7f800001, 0x7f800000, 0xff800000, 1, 0x80000001, 0x00800000, 0x80800000, 0x3f800000, 0xbf800000, 0x7f7fffff, 0xff7fffff}
		}
		check := func(x, y uint64) {
			for _, op := range []token.Token{token.LSS, token.GTR} {
				a := llvm.ConstBitCast(llvm.ConstInt(intType, x, false), floatType)
				c := llvm.ConstBitCast(llvm.ConstInt(intType, y, false), floatType)
				result := b.floatMinMaxBits(op, a, c)
				raw := llvm.ConstBitCast(result, intType)
				if raw.IsAConstantInt().IsNil() {
					t.Fatalf("constant inputs did not fold: %s", raw.String())
				}
				got := raw.ZExtValue()
				var want, actual float64
				if width == 32 {
					a, c := math.Float32frombits(uint32(x)), math.Float32frombits(uint32(y))
					v := min(a, c)
					if op == token.GTR {
						v = max(a, c)
					}
					want, actual = float64(v), float64(math.Float32frombits(uint32(got)))
				} else {
					a, c := math.Float64frombits(x), math.Float64frombits(y)
					want = min(a, c)
					if op == token.GTR {
						want = max(a, c)
					}
					actual = math.Float64frombits(got)
				}
				if math.IsNaN(want) {
					if !math.IsNaN(actual) {
						t.Fatalf("f%d %s(%x,%x) = %x, want NaN", width, op, x, y, got)
					}
				} else if math.Float64bits(actual) != math.Float64bits(want) {
					t.Fatalf("f%d %s(%x,%x) = %x, want %g (sign=%v)", width, op, x, y, got, want, math.Signbit(want))
				}
			}
		}
		for _, a := range values {
			for _, c := range values {
				check(a, c)
			}
		}
		// Exercise finite values across the exponent and mantissa ranges, as well
		// as the exhaustive edge pairs above. The fixed seed keeps failures stable.
		seed := uint64(0x123456789abcdef)
		next := func() uint64 { seed ^= seed << 13; seed ^= seed >> 7; seed ^= seed << 17; return seed }
		for i := 0; i < 2048; i++ {
			check(next(), next())
		}
	}
	b.Return()
}
