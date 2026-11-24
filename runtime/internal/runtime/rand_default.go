//go:build !windows && (!llgo || !baremetal || !llgo.embedded.startup)

package runtime

import _ "unsafe"

//go:linkname fastrand C.rand
func fastrand() uint32
