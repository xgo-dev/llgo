//go:build windows

package meta

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func mapFile(f *os.File, size int) ([]byte, error) {
	mapping, err := windows.CreateFileMapping(
		windows.Handle(f.Fd()), nil, windows.PAGE_READONLY, 0, 0, nil,
	)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(mapping)

	addr, err := windows.MapViewOfFile(mapping, windows.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		return nil, err
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(addr)), size), nil
}

func unmapFile(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	return windows.UnmapViewOfFile(uintptr(unsafe.Pointer(&raw[0])))
}
