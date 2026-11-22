//go:build llgo && baremetal && linux && arm && llgo.baremetal.stdio && llgo.size.embedded

package syscall

// The bare-metal dispatcher implements read and write only. Resource limits
// cannot be queried, and the result structure remains unchanged.
//
//llgo:skip Getrlimit
func Getrlimit(_ int, _ *Rlimit) error { return ENOSYS }
