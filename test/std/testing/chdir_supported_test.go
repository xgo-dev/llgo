//go:build !llgo || !js || !wasm || llgo.wasm.emscripten

package testing_test

type chdirTestContext interface {
	Helper()
	Fatalf(string, ...any)
}

func testChdir(tb chdirTestContext, _ string, chdir func()) {
	tb.Helper()
	chdir()
}
