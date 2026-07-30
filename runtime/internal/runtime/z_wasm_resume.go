//go:build llgo && wasm && llgo.wasm_resume && (js || wasip1) && !(wasip1 && llgo.wasi_threads)

package runtime

import (
	c "github.com/goplus/llgo/runtime/internal/clite"
	"github.com/goplus/llgo/runtime/internal/clite/debug"
)

var (
	printFormatPrefixInt  = c.Str("%lld")
	printFormatPrefixUInt = c.Str("%llu")
	printFormatPrefixHex  = c.Str("%llx")
)

// Rethrow transfers pending panic/Goexit processing to the scheduler's active
// native catch. The scheduler then redirects the explicit frame chain to the
// defer owner recorded by the compiler.
func Rethrow(link *Defer) {
	gp := getg()
	if gp.panic_ == nil && !gp.goexit {
		return
	}
	gp.defer_ = link
	if unwind := gp.context.platform.unwind; unwind != nil {
		c.Siglongjmp(unwind, 1)
		return
	}
	if ptr := gp.panic_; ptr != nil {
		TracePanic(*(*any)(ptr))
		if PanicTraceback == nil || !PanicTraceback(2) {
			debug.PrintStack(2)
		}
		c.Free(ptr)
		c.Exit(2)
		return
	}
	fatal("runtime: Goexit outside WebAssembly scheduler")
}
