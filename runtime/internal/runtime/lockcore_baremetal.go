//go:build llgo && baremetal && tinygo.riscv32 && !llgo.idf

package runtime

// LockCore on the RV32 non-IDF baremetal targets always succeeds: the fiber
// scheduler is single-hart and the target brings no second scheduler core,
// so the running context is always on the only core there is - the request
// is trivially satisfied whatever id asks for, and there is no second core
// to be wrong about. (The IDF kernel targets, where a real pin exists, build
// lockcore_idf.go instead.)
func LockCore(id int) error {
	return nil
}
