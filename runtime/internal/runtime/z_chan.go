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

	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/pthread/sync"
	"github.com/goplus/llgo/runtime/internal/runtime/math"
)

// -----------------------------------------------------------------------------

const (
	chanNoSendRecv = 0
	chanHasRecv    = 1
)

type Chan struct {
	mutex sync.Mutex
	cond  sync.Cond
	data  unsafe.Pointer
	getp  int
	// seq changes when an unbuffered send/recv rendezvous completes. A later
	// recv on the same channel can set getp back to chanHasRecv before the
	// previous recv wakes up, so recv waiters must not use getp alone to decide
	// whether their own rendezvous completed.
	seq  uint64
	len  int
	cap  int
	sop  *selectOp
	sops []*selectOp
	// sends counts goroutines blocked in unbuffered send, including select-send.
	sends uint16
	// selsends is the subset of sends originating from select operations.
	selsends uint16
	close    bool
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
	if cap > 0 {
		ret.data = AllocU(mem)
		ret.cap = cap
	}
	ret.mutex.Init(nil)
	ret.cond.Init(nil)
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
	n = p.len
	p.mutex.Unlock()
	return
}

func ChanCap(p *Chan) int {
	if p == nil {
		return 0
	}
	return p.cap
}

func notifyOps(p *Chan) {
	if p.sop != nil {
		p.sop.notify()
	}
	for _, sop := range p.sops {
		sop.notify()
	}
}

func ChanClose(p *Chan) {
	if p == nil {
		panic("close of nil channel")
	}
	p.mutex.Lock()
	if p.close {
		p.mutex.Unlock()
		panic("close of closed channel")
	}
	p.close = true
	notifyOps(p)
	p.mutex.Unlock()
	p.cond.Broadcast()
}

func panicSendOnClosedChan() {
	panic("send on closed channel")
}

func zeroChanRecv(v unsafe.Pointer, eltSize int) {
	if v != nil && eltSize > 0 {
		c.Memset(v, 0, uintptr(eltSize))
	}
}

func ChanTrySend(p *Chan, v unsafe.Pointer, eltSize int) bool {
	if p == nil {
		return false
	}
	n := p.cap
	p.mutex.Lock()
	if p.close {
		p.mutex.Unlock()
		panicSendOnClosedChan()
	}
	if n == 0 {
		if p.getp != chanHasRecv {
			p.mutex.Unlock()
			return false
		}
		if p.data != nil {
			c.Memcpy(p.data, v, uintptr(eltSize))
		}
		p.getp = chanNoSendRecv
		p.seq++
	} else {
		if p.len == n {
			p.mutex.Unlock()
			return false
		}
		off := (p.getp + p.len) % n
		c.Memcpy(c.Advance(p.data, off*eltSize), v, uintptr(eltSize))
		p.len++
	}
	notifyOps(p)
	p.mutex.Unlock()
	p.cond.Broadcast()
	return true
}

func ChanSend(p *Chan, v unsafe.Pointer, eltSize int) bool {
	if p == nil {
		blockForever()
		return false
	}
	n := p.cap
	p.mutex.Lock()
	if n == 0 {
		for p.getp != chanHasRecv && !p.close {
			p.sends++
			// A blocked unbuffered send must wake select-based receivers so they can
			// retry ChanTryRecv after observing that a sender is now waiting.
			if p.sends == 1 || p.sends-1 == p.selsends {
				notifyOps(p)
			}
			p.cond.Wait(&p.mutex)
			p.sends--
		}
		if p.close {
			p.mutex.Unlock()
			panicSendOnClosedChan()
		}
		if p.data != nil {
			c.Memcpy(p.data, v, uintptr(eltSize))
		}
		p.getp = chanNoSendRecv
		p.seq++
	} else {
		for p.len == n && !p.close {
			p.cond.Wait(&p.mutex)
		}
		if p.close {
			p.mutex.Unlock()
			panicSendOnClosedChan()
		}
		off := (p.getp + p.len) % n
		c.Memcpy(c.Advance(p.data, off*eltSize), v, uintptr(eltSize))
		p.len++
	}
	notifyOps(p)
	p.mutex.Unlock()
	p.cond.Broadcast()
	return true
}

func ChanTryRecv(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool, tryOK bool) {
	if p == nil {
		return false, false
	}
	return chanTryRecv(p, v, eltSize, true)
}

func chanTryRecv(p *Chan, v unsafe.Pointer, eltSize int, acceptSelectSend bool) (recvOK bool, tryOK bool) {
	n := p.cap
	p.mutex.Lock()
	if n == 0 {
		if p.sends == 0 || p.getp == chanHasRecv || p.close {
			tryOK = p.close
			if p.close {
				zeroChanRecv(v, eltSize)
			}
			p.mutex.Unlock()
			return
		}
		if !acceptSelectSend && p.sends == p.selsends {
			p.mutex.Unlock()
			return
		}
		p.getp = chanHasRecv
		p.data = v
		seq := p.seq
		notifyOps(p)
		p.mutex.Unlock()
		p.cond.Broadcast()

		p.mutex.Lock()
		for p.getp == chanHasRecv && p.seq == seq && !p.close {
			p.cond.Wait(&p.mutex)
		}
		recvOK = p.seq != seq || p.getp != chanHasRecv
		if !recvOK {
			zeroChanRecv(v, eltSize)
		}
		tryOK = true
		p.mutex.Unlock()
		return
	} else {
		if p.len == 0 {
			tryOK = p.close
			if p.close {
				zeroChanRecv(v, eltSize)
			}
			p.mutex.Unlock()
			return
		}
		if v != nil {
			c.Memcpy(v, c.Advance(p.data, p.getp*eltSize), uintptr(eltSize))
		}
		p.getp = (p.getp + 1) % n
		p.len--
	}
	notifyOps(p)
	p.mutex.Unlock()
	p.cond.Broadcast()
	recvOK, tryOK = true, true
	return
}

func ChanRecv(p *Chan, v unsafe.Pointer, eltSize int) (recvOK bool) {
	if p == nil {
		blockForever()
		return false
	}
	n := p.cap
	p.mutex.Lock()
	if n == 0 {
		for p.getp == chanHasRecv && !p.close {
			p.cond.Wait(&p.mutex)
		}
		if p.close {
			zeroChanRecv(v, eltSize)
			p.mutex.Unlock()
			return false
		}
		p.getp = chanHasRecv
		p.data = v
		seq := p.seq
		notifyOps(p)
		p.mutex.Unlock()
		p.cond.Broadcast()

		p.mutex.Lock()
		for p.getp == chanHasRecv && p.seq == seq && !p.close {
			p.cond.Wait(&p.mutex)
		}
		recvOK = p.seq != seq || p.getp != chanHasRecv
		if !recvOK {
			zeroChanRecv(v, eltSize)
		}
		p.mutex.Unlock()
		return
	} else {
		for p.len == 0 {
			if p.close {
				zeroChanRecv(v, eltSize)
				p.mutex.Unlock()
				return false
			}
			p.cond.Wait(&p.mutex)
		}
		if v != nil {
			c.Memcpy(v, c.Advance(p.data, p.getp*eltSize), uintptr(eltSize))
		}
		p.getp = (p.getp + 1) % n
		p.len--
	}
	notifyOps(p)
	p.mutex.Unlock()
	p.cond.Broadcast()
	recvOK = true
	return
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

type selectOp struct {
	mutex sync.Mutex
	cond  sync.Cond
	sem   bool
}

func (p *selectOp) init() {
	p.mutex.Init(nil)
	p.cond.Init(nil)
	p.sem = false
}

func (p *selectOp) end() {
	p.mutex.Destroy()
	p.cond.Destroy()
}

func (p *selectOp) notify() {
	p.mutex.Lock()
	p.sem = true
	p.mutex.Unlock()
	p.cond.Signal()
}

func (p *selectOp) wait() {
	p.mutex.Lock()
	if !p.sem {
		p.cond.Wait(&p.mutex)
	}
	p.sem = false
	p.mutex.Unlock()
}

// ChanOp represents a channel operation.
type ChanOp struct {
	C *Chan

	Val  unsafe.Pointer
	Size int32

	Send bool
}

// TrySelect executes a non-blocking select operation.
func TrySelect(ops ...ChanOp) (isel int, recvOK, tryOK bool) {
	for isel = range ops {
		op := ops[isel]
		if op.C == nil {
			// Nil-channel select cases are permanently disabled.
			continue
		}
		if op.Send {
			if tryOK = ChanTrySend(op.C, op.Val, int(op.Size)); tryOK {
				return
			}
		} else {
			if recvOK, tryOK = ChanTryRecv(op.C, op.Val, int(op.Size)); tryOK {
				return
			}
		}
	}
	return
}

// Select executes a blocking select operation.
func Select(ops ...ChanOp) (isel int, recvOK bool) {
	selOp := (*selectOp)(c.Alloca(unsafe.Sizeof(selectOp{})))
	selOp.init()
	sendFirst := selectSendFirst(ops)
	for _, op := range ops {
		if op.C == nil {
			continue
		}
		prepareSelect(op.C, selOp, op.Send)
	}
	var tryOK bool
	for {
		if isel, recvOK, tryOK = trySelect(ops, sendFirst); tryOK {
			break
		}
		selOp.wait()
	}
	for _, op := range ops {
		if op.C == nil {
			continue
		}
		endSelect(op.C, selOp, op.Send)
	}
	selOp.end()
	return
}

func trySelect(ops []ChanOp, sendFirst bool) (isel int, recvOK, tryOK bool) {
	// Split probing by direction. If sends are probed first, the recv phase must
	// not accept select-only senders, because this select's own sends already
	// failed to commit to a peer. If recvs are probed first, they may accept
	// select-senders because our own sends have not been attempted yet.
	if sendFirst {
		if isel, recvOK, tryOK = trySelectDir(ops, true, false, nil); tryOK {
			return
		}
		return trySelectDir(ops, false, false, ops)
	}
	if isel, recvOK, tryOK = trySelectDir(ops, false, true, ops); tryOK {
		return
	}
	return trySelectDir(ops, true, true, nil)
}

func trySelectDir(ops []ChanOp, send bool, acceptSelectSend bool, allOps []ChanOp) (isel int, recvOK, tryOK bool) {
	for isel = range ops {
		op := ops[isel]
		if op.C == nil || op.Send != send {
			continue
		}
		if op.Send {
			if tryOK = ChanTrySend(op.C, op.Val, int(op.Size)); tryOK {
				return
			}
			continue
		}
		accept := acceptSelectSend
		if accept && selectHasSend(allOps, op.C) {
			accept = false
		}
		if recvOK, tryOK = chanTryRecv(op.C, op.Val, int(op.Size), accept); tryOK {
			return
		}
	}
	return
}

func selectHasSend(ops []ChanOp, ch *Chan) bool {
	for _, op := range ops {
		if op.C == ch && op.Send {
			return true
		}
	}
	return false
}

func selectSendFirst(ops []ChanOp) bool {
	const maxUintptr = ^uintptr(0)
	minRecv := maxUintptr
	minSend := maxUintptr
	for _, op := range ops {
		if op.C == nil {
			continue
		}
		addr := uintptr(unsafe.Pointer(op.C))
		if op.Send {
			if addr < minSend {
				minSend = addr
			}
		} else if addr < minRecv {
			minRecv = addr
		}
	}
	if minSend == maxUintptr {
		return false
	}
	if minRecv == maxUintptr {
		return true
	}
	// Deterministically break mirrored select ties by channel address. Peers
	// operating on the same channel pair with swapped directions then probe in
	// opposite orders, allowing at least one side to make progress.
	return minSend < minRecv
}

func prepareSelect(c *Chan, selOp *selectOp, isSend bool) {
	c.mutex.Lock()
	// Unbuffered channel select support:
	// ChanTryRecv uses c.sends to decide whether a recv can proceed (without
	// blocking forever). If both sides use select, ChanTrySend/ChanTryRecv won't
	// see each other unless select-send contributes to c.sends.
	//
	// This is intentionally minimal and targets common stdlib patterns (e.g.
	// net.Pipe) to avoid deadlocks.
	if c.cap == 0 && isSend {
		c.sends++
		c.selsends++
	}
	if c.sop == nil {
		c.sop = selOp
	} else {
		c.sops = append(c.sops, selOp)
	}
	if c.cap == 0 && isSend {
		// A newly-registered select-send can make a select-recv runnable.
		notifyOps(c)
	}
	c.mutex.Unlock()
}

func endSelect(c *Chan, selOp *selectOp, isSend bool) {
	c.mutex.Lock()
	if c.cap == 0 && isSend {
		c.sends--
		c.selsends--
	}
	if c.sop == selOp {
		c.sop = nil
		c.mutex.Unlock()
		return
	}
	for i, op := range c.sops {
		if op == selOp {
			copy(c.sops[i:], c.sops[i+1:])
			c.sops[len(c.sops)-1] = nil
			c.sops = c.sops[:len(c.sops)-1]
			break
		}
	}
	c.mutex.Unlock()
}

// -----------------------------------------------------------------------------
