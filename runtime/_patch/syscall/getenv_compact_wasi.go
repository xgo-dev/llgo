//go:build llgo && wasip1 && llgo.size.wasi && llgo.wasi.mutatorselect

package syscall

import _ "unsafe"

//go:linkname llgoWASIUseOrdinaryStartup C.__llgo_wasi_ordinary_startup
var llgoWASIUseOrdinaryStartup bool

func llgoUsePinnedProcess() bool { return !llgoWASIUseOrdinaryStartup }

func llgoLookupPinnedEnvironment(key string) (string, bool)
