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
	"github.com/xgo-dev/llgo/runtime/internal/clite/pthread/sync"
	"github.com/xgo-dev/llgo/runtime/internal/runtime/math"
)

// -----------------------------------------------------------------------------

type Chan struct {
	mutex sync.Mutex

	qcount   int
	dataqsiz int
	buf      unsafe.Pointer
	elemsize int
	closed   bool
	recvx    int
	sendx    int

	sendq chanWaitq
	recvq chanWaitq
}

type chanWaitq struct {
	first *chanWaiter
	last  *chanWaiter
}

type chanWaiter struct {
	prev *chanWaiter
	next *chanWaiter
	all  *chanWaiter

	ch   *Chan
	elem unsafe.Pointer
	size int
	send bool

	queued bool
	status waitStatus

	mutex sync.Mutex
	cond  sync.Cond

	sel       *selectState
	caseIndex int
}

type selectState struct {
	mutex sync.Mutex
	cond  sync.Cond

	status waitStatus
	chosen int
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
	if cap < 0 {
		panicMakeChanSize()
	}
	mem, overflow := math.MulUintptr(uintptr(eltSize), uintptr(cap))
	if overflow || mem > maxAlloc {
		panicMakeChanSize()
	}
	ret := new(Chan)
	ret.elemsize = eltSize
	ret.dataqsiz = cap
	if cap > 0 {
		ret.buf = AllocU(mem)
	}
	ret.mutex.Init(nil)
	return ret
}

func panicMakeChanSize() {
	panic(errorString("makechan: size out of range"))
}

func ChanLen(p *Chan) (n int) {
	if p == nil {
		return 0
	}
	p.mutex.Lock()
	n = p.qcount
	p.mutex.Unlock()
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
	w := new(chanWaiter)
	w.ch = ch
	w.elem = elem
	w.size = eltSize
	w.send = send
	w.mutex.Init(nil)
	w.cond.Init(nil)
	w.mutex.Lock()
	return w
}

func newSelectState() *selectState {
	state := (*selectState)(c.Malloc(unsafe.Sizeof(selectState{})))
	if state == nil {
		panic("out of memory")
	}
	c.Memset(unsafe.Pointer(state), 0, unsafe.Sizeof(selectState{}))
	state.chosen = -1
	state.mutex.Init(nil)
	state.cond.Init(nil)
	return state
}

func freeSelectState(state *selectState) {
	c.Free(unsafe.Pointer(state))
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

func freeSelectWaiters(w *chanWaiter) {
	for w != nil {
		next := w.all
		c.Free(unsafe.Pointer(w))
		w = next
	}
}

func (w *chanWaiter) wait() {
	for !w.status.done() {
		w.cond.Wait(&w.mutex)
	}
	w.mutex.Unlock()
	w.cond.Destroy()
	w.mutex.Destroy()
}

func (w *chanWaiter) finish(status waitStatus) {
	if w.sel != nil {
		w.sel.mutex.Lock()
		w.sel.status = status
		w.sel.mutex.Unlock()
		w.sel.cond.Signal()
		return
	}
	w.mutex.Lock()
	w.status = status
	w.mutex.Unlock()
	w.cond.Signal()
}

func claimWaiter(w *chanWaiter) bool {
	if w.sel != nil {
		w.sel.mutex.Lock()
		if w.sel.status != waitPending {
			w.sel.mutex.Unlock()
			return false
		}
		w.sel.status = waitClaimed
		w.sel.chosen = w.caseIndex
		w.sel.mutex.Unlock()
		return true
	}
	return true
}

func completeRecvWaiter(w *chanWaiter, src unsafe.Pointer, eltSize int, status waitStatus) bool {
	if !claimWaiter(w) {
		return false
	}
	if status.recvOK() {
		copyChanElem(w.elem, src, eltSize)
	} else {
		zeroChanRecv(w.elem, eltSize)
	}
	w.finish(status)
	return true
}

func completeSendWaiter(w *chanWaiter, status waitStatus) bool {
	if !claimWaiter(w) {
		return false
	}
	w.finish(status)
	return true
}

func recvFromSendWaiter(dst unsafe.Pointer, w *chanWaiter, eltSize int) bool {
	if !claimWaiter(w) {
		return false
	}
	copyChanElem(dst, w.elem, eltSize)
	w.finish(waitSendOK)
	return true
}

func dequeueRecvAndComplete(p *Chan, src unsafe.Pointer, eltSize int, status waitStatus) bool {
	for {
		w := p.recvq.dequeue()
		if w == nil {
			return false
		}
		if completeRecvWaiter(w, src, eltSize, status) {
			return true
		}
	}
}

func dequeueSendAndRecv(p *Chan, dst unsafe.Pointer, eltSize int) bool {
	for {
		w := p.sendq.dequeue()
		if w == nil {
			return false
		}
		if recvFromSendWaiter(dst, w, eltSize) {
			return true
		}
	}
}

func chanTrySendLocked(p *Chan, v unsafe.Pointer, eltSize int) (tryOK bool, closed bool) {
	elemSize := p.elemsize
	if p.closed {
		return false, true
	}
	if dequeueRecvAndComplete(p, v, elemSize, waitRecvOK) {
		return true, false
	}
	if p.qcount < p.dataqsiz {
		copyChanElem(chanBuf(p, p.sendx), v, elemSize)
		p.sendx++
		if p.sendx == p.dataqsiz {
			p.sendx = 0
		}
		p.qcount++
		return true, false
	}
	return false, false
}

func ChanTrySend(p *Chan, v unsafe.Pointer, eltSize int) bool {
	if p == nil {
		return false
	}
	p.mutex.Lock()
	ok, closed := chanTrySendLocked(p, v, eltSize)
	p.mutex.Unlock()
	if closed {
		panicSendOnClosedChan()
	}
	return ok
}

func ChanSend(p *Chan, v unsafe.Pointer, eltSize int) bool {
	if p == nil {
		blockForever()
		return false
	}
	p.mutex.Lock()
	ok, closed := chanTrySendLocked(p, v, eltSize)
	if closed {
		p.mutex.Unlock()
		panicSendOnClosedChan()
	}
	if ok {
		p.mutex.Unlock()
		return true
	}
	w := newChanWaiter(p, v, eltSize, true)
	p.sendq.enqueue(w)
	p.mutex.Unlock()

	w.wait()
	if w.status.panicOnWake() {
		panicSendOnClosedChan()
	}
	return true
}

func chanTryRecvLocked(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool, tryOK bool) {
	elemSize := p.elemsize
	if p.dataqsiz == 0 {
		if dequeueSendAndRecv(p, v, elemSize) {
			return true, true
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
			w.finish(waitSendOK)
			break
		}
		return true, true
	}
	if p.closed {
		zeroChanRecv(v, elemSize)
		return false, true
	}
	return false, false
}

func ChanTryRecv(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool, tryOK bool) {
	if p == nil {
		return false, false
	}
	p.mutex.Lock()
	recvOK, tryOK = chanTryRecvLocked(p, v, eltSize)
	p.mutex.Unlock()
	return
}

func ChanRecv(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool) {
	if p == nil {
		blockForever()
		return false
	}
	p.mutex.Lock()
	if recvOK, tryOK := chanTryRecvLocked(p, v, eltSize); tryOK {
		p.mutex.Unlock()
		return recvOK
	}
	w := newChanWaiter(p, v, eltSize, false)
	p.recvq.enqueue(w)
	p.mutex.Unlock()

	w.wait()
	return w.status.recvOK()
}

func ChanClose(p *Chan) {
	if p == nil {
		panic("close of nil channel")
	}
	p.mutex.Lock()
	if p.closed {
		p.mutex.Unlock()
		panic("close of closed channel")
	}
	p.closed = true
	for {
		w := p.recvq.dequeue()
		if w == nil {
			break
		}
		if completeRecvWaiter(w, nil, p.elemsize, waitRecvClosed) {
			continue
		}
	}
	for {
		w := p.sendq.dequeue()
		if w == nil {
			break
		}
		if completeSendWaiter(w, waitSendClosed) {
			continue
		}
	}
	p.mutex.Unlock()
}

func blockForever() {
	var mutex sync.Mutex
	var cond sync.Cond
	mutex.Init(nil)
	cond.Init(nil)
	mutex.Lock()
	for {
		cond.Wait(&mutex)
	}
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
	n := len(ops)
	if n == 0 {
		return
	}
	start := selectStart(n)
	for i := 0; i < n; i++ {
		isel = (start + i) % n
		op := ops[isel]
		if op.C == nil {
			continue
		}
		op.C.mutex.Lock()
		if op.Send {
			var closed bool
			tryOK, closed = chanTrySendLocked(op.C, op.Val, int(op.Size))
			recvOK = true
			op.C.mutex.Unlock()
			if closed {
				panicSendOnClosedChan()
			}
		} else {
			recvOK, tryOK = chanTryRecvLocked(op.C, op.Val, int(op.Size))
			op.C.mutex.Unlock()
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
		if op.Send {
			var closed bool
			ready, closed = chanTrySendLocked(ch, op.Val, int(op.Size))
			if closed {
				unlockSelectChannels(&chans)
				panicSendOnClosedChan()
			}
			if ready {
				unlockSelectChannels(&chans)
				return i, true
			}
		} else {
			recvOK, ready = chanTryRecvLocked(ch, op.Val, int(op.Size))
			if ready {
				unlockSelectChannels(&chans)
				return i, recvOK
			}
		}
	}

	state := newSelectState()

	var waiters *chanWaiter
	var lastWaiter *chanWaiter
	for n := 0; n < len(ops); n++ {
		i := (start + n) % len(ops)
		op := ops[i]
		if op.C == nil {
			continue
		}
		w := newSelectWaiter(op.C, op.Val, int(op.Size), op.Send, state, i)
		if op.Send {
			op.C.sendq.enqueue(w)
		} else {
			op.C.recvq.enqueue(w)
		}
		if lastWaiter == nil {
			waiters = w
		} else {
			lastWaiter.all = w
		}
		lastWaiter = w
	}
	unlockSelectChannels(&chans)

	state.mutex.Lock()
	for !state.status.done() {
		state.cond.Wait(&state.mutex)
	}
	isel = state.chosen
	status := state.status
	recvOK = status.recvOK()
	state.mutex.Unlock()

	for w := waiters; w != nil; w = w.all {
		cleanupSelectWaiter(w)
	}
	state.cond.Destroy()
	state.mutex.Destroy()
	freeSelectState(state)
	freeSelectWaiters(waiters)
	if status.panicOnWake() {
		panicSendOnClosedChan()
	}
	return
}

func lockSelectChannels(chans *selectChanList) {
	for i := 0; i < chans.len; i++ {
		ch := chans.get(i)
		ch.mutex.Lock()
	}
}

func unlockSelectChannels(chans *selectChanList) {
	for i := chans.len - 1; i >= 0; i-- {
		chans.get(i).mutex.Unlock()
	}
}

func cleanupSelectWaiter(w *chanWaiter) {
	w.ch.mutex.Lock()
	if w.send {
		w.ch.sendq.remove(w)
	} else {
		w.ch.recvq.remove(w)
	}
	w.ch.mutex.Unlock()
}

func selectStart(n int) int {
	if n <= 1 {
		return 0
	}
	return int(fastrand() % uint32(n))
}

// -----------------------------------------------------------------------------
