//go:build llgo && wasip1 && llgo.size.wasi

package os

import (
	"internal/poll"
	"io"
	"strconv"
	"sync"
	"syscall"
	"unsafe"
)

// A closed WASI link may use this wrapper when no code observes the standard
// File globals. WASI has no LLGo poll registration for these descriptors.
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

// Only the closed executable's literal output calls can reach this writer.
// Descriptor 1 cannot be closed through its Go File, and the WASI runtime
// never registers it with a poller. Serialize writes and preserve the stream
// limits, retry, partial write, and error behavior of File.WriteString.
func llgoWriteStdoutDescriptor(s string) (int, error) {
	const maxWrite = 1 << 30
	llgoStdoutMu.Lock()
	defer llgoStdoutMu.Unlock()
	data := unsafe.Slice(unsafe.StringData(s), len(s))
	var written int
	for {
		p := data[written:]
		if len(p) > maxWrite {
			p = p[:maxWrite]
		}
		n, err := syscall.Write(syscall.Stdout, p)
		if err == syscall.EINTR {
			continue
		}
		if n > 0 {
			if n > len(p) {
				panic("invalid return from write: got " + strconv.Itoa(n) + " from a write of " + strconv.Itoa(len(p)))
			}
			written += n
		}
		if written == len(data) && err == nil {
			return written, nil
		}
		if err == nil && n == 0 {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			return written, &PathError{Op: "write", Path: "/dev/stdout", Err: err}
		}
	}
}
