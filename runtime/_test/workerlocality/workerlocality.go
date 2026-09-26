//go:build llgo && js && wasm && llgo.wasm.workers

package workerlocality

// These values exercise cross-package GLS through the bounded worker runtime.
//
//llgointernal:gls
var localState *int

//llgointernal:gls
var initializedState = new(int)

func CurrentLocalState() *int       { return localState }
func SetLocalState(value *int)      { localState = value }
func CurrentInitializedState() *int { return initializedState }
func SetInitializedState(value int) { *initializedState = value }

type TLSProbe struct {
	Value   int
	Padding [128]uintptr
}

//llgointernal:tls
var tlsValue *TLSProbe

//llgointernal:tls
var tlsFunc func() int

//go:noinline
func InstallTLSValue(value int) {
	tlsValue = &TLSProbe{Value: value}
}

//go:noinline
func LoadTLSValue() int {
	return tlsValue.Value
}

//go:noinline
func InstallTLSFunc(value int) {
	tlsFunc = func() int { return value }
}

//go:noinline
func CallTLSFunc() int {
	return tlsFunc()
}
