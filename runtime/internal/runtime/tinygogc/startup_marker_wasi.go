//go:build llgo && wasip1 && llgo.wasi.mutatorselect

package tinygogc

import _ "unsafe"

//go:linkname requireManagedStartup C.__llgo_wasi_requires_managed_startup
func requireManagedStartup()
