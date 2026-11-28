//go:build !windows

package meta

import (
	"os"
	"syscall"
)

func mapFile(f *os.File, size int) ([]byte, error) {
	return syscall.Mmap(int(f.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
}

func unmapFile(raw []byte) error {
	return syscall.Munmap(raw)
}
