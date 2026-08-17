package libuv

import (
	_ "unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// ----------------------------------------------

/* Handle types. */

// TODO(spongehah): Timer
type Timer struct {
	Unused [uvTimerSize]byte
}

// ----------------------------------------------

// llgo:type C
type TimerCb func(timer *Timer)

// ----------------------------------------------

/* Timer related function and method */

//go:linkname InitTimer C.uv_timer_init
func InitTimer(loop *Loop, timer *Timer) c.Int

//go:linkname uvTimerStart C.uv_timer_start
func uvTimerStart(timer *Timer, cb TimerCb, timeoutMs uint64, repeat uint64) c.Int

func (timer *Timer) Start(cb TimerCb, timeoutMs uint64, repeat uint64) c.Int {
	return uvTimerStart(timer, cb, timeoutMs, repeat)
}

//go:linkname uvTimerStop C.uv_timer_stop
func uvTimerStop(timer *Timer) c.Int

func (timer *Timer) Stop() c.Int {
	return uvTimerStop(timer)
}

//go:linkname uvTimerAgain C.uv_timer_again
func uvTimerAgain(timer *Timer) c.Int

func (timer *Timer) Again() c.Int {
	return uvTimerAgain(timer)
}

//go:linkname uvTimerSetRepeat C.uv_timer_set_repeat
func uvTimerSetRepeat(timer *Timer, repeat uint64)

func (timer *Timer) SetRepeat(repeat uint64) {
	uvTimerSetRepeat(timer, repeat)
}

//go:linkname uvTimerGetRepeat C.uv_timer_get_repeat
func uvTimerGetRepeat(timer *Timer) uint64

func (timer *Timer) GetRepeat() uint64 {
	return uvTimerGetRepeat(timer)
}

//go:linkname uvTimerGetDueIn C.uv_timer_get_due_in
func uvTimerGetDueIn(timer *Timer) uint64

func (timer *Timer) GetDueIn() uint64 {
	return uvTimerGetDueIn(timer)
}
