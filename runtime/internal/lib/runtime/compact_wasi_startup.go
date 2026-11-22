//go:build llgo && wasip1 && llgo.wasi.mutatorselect

package runtime

import "unsafe"

//go:linkname wasiUseOrdinaryStartup C.__llgo_wasi_ordinary_startup
var wasiUseOrdinaryStartup bool

func usePinnedProcess() bool { return !wasiUseOrdinaryStartup }

const wasiPinnedProcessFiles = "; _wrap/wasi_compact_startup.c"

//go:linkname pinnedProcessArgs C.llgo_pinned_argv
var pinnedProcessArgs []string

//go:linkname pinnedEnvironmentLookup C.llgo_pinned_lookup
//go:noescape
func pinnedEnvironmentLookup(key *byte, size uint32) *string

//go:linkname syscallPinnedLookup syscall.llgoLookupPinnedEnvironment
func syscallPinnedLookup(key string) (string, bool) {
	if value := pinnedEnvironmentLookup(unsafe.StringData(key), uint32(len(key))); value != nil {
		return *value, true
	}
	return "", false
}
