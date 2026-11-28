//go:build windows

package syscall

type Errno uintptr

type Signal int

type Timespec struct {
	Sec  int64
	Nsec int64
}

type Timeval struct {
	Sec  int32
	Usec int32
}
