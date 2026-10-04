//go:build llgo

package dep1

// Recover stores the panic value seen by a directly deferred dependency call.
func Recover(dst *any) {
	*dst = recover()
}
