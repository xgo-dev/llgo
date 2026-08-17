package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

/* Handle types. */

type Signal struct {
	Unused [uvSignalSize]byte
}

// ----------------------------------------------

/* Function type */

// llgo:type C
type SignalCb func(handle *Signal, sigNum c.Int)

// ----------------------------------------------

/* Signal related functions and method. */

//go:linkname SignalInit C.uv_signal_init
func SignalInit(loop *Loop, handle *Signal) c.Int

//go:linkname uvSignalStart C.uv_signal_start
func uvSignalStart(handle *Signal, cb SignalCb, signum c.Int) c.Int

func (handle *Signal) Start(cb SignalCb, signum c.Int) c.Int {
	return uvSignalStart(handle, cb, signum)
}

//go:linkname uvSignalStartOneshot C.uv_signal_start_oneshot
func uvSignalStartOneshot(handle *Signal, cb SignalCb, signum c.Int) c.Int

func (handle *Signal) StartOneshot(cb SignalCb, signum c.Int) c.Int {
	return uvSignalStartOneshot(handle, cb, signum)
}

//go:linkname uvSignalStop C.uv_signal_stop
func uvSignalStop(handle *Signal) c.Int

func (handle *Signal) Stop() c.Int {
	return uvSignalStop(handle)
}
