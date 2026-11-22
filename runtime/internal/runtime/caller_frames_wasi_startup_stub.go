//go:build !llgo || !wasip1 || !llgo.wasi.mutatorselect

package runtime

const wasiCallerFrames = true
