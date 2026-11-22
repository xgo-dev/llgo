//go:build !llgo || !wasip1 || !llgo.wasi.mutatorselect

package tinygogc

func requireManagedStartup() {}
