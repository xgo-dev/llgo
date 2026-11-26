//go:build baremetal && !(llgo && tinygo.riscv32 && !nogc)

package runtime

// Live-G bookkeeping for every baremetal configuration without the fiber
// scheduler: the pthread-style single-context backends and the nogc riscv32
// builds that keep that backend. Bare-metal has no atomic goroutine counter.
// Returning one remaining goroutine intentionally disables native pthread
// main-Goexit deadlock detection for the single-context backend.
func retainG() {
}

func releaseG() (remaining uint64, mainExited bool) {
	return 1, false
}

func markMainExited() {}

func gState() (count uint64, mainExited bool) {
	return 1, false
}
