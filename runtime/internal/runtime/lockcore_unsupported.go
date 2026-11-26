//go:build llgo && baremetal && !tinygo.riscv32

package runtime

// LockCore has no implementation on non-RISC-V baremetal targets: the ESP32
// classic (xtensa) chip has two cores and no pinning facility is wired up
// for it, so claiming "locked" would be a lie. The error keeps hosted
// semantics - an explicit refusal, never a silent pretend-pin.
func LockCore(id int) error {
	return errorString("runtime.LockCore: not implemented on this target")
}
