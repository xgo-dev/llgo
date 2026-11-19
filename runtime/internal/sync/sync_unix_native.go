//go:build !windows && !(cortexm && qemu && lm3s6965)

package sync

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
	_ "unsafe"
)

//go:linkname pthreadMutexInit C.pthread_mutex_init
func pthreadMutexInit(m *Mutex, attr *MutexAttr) c.Int

//go:linkname pthreadMutexDestroy C.pthread_mutex_destroy
func pthreadMutexDestroy(m *Mutex) c.Int

//go:linkname pthreadMutexLock C.pthread_mutex_lock
func pthreadMutexLock(m *Mutex) c.Int

//go:linkname pthreadMutexTryLock C.pthread_mutex_trylock
func pthreadMutexTryLock(m *Mutex) c.Int

//go:linkname pthreadMutexUnlock C.pthread_mutex_unlock
func pthreadMutexUnlock(m *Mutex) c.Int

// llgo:link (*Once).Do C.pthread_once
func (o *Once) Do(f OnceFunc) c.Int { return 0 }

//go:linkname pthreadCondInit C.pthread_cond_init
func pthreadCondInit(cond *Cond, attr *CondAttr) c.Int

//go:linkname pthreadCondDestroy C.pthread_cond_destroy
func pthreadCondDestroy(cond *Cond) c.Int

//go:linkname pthreadCondSignal C.pthread_cond_signal
func pthreadCondSignal(cond *Cond) c.Int

//go:linkname pthreadCondBroadcast C.pthread_cond_broadcast
func pthreadCondBroadcast(cond *Cond) c.Int

//go:linkname pthreadCondWait C.pthread_cond_wait
func pthreadCondWait(cond *Cond, m *Mutex) c.Int
