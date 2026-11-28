//go:build llgo && linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded

package os

import "sync"

var llgoStdoutMu sync.Mutex

func llgoStdoutLock()   { llgoStdoutMu.Lock() }
func llgoStdoutUnlock() { llgoStdoutMu.Unlock() }
