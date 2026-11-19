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
