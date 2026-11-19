//go:build llgo && linux && baremetal && llgo.baremetal.stdio && go1.27

package os

import "internal/poll"

// The bare-metal syscall provider exposes the three console descriptors.
//
//llgo:skip newFileFromNewFile
func newFileFromNewFile(fd uintptr, name string) *File {
	if fd > 2 {
		return nil
	}
	return &File{&file{
		pfd:         poll.FD{Sysfd: int(fd), IsStream: true, ZeroReadIsEOF: true},
		name:        name,
		stdoutOrErr: fd == 1 || fd == 2,
	}}
}
