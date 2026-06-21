//go:build (darwin || linux) && !baremetal

package runtime

import (
	"unsafe"

	c "github.com/goplus/llgo/runtime/internal/clite"
	cliteos "github.com/goplus/llgo/runtime/internal/clite/os"
	psync "github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	csyscall "github.com/goplus/llgo/runtime/internal/clite/syscall"
	latomic "github.com/goplus/llgo/runtime/internal/lib/sync/atomic"
)

// Minimal runtime netpoll backing for internal/poll.
//
// internal/poll assumes non-blocking I/O and a working runtime poller when
// pollDesc.init succeeds. Our previous stubs returned "success" but never
// actually blocked or unblocked, which caused tight spins and deadlocks (e.g.
// listener Close waiting forever because Accept loops never observed "closing").
//
// This implementation uses libc poll(2) and a global wake pipe to interrupt
// poll waits on close/unblock/deadline changes. It is intentionally minimal:
// it aims for correctness for stdlib and tests, not high-performance netpoll.

// Must match the error codes in internal/poll/fd_poll_runtime.go.
const (
	pollNoError        = 0
	pollErrClosing     = 1
	pollErrTimeout     = 2
	pollErrNotPollable = 3
)

const (
	pollIn  int16 = 0x0001
	pollOut int16 = 0x0004
)

type pollfd struct {
	fd      c.Int
	events  int16
	revents int16
}

//go:linkname c_poll C.poll
func c_poll(fds *pollfd, nfds uintptr, timeout c.Int) c.Int

type llgoPollDesc struct {
	fd c.Int

	closing uint32

	// Absolute deadlines in nanoseconds. 0 means no deadline.
	rd int64
	wd int64
}

var pollOnce psync.Once
var wakeR, wakeW c.Int
var pollDescMu psync.Mutex
var pollDescRoots map[uintptr]*llgoPollDesc

func pollInit() {
	pollDescMu.Init(nil)
	pollDescRoots = make(map[uintptr]*llgoPollDesc)

	var fds [2]c.Int
	if cliteos.Pipe(&fds) != 0 {
		// If we can't create the wake pipe, we fall back to a non-pollable mode.
		// internal/poll will then use blocking operations (with reduced features).
		wakeR, wakeW = -1, -1
		return
	}
	wakeR, wakeW = fds[0], fds[1]
	setCloseOnExec(wakeR)
	setCloseOnExec(wakeW)
	setNonblock(wakeR)
	setNonblock(wakeW)
}

func setCloseOnExec(fd c.Int) {
	flags := cliteos.Fcntl(fd, c.Int(csyscall.F_GETFD))
	if flags < 0 {
		return
	}
	_ = cliteos.Fcntl(fd, c.Int(csyscall.F_SETFD), uintptr(flags)|uintptr(csyscall.FD_CLOEXEC))
}

func setNonblock(fd c.Int) {
	flags := cliteos.Fcntl(fd, c.Int(csyscall.F_GETFL))
	if flags < 0 {
		return
	}
	_ = cliteos.Fcntl(fd, c.Int(csyscall.F_SETFL), uintptr(flags)|uintptr(csyscall.O_NONBLOCK))
}

func pollRootAdd(pd *llgoPollDesc) uintptr {
	if pd == nil {
		return 0
	}
	ctx := uintptr(unsafe.Pointer(pd))
	pollDescMu.Lock()
	pollDescRoots[ctx] = pd
	pollDescMu.Unlock()
	return ctx
}

func pollRootDel(ctx uintptr) {
	if ctx == 0 {
		return
	}
	pollDescMu.Lock()
	delete(pollDescRoots, ctx)
	pollDescMu.Unlock()
}

func pollWake() {
	pollOnce.Do(pollInit)
	if wakeW < 0 {
		return
	}
	var b [1]byte
	// Best-effort; ignore EAGAIN (pipe full) and other errors.
	_ = cliteos.Write(wakeW, unsafe.Pointer(&b[0]), 1)
}

func pollDrainWake() {
	if wakeR < 0 {
		return
	}
	var buf [64]byte
	for {
		n := cliteos.Read(wakeR, unsafe.Pointer(&buf[0]), uintptr(len(buf)))
		if n <= 0 {
			// Either EAGAIN (empty) or an error; both are fine.
			return
		}
	}
}

func pollDeadline(pd *llgoPollDesc, mode int) int64 {
	switch mode {
	case 'r':
		return latomic.LoadInt64(&pd.rd)
	case 'w':
		return latomic.LoadInt64(&pd.wd)
	default:
		rd := latomic.LoadInt64(&pd.rd)
		wd := latomic.LoadInt64(&pd.wd)
		if rd == 0 {
			return wd
		}
		if wd == 0 || rd < wd {
			return rd
		}
		return wd
	}
}

func pollTimeoutMs(deadline int64) (c.Int, int) {
	if deadline == 0 {
		return -1, pollNoError
	}
	now := runtimeNano()
	remain := deadline - now
	if remain <= 0 {
		return 0, pollErrTimeout
	}
	// poll expects milliseconds. Round up to avoid premature timeout.
	ms := (remain + 1e6 - 1) / 1e6
	if ms > int64(^uint32(0)>>1) { // int32 max-ish
		ms = int64(^uint32(0) >> 1)
	}
	return c.Int(ms), pollNoError
}

//go:linkname poll_runtime_pollServerInit internal/poll.runtime_pollServerInit
func poll_runtime_pollServerInit() {
	pollOnce.Do(pollInit)
}

//go:linkname poll_runtime_pollOpen internal/poll.runtime_pollOpen
func poll_runtime_pollOpen(fd uintptr) (uintptr, int) {
	pollOnce.Do(pollInit)
	if wakeR < 0 {
		return 0, int(csyscall.EOPNOTSUPP)
	}
	pd := &llgoPollDesc{fd: c.Int(fd)}
	return pollRootAdd(pd), 0
}

//go:linkname poll_runtime_pollClose internal/poll.runtime_pollClose
func poll_runtime_pollClose(ctx uintptr) {
	pollOnce.Do(pollInit)
	if ctx == 0 {
		return
	}
	pd := (*llgoPollDesc)(unsafe.Pointer(ctx))
	latomic.StoreUint32(&pd.closing, 1)
	pollWake()
	pollRootDel(ctx)
}

//go:linkname poll_runtime_pollWait internal/poll.runtime_pollWait
func poll_runtime_pollWait(ctx uintptr, mode int) int {
	if ctx == 0 {
		return pollErrNotPollable
	}
	pd := (*llgoPollDesc)(unsafe.Pointer(ctx))
	if latomic.LoadUint32(&pd.closing) != 0 {
		return pollErrClosing
	}
	pollOnce.Do(pollInit)
	if wakeR < 0 {
		return pollErrNotPollable
	}

	ev := pollIn
	if mode == 'w' {
		ev = pollOut
	}
	const maxWaitMs c.Int = 1000 // Ensure close/unblock is observed even if wake pipe misses.
	for {
		if latomic.LoadUint32(&pd.closing) != 0 {
			return pollErrClosing
		}

		timeout, derr := pollTimeoutMs(pollDeadline(pd, mode))
		if derr != pollNoError {
			return derr
		}
		// Avoid indefinite poll waits so close/unblock is always observed in
		// bounded time, even on platforms where closing the fd doesn't wake poll.
		allowTimeout := true
		if timeout < 0 || timeout > maxWaitMs {
			timeout = maxWaitMs
			allowTimeout = false
		}

		var fds [2]pollfd
		fds[0] = pollfd{fd: pd.fd, events: ev}
		fds[1] = pollfd{fd: wakeR, events: pollIn}

		n := c_poll(&fds[0], 2, timeout)
		if n < 0 {
			errno := cliteos.Errno()
			if int(errno) == int(csyscall.EINTR) {
				continue
			}
			// Treat unexpected poll errors as readiness; syscall will return the real error.
			return pollNoError
		}
		if n == 0 {
			if allowTimeout {
				return pollErrTimeout
			}
			continue
		}

		// Wake pipe readable: drain and re-check state/deadline.
		if fds[1].revents != 0 {
			pollDrainWake()
			if latomic.LoadUint32(&pd.closing) != 0 {
				return pollErrClosing
			}
			continue
		}

		// The fd is ready (or has an error/hangup); let the caller retry the syscall.
		if fds[0].revents != 0 {
			if latomic.LoadUint32(&pd.closing) != 0 {
				return pollErrClosing
			}
			return pollNoError
		}
	}
}

//go:linkname poll_runtime_pollWaitCanceled internal/poll.runtime_pollWaitCanceled
func poll_runtime_pollWaitCanceled(ctx uintptr, mode int) {
	// No-op: our poller doesn't track per-wait cancellation state.
}

//go:linkname poll_runtime_pollReset internal/poll.runtime_pollReset
func poll_runtime_pollReset(ctx uintptr, mode int) int {
	if ctx == 0 {
		return pollErrNotPollable
	}
	pd := (*llgoPollDesc)(unsafe.Pointer(ctx))
	if latomic.LoadUint32(&pd.closing) != 0 {
		return pollErrClosing
	}
	_, derr := pollTimeoutMs(pollDeadline(pd, mode))
	return derr
}

//go:linkname poll_runtime_pollSetDeadline internal/poll.runtime_pollSetDeadline
func poll_runtime_pollSetDeadline(ctx uintptr, d int64, mode int) {
	if ctx == 0 {
		return
	}
	pd := (*llgoPollDesc)(unsafe.Pointer(ctx))

	var abs int64
	if d != 0 {
		abs = runtimeNano() + d
	}

	switch mode {
	case 'r':
		latomic.StoreInt64(&pd.rd, abs)
	case 'w':
		latomic.StoreInt64(&pd.wd, abs)
	default:
		latomic.StoreInt64(&pd.rd, abs)
		latomic.StoreInt64(&pd.wd, abs)
	}
	// Wake any poll waiters to re-evaluate deadlines.
	pollWake()
}

//go:linkname poll_runtime_pollUnblock internal/poll.runtime_pollUnblock
func poll_runtime_pollUnblock(ctx uintptr) {
	if ctx == 0 {
		return
	}
	pd := (*llgoPollDesc)(unsafe.Pointer(ctx))
	latomic.StoreUint32(&pd.closing, 1)
	pollWake()
}

//go:linkname poll_runtime_isPollServerDescriptor internal/poll.runtime_isPollServerDescriptor
func poll_runtime_isPollServerDescriptor(fd uintptr) bool { return false }

func nanotime() int64 { return runtimeNano() }
