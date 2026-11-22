//go:build llgo && wasip1 && llgo.wasi.mutatorselect

package runtime

import _ "unsafe"

//go:linkname wasiCallerFrames C.__llgo_wasi_ordinary_startup
var wasiCallerFrames bool
