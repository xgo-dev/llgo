//go:build (darwin || linux) && !baremetal

package runtime

import (
	"unsafe"

	latomic "sync/atomic"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	cliteos "github.com/xgo-dev/llgo/runtime/internal/clite/os"
	csyscall "github.com/xgo-dev/llgo/runtime/internal/clite/syscall"
	psync "github.com/xgo-dev/llgo/runtime/internal/sync"
)

// Minimal runtime netpoll backing for internal/poll.
//
// internal/poll assumes non-blocking I/O and a working runtime poller when
// pollDesc.init succeeds. Our previous stubs returned "success" but never
// actually blocked or unblocked, which caused tight spins and deadlocks (e.g.
// listener Close waiting forever because Accept loops never observed "closing").
//
// Each blocked operation uses libc poll(2) with its own wake pipe. Close and
// deadline changes signal every affected operation: a different descriptor or
// a concurrent read/write waiter must not consume its notification.

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

	// Protected by pollDescMu, including writes and removal/closing of pipes.
	waiters *pollWaiter
}

type pollWaiter struct {
	next *pollWaiter
	mode int
	r, w c.Int
}

var pollOnce psync.Once
var pollDescMu psync.Mutex
var pollDescRoots map[uintptr]*llgoPollDesc

func pollInit() {
	pollDescMu.Init(nil)
	pollDescRoots = make(map[uintptr]*llgoPollDesc)
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

// Register before inspecting the deadline or closing state. A state change
// either precedes that inspection or finds this waiter under pollDescMu.
func pollAddWaiter(pd *llgoPollDesc, mode int) *pollWaiter {
	var fds [2]c.Int
	if !pollPipe(&fds) {
		return nil
	}
	w := &pollWaiter{mode: mode, r: fds[0], w: fds[1]}
	pollDescMu.Lock()
	w.next = pd.waiters
	pd.waiters = w
	pollDescMu.Unlock()
	return w
}

func pollRemoveWaiter(pd *llgoPollDesc, w *pollWaiter) {
	pollDescMu.Lock()
	for link := &pd.waiters; *link != nil; link = &(*link).next {
		if *link == w {
			*link = w.next
			break
		}
	}
	// A notifier cannot write after close, even if these descriptor numbers
	// are immediately reused by a different thread.
	cliteos.Close(w.r)
	cliteos.Close(w.w)
	pollDescMu.Unlock()
}

func pollWake(pd *llgoPollDesc, mode int) {
	pollDescMu.Lock()
	for w := pd.waiters; w != nil; w = w.next {
		if mode != 0 && mode != 'r'+'w' && mode != w.mode {
			continue
		}
		// Only readability matters; repeated notifications may coalesce.
		var b byte
		for cliteos.Write(w.w, unsafe.Pointer(&b), 1) < 0 {
			if int(cliteos.Errno()) != int(csyscall.EINTR) {
				// EAGAIN means a notification is already pending. Other
				// errors also stop retries so remaining waiters can be signaled.
				break
			}
		}
	}
	pollDescMu.Unlock()
}

func pollDrainWake(w *pollWaiter) {
	var buf [64]byte
	for {
		n := cliteos.Read(w.r, unsafe.Pointer(&buf[0]), uintptr(len(buf)))
		if n < 0 && int(cliteos.Errno()) == int(csyscall.EINTR) {
			continue
		}
		if n <= 0 {
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
	if pollDescriptorUnsupported(c.Int(fd)) {
		// Linux epoll rejects regular files and directories with EPERM. Match
		// that contract so internal/poll leaves their runtime context unset and
		// reports ErrNoDeadline from os.File deadline methods.
		return 0, int(csyscall.EPERM)
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
	pollWake(pd, 0)
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
	w := pollAddWaiter(pd, mode)
	nfds := uintptr(1)
	if w != nil {
		defer pollRemoveWaiter(pd, w)
		nfds = 2
	}

	ev := pollIn
	if mode == 'w' {
		ev = pollOut
	}
	for {
		if latomic.LoadUint32(&pd.closing) != 0 {
			return pollErrClosing
		}

		timeout, derr := pollTimeoutMs(pollDeadline(pd, mode))
		if derr != pollNoError {
			return derr
		}
		// Under fd pressure a wake pipe may be unavailable. Keep I/O working
		// and periodically observe Close/deadline changes without that pipe.
		if w == nil && (timeout < 0 || timeout > 1000) {
			timeout = 1000
		}

		var fds [2]pollfd
		fds[0] = pollfd{fd: pd.fd, events: ev}
		if w != nil {
			fds[1] = pollfd{fd: w.r, events: pollIn}
		}

		n := c_poll(&fds[0], nfds, timeout)
		if n < 0 {
			errno := cliteos.Errno()
			if int(errno) == int(csyscall.EINTR) {
				continue
			}
			// Treat unexpected poll errors as readiness; syscall will return the real error.
			return pollNoError
		}
		if n == 0 {
			// Re-read the deadline: it may have been extended or cleared
			// concurrently with poll's timeout.
			continue
		}

		// Wake pipe readable: drain and re-check state/deadline.
		if w != nil && fds[1].revents != 0 {
			pollDrainWake(w)
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
	// Wake affected poll waiters to re-evaluate deadlines.
	pollWake(pd, mode)
}

//go:linkname poll_runtime_pollUnblock internal/poll.runtime_pollUnblock
func poll_runtime_pollUnblock(ctx uintptr) {
	if ctx == 0 {
		return
	}
	pd := (*llgoPollDesc)(unsafe.Pointer(ctx))
	latomic.StoreUint32(&pd.closing, 1)
	pollWake(pd, 0)
}

//go:linkname poll_runtime_isPollServerDescriptor internal/poll.runtime_isPollServerDescriptor
func poll_runtime_isPollServerDescriptor(fd uintptr) bool { return false }
