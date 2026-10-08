//go:build js && wasm

package js

// Keep forwarding closures and their captured result slots out of the local
// call frame. Otherwise SSA hoists their heap allocations before the worker
// check, even when the call never leaves worker zero.

//go:noinline
func remoteEqual(v Value, w Value) (result bool) {
	onJSWorker(func() { result = v.Equal(w) })
	return
}

//go:noinline
func remoteIsNaN(v Value) (result bool) {
	onJSWorker(func() { result = v.IsNaN() })
	return
}

//go:noinline
func remoteGlobal() (result Value) {
	onJSWorker(func() { result = Global() })
	return
}

//go:noinline
func remoteGlobalForHost() (result Value) {
	onJSWorker(func() { result = GlobalForHost() })
	return
}

//go:noinline
func remoteValueOf(x any) Value {
	// These conversions copy existing values or reserved handles; they do not
	// access the owning realm. Keep request allocations in the host-only helper.
	switch x := x.(type) {
	case Value:
		return x
	case Func:
		return x.Value
	case nil:
		return valueNull
	case bool:
		if x {
			return valueTrue
		}
		return valueFalse
	}
	return forwardValueOf(x)
}

//go:noinline
func forwardValueOf(x any) (result Value) {
	onJSWorker(func() { result = ValueOf(x) })
	return
}

//go:noinline
func remoteType(v Value) (result Type) {
	onJSWorker(func() { result = v.Type() })
	return
}

//go:noinline
func remoteGet(v Value, p string) (result Value) {
	onJSWorker(func() { result = v.Get(p) })
	return
}

//go:noinline
func remoteSet(v Value, p string, x any) {
	onJSWorker(func() { v.Set(p, x) })
}

//go:noinline
func remoteDelete(v Value, p string) {
	onJSWorker(func() { v.Delete(p) })
}

//go:noinline
func remoteIndex(v Value, i int) (result Value) {
	onJSWorker(func() { result = v.Index(i) })
	return
}

//go:noinline
func remoteSetIndex(v Value, i int, x any) {
	onJSWorker(func() { v.SetIndex(i, x) })
}

//go:noinline
func remoteLength(v Value) (result int) {
	onJSWorker(func() { result = v.Length() })
	return
}

//go:noinline
func remoteCall(v Value, m string, args ...any) (res Value) {
	onJSWorker(func() { res = v.Call(m, args...) })
	return
}

//go:noinline
func remoteInvoke(v Value, args ...any) (res Value) {
	onJSWorker(func() { res = v.Invoke(args...) })
	return
}

//go:noinline
func remoteNew(v Value, args ...any) (res Value) {
	onJSWorker(func() { res = v.New(args...) })
	return
}

//go:noinline
func remoteFloat(v Value, method string) (result float64) {
	onJSWorker(func() { result = v.float(method) })
	return
}

//go:noinline
func remoteTruthy(v Value) (result bool) {
	onJSWorker(func() { result = v.Truthy() })
	return
}

//go:noinline
func remoteString(v Value) (result string) {
	onJSWorker(func() { result = v.String() })
	return
}

//go:noinline
func remoteInstanceOf(v Value, t Value) (result bool) {
	onJSWorker(func() { result = v.InstanceOf(t) })
	return
}

//go:noinline
func remoteCopyBytesToGo(dst []byte, src Value) (result int) {
	onJSWorker(func() { result = CopyBytesToGo(dst, src) })
	return
}

//go:noinline
func remoteCopyBytesToJS(dst Value, src []byte) (result int) {
	onJSWorker(func() { result = CopyBytesToJS(dst, src) })
	return
}

//go:noinline
func remoteFuncOf(fn func(this Value, args []Value) any) (result Func) {
	onJSWorker(func() { result = FuncOf(fn) })
	return
}

//go:noinline
func remoteRelease(c Func) {
	onJSWorker(func() { c.Release() })
}
