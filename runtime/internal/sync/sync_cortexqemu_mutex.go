//go:build baremetal && llgo.singlecontext

package sync

import c "github.com/xgo-dev/llgo/runtime/internal/clite"

// The Cortex-M QEMU runtime executes Go on a single context. Keep lock
// state to catch reentrant runtime calls and preserve TryLock behavior.
func pthreadMutexInit(m *Mutex, _ *MutexAttr) c.Int {
	m.unused[0] = 0
	return 0
}

func pthreadMutexDestroy(m *Mutex) c.Int {
	return pthreadMutexInit(m, nil)
}

func pthreadMutexLock(m *Mutex) c.Int {
	if pthreadMutexTryLock(m) != 0 {
		c.Fputs(c.Str("mutex deadlock on the single-context Cortex-M runtime\n"), c.Stderr)
		c.Exit(1)
	}
	return 0
}

func pthreadMutexTryLock(m *Mutex) c.Int {
	if m.unused[0] != 0 {
		return 1
	}
	m.unused[0] = 1
	return 0
}

func pthreadMutexUnlock(m *Mutex) c.Int {
	m.unused[0] = 0
	return 0
}

func (o *Once) Do(f OnceFunc) c.Int {
	if o.unused[0] == 0 {
		o.unused[0] = 1
		f()
	}
	return 0
}

func pthreadCondInit(_ *Cond, _ *CondAttr) c.Int { return 0 }
func pthreadCondDestroy(_ *Cond) c.Int           { return 0 }
func pthreadCondSignal(_ *Cond) c.Int            { return 0 }
func pthreadCondBroadcast(_ *Cond) c.Int         { return 0 }

func pthreadCondWait(_ *Cond, _ *Mutex) c.Int {
	// There is no second execution context to wake the waiter.
	c.Fputs(c.Str("condition wait on the single-context Cortex-M runtime\n"), c.Stderr)
	c.Exit(1)
	return 0
}
