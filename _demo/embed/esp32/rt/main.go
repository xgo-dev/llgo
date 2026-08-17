package main

import (
	_ "unsafe"

	"github.com/xgo-dev/llgo/_demo/embed/esp32/watchdog"
)

//
//go:linkname absvdi2 __absvdi2
func absvdi2(a int64) int64

//go:linkname absvsi2 __absvsi2
func absvsi2(a int32) int32

//go:linkname adddf3 __adddf3
func adddf3(a, b float64) float64

//go:linkname addsf3 __addsf3
func addsf3(a, b float32) float32

//go:linkname addvdi3 __addvdi3
func addvdi3(a, b int64) int64

//go:linkname addvsi3 __addvsi3
func addvsi3(a, b int32) int32

//go:linkname udivdi3 __udivdi3
func udivdi3(a, b uint64) uint64

//go:linkname clzdi2 __clzdi2
func clzdi2(a uint64) int32

//go:linkname clzsi2 __clzsi2
func clzsi2(a uint32) int32

//go:linkname ctzdi2 __ctzdi2
func ctzdi2(a uint64) int32

//go:linkname ctzsi2 __ctzsi2
func ctzsi2(a uint32) int32

//go:linkname popcountdi2 __popcountdi2
func popcountdi2(a uint64) int32

//go:linkname popcountsi2 __popcountsi2
func popcountsi2(a uint32) int32

//go:linkname divdf3 __divdf3
func divdf3(a, b float64) float64

//go:linkname divsf3 __divsf3
func divsf3(a, b float32) float32

//go:linkname mulsf3 __mulsf3
func mulsf3(a, b float32) float32

//go:linkname divdi3 __divdi3
func divdi3(a, b int64) int64

//go:linkname muldf3 __muldf3
func muldf3(a, b float64) float64

//go:linkname muldi3 __muldi3
func muldi3(a, b int64) int64

//go:linkname subdf3 __subdf3
func subdf3(a, b float64) float64

//go:linkname subsf3 __subsf3
func subsf3(a, b float32) float32

//go:linkname extendsfdf2 __extendsfdf2
func extendsfdf2(a float32) float64

//go:linkname fixdfdi __fixdfdi
func fixdfdi(a float64) int64

//go:linkname fixdfsi __fixdfsi
func fixdfsi(a float64) int32

//go:linkname fixsfdi __fixsfdi
func fixsfdi(a float32) int64

//go:linkname fixsfsi __fixsfsi
func fixsfsi(a float32) int32

//go:linkname floatdidf __floatdidf
func floatdidf(a int64) float64

//go:linkname floatsidf __floatsidf
func floatsidf(a int32) float64

//go:linkname ashldi3 __ashldi3
func ashldi3(a int64, b int32) int64

//go:linkname ashrdi3 __ashrdi3
func ashrdi3(a int64, b int32) int64

//go:linkname lshrdi3 __lshrdi3
func lshrdi3(a uint64, b int32) uint64

//go:linkname bswapdi2 __bswapdi2
func bswapdi2(a uint64) uint64

//go:linkname bswapsi2 __bswapsi2
func bswapsi2(a uint32) uint32

var totalTests = 0
var passedTests = 0
var failedTests = 0

func assertEqualInt32(name string, actual, expected int32) {
	totalTests++
	if actual != expected {
		println("FAIL: %s: expected %d, got %d\n", name, expected, actual)
		failedTests++
	} else {
		passedTests++
	}
}

func assertEqualInt64(name string, actual, expected int64) {
	totalTests++
	if actual != expected {
		println("FAIL: %s: expected %d, got %d\n", name, expected, actual)
		failedTests++
	} else {
		passedTests++
	}
}

func assertEqualUint32(name string, actual, expected uint32) {
	totalTests++
	if actual != expected {
		println("FAIL: %s: expected %d, got %d\n", name, expected, actual)
		failedTests++
	} else {
		passedTests++
	}
}

func assertEqualUint64(name string, actual, expected uint64) {
	totalTests++
	if actual != expected {
		println("FAIL: %s: expected %d, got %d\n", name, expected, actual)
		failedTests++
	} else {
		passedTests++
	}
}

func assertEqualFloat32(name string, actual, expected float32, epsilon float32) {
	totalTests++
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > epsilon {
		println("FAIL: %s: expected %f, got %f\n", name, expected, actual)
		failedTests++
	} else {
		passedTests++
	}
}

func assertEqualFloat64(name string, actual, expected float64, epsilon float64) {
	totalTests++
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}
	if diff > epsilon {
		println("FAIL: %s: expected %f, got %f\n", name, expected, actual)
		failedTests++
	} else {
		passedTests++
	}
}

func testAbsFunctions() {
	println("Testing absolute value functions...")

	// Test absvsi2
	assertEqualInt32("absvsi2", absvsi2(12345), 12345)
	assertEqualInt32("absvsi2", absvsi2(-12345), 12345)
	assertEqualInt32("absvsi2", absvsi2(0), 0)

	// Test absvdi2
	assertEqualInt64("absvdi2", absvdi2(1234567890123456789), 1234567890123456789)
	assertEqualInt64("absvdi2", absvdi2(-1234567890123456789), 1234567890123456789)
	assertEqualInt64("absvdi2", absvdi2(0), 0)
}

func testAddFunctions() {
	println("Testing addition functions...")

	// Test addvsi3
	assertEqualInt32("addvsi3", addvsi3(1000, 2000), 3000)
	assertEqualInt32("addvsi3", addvsi3(-1000, -2000), -3000)
	assertEqualInt32("addvsi3", addvsi3(0, 0), 0)

	// Test addvdi3
	assertEqualInt64("addvdi3", addvdi3(1000000000, 2000000000), 3000000000)
	assertEqualInt64("addvdi3", addvdi3(-1000000000, -2000000000), -3000000000)
	assertEqualInt64("addvdi3", addvdi3(0, 0), 0)

	// Test adddf3
	assertEqualFloat64("adddf3", adddf3(3.14, 2.71), 5.85, 1e-10)
	assertEqualFloat64("adddf3", adddf3(-3.14, -2.71), -5.85, 1e-10)
	assertEqualFloat64("adddf3", adddf3(0.0, 0.0), 0.0, 1e-10)

	// Test addsf3
	assertEqualFloat32("addsf3", addsf3(3.14, 2.71), 5.85, 1e-6)
	assertEqualFloat32("addsf3", addsf3(-3.14, -2.71), -5.85, 1e-6)
	assertEqualFloat32("addsf3", addsf3(0.0, 0.0), 0.0, 1e-6)
}

func testCountFunctions() {
	println("Testing count functions...")

	// Test clzsi2 - count leading zeros in 32-bit integer
	assertEqualInt32("clzsi2", clzsi2(1), 31)         // 0x00000001 has 31 leading zeros
	assertEqualInt32("clzsi2", clzsi2(0x80000000), 0) // 0x80000000 has 0 leading zeros
	assertEqualInt32("clzsi2", clzsi2(0), 32)         // 0 has 32 leading zeros

	// FIXME
	// // Test clzdi2 - count leading zeros in 64-bit integer
	// assertEqualInt32("clzdi2", clzdi2(1), 63)                 // 0x0000000000000001 has 63 leading zeros
	// assertEqualInt32("clzdi2", clzdi2(0x8000000000000000), 0) // 0x8000000000000000 has 0 leading zeros
	// assertEqualInt32("clzdi2", clzdi2(0), 64)                 // 0 has 64 leading zeros

	// Test ctzsi2 - count trailing zeros in 32-bit integer
	assertEqualInt32("ctzsi2", ctzsi2(1<<5), 5)        // 0x00000020 has 5 trailing zeros
	assertEqualInt32("ctzsi2", ctzsi2(0x80000000), 31) // 0x80000000 has 31 trailing zeros
	assertEqualInt32("ctzsi2", ctzsi2(0), 32)          // 0 has 32 trailing zeros

	// Test ctzdi2 - count trailing zeros in 64-bit integer
	assertEqualInt32("ctzdi2", ctzdi2(1<<10), 10)              // 0x0000000000000400 has 10 trailing zeros
	assertEqualInt32("ctzdi2", ctzdi2(0x8000000000000000), 63) // 0x8000000000000000 has 63 trailing zeros
	assertEqualInt32("ctzdi2", ctzdi2(0), 64)                  // 0 has 64 trailing zeros

	// Test popcountsi2 - population count of 32-bit integer
	assertEqualInt32("popcountsi2", popcountsi2(0xF0F0F0F0), 16) // 0xF0F0F0F0 has 16 ones
	assertEqualInt32("popcountsi2", popcountsi2(0), 0)           // 0 has 0 ones
	assertEqualInt32("popcountsi2", popcountsi2(0xFFFFFFFF), 32) // 0xFFFFFFFF has 32 ones

	// Test popcountdi2 - population count of 64-bit integer
	assertEqualInt32("popcountdi2", popcountdi2(0xFFFF0000FFFF0000), 32) // 0xFFFF0000FFFF0000 has 32 ones
	assertEqualInt32("popcountdi2", popcountdi2(0), 0)                   // 0 has 0 ones
	assertEqualInt32("popcountdi2", popcountdi2(0xFFFFFFFFFFFFFFFF), 64) // 0xFFFFFFFFFFFFFFFF has 64 ones
}

func testDivisionFunctions() {
	println("Testing division functions...")

	// Test udivdi3 - unsigned 64-bit division
	assertEqualUint64("udivdi3", udivdi3(100, 5), 20)
	assertEqualUint64("udivdi3", udivdi3(18446744073709551615, 3), 6148914691236517205)
	assertEqualUint64("udivdi3", udivdi3(0, 123456789), 0)

	// Test divdi3 - signed 64-bit division
	assertEqualInt64("divdi3", divdi3(20, 3), 6)
	assertEqualInt64("divdi3", divdi3(-20, 3), -6)
	assertEqualInt64("divdi3", divdi3(20, -3), -6)

	// Test divdf3 - double precision division
	assertEqualFloat64("divdf3", divdf3(20.0, 3.0), 6.666666666666667, 1e-10)
	assertEqualFloat64("divdf3", divdf3(-20.0, 3.0), -6.666666666666667, 1e-10)

	// Test divsf3 - single precision division
	assertEqualFloat32("divsf3", divsf3(20.0, 3.0), 6.6666665, 1e-6)
	assertEqualFloat32("divsf3", divsf3(-20.0, 3.0), -6.6666665, 1e-6)
}

func testMultiplicationFunctions() {
	println("Testing multiplication functions...")

	// Test muldi3 - signed 64-bit multiplication
	assertEqualInt64("muldi3", muldi3(5, 4), 20)
	assertEqualInt64("muldi3", muldi3(-5, 4), -20)
	assertEqualInt64("muldi3", muldi3(5, -4), -20)

	// Test muldf3 - double precision multiplication
	assertEqualFloat64("muldf3", muldf3(3.0, 4.0), 12.0, 1e-10)
	assertEqualFloat64("muldf3", muldf3(-3.0, 4.0), -12.0, 1e-10)

	// Test mulsf3 - single precision multiplication
	assertEqualFloat32("mulsf3", mulsf3(3.0, 4.0), 12.0, 1e-6)
	assertEqualFloat32("mulsf3", mulsf3(-3.0, 4.0), -12.0, 1e-6)
}

func testSubtractionFunctions() {
	println("Testing subtraction functions...")

	// Test subdf3 - double precision subtraction
	assertEqualFloat64("subdf3", subdf3(5.0, 3.0), 2.0, 1e-10)
	assertEqualFloat64("subdf3", subdf3(3.0, 5.0), -2.0, 1e-10)

	// Test subsf3 - single precision subtraction
	assertEqualFloat32("subsf3", subsf3(5.0, 3.0), 2.0, 1e-6)
	assertEqualFloat32("subsf3", subsf3(3.0, 5.0), -2.0, 1e-6)
}

func testConversionFunctions() {
	println("Testing conversion functions...")

	// Test extendsfdf2 - single to double precision conversion
	// FIXME
	// assertEqualFloat64("extendsfdf2", extendsfdf2(3.14), 3.14, 1e-10)

	// Test fixdfsi - double precision to int32 conversion
	assertEqualInt32("fixdfsi", fixdfsi(123.45), 123)
	assertEqualInt32("fixdfsi", fixdfsi(-123.45), -123)

	// Test fixsfsi - single precision to int32 conversion
	assertEqualInt32("fixsfsi", fixsfsi(123.45), 123)
	assertEqualInt32("fixsfsi", fixsfsi(-123.45), -123)

	// Test fixdfdi - double precision to int64 conversion
	assertEqualInt64("fixdfdi", fixdfdi(123456789.123), 123456789)
	assertEqualInt64("fixdfdi", fixdfdi(-123456789.123), -123456789)

	// Test fixsfdi - single precision to int64 conversion
	// FIXME
	// assertEqualInt64("fixsfdi", fixsfdi(123456789.123), 123456789)
	// assertEqualInt64("fixsfdi", fixsfdi(-123456789.123), -123456789)

	// Test floatsidf - int32 to double precision conversion
	assertEqualFloat64("floatsidf", floatsidf(42), 42.0, 1e-10)
	assertEqualFloat64("floatsidf", floatsidf(-100), -100.0, 1e-10)

	// Test floatdidf - int64 to double precision conversion
	assertEqualFloat64("floatdidf", floatdidf(123456789), 123456789.0, 1e-10)
	assertEqualFloat64("floatdidf", floatdidf(-123456789), -123456789.0, 1e-10)
}

func testShiftFunctions() {
	println("Testing shift functions...")

	// Test ashldi3 - arithmetic shift left
	assertEqualInt64("ashldi3", ashldi3(1, 10), 1024)

	// Test ashrdi3 - arithmetic shift right
	assertEqualInt64("ashrdi3", ashrdi3(1024, 10), 1)

	// Test lshrdi3 - logical shift right
	assertEqualUint64("lshrdi3", lshrdi3(1024, 10), 1)
	assertEqualUint64("lshrdi3", lshrdi3(0x8000000000000000, 63), 1)
}

func testBitManipulationFunctions() {
	println("Testing bit manipulation functions...")

	// Test bswapsi2 - byte swap 32-bit integer
	assertEqualUint32("bswapsi2", bswapsi2(0x12345678), 0x78563412)

	// Test bswapdi2 - byte swap 64-bit integer
	assertEqualUint64("bswapdi2", bswapdi2(0x1234567890ABCDEF), 0xEFCDAB9078563412)
}

func main() {
	watchdog.Disable()
	println("Testing Compiler-RT Builtins Functions")
	println("=====================================")
	testAbsFunctions()
	testAddFunctions()
	testCountFunctions()
	testDivisionFunctions()
	testMultiplicationFunctions()
	testSubtractionFunctions()
	testConversionFunctions()
	testShiftFunctions()
	testBitManipulationFunctions()

	println("\n=====================================")
	println("Test Results: %d total, %d passed, %d failed\n", totalTests, passedTests, failedTests)
	println("=====================================")

	if failedTests == 0 {
		println("All tests PASSED!")
	} else {
		println("Some tests FAILED!")
	}
}
