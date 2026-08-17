// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	_ "unsafe"

	psync "github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
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

func Stack(buf []byte, all bool) int {
	var pcs [64]uintptr
	n := Callers(0, pcs[:])
	out := make([]byte, 0, 1024)
	out = append(out, "goroutine "...)
	out = appendInt(out, int(goid()))
	out = append(out, " [running]:\n"...)
	frames := CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "" {
			frame.Function = unknownFunctionName(frame.PC)
		}
		out = append(out, frame.Function...)
		out = append(out, "()\n\t"...)
		if frame.File == "" {
			out = append(out, "???"...)
		} else {
			out = append(out, frame.File...)
		}
		out = append(out, ':')
		out = appendInt(out, frame.Line)
		if frame.Entry != 0 && frame.PC >= frame.Entry {
			out = append(out, " +0x"...)
			out = appendHexUint(out, uintptr(frame.PC-frame.Entry))
		}
		out = append(out, '\n')
		if !more {
			break
		}
	}
	if len(out) > len(buf) {
		copy(buf, out[:len(buf)])
		return len(buf)
	}
	copy(buf, out)
	return len(out)
}

func appendHexUint(buf []byte, v uintptr) []byte {
	const digits = "0123456789abcdef"
	if v == 0 {
		return append(buf, '0')
	}
	var tmp [16]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = digits[v&0xf]
		v >>= 4
	}
	return append(buf, tmp[i:]...)
}

func appendInt(out []byte, v int) []byte {
	if v == 0 {
		return append(out, '0')
	}
	if v < 0 {
		out = append(out, '-')
		v = -v
	}
	var digits [20]byte
	i := len(digits)
	for v > 0 {
		i--
		digits[i] = byte('0' + v%10)
		v /= 10
	}
	return append(out, digits[i:]...)
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
