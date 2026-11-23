//go:build llgo && linux && baremetal && llgo.baremetal.stdio

package os

import "internal/poll"

func llgoConsoleFile(fd uintptr, name string) *File {
	if fd > 2 {
		return nil
	}
	return &File{&file{
		pfd:         poll.FD{Sysfd: int(fd), IsStream: true, ZeroReadIsEOF: true},
		name:        name,
		stdoutOrErr: fd == 1 || fd == 2,
	}}
}
