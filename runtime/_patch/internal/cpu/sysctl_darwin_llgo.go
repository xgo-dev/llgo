//go:build darwin && !ios && !baremetal

package cpu

import "unsafe"

// Keep CPU detection usable in programs that do not import the public runtime
// package. These hooks belong to internal/cpu's own initialization path.
func sysctlbynameInt32(name []byte) (int32, int32) {
	var value int32
	size := unsafe.Sizeof(value)
	status := llgoSysctlbyname(&name[0], unsafe.Pointer(&value), &size, nil, 0)
	return status, value
}

func sysctlbynameBytes(name, out []byte) int32 {
	size := uintptr(len(out))
	return llgoSysctlbyname(&name[0], unsafe.Pointer(&out[0]), &size, nil, 0)
}

// Older supported Go sources use this name for the same query.
func getsysctlbyname(name []byte) (int32, int32) {
	return sysctlbynameInt32(name)
}

//go:noescape
//go:linkname llgoSysctlbyname C.sysctlbyname
func llgoSysctlbyname(name *byte, out unsafe.Pointer, size *uintptr, new unsafe.Pointer, newSize uintptr) int32
