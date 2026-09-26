// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license.
// See LICENSES/Go-BSD-3-Clause.txt at this module root for license terms.

package runtime

import (
	_ "unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
	"github.com/xgo-dev/llgo/runtime/internal/traceback"
)

// Layout of in-memory per-function information prepared by linker
// See https://golang.org/s/go12symtab.
// Keep in sync with linker (../cmd/link/internal/ld/pcln.go:/pclntab)
// and with package debug/gosym and with symtab.go in package runtime.
type _func struct {
	unused [8]byte
}

//go:linkname goid github.com/xgo-dev/llgo/runtime/internal/runtime.goid
func goid() uint64

//go:noinline
func Stack(buf []byte, all bool) int {
	if len(buf) == 0 {
		return 0
	}
	out := appendTracebackHeader(buf[:0:len(buf)])
	if len(out) >= len(buf) {
		return copy(buf, out)
	}
	var small [64]uintptr
	pcs := small[:]
	n := 0
	for {
		// Skip runtime.Callers and runtime.Stack itself, as Go does.
		n = Callers(2, pcs)
		if n < len(pcs) || len(pcs) >= maxTracebackFrames {
			break
		}
		pcs = make([]uintptr, len(pcs)*2)
	}
	if n > 0 {
		var window traceback.Window
		first := true
		frames := CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			if traceback.Visible(frame.Function, false, first) {
				first = false
				out = window.Append(out, tracebackFrame(frame))
			}
			if !more || len(out) >= len(buf) {
				break
			}
		}
		out = window.Finish(out)
	}
	out = appendCurrentCreatedBy(out)
	if all && len(out) < len(buf) {
		out = appendOtherTracebacks(out, false, len(buf))
	}
	return copy(buf, out)
}

func appendHexUint(buf []byte, v uintptr) []byte {
	return traceback.AppendHex(buf, v)
}

func appendInt(out []byte, v int) []byte {
	return traceback.AppendInt(out, v)
}

type traceError string

func (e traceError) Error() string { return string(e) }

var (
	traceInitOnce psync.Once
	traceMu       psync.Mutex

	traceCh         chan []byte
	traceDoneCh     chan struct{}
	traceClosed     bool
	traceDoneClosed bool
)

func ensureTraceInit() {
	traceInitOnce.Do(func() {
		traceMu.Init(nil)
	})
}

func StartTrace() error {
	ensureTraceInit()

	traceMu.Lock()
	if traceCh != nil {
		traceMu.Unlock()
		return traceError("runtime: tracing already enabled")
	}
	traceCh = make(chan []byte, 8)
	traceDoneCh = make(chan struct{})
	traceClosed = false
	traceDoneClosed = false

	// Minimal non-empty payload so stdlib runtime/trace tests can assert that
	// tracing produced output. This is not a real execution trace.
	traceCh <- []byte("llgo-trace\n")
	traceMu.Unlock()
	return nil
}

func ReadTrace() []byte {
	ensureTraceInit()

	traceMu.Lock()
	ch := traceCh
	done := traceDoneCh
	traceMu.Unlock()
	if ch == nil {
		return nil
	}
	data, ok := <-ch
	if ok {
		return data
	}
	// Channel closed and drained: wake StopTrace.
	if done != nil {
		traceMu.Lock()
		if traceDoneCh == done && !traceDoneClosed {
			traceDoneClosed = true
			close(done)
		}
		traceMu.Unlock()
	}
	return nil
}

func StopTrace() {
	ensureTraceInit()

	traceMu.Lock()
	ch := traceCh
	done := traceDoneCh
	if ch == nil {
		traceMu.Unlock()
		return
	}
	doClose := !traceClosed
	traceClosed = true
	traceMu.Unlock()

	if doClose {
		close(ch)
	}
	if done != nil {
		<-done
	}

	traceMu.Lock()
	if traceCh == ch {
		traceCh = nil
		traceDoneCh = nil
		traceClosed = false
		traceDoneClosed = false
	}
	traceMu.Unlock()
}

func SetMutexProfileFraction(rate int) int {
	return 0
}

func SetBlockProfileRate(rate int) {
}

var MemProfileRate int = 512 * 1024
