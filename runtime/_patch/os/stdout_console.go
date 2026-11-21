//go:build llgo && ((wasip1 && llgo.size.wasi) || (linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded))

package os

import (
	"internal/poll"
	"io"
	"strconv"
	"sync"
	"syscall"
	"unsafe"
)

// A closed link may use this wrapper when no code observes the standard
// File globals and the target uses synchronous console I/O.
// Keeping File.WriteString preserves write serialization and error handling.
var llgoClosedStdout = File{&file{
	pfd:         poll.FD{Sysfd: 1, IsStream: true, ZeroReadIsEOF: true},
	name:        "/dev/stdout",
	stdoutOrErr: true,
}}

func llgoWriteClosedStdout(s string) (int, error) {
	return llgoClosedStdout.WriteString(s)
}

var llgoStdoutMu sync.Mutex

const llgoStdoutMaxWrite = 1 << 30

// Only the closed executable's literal output calls can reach this writer.
// The standard File wrappers are unobserved and the target never registers
// these descriptors with a poller. Serialize writes and preserve the stream
// limits, retry, partial write, and error behavior of File.WriteString.
func llgoWriteStdoutDescriptor(s string) (int, error) {
	data := unsafe.Slice(unsafe.StringData(s), len(s))
	llgoStdoutMu.Lock()
	written, err, invalidCount := llgoWriteStdoutLocked(data)
	llgoStdoutMu.Unlock()
	if invalidCount != 0 {
		size := len(data) - written
		if size > llgoStdoutMaxWrite {
			size = llgoStdoutMaxWrite
		}
		panic("invalid return from write: got " + strconv.Itoa(invalidCount) + " from a write of " + strconv.Itoa(size))
	}
	if err != nil {
		return written, &PathError{Op: "write", Path: "/dev/stdout", Err: err}
	}
	return written, nil
}

// The write path uses synchronous console syscalls. Report invalid host
// results to the caller so it can release the lock before panicking.
func llgoWriteStdoutLocked(data []byte) (int, error, int) {
	var written int
	for {
		p := data[written:]
		if len(p) > llgoStdoutMaxWrite {
			p = p[:llgoStdoutMaxWrite]
		}
		n, err := syscall.Write(1, p)
		if err == syscall.EINTR {
			continue
		}
		if n > 0 {
			if n > len(p) {
				return written, nil, n
			}
			written += n
		}
		if written == len(data) && err == nil {
			return written, nil, 0
		}
		if err == nil && n == 0 {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			return written, err, 0
		}
	}
}
