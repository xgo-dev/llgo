//go:build llgo && ((wasip1 && llgo.size.wasi) || (linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded))

package os

import "internal/poll"

const llgoStdoutNative = false

// With synchronous console I/O and unobserved standard file wrappers, use
// the regular File writer without constructing and registering the wrapper.
var llgoClosedStdout = File{&file{
	pfd:         poll.FD{Sysfd: 1, IsStream: true, ZeroReadIsEOF: true},
	name:        "/dev/stdout",
	stdoutOrErr: true,
}}

func llgoWriteClosedStdout(s string) (int, error) {
	return llgoClosedStdout.WriteString(s)
}

func llgoStdoutWaitForSpace() bool { return false }
func llgoStdoutCheckError(error)   {}
