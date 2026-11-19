//go:build llgo && baremetal && linux && arm && llgo.baremetal.stdio && go1.27

package syscall

//llgo:skip seek
func seek(fd int, _ int64, _ int) (int64, Errno) {
	if fd >= 0 && fd <= 2 {
		return 0, ESPIPE
	}
	return 0, EBADF
}
