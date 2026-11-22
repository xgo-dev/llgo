//go:build llgo && ((wasip1 && llgo.size.wasi && !llgo.wasi.mutatorselect) || (!wasip1 && (((darwin || linux) && llgo.size.native) || (linux && baremetal && llgo.baremetal.stdio && llgo.size.embedded))))

package syscall

func llgoUsePinnedProcess() bool                            { return false }
func llgoLookupPinnedEnvironment(key string) (string, bool) { return "", false }
