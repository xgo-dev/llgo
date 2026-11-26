//go:build baremetal && !(llgo && tinygo.riscv32 && !nogc)

package runtime

import (
	c "github.com/xgo-dev/llgo/runtime/internal/clite"
)

// goexitBaremetal keeps the single-context bare-metal behavior (the pthread
// single-context backends, including the nogc riscv32 builds that retain
// them): Goexit drains every defer through the longjmp path, then retains the
// fatal termination because there is no scheduler to return to.
func goexitBaremetal(gp *g) {
	_ = gp
	c.Printf(c.Str("fatal error\n"))
	c.Exit(2)
}
