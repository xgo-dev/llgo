//go:build llgo && js && wasm && !llgo.wasm.emscripten

// Copyright (c) 2026 The XGo Authors. All rights reserved.
// Use of this source code is governed by the Apache License 2.0.

package js

import "unsafe"

// hostFrame has fixed-width slots; Go refs keep GOROOT's NaN-boxed format.
type hostFrame [6]uint64

const (
	hostFinalize = iota
	hostString
	hostGet
	hostSet
	hostDelete
	hostIndex
	hostSetIndex
	hostLength
	hostCall
	hostInvoke
	hostNew
	hostPrepareString
	hostLoadString
	hostInstanceOf
	hostCopyToGo
	hostCopyToJS
	hostInstallHandler
	hostTakeEvent
)

//go:linkname hostCallOp C.llgo_js_host
func hostCallOp(op int32, frame *hostFrame)

//llgo:type C
type hostHandler func()

//go:linkname hostInstall C.llgo_js_install
func hostInstall(handler hostHandler, pending *uint32)

//go:linkname registerHostPoll github.com/xgo-dev/llgo/runtime/internal/runtime.RegisterWasmCallbackPoll
func registerHostPoll(poll func())

//go:linkname handleHostEvent github.com/xgo-dev/llgo/runtime/internal/runtime.HandleWasmEvent
func handleHostEvent(fn func())

//go:linkname pollHostEvent github.com/xgo-dev/llgo/runtime/internal/runtime.PollWasmEvent
func pollHostEvent()

// These functions replace only GOROOT's go:wasmimport declarations. Keep the
// Value representation, public methods, finalizers, KeepAlive calls, argument
// conversion, and Func registry in GOROOT rather than maintaining a copy.
func finalizeRef(r ref) {
	f := hostFrame{uint64(r)}
	hostCallOp(hostFinalize, &f)
}

func stringVal(s string) ref {
	f := hostFrame{0, 0, uint64(uintptr(unsafe.Pointer(unsafe.StringData(s)))), uint64(len(s))}
	hostCallOp(hostString, &f)
	return ref(f[0])
}

func valueGet(v ref, key string) ref {
	f := propertyFrame(v, key)
	hostCallOp(hostGet, &f)
	return ref(f[0])
}

func valueSet(v ref, key string, x ref) {
	f := propertyFrame(v, key)
	f[1] = uint64(x)
	hostCallOp(hostSet, &f)
}

func valueDelete(v ref, key string) {
	f := propertyFrame(v, key)
	hostCallOp(hostDelete, &f)
}

func propertyFrame(v ref, key string) hostFrame {
	return hostFrame{uint64(v), 0, uint64(uintptr(unsafe.Pointer(unsafe.StringData(key)))), uint64(len(key))}
}

func valueIndex(v ref, i int) ref {
	f := hostFrame{uint64(v), uint64(int64(i))}
	hostCallOp(hostIndex, &f)
	return ref(f[0])
}

func valueSetIndex(v ref, i int, x ref) {
	f := hostFrame{uint64(v), uint64(int64(i)), uint64(x)}
	hostCallOp(hostSetIndex, &f)
}

func valueLength(v ref) int {
	f := hostFrame{uint64(v)}
	hostCallOp(hostLength, &f)
	return int(f[0])
}

func valueCall(v ref, method string, args []ref) (ref, bool) {
	f := propertyFrame(v, method)
	return callFrame(hostCall, &f, args)
}

func valueInvoke(v ref, args []ref) (ref, bool) {
	f := hostFrame{uint64(v)}
	return callFrame(hostInvoke, &f, args)
}

func valueNew(v ref, args []ref) (ref, bool) {
	f := hostFrame{uint64(v)}
	return callFrame(hostNew, &f, args)
}

func callFrame(op int32, f *hostFrame, args []ref) (ref, bool) {
	f[4] = uint64(uintptr(unsafe.Pointer(unsafe.SliceData(args))))
	f[5] = uint64(len(args))
	hostCallOp(op, f)
	return ref(f[0]), f[1] != 0
}

func valuePrepareString(v ref) (ref, int) {
	f := hostFrame{uint64(v)}
	hostCallOp(hostPrepareString, &f)
	return ref(f[0]), int(f[1])
}

func valueLoadString(v ref, b []byte) {
	f := bytesFrame(v, b)
	hostCallOp(hostLoadString, &f)
}

func valueInstanceOf(v, constructor ref) bool {
	f := hostFrame{uint64(v), uint64(constructor)}
	hostCallOp(hostInstanceOf, &f)
	return f[0] != 0
}

func bytesFrame(v ref, b []byte) hostFrame {
	return hostFrame{uint64(v), 0, uint64(uintptr(unsafe.Pointer(unsafe.SliceData(b)))), uint64(len(b))}
}

func copyBytesToGo(dst []byte, src ref) (int, bool) {
	f := bytesFrame(src, dst)
	hostCallOp(hostCopyToGo, &f)
	return int(f[0]), f[1] != 0
}

func copyBytesToJS(dst ref, src []byte) (int, bool) {
	f := bytesFrame(dst, src)
	hostCallOp(hostCopyToJS, &f)
	return int(f[0]), f[1] != 0
}

var hostEventHandler func() bool
var hostEventPending uint32
var hostEventDispatching bool

func setEventHandler(fn func() bool) {
	hostEventHandler = fn
	hostInstall(dispatchHostEvent, &hostEventPending)
	registerHostPoll(pollHostEvents)
}

// Nested Go-to-JS-to-Go calls run on the calling G, matching Go's event
// handler contract. External events enter through the scheduler below.
func dispatchHostEvent() {
	handleHostEvent(func() { hostEventHandler() })
}

func pollHostEvents() {
	if hostEventPending != 0 && !hostEventDispatching {
		hostEventDispatching = true
		go func() {
			var f hostFrame
			hostCallOp(hostTakeEvent, &f)
			hostEventDispatching = false
			dispatchHostEvent()
		}()
	}
	pollHostEvent()
}
