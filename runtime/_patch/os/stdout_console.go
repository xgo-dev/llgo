//go:build llgo && ((wasip1 && llgo.size.wasi) || (linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded) || ((darwin || (linux && !baremetal)) && llgo.size.native))

package os

import (
	"io"
	"strconv"
	"syscall"
	"unsafe"
)

const llgoStdoutMaxWrite = 1 << 30

// Only the closed executable's specialized output calls can reach this writer.
// The standard File wrappers are unobserved. Serialize writes and preserve
// File.WriteString's stream limits, partial writes and errors.
func llgoWriteStdoutDescriptor(s string) (int, error) {
	return llgoWriteStdoutDescriptorBytes(unsafe.Slice(unsafe.StringData(s), len(s)))
}

func llgoWriteStdoutDescriptorBytes(data []byte) (int, error) {
	llgoStdoutLock()
	written, err, invalidCount := llgoWriteStdoutLocked(data)
	llgoStdoutUnlock()
	if invalidCount != 0 {
		size := len(data) - written
		if size > llgoStdoutMaxWrite {
			size = llgoStdoutMaxWrite
		}
		panic("invalid return from write: got " + strconv.Itoa(invalidCount) + " from a write of " + strconv.Itoa(size))
	}
	if err != nil {
		if llgoStdoutNative {
			llgoStdoutCheckError(err)
		}
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
		if llgoStdoutNative && err == syscall.EAGAIN && llgoStdoutWaitForSpace() {
			continue
		}
		if err == nil && n == 0 {
			err = io.ErrUnexpectedEOF
		}
		if err != nil {
			return written, err, 0
		}
	}
}
