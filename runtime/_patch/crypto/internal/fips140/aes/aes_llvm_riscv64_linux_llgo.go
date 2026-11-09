//go:build llgo && riscv64 && linux && !baremetal && go1.27 && !go1.28

package aes

import _ "unsafe"

const llgoAESName = "RISC-V Crypto"
const llgoAESPreKey = false

func llgoAESHardware() bool {
	// Linux riscv_hwprobe intersects capabilities over online CPUs when
	// cpu_count is zero. Older kernels and unknown keys safely leave AES off.
	probe := struct {
		key   int64
		value uint64
	}{key: 4} // RISCV_HWPROBE_KEY_IMA_EXT_0
	const aes = uint64(1<<11 | 1<<12) // Zknd and Zkne
	return llgoAESHWProbe(258, &probe, 1, 0, nil, 0, 0) == 0 && probe.key == 4 && probe.value&aes == aes
}

//go:linkname llgoAESHWProbe C.syscall
func llgoAESHWProbe(uintptr, *struct {
	key   int64
	value uint64
}, uintptr, uintptr, *uintptr, uintptr, uintptr) int64
