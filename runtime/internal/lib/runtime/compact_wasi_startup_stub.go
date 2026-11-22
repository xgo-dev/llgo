//go:build !llgo || !wasip1 || !llgo.wasi.mutatorselect

package runtime

func usePinnedProcess() bool { return false }

const wasiPinnedProcessFiles = ""

var pinnedProcessArgs []string
