package runtime

import (
	"unsafe"

	"github.com/xgo-dev/llgo/runtime/internal/sync/atomic"
)

var synctestRunning uint32

// SynctestActive keeps ordinary semaphore releases on their native path again
// after the last bubble finishes, including concurrently running bubbles.
func SynctestActive() bool { return atomic.Load(&synctestRunning) != 0 }

// SynctestBubble implements the Go 1.27 internal/synctest contract on LLGo's
// native threads and cooperative fibers. Only durable object waits reduce
// running; mutexes, syscalls and I/O deliberately remain active. A ready
// operation restores the count before publishing the wake, under the object's
// own lock, so fake time cannot advance between a wake and its G resuming.
type SynctestBubble struct {
	parked                bool
	event                 chanSignal // single controller, with a portable lock/wait backend
	total, running        int
	main                  *g
	done, closed, waiting bool
	waiter                *synctestWait
	now                   int64
	timers                map[*SynctestTimer]bool
	refs                  int
}

type synctestWait struct {
	signal chanSignal
	gp     *g
	ready  bool
}

func CurrentSynctestBubble() *SynctestBubble {
	// Worker scheduler timer callbacks run without a current G. They use the
	// ordinary clock; reading through a nil G also reads linear memory on Wasm.
	if gp := getg(); gp != nil {
		return gp.bubble
	}
	return nil
}

func inheritSynctest(gp, parent *g) {
	if parent == nil || parent.bubble == nil {
		return
	}
	b := parent.bubble
	b.event.lock()
	gp.bubble = b
	b.total++
	b.running++
	b.event.unlock()
}

func leaveSynctest(gp *g) {
	b := gp.bubble
	if b == nil {
		return
	}
	b.event.lock()
	b.total--
	if !gp.bubbleBlocked {
		b.running--
	}
	if b.main == gp {
		b.done = true
	}
	gp.bubble = nil
	gp.bubbleBlocked = false
	b.notifyLocked()
	b.event.unlock()
}

// SynctestWaiter records a potentially durable wait, not a scheduler park.
// Block and Wake must be protected by the synchronization object's lock.
type SynctestWaiter struct{ gp *g }

func NewSynctestWaiter() *SynctestWaiter {
	gp := getg()
	if gp == nil || gp.bubble == nil {
		return nil
	}
	return &SynctestWaiter{gp: gp}
}

func (w *SynctestWaiter) Block() {
	if w == nil {
		return
	}
	b := w.gp.bubble
	b.event.lock()
	if !w.gp.bubbleBlocked {
		w.gp.bubbleBlocked = true
		b.running--
		b.notifyLocked()
	}
	b.event.unlock()
}

// CheckWake validates ownership before a multi-waiter operation changes any
// accounting or publishes a wake. The object's lock must still be held.
func (w *SynctestWaiter) CheckWake() { w.wake(false) }

func (w *SynctestWaiter) Wake() { w.wake(true) }

func (w *SynctestWaiter) wake(ready bool) {
	if w == nil {
		return
	}
	b := w.gp.bubble
	b.event.lock()
	defer b.event.unlock()
	if w.gp.bubbleBlocked {
		if b != CurrentSynctestBubble() {
			panic("synctest: wake from outside bubble")
		}
		if ready {
			w.gp.bubbleBlocked = false
			b.running++
		}
	}
}

func SynctestRun(f func()) {
	if CurrentSynctestBubble() != nil {
		panic("synctest.Run called from within a synctest bubble")
	}
	b := &SynctestBubble{total: 1, running: 1, now: 946684800000000000}
	b.event.init()
	atomic.Add(&synctestRunning, 1)
	completed := false
	defer func() {
		clearSynctestAssociations(b)
		if completed {
			b.event.destroy()
		}
		atomic.Sub(&synctestRunning, 1)
	}()
	// Reserve root membership before spawning. Otherwise the controller could
	// observe an empty bubble before the new native thread/fiber starts.
	go func() {
		gp := getg()
		b.event.lock()
		gp.bubble = b
		b.main = gp
		b.event.unlock()
		defer leaveSynctest(gp)
		f()
	}()
	b.event.lock()
	for {
		for b.running != 0 || b.refs != 0 {
			b.parked = true
			b.event.park()
			b.parked = false
		}
		if b.total == 0 {
			b.closed = true
			completed = true
			b.event.unlock()
			return
		}
		var first *SynctestTimer
		for timer := range b.timers {
			if first == nil || timer.when < first.when {
				first = timer
			}
		}
		// Timers already due run before Wait returns. Future timers advance
		// only if nobody is waiting and the bubble's main has not exited.
		if first != nil && (first.when <= b.now || (b.waiter == nil && !b.done)) {
			if first.when > b.now {
				b.now = first.when
			}
			now, when, callback := b.now, first.when, first.callback
			if first.period > 0 {
				first.when += first.period * (1 + (now-first.when)/first.period)
				if first.when < 0 {
					first.when = int64(1<<63 - 1)
				}
			} else {
				delete(b.timers, first)
			}
			b.running++ // the timer dispatcher is active until callback ends
			b.event.unlock()
			func() {
				gp := getg()
				gp.bubble = b
				defer func() {
					gp.bubble = nil
					b.event.lock()
					b.running--
					b.notifyLocked()
					b.event.unlock()
				}()
				callback(now - when)
			}()
			b.event.lock()
			continue
		}
		if w := b.waiter; w != nil {
			b.running++
			w.gp.bubbleBlocked = false
			b.waiter = nil
			b.event.unlock()
			w.signal.lock()
			w.ready = true
			w.signal.ready()
			w.signal.unlock()
			b.event.lock()
			continue
		}
		b.event.unlock()
		if b.done {
			panic(errorString("deadlock: main bubble goroutine has exited but blocked goroutines remain"))
		}
		panic(errorString("deadlock: all goroutines in bubble are blocked"))
	}
}

func SynctestWait() {
	gp := getg()
	b := CurrentSynctestBubble()
	if b == nil {
		panic("goroutine is not in a bubble")
	}
	w := &synctestWait{gp: gp}
	w.signal.init()
	w.signal.lock()
	b.event.lock()
	if b.waiting {
		b.event.unlock()
		w.signal.unlock()
		w.signal.destroy()
		panic("wait already in progress")
	}
	b.waiting = true
	b.waiter = w
	gp.bubbleBlocked = true
	b.running--
	b.notifyLocked()
	b.event.unlock()
	for !w.ready {
		w.signal.park()
	}
	w.signal.unlock()
	w.signal.destroy()
	b.event.lock()
	b.waiting = false
	b.event.unlock()
}

func SynctestAcquire() *SynctestBubble {
	b := CurrentSynctestBubble()
	if b == nil {
		return nil
	}
	b.event.lock()
	b.refs++
	b.event.unlock()
	return b
}

func SynctestRelease(b *SynctestBubble) {
	if b == nil {
		return
	}
	b.event.lock()
	if b.refs == 0 {
		b.event.unlock()
		panic("synctest: release without acquire")
	}
	b.refs--
	b.notifyLocked()
	b.event.unlock()
}

func SynctestInBubble(b *SynctestBubble, f func()) {
	if b == nil {
		f()
		return
	}
	gp := getg()
	if gp.bubble != nil {
		panic("synctest: callback entered a different bubble")
	}
	b.event.lock()
	if b.closed {
		b.event.unlock()
		panic("synctest: callback entered a finished bubble")
	}
	gp.bubble = b
	b.total++
	b.running++
	b.event.unlock()
	defer leaveSynctest(gp)
	f()
}

func (b *SynctestBubble) Now() int64 {
	b.event.lock()
	now := b.now
	b.event.unlock()
	return now
}

// A fake timer belongs to its creation bubble even after Stop. Callback
// execution occurs on the bubble controller and never on a wall-clock worker.
type SynctestTimer struct {
	bubble       *SynctestBubble
	when, period int64
	callback     func(delay int64)
}

func NewSynctestTimer(when, period int64, callback func(int64)) *SynctestTimer {
	b := CurrentSynctestBubble()
	t := &SynctestTimer{bubble: b}
	b.event.lock()
	if b.timers == nil {
		b.timers = make(map[*SynctestTimer]bool)
	}
	t.when, t.period, t.callback = when, period, callback
	b.timers[t] = true
	b.event.unlock()
	return t
}

func (t *SynctestTimer) Check() {
	if CurrentSynctestBubble() != t.bubble {
		panic("timer operation from outside synctest bubble")
	}
}

func (t *SynctestTimer) Stop() bool {
	t.Check()
	b := t.bubble
	b.event.lock()
	active := b.timers[t]
	delete(b.timers, t)
	b.event.unlock()
	return active
}

func (t *SynctestTimer) Reset(when, period int64, callback func(int64)) bool {
	t.Check()
	b := t.bubble
	b.event.lock()
	active := b.timers[t]
	t.when, t.period, t.callback = when, period, callback
	b.timers[t] = true
	b.event.unlock()
	return active
}

var synctestAssociations struct {
	once   chanMutex
	values map[unsafe.Pointer]*SynctestBubble
}

// init before other packages can run concurrently, without importing sync
// back into the scheduler. Entries are removed when a WaitGroup is reset and
// at bubble teardown, so the side table cannot keep completed bubbles alive.
func init() { synctestAssociations.once.init() }

//go:linkname synctestCanAssociate C.llgo_synctest_can_associate
func synctestCanAssociate(unsafe.Pointer) bool

func SynctestAssociate(p unsafe.Pointer) int {
	if !synctestCanAssociate(p) {
		return 0
	}
	b := CurrentSynctestBubble()
	synctestAssociations.once.Lock()
	defer synctestAssociations.once.Unlock()
	if old := synctestAssociations.values[p]; old != nil {
		if old == b {
			return 1
		}
		return 2
	}
	if b == nil {
		return 0
	}
	if synctestAssociations.values == nil {
		synctestAssociations.values = make(map[unsafe.Pointer]*SynctestBubble)
	}
	synctestAssociations.values[p] = b
	return 1
}

func SynctestDisassociate(p unsafe.Pointer) {
	synctestAssociations.once.Lock()
	delete(synctestAssociations.values, p)
	synctestAssociations.once.Unlock()
}

func SynctestIsAssociated(p unsafe.Pointer) bool {
	synctestAssociations.once.Lock()
	b := synctestAssociations.values[p]
	synctestAssociations.once.Unlock()
	return b != nil && b == CurrentSynctestBubble()
}

func clearSynctestAssociations(b *SynctestBubble) {
	synctestAssociations.once.Lock()
	for p, owner := range synctestAssociations.values {
		if owner == b {
			delete(synctestAssociations.values, p)
		}
	}
	synctestAssociations.once.Unlock()
}

// SynctestOutside initializes runtime services without inheriting test object
// or G membership. The calling G remains active while its init handshake runs.
func SynctestOutside(f func()) {
	gp := getg()
	if gp == nil {
		f()
		return
	}
	b := gp.bubble
	gp.bubble = nil
	defer func() { gp.bubble = b }()
	f()
}

func (b *SynctestBubble) notifyLocked() {
	if b.parked {
		b.parked = false
		b.event.ready()
	}
}
