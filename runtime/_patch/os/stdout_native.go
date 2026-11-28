//go:build llgo && llgo.size.native && (darwin || (linux && !baremetal))

package os

import (
	"sync"
	"syscall"
)

const llgoStdoutNative = true

var llgoStdoutMu sync.Mutex

func llgoStdoutLock()   { llgoStdoutMu.Lock() }
func llgoStdoutUnlock() { llgoStdoutMu.Unlock() }

func llgoStdoutPollableAtStartup() bool
func llgoStdoutPoll()

// Capture the same inherited descriptor state as NewFile, before application
// initialization. The linker enables this snapshot only for descriptor writes.
var llgoSkipStdoutPollInit bool
var llgoWaitStdout = llgoInitStdoutPoll()

func llgoInitStdoutPoll() bool {
	if llgoSkipStdoutPollInit {
		return false
	}
	return llgoStdoutPollableAtStartup()
}

func llgoStdoutWaitForSpace() bool {
	if !llgoWaitStdout {
		return false
	}
	llgoStdoutPoll()
	return true
}

func llgoStdoutCheckError(err error) {
	if err == syscall.EPIPE {
		sigpipe()
	}
}
