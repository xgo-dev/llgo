package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/luoliwoshang/mquickjs/mquickjs"
)

// Link to the js_stdlib symbol exported from libmquickjs
//
//go:linkname jsStdlib js_stdlib
var jsStdlib mquickjs.JSSTDLibraryDef

func myprint(s *c.Char) {
	for i := 0; i < int(c.Strlen(s)); i++ {
		WriteByte(byte(c.Index(s, i)))
	}
}

// js_print - Override the weak symbol in libmquickjs to provide actual print functionality
// This function is called when JavaScript code calls print()
//
//export js_print
func js_print(ctx *mquickjs.JSContext, thisVal *mquickjs.JSValue, argc c.Int, argv *mquickjs.JSValue) mquickjs.JSValue {
	// Iterate through all arguments
	for i := c.Int(0); i < argc; i++ {
		// Get argument at index i
		argPtr := (*mquickjs.JSValue)(unsafe.Pointer(uintptr(unsafe.Pointer(argv)) + uintptr(i)*unsafe.Sizeof(*argv)))

		// Convert JS value to C string
		var buf mquickjs.JSCStringBuf
		str := ctx.JSToCString(*argPtr, &buf)
		if str != nil {
			myprint(str)
		}

		// Add space between arguments
		if i < argc-1 {
			WriteByte(' ')
		}
	}
	// Add newline at the end
	WriteByte('\n')

	return mquickjs.JSValue(mquickjs.JS_TAG_UNDEFINED)
}

func runTest(ctx *mquickjs.JSContext, name *c.Char, code *c.Char) {
	myprint(c.Str("Test: "))
	myprint(name)
	myprint(c.Str("\n"))

	result := ctx.JSEval(code, c.Int(c.Strlen(code)), c.Str("<eval>"), 0)

	// Check for exception
	if mquickjs.JSValue(result) == mquickjs.JSValue(mquickjs.JS_TAG_EXCEPTION) {
		myprint(c.Str("  Exception occurred!\n"))
	}
	myprint(c.Str("\n"))
}

func main() {
	// Allocate memory for JS context (64KB)
	const memSize = 64 * 1024
	mem := c.Malloc(memSize)

	// Create JS context with js_stdlib
	ctx := mquickjs.JSNewContext(mem, c.Int(memSize), &jsStdlib)
	if ctx == nil {
		myprint(c.Str("Failed to create JS context\n"))
		return
	}

	myprint(c.Str("=== MQuickJS on ESP32 ===\n\n"))

	// Test 1: String output
	runTest(ctx, c.Str("String output"), c.Str("print('Hello from MQuickJS on ESP32!')"))

	// Test 2: Number output
	runTest(ctx, c.Str("Number output"), c.Str("print(123)"))

	// Test 3: Math expression
	runTest(ctx, c.Str("Math expression"), c.Str("print(1 + 2 * 3)"))

	// Test 4: Multiple arguments
	runTest(ctx, c.Str("Multiple args"), c.Str("print('Result:', 10 + 20)"))

	// Test 5: Define JS functions (no output expected)
	runTest(ctx, c.Str("Define functions"), c.Str(`
function add(a, b) {
    return a + b;
}

function factorial(n) {
    if (n <= 1) return 1;
    return n * factorial(n - 1);
}

var multiply = function(a, b) { return a * b; };
`))

	// Test 6: Call the previously defined functions
	runTest(ctx, c.Str("Call add"), c.Str("print('add(3, 5) =', add(3, 5))"))

	runTest(ctx, c.Str("Call factorial"), c.Str("print('factorial(5) =', factorial(5))"))

	runTest(ctx, c.Str("Call multiply"), c.Str("print('multiply(4, 7) =', multiply(4, 7))"))

	// Free context
	ctx.JSFreeContext()
	c.Free(unsafe.Pointer(mem))

	myprint(c.Str("=== Tests complete ===\n"))

	for {
		sleep(1)
	}
}
