/*
 * Copyright (c) 2024 The XGo Authors (xgo.dev). All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package runtime

import (
	"unsafe"

	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/math"
	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

// -----------------------------------------------------------------------------

type Chan struct {
	bubble *SynctestBubble
	mutex  chanMutex

	qcount   int
	dataqsiz int
	buf      unsafe.Pointer
	elemsize int
	closed   bool

	// When enabled, published under mutex: nonzero when closed, buffered, or a
	// sender is queued. Nonblocking receivers can skip the lock when it is zero.
	// Blocking receives and all sends still inspect state under the lock.
	recvReady uint32
	recvx     int
	sendx     int

	sendq chanWaitq
	recvq chanWaitq

	// Lock-free exponential-backoff spin cooldown word (address-wait fast-path
	// parkFast). Placed after the mutex-protected queue headers to keep
	// sleeping-waiter backoff writes away from recvReady/mutex hot fields.
	spinCooldown uint32
}

type chanWaitq struct {
	first *chanWaiter
	last  *chanWaiter
}

type chanWaiter struct {
	synctest *SynctestWaiter
	prev     *chanWaiter
	next     *chanWaiter
	all      *chanWaiter

	ch   *Chan
	elem unsafe.Pointer
	size int
	send bool

	queued   bool
	fastWait bool
	isHead   bool
	status   waitStatus

	// Non-nil for ordinary single-channel waiters; select waiters share
	// sel.signal so they do not carry unused per-case chanSignal storage.
	signal *chanSignal

	sel       *selectState
	caseIndex int
}

type singleChanWaiter struct {
	waiter      chanWaiter
	signal      chanSignal
	signalReady bool
}

type selectState struct {
	synctest *SynctestWaiter
	signal   chanSignal

	status   uint32
	chosen   int
	fastWait bool
}

type selectCache struct {
	state        selectState
	spinCooldown uint32
	waiters      [selectInlineChanCount]chanWaiter
}

func resetSelectState(state *selectState) {
	state.synctest = nil
	atomic.Store(&state.status, uint32(waitPending))
	state.chosen = -1
	state.fastWait = false
}

func acquireSelectCache() *selectCache {
	if gp := getg(); gp != nil && gp.selectCache != nil {
		sc := gp.selectCache
		gp.selectCache = nil
		resetSelectState(&sc.state)
		return sc
	}
	sc := (*selectCache)(c.Malloc(unsafe.Sizeof(selectCache{})))
	if sc == nil {
		panic("out of memory")
	}
	c.Memset(unsafe.Pointer(sc), 0, unsafe.Sizeof(selectCache{}))
	sc.state.signal.initFast()
	resetSelectState(&sc.state)
	return sc
}

func releaseSelectCache(sc *selectCache, used int) {
	sc.state.synctest = nil
	for i := 0; i < used; i++ {
		sc.waiters[i] = chanWaiter{}
	}
	if gp := getg(); gp != nil && gp.selectCache == nil {
		gp.selectCache = sc
		return
	}
	sc.state.signal.destroy()
	c.Free(unsafe.Pointer(sc))
}

func freeSelectCache(gp *g) {
	if gp != nil && gp.selectCache != nil {
		gp.selectCache.state.signal.destroy()
		c.Free(unsafe.Pointer(gp.selectCache))
		gp.selectCache = nil
	}
}

func prepareSelectCacheForTesting() bool {
	if getg().selectCache != nil {
		return true
	}
	releaseSelectCache(acquireSelectCache(), 0)
	return false
}

func freeChanWaiter(gp *g) {
	if gp != nil && gp.chanWaiter != nil {
		if gp.chanWaiter.signalReady {
			gp.chanWaiter.signal.destroy()
		}
		gp.chanWaiter = nil
	}
}

type waitStatus uint8

const (
	waitPending waitStatus = iota
	waitClaimed
	waitRecvOK
	waitRecvClosed
	waitSendOK
	waitSendClosed
)

func (s waitStatus) done() bool {
	return s >= waitRecvOK
}

func (s waitStatus) recvOK() bool {
	return s == waitRecvOK || s == waitSendOK
}

func (s waitStatus) panicOnWake() bool {
	return s == waitSendClosed
}

func (q *chanWaitq) enqueue(w *chanWaiter) {
	w.prev = q.last
	w.next = nil
	w.queued = true
	w.isHead = q.last == nil
	if q.last == nil {
		q.first = w
	} else {
		q.last.next = w
	}
	q.last = w
}

func (q *chanWaitq) dequeue() *chanWaiter {
	w := q.first
	if w != nil {
		q.remove(w)
	}
	return w
}

func (q *chanWaitq) remove(w *chanWaiter) {
	if !w.queued {
		return
	}
	if w.prev == nil {
		q.first = w.next
	} else {
		w.prev.next = w.next
	}
	if w.next == nil {
		q.last = w.prev
	} else {
		w.next.prev = w.prev
	}
	w.prev = nil
	w.next = nil
	w.queued = false
}

func NewChan(eltSize, cap int) *Chan {
	return NewChanAligned(eltSize, cap, 1)
}

func NewChanAligned(eltSize, cap, eltAlign int) *Chan {
	if cap < 0 {
		panicMakeChanSize()
	}
	mem, overflow := math.MulUintptr(uintptr(eltSize), uintptr(cap))
	if overflow || mem > maxAlloc {
		panicMakeChanSize()
	}
	ret := new(Chan)
	ret.bubble = CurrentSynctestBubble()
	ret.elemsize = eltSize
	ret.dataqsiz = cap
	if cap > 0 {
		ret.buf = AllocUAligned(mem, uintptr(eltAlign))
	}
	ret.mutex.init()
	return ret
}

func panicMakeChanSize() {
	panic(errorString("makechan: size out of range"))
}

// unlock publishes a receive-readiness snapshot where enabled. A nonblocking
// receiver may return default while another operation is still in progress;
// once that operation unlocks, its readiness is visible through the atomic.
// A positive snapshot is only a hint: selection and data access still lock.
func (p *Chan) unlock() {
	p.publishRecvReady()
	p.mutex.Unlock()
}

func ChanLen(p *Chan) (n int) {
	if p == nil {
		return 0
	}
	p.mutex.Lock()
	n = p.qcount
	p.unlock()
	return
}

func ChanCap(p *Chan) int {
	if p == nil {
		return 0
	}
	return p.dataqsiz
}

func panicSendOnClosedChan() {
	panic("send on closed channel")
}

func zeroChanRecv(v unsafe.Pointer, eltSize int) {
	if v != nil && eltSize > 0 {
		c.Memset(v, 0, uintptr(eltSize))
	}
}

func copyChanElem(dst, src unsafe.Pointer, eltSize int) {
	if dst != nil && src != nil && eltSize > 0 {
		c.Memcpy(dst, src, uintptr(eltSize))
	}
}

func chanBuf(p *Chan, i int) unsafe.Pointer {
	return c.Advance(p.buf, i*p.elemsize)
}

func newChanWaiter(ch *Chan, elem unsafe.Pointer, eltSize int, send bool) *chanWaiter {
	var sw *singleChanWaiter
	if gp := getg(); gp != nil {
		sw = gp.chanWaiter
		gp.chanWaiter = nil
	}
	if sw == nil {
		sw = new(singleChanWaiter)
	}
	if !sw.signalReady {
		sw.signal.initFast()
		sw.signalReady = true
	}
	w := &sw.waiter
	w.synctest = nil
	w.prev = nil
	w.next = nil
	w.all = nil
	w.queued = false
	w.isHead = false
	w.status = waitPending
	w.sel = nil
	w.caseIndex = 0
	if ch.bubble != nil {
		w.synctest = NewSynctestWaiter()
	}
	w.ch = ch
	w.elem = elem
	w.size = eltSize
	w.send = send
	w.signal = &sw.signal
	if w.synctest == nil && sw.signal.useFast() {
		w.fastWait = true
		sw.signal.resetFast()
		return w
	}
	w.fastWait = false
	sw.signal.ensureMutexInit()
	sw.signal.lock()
	return w
}

func newSelectWaiter(ch *Chan, elem unsafe.Pointer, eltSize int, send bool, state *selectState, caseIndex int) *chanWaiter {
	w := (*chanWaiter)(c.Malloc(unsafe.Sizeof(chanWaiter{})))
	if w == nil {
		panic("out of memory")
	}
	c.Memset(unsafe.Pointer(w), 0, unsafe.Sizeof(chanWaiter{}))
	w.ch = ch
	w.elem = elem
	w.size = eltSize
	w.send = send
	w.sel = state
	w.caseIndex = caseIndex
	return w
}

func (w *chanWaiter) wait() waitStatus {
	sig := w.signal
	var status waitStatus
	if w.fastWait {
		sig.parkFast(&w.ch.spinCooldown, w.isHead)
		status = w.status
		// Clear channel/element pointers before caching, including on panic wakes,
		// while keeping the lock-free signal initialized for reuse.
		w.synctest = nil
		w.prev = nil
		w.next = nil
		w.all = nil
		w.ch = nil
		w.elem = nil
		w.sel = nil
		w.queued = false
		w.fastWait = false
		w.isHead = false
		w.status = waitPending
		sw := (*singleChanWaiter)(unsafe.Pointer(w))
		if gp := getg(); gp != nil && gp.chanWaiter == nil {
			gp.chanWaiter = sw
		} else {
			sig.destroy()
		}
		return status
	}
	for !w.status.done() {
		w.synctest.Block()
		sig.park()
	}
	sig.unlock()
	sig.destroy()
	status = w.status
	sw := (*singleChanWaiter)(unsafe.Pointer(w))
	*sw = singleChanWaiter{}
	if gp := getg(); gp != nil && gp.chanWaiter == nil {
		gp.chanWaiter = sw
	}
	return status
}

// finish keeps the signal locked through ready on the mutex/cond backend. On
// the lock-free fast path (non-synctest Darwin/Linux/Windows address-wait),
// readyFast uses a 4-state atomic handshake (sigPending/sigSleeping ->
// sigWaking -> sigDone) so the waiter never reuses or destroys its signal until
// readyFast completes.
func (w *chanWaiter) finish(status waitStatus) {
	if w.sel != nil {
		if w.sel.fastWait {
			w.sel.chosen = w.caseIndex
			atomic.Store(&w.sel.status, uint32(status))
			w.sel.signal.readyFast()
			return
		}
		w.sel.signal.lock()
		atomic.Store(&w.sel.status, uint32(status))
		w.sel.synctest.Wake()
		w.sel.signal.ready()
		w.sel.signal.unlock()
		return
	}
	if w.fastWait {
		w.status = status
		w.signal.readyFast()
		return
	}
	w.signal.lock()
	w.status = status
	w.synctest.Wake()
	w.signal.ready()
	w.signal.unlock()
}

func claimWaiter(w *chanWaiter) bool {
	if w.sel != nil {
		if w.sel.fastWait {
			_, won := atomic.CompareAndExchange(&w.sel.status, uint32(waitPending), uint32(waitClaimed))
			return won
		}
		w.sel.signal.lock()
		if waitStatus(atomic.Load(&w.sel.status)) != waitPending {
			w.sel.signal.unlock()
			return false
		}
		atomic.Store(&w.sel.status, uint32(waitClaimed))
		w.sel.chosen = w.caseIndex
		w.sel.signal.unlock()
		return true
	}
	return true
}

// wakeUnderLock reports whether w must be woken before releasing Chan.mutex.
// Durable synctest waiters keep in-lock completion so bubble wake/idle accounting
// stays serialized with the channel state transition. Non-durable waiters
// (including a bubbled goroutine operating on an unbubbled channel, where
// synctest is nil) defer finish() until after Chan.mutex is released so OS wake
// syscalls do not extend the channel critical section.
func (w *chanWaiter) wakeUnderLock() bool {
	return w.synctest != nil || (w.sel != nil && w.sel.synctest != nil)
}

func completeRecvWaiter(w *chanWaiter, src unsafe.Pointer, eltSize int, status waitStatus) (bool, *chanWaiter) {
	if !claimWaiter(w) {
		return false, nil
	}
	if status.recvOK() {
		copyChanElem(w.elem, src, eltSize)
	} else {
		zeroChanRecv(w.elem, eltSize)
	}
	if w.wakeUnderLock() {
		w.finish(status)
		return true, nil
	}
	return true, w
}

func completeSendWaiter(w *chanWaiter, status waitStatus) (bool, *chanWaiter) {
	if !claimWaiter(w) {
		return false, nil
	}
	if w.wakeUnderLock() {
		w.finish(status)
		return true, nil
	}
	return true, w
}

func recvFromSendWaiter(dst unsafe.Pointer, w *chanWaiter, eltSize int) (bool, *chanWaiter) {
	if !claimWaiter(w) {
		return false, nil
	}
	copyChanElem(dst, w.elem, eltSize)
	if w.wakeUnderLock() {
		w.finish(waitSendOK)
		return true, nil
	}
	return true, w
}

func dequeueRecvAndComplete(p *Chan, src unsafe.Pointer, eltSize int, status waitStatus) (bool, *chanWaiter) {
	for {
		w := p.recvq.dequeue()
		if w == nil {
			return false, nil
		}
		if ok, wake := completeRecvWaiter(w, src, eltSize, status); ok {
			return true, wake
		}
	}
}

func dequeueSendAndRecv(p *Chan, dst unsafe.Pointer, eltSize int) (bool, *chanWaiter) {
	for {
		w := p.sendq.dequeue()
		if w == nil {
			return false, nil
		}
		if ok, wake := recvFromSendWaiter(dst, w, eltSize); ok {
			return true, wake
		}
	}
}

func chanTrySendLocked(p *Chan, v unsafe.Pointer, eltSize int) (tryOK bool, closed bool, wake *chanWaiter) {
	elemSize := p.elemsize
	if p.closed {
		return false, true, nil
	}
	if ok, wake := dequeueRecvAndComplete(p, v, elemSize, waitRecvOK); ok {
		return true, false, wake
	}
	if p.qcount < p.dataqsiz {
		copyChanElem(chanBuf(p, p.sendx), v, elemSize)
		p.sendx++
		if p.sendx == p.dataqsiz {
			p.sendx = 0
		}
		p.qcount++
		return true, false, nil
	}
	return false, false, nil
}

func ChanTrySend(p *Chan, v unsafe.Pointer, eltSize int) bool {
	checkSynctestChannel(p)
	if p == nil {
		return false
	}
	p.mutex.Lock()
	ok, closed, wake := chanTrySendLocked(p, v, eltSize)
	p.unlock()
	if wake != nil {
		wake.finish(waitRecvOK)
	}
	if closed {
		panicSendOnClosedChan()
	}
	return ok
}

func ChanSend(p *Chan, v unsafe.Pointer, eltSize int) bool {
	checkSynctestChannel(p)
	if p == nil {
		blockForever()
		return false
	}
	p.mutex.Lock()
	ok, closed, wake := chanTrySendLocked(p, v, eltSize)
	if closed {
		p.unlock()
		panicSendOnClosedChan()
	}
	if ok {
		p.unlock()
		if wake != nil {
			wake.finish(waitRecvOK)
		}
		return true
	}
	w := newChanWaiter(p, v, eltSize, true)
	p.sendq.enqueue(w)
	p.unlock()

	if w.wait().panicOnWake() {
		panicSendOnClosedChan()
	}
	return true
}

func chanTryRecvLocked(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool, tryOK bool, wake *chanWaiter) {
	elemSize := p.elemsize
	if p.dataqsiz == 0 {
		if ok, wake := dequeueSendAndRecv(p, v, elemSize); ok {
			return true, true, wake
		}
	} else if p.qcount > 0 {
		copyChanElem(v, chanBuf(p, p.recvx), elemSize)
		zeroChanRecv(chanBuf(p, p.recvx), elemSize)
		p.recvx++
		if p.recvx == p.dataqsiz {
			p.recvx = 0
		}
		p.qcount--
		for p.qcount < p.dataqsiz {
			w := p.sendq.dequeue()
			if w == nil {
				break
			}
			if !claimWaiter(w) {
				continue
			}
			copyChanElem(chanBuf(p, p.sendx), w.elem, elemSize)
			p.sendx++
			if p.sendx == p.dataqsiz {
				p.sendx = 0
			}
			p.qcount++
			if w.wakeUnderLock() {
				w.finish(waitSendOK)
			} else {
				wake = w
			}
			break
		}
		return true, true, wake
	}
	if p.closed {
		zeroChanRecv(v, elemSize)
		return false, true, nil
	}
	return false, false, nil
}

func ChanTryRecv(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool, tryOK bool) {
	checkSynctestChannel(p)
	if p == nil || !p.recvMayBeReady() {
		return false, false
	}
	p.mutex.Lock()
	var wake *chanWaiter
	recvOK, tryOK, wake = chanTryRecvLocked(p, v, eltSize)
	p.unlock()
	if wake != nil {
		wake.finish(waitSendOK)
	}
	return
}

func ChanRecv(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool) {
	checkSynctestChannel(p)
	if p == nil {
		blockForever()
		return false
	}
	p.mutex.Lock()
	if recvOK, tryOK, wake := chanTryRecvLocked(p, v, eltSize); tryOK {
		p.unlock()
		if wake != nil {
			wake.finish(waitSendOK)
		}
		return recvOK
	}
	w := newChanWaiter(p, v, eltSize, false)
	p.recvq.enqueue(w)
	p.unlock()

	return w.wait().recvOK()
}

func ChanClose(p *Chan) {
	checkSynctestChannel(p)
	if p == nil {
		panic("close of nil channel")
	}
	p.mutex.Lock()
	if p.closed {
		p.unlock()
		panic("close of closed channel")
	}
	p.closed = true
	// Publish the closed bit before waking any synctest waiter in-lock and before
	// releasing p.mutex so subsequent lock-free close checks observe close.
	p.publishRecvClosed()
	// Reusing w.next for the deferred wake lists is safe because dequeue cleared
	// next/queued, at most one waiter per select can win claimWaiter, and each
	// claimed waiter cannot return or be cleaned up until w.finish runs below.
	var wakeRecvs, wakeRecvsTail *chanWaiter
	for {
		w := p.recvq.dequeue()
		if w == nil {
			break
		}
		if ok, wake := completeRecvWaiter(w, nil, p.elemsize, waitRecvClosed); ok && wake != nil {
			if wakeRecvsTail == nil {
				wakeRecvs = wake
			} else {
				wakeRecvsTail.next = wake
			}
			wakeRecvsTail = wake
		}
	}
	var wakeSends, wakeSendsTail *chanWaiter
	for {
		w := p.sendq.dequeue()
		if w == nil {
			break
		}
		if ok, wake := completeSendWaiter(w, waitSendClosed); ok && wake != nil {
			if wakeSendsTail == nil {
				wakeSends = wake
			} else {
				wakeSendsTail.next = wake
			}
			wakeSendsTail = wake
		}
	}
	p.unlock()
	for w := wakeRecvs; w != nil; {
		next := w.next
		w.next = nil
		w.finish(waitRecvClosed)
		w = next
	}
	for w := wakeSends; w != nil; {
		next := w.next
		w.next = nil
		w.finish(waitSendClosed)
		w = next
	}
}

func blockForever() {
	chanBlockForever()
}

// -----------------------------------------------------------------------------

// ChanOp represents a channel operation.
type ChanOp struct {
	C *Chan

	Val  unsafe.Pointer
	Size int32

	Send bool
}

const selectInlineChanCount = 8

type selectChanList struct {
	len    int
	inline [selectInlineChanCount]*Chan
	extra  []*Chan
}

func (l *selectChanList) get(i int) *Chan {
	if l.extra != nil {
		return l.extra[i]
	}
	return l.inline[i]
}

func (l *selectChanList) set(i int, ch *Chan) {
	if l.extra != nil {
		l.extra[i] = ch
		return
	}
	l.inline[i] = ch
}

func (l *selectChanList) insert(pos int, ch *Chan) {
	if l.extra != nil {
		l.extra = append(l.extra, nil)
		copy(l.extra[pos+1:], l.extra[pos:])
		l.extra[pos] = ch
		l.len++
		return
	}
	if l.len == selectInlineChanCount {
		extra := make([]*Chan, selectInlineChanCount+1, selectInlineChanCount*2)
		copy(extra, l.inline[:pos])
		extra[pos] = ch
		copy(extra[pos+1:], l.inline[pos:])
		l.extra = extra
		l.len++
		return
	}
	l.len++
	for i := l.len - 1; i > pos; i-- {
		l.set(i, l.get(i-1))
	}
	l.set(pos, ch)
}

func (l *selectChanList) add(ch *Chan) {
	addr := uintptr(unsafe.Pointer(ch))
	pos := 0
	for pos < l.len {
		cur := uintptr(unsafe.Pointer(l.get(pos)))
		if cur == addr {
			return
		}
		if cur > addr {
			break
		}
		pos++
	}
	l.insert(pos, ch)
}

// TrySelect executes a non-blocking select operation.
func TrySelect(ops ...ChanOp) (isel int, recvOK, tryOK bool) {
	for _, op := range ops {
		checkSynctestChannel(op.C)
	}
	n := len(ops)
	if n == 0 {
		return
	}
	start := selectStart(n)
	for i := 0; i < n; i++ {
		isel = (start + i) % n
		op := ops[isel]
		if op.C == nil || (!op.Send && !op.C.recvMayBeReady()) {
			continue
		}
		op.C.mutex.Lock()
		var wake *chanWaiter
		if op.Send {
			var closed bool
			tryOK, closed, wake = chanTrySendLocked(op.C, op.Val, int(op.Size))
			recvOK = true
			op.C.unlock()
			if wake != nil {
				wake.finish(waitRecvOK)
			}
			if closed {
				panicSendOnClosedChan()
			}
		} else {
			recvOK, tryOK, wake = chanTryRecvLocked(op.C, op.Val, int(op.Size))
			op.C.unlock()
			if wake != nil {
				wake.finish(waitSendOK)
			}
		}
		if tryOK {
			return
		}
	}
	return
}

// Select executes a blocking select operation.
func Select(ops ...ChanOp) (isel int, recvOK bool) {
	if isel, recvOK, ok := TrySelect(ops...); ok {
		return isel, recvOK
	}

	var chans selectChanList
	for _, op := range ops {
		ch := op.C
		if ch == nil {
			continue
		}
		chans.add(ch)
	}
	if chans.len == 0 {
		blockForever()
	}
	lockSelectChannels(&chans)

	start := selectStart(len(ops))
	for n := 0; n < len(ops); n++ {
		i := (start + n) % len(ops)
		op := ops[i]
		if op.C == nil {
			continue
		}
		ch := op.C
		var ready bool
		var wake *chanWaiter
		if op.Send {
			var closed bool
			ready, closed, wake = chanTrySendLocked(ch, op.Val, int(op.Size))
			if closed {
				unlockSelectChannels(&chans)
				panicSendOnClosedChan()
			}
			if ready {
				unlockSelectChannels(&chans)
				if wake != nil {
					wake.finish(waitRecvOK)
				}
				return i, true
			}
		} else {
			recvOK, ready, wake = chanTryRecvLocked(ch, op.Val, int(op.Size))
			if ready {
				unlockSelectChannels(&chans)
				if wake != nil {
					wake.finish(waitSendOK)
				}
				return i, recvOK
			}
		}
	}

	sc := acquireSelectCache()
	state := &sc.state
	durable := CurrentSynctestBubble() != nil
	for _, op := range ops {
		if op.C != nil && op.C.bubble == nil {
			durable = false
		}
	}
	if durable {
		state.synctest = NewSynctestWaiter()
	}
	if state.synctest == nil && state.signal.useFast() {
		state.fastWait = true
		state.signal.resetFast()
	} else {
		state.fastWait = false
		state.signal.ensureMutexInit()
	}

	inlineUsed := 0
	hasHeapWaiters := false
	anyHead := false
	var waiters *chanWaiter
	var lastWaiter *chanWaiter
	for n := 0; n < len(ops); n++ {
		i := (start + n) % len(ops)
		op := ops[i]
		if op.C == nil {
			continue
		}
		var w *chanWaiter
		if inlineUsed < len(sc.waiters) {
			w = &sc.waiters[inlineUsed]
			inlineUsed++
			w.ch = op.C
			w.elem = op.Val
			w.size = int(op.Size)
			w.send = op.Send
			w.sel = state
			w.caseIndex = i
		} else {
			w = newSelectWaiter(op.C, op.Val, int(op.Size), op.Send, state, i)
			hasHeapWaiters = true
		}
		if op.Send {
			op.C.sendq.enqueue(w)
		} else {
			op.C.recvq.enqueue(w)
		}
		if w.isHead {
			anyHead = true
		}
		if lastWaiter == nil {
			waiters = w
		} else {
			lastWaiter.all = w
		}
		lastWaiter = w
	}
	unlockSelectChannels(&chans)

	var status waitStatus
	if state.fastWait {
		state.signal.parkFast(&sc.spinCooldown, anyHead)
		isel = state.chosen
		status = waitStatus(atomic.Load(&state.status))
		recvOK = status.recvOK()
	} else {
		state.signal.lock()
		for !waitStatus(atomic.Load(&state.status)).done() {
			state.synctest.Block()
			state.signal.park()
		}
		isel = state.chosen
		status = waitStatus(atomic.Load(&state.status))
		recvOK = status.recvOK()
		state.signal.unlock()
	}

	for w := waiters; w != nil; w = w.all {
		if w.caseIndex == isel {
			// Safe to skip cleanupSelectWaiter for the winner: every claimWaiter
			// call runs immediately after dequeue under ch.mutex, so the winner
			// is already unlinked and publishRecvReady ran when ch unlocked.
			continue
		}
		cleanupSelectWaiter(w)
	}
	if hasHeapWaiters {
		for w := waiters; w != nil; {
			next := w.all
			if !isInlineSelectWaiter(w, &sc.waiters) {
				c.Free(unsafe.Pointer(w))
			}
			w = next
		}
	}
	releaseSelectCache(sc, inlineUsed)
	if status.panicOnWake() {
		panicSendOnClosedChan()
	}
	return
}

func isInlineSelectWaiter(w *chanWaiter, buf *[selectInlineChanCount]chanWaiter) bool {
	p := uintptr(unsafe.Pointer(w))
	lo := uintptr(unsafe.Pointer(&buf[0]))
	hi := lo + unsafe.Sizeof(*buf)
	return p >= lo && p < hi
}

func lockSelectChannels(chans *selectChanList) {
	for i := 0; i < chans.len; i++ {
		ch := chans.get(i)
		ch.mutex.Lock()
	}
}

func unlockSelectChannels(chans *selectChanList) {
	for i := chans.len - 1; i >= 0; i-- {
		chans.get(i).unlock()
	}
}

func cleanupSelectWaiter(w *chanWaiter) {
	w.ch.mutex.Lock()
	if w.send {
		w.ch.sendq.remove(w)
	} else {
		w.ch.recvq.remove(w)
	}
	w.ch.unlock()
}

func selectStart(n int) int {
	if n <= 1 {
		return 0
	}
	return int(fastrand() % uint32(n))
}

// -----------------------------------------------------------------------------

func checkSynctestChannel(p *Chan) {
	if p != nil && p.bubble != nil && p.bubble != CurrentSynctestBubble() {
		panic("channel operation from outside synctest bubble")
	}
}
